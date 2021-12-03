package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// util functions
func initPrimaryNode(initailView []string, shardCount int) {
	kvsDb = NewKeyValStoreDatabase(localAddress)
	view = NewView()
	for _, v := range initailView {
		view.PutView(v)
	}
	ring = NewRing(shardCount, view.Nodes)
	localShardId = ring.GetShardIdFromNode(localAddress)
}

func initTertiaryNode(initailView []string) {
	kvsDb = NewKeyValStoreDatabase(localAddress)
	view = NewView()
	for _, v := range initailView {
		view.PutView(v)
	}
	ring = getRingData()
	localShardId = -1

	broadcastPutView(localAddress)
}

func deleteNode(node string) {
	view.DeleteView(node)
	ring.RemoveNode(node)
	broadcastDeleteView(node)
}

// asks the other nodes in the view for ring data
func getRingData() *Ring {
	res, err := sendMsgToGroup(
		removeLocalAddressFromMap(view.Nodes),
		"/rep/shard",
		http.MethodGet,
		"application/json",
		make([]byte, 0),
		true)
	if err != nil {
		log.Fatal(err)
	}

	type TempSt struct {
		Ring Ring `json:"ring"`
	}
	var newRing TempSt
	reqBody, _ := io.ReadAll(res.Body)
	json.Unmarshal(reqBody, &newRing)

	return &newRing.Ring
}

// Clones all the kvs data from the specified shard
// to local kvs database
func getShardData(shardId int) {

	res, err := sendMsgToGroup(
		removeLocalAddressFromMap(ring.Shards[shardId].Replicas),
		"/rep/clone-shard-data",
		http.MethodGet,
		"application/json",
		make([]byte, 1),
		true)

	if err != nil {
		log.Fatal(err)
	}

	type TempKvs struct {
		Kvs KeyValStoreDatabase `json:"data"`
	}

	var newKvsDb TempKvs

	resBody, _ := io.ReadAll(res.Body)
	json.Unmarshal(resBody, &newKvsDb)
	kvsDb = &newKvsDb.Kvs
}

// This function runs through every key in the database
// and re checks what shard it belongs to
// if it belongs to a different node
func shuffleKvsData() {
	kvsDb.Lock()
	defer kvsDb.Unlock()

	toDelete := make(map[string]interface{})

	for key, val := range kvsDb.Data {
		// if key no longer belongs, add to the delete list
		keyShardId := ring.GetShardId(key)
		if keyShardId != localShardId {
			toDelete[key] = val
		}
		// Broadcast to new shard
		go broadcastKeyValNoChecks(key, val, removeLocalAddressFromMap(ring.Shards[keyShardId].Replicas))
	}

	// Delete all shards the don't belong to new shard
	for key := range toDelete {
		delete(kvsDb.Data, key)
	}
}

func removeLocalAddressFromMap(mp map[string]struct{}) map[string]struct{} {
	copyMp := make(map[string]struct{})
	for key := range mp {
		if key != localAddress {
			copyMp[key] = struct{}{}
		}
	}
	return copyMp
}

func proxyToShard(c *gin.Context, endpoint string, shardId int) {

	// Get body data from context
	reqData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(123, gin.H{"error": err.Error()})
		return
	}

	// send client request to shard and get responce
	res, err := sendMsgToGroup(
		removeLocalAddressFromMap(ring.Shards[shardId].Replicas),
		endpoint,
		c.Request.Method,
		c.ContentType(),
		reqData,
		false)
	if err != nil {
		c.JSON(123, gin.H{"error": err.Error()})
		return
	}

	// send the responce from the shard back to the OG client
	resData, _ := io.ReadAll(res.Body)
	c.Data(res.StatusCode, res.Header.Get("Content-Type"), resData)
}

func getMetadataFromInterface(i interface{}) map[string]int {
	metadata := make(map[string]int)
	if i == nil {
		metadata = make(map[string]int)
	} else {
		tempData := i.(map[string]interface{})
		for key, val := range tempData {
			metadata[key] = int(val.(float64))
		}
	}
	return metadata
}
