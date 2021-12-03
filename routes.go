package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

//// ----- gin router handler functions -----

/// --- view routes ---
// Returns an array of the current view
func getView(c *gin.Context) {
	viewArr := view.GetViewAsSlice()
	// send list back in JSON form
	c.JSON(http.StatusOK, gin.H{"view": viewArr})
}

// Checks if the replica exists, and if not, adds it to the view
func putView(c *gin.Context) {
	// get the json data from the body
	data, err := parseKeysFromBody(c, "socket-address")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no address specified"})
		return
	}
	nodeAddress := data["socket-address"].(string)

	// add to view
	existed := view.PutView(nodeAddress)

	// respond to client
	if existed {
		c.JSON(http.StatusOK, gin.H{"result": "already present"})
	} else {
		c.JSON(http.StatusCreated, gin.H{"result": "added"})
	}
}

// Checks if the replica exists, and if so, delete it from the view
func deleteView(c *gin.Context) {
	view.Lock()
	defer view.Unlock()

	// get the json data from the body
	data, err := parseKeysFromBody(c, "socket-address")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no address specified"})
		return
	}
	nodeAddress := data["socket-address"].(string)

	// delete from view
	existed := view.DeleteView(nodeAddress)
	ring.RemoveNode(nodeAddress)

	// respond to client
	if existed {
		c.JSON(http.StatusOK, gin.H{"result": "deleted"})
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "View has no such replica"})
	}
}

/// --- kvs routes

// ---Key Value Store Functions---
// Retrieves the value for the specified key and returns it to the client
func getKey(c *gin.Context) {
	// get key from URL
	key := c.Param(("key"))

	shardId := ring.GetShardId(key)
	if shardId != localShardId {
		proxyToShard(c, "/kvs/"+key, shardId)
		return
	}

	// get the json data from the body
	data, err := parseKeysFromBody(c, "causal-metadata")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no address specified"})
		return
	}
	metadata := data["causal-metadata"].(map[string]int)

	// Make change in local kvs database and check for errors
	val, currMetadata, err := kvsDb.GetData(key, metadata)
	if err == ErrInvalidMetadata {
		sendServiceUnavailable(c)
		return
	} else if err == ErrKeyNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Key does not exist"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// send success to client
	c.JSON(http.StatusOK, gin.H{"result": "found", "value": val, "causal-metadata": currMetadata})
}

// Tries to add the kv pair to the kvs
func putKey(c *gin.Context) {
	// get key from URL
	key := c.Param(("key"))

	shardId := ring.GetShardId(key)
	if shardId != localShardId {
		proxyToShard(c, "/kvs/"+key, shardId)
		return
	}

	// get the json data from the body
	data, err := parseKeysFromBody(c, "value", "causal-metadata")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "causal-metadata or value not specified"})
		return
	}
	value := data["value"]
	metadata := data["causal-metadata"].(map[string]int)

	// check if key is under char limit
	if len(key) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Key is too long"})
		return
	}

	// check if value in nil
	if value == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PUT request does not specify a value"})
		return
	}

	// put key and check for errors
	wasCreated, currMetadata, err := kvsDb.PutData(key, value, metadata, localAddress)
	if err == ErrInvalidMetadata {
		sendServiceUnavailable(c)
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// check if updated of created
	if wasCreated {
		c.JSON(http.StatusCreated, gin.H{"result": "created", "causal-metadata": currMetadata})
	} else {
		c.JSON(http.StatusOK, gin.H{"result": "updated", "causal-metadata": currMetadata})
	}

	// broadcast
	go broadcastKvsPut(key, value, metadata, localAddress)
}

func deleteKey(c *gin.Context) {
	// get key from URL
	key := c.Param(("key"))

	shardId := ring.GetShardId(key)
	if shardId != localShardId {
		proxyToShard(c, "/kvs/"+key, shardId)
		return
	}

	// get the json data from the body
	data, err := parseKeysFromBody(c, "causal-metadata")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "causal-metadata not specified"})
		return
	}
	metadata := data["causal-metadata"].(map[string]int)

	// put key and check for errors
	currMetadata, err := kvsDb.DeleteData(key, metadata, localAddress)
	if err == ErrInvalidMetadata {
		sendServiceUnavailable(c)
		return
	} else if err == ErrKeyNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Key does not exist"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// send success to client
	c.JSON(http.StatusOK, gin.H{"result": "deleted", "causal-metadata": currMetadata})

	// broadcast
	go broadcastKvsDelete(key, metadata, localAddress)
}

/// --- shard routes ---

// Client to Node Endpoints
func getNodeShardId(c *gin.Context) {
	if localShardId != -1 {
		c.JSON(http.StatusOK, gin.H{"node-shard-id": localShardId})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Node not assigned to a Shard"})
	}
}

func getShardIds(c *gin.Context) {
	// Since the Shard IDs are just their index in the ring.Shards slice
	// we must create a slice of numbers from 0 to number of shards
	idArr := make([]int, len(ring.Shards))
	for i := 0; i < len(idArr); i++ {
		idArr[i] = i
	}

	// Respond with 200 OK and the idArr
	c.JSON(http.StatusOK, gin.H{"shard-ids": idArr})
}

