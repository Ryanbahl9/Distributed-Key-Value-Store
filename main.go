package main

import (
	kvs_pkg "cse138/assignment_4/kvs"
	ring_pkg "cse138/assignment_4/sharding"
	view_pkg "cse138/assignment_4/view"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const DEFAULT_TIMEOUT = time.Second

var kvsDb kvs_pkg.KeyValStoreDatabase
var view view_pkg.View
var ring ring_pkg.Ring
var localShardId int
var localAddress string

func deleteNode(node string) {

}

// Clones all the kvs data from the specified shard
// to local kvs database
// todo add more error checking
func getShardData(shardId int) {

	res, err := sendMsgToGroup(
		removeLocalAddressFromMap(ring.Shards[shardId].Replicas),
		"/rep/clone-shard-data",
		http.MethodGet,
		" application/json",
		make([]byte, 1))

	if err != nil {
		log.Fatal(err)
	}

	var newKvsDb kvs_pkg.KeyValStoreDatabase

	resBody, _ := io.ReadAll(res.Body)
	json.Unmarshal(resBody, &newKvsDb)
	kvsDb = newKvsDb
}

/// ----- gin router handler functions -----

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
	for key, _ := range ring.Shards[id].Replicas {
		members[i] = key
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

	// add the node the the local shard
	ring.AddNodeToShard(shardId, nodeAddress)

	// build responce to broadcast
	dataMap := make(map[string]interface{})
	dataMap["socket-address"] = nodeAddress
	dataMap["shard-id"] = shardId

	// turn body data into string JSON
	jsonData, err := json.Marshal(dataMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	sendBroadcastMsg(
		removeLocalAddressFromMap(view.Nodes),
		"/rep/shard/add-member",
		http.MethodPut,
		c.ContentType(),
		jsonData)

	c.JSON(http.StatusOK, gin.H{"result": "node added to shard"})

	// if the nodeAddress is this node start cloning data
	if nodeAddress == localAddress {
		go getShardData(shardId)
	}
}

func putReshard(c *gin.Context) {
	// get the json data from the body
	data, err := parseKeysFromBody(c, "shard-count")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no shard count specified"})
		return
	}
	shardCount := data["shard-count"].(int)

	ring, err = ring.Reshard(shardCount, view.Nodes)
}

// ------------ Node to Node endpoints -----------------
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
}

func repGetCloneShardData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": kvsDb})
}

func main() {

	localShardId = -1

	router := gin.Default()

	router.GET("/shard/ids", getShardIds)
	router.GET("/shard/node-shard-id", getNodeShardId)
	router.GET("/shard/members/:id", getShardMembers)
	router.GET("/shard/key-count/:id", getShardKeyCount)
	router.PUT("/shard/add-member/:id", addNodeToShard)
	router.PUT("/shard/reshard", putReshard)

	router.PUT("/rep/shard/add-member", repAddNodeToShard)
	router.GET("/rep/clone-shard-data", repGetCloneShardData)

	router.Run("localhost:8080")
}

// util functions
func removeLocalAddressFromMap(mp map[string]struct{}) map[string]struct{} {
	copyMp := make(map[string]struct{})
	for key := range mp {
		if key != localAddress {
			copyMp[key] = struct{}{}
		}
	}
	return copyMp
}