func getShardMembers(c *gin.Context) {
	// get ID from URL
	id, err := parseShardIdFromURL(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ID not found"})
		return
	}

	// turn the map of replicas (aka shard members) into a slice so the JSON formatts correctly
	members := make([]string, len(ring.Shards[id].Replicas))

	i := 0
	for key := range ring.Shards[id].Replicas {
		members[i] = key
		i++
	}

	// respond with members
	c.JSON(http.StatusOK, gin.H{"shard-members": members})
}

func getShardKeyCount(c *gin.Context) {
	// get ID from URL
	id, err := parseShardIdFromURL(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ID not found"})
		return
	}

	if id != localShardId {
		proxyToShard(c, "/shard/key-count/"+strconv.Itoa(id), id)
		return
	}

	c.JSON(http.StatusOK, gin.H{"shard-key-count": len(kvsDb.Data)})
}

func addNodeToShard(c *gin.Context) {
	/// ----Parsing Data----
	// get shard ID from URL
	shardId, err := parseShardIdFromURL(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ID not found"})
		return
	}

	// get the json data from the body
	data, err := parseKeysFromBody(c, "socket-address")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no address specified"})
		return
	}
	nodeAddress := data["socket-address"].(string)

	// check if nodeAddress is in the view
	if !view.Contains(nodeAddress) {
		c.JSON(http.StatusNotFound, gin.H{"error": "socket-address not found"})
		return
	}

	/// ----Adding Node Local----
	// add the node the the local shard
	ring.AddNodeToShard(shardId, nodeAddress)
	// Respond to client
	c.JSON(http.StatusOK, gin.H{"result": "node added to shard"})

	go broadcastAddNodeToShard(nodeAddress, shardId)

	// if the nodeAddress is this node start cloning data
	if nodeAddress == localAddress {
		go getShardData(shardId)
	}
}

func putReshard(c *gin.Context) {
	/// ----Parsing Data----
	// get the json data from the body
	data, err := parseKeysFromBody(c, "shard-count")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no shard count specified"})
		return
	}
	shardCount := data["shard-count"].(int)

	/// ----Resharding Local----
	// reshard local ring and check for insufficient node count
	ring, err = ring.Reshard(shardCount, view.Nodes)
	if err == ErrNotEnoughNodes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough nodes to provide fault tolerance with requested shard count"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	localShardId = ring.GetShardIdFromNode(localAddress)

	// Respond to client
	c.JSON(http.StatusOK, gin.H{"result": "resharded"})

	// recheck data with new shards
	go shuffleKvsData()

	/// Broadcast
	go broadcastReshard(ring)
}

// ------------ Node to Node endpoints -----------------

// adds keys to kvsDb but with less error checking and does not broadcast
func repPutKey(c *gin.Context) {
	// get data from request body
	data, err := parseKeysFromBody(c, "key", "value", "causal-metadata", "sender")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	key := data["key"].(string)
	value := data["value"]
	metadata := data["causal-metadata"].(map[string]int)
	sender := data["sender"].(string)

	// add data to kvs database
	_, _, err = kvsDb.PutData(key, value, metadata, sender)
	if err == ErrInvalidMetadata {
		sendServiceUnavailable(c)
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// respond with success
	c.JSON(http.StatusOK, gin.H{"result": "added"})
}

func repDeleteKey(c *gin.Context) {
	// get data from request body
	data, err := parseKeysFromBody(c, "key", "causal-metadata", "sender")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	key := data["key"].(string)
	metadata := data["causal-metadata"].(map[string]int)
	sender := data["sender"].(string)

	// add data to kvs database
	_, err = kvsDb.DeleteData(key, metadata, sender)
	if err == ErrInvalidMetadata {
		sendServiceUnavailable(c)
		return
	} else if err == ErrKeyNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Key does not exist"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// respond with success
	c.JSON(http.StatusOK, gin.H{"result": "deleted"})
}

func repAddNodeToShard(c *gin.Context) {
	// get shard-id and socket-address of the node from the json data
	data, err := parseKeysFromBody(c, "shard-id", "socket-address")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	shardId, _ := data["shard-id"].(int)
	nodeAddress, _ := data["socket-address"].(string)

	// add the node to the local ring
	ring.AddNodeToShard(shardId, nodeAddress)

	if nodeAddress == localAddress {
		go getShardData(shardId)
	}
	c.JSON(http.StatusOK, gin.H{"result": "added"})
}

func repReshard(c *gin.Context) {
	oldRing := ring
	oldRing.Lock()
	defer oldRing.Unlock()

	var newRing Ring
	reqBody, _ := io.ReadAll(c.Request.Body)
	json.Unmarshal(reqBody, &newRing)
	ring = &newRing

	localShardId = ring.GetShardIdFromNode(localAddress)

	go shuffleKvsData()
	c.JSON(http.StatusOK, gin.H{"result": "resharded"})
}

func repPutKeyNoChecks(c *gin.Context) {
	// get data from request body
	data, err := parseKeysFromBody(c, "key", "value")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	key := data["key"].(string)
	value := data["value"]

	// add data to kvs database
	kvsDb.PutDataNoChecks(key, value)

	// respond with success
	c.JSON(http.StatusOK, gin.H{"result": "added"})
}

func repCloneRing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ring": ring})
}

func repCloneShardData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": kvsDb})
}

func testDataDump(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"kvsDb": kvsDb,
		"ring":  ring,
		"view":  view,
	})
}
