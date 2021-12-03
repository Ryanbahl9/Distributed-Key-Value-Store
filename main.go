package main

import (
	kvs_pkg "cse138/assignment_4/kvs"
	ring_pkg "cse138/assignment_4/sharding"
	view_pkg "cse138/assignment_4/view"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const DEFAULT_TIMEOUT = time.Second

var kvsDb kvs_pkg.KeyValStoreDatabase
var view view_pkg.View
var ring ring_pkg.Ring
var localShardId int
var localAddress string

//TODO: Finish
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
		"application/json",
		make([]byte, 1))

	if err != nil {
		log.Fatal(err)
	}

	var newKvsDb kvs_pkg.KeyValStoreDatabase

	resBody, _ := io.ReadAll(res.Body)
	json.Unmarshal(resBody, &newKvsDb)
	kvsDb = newKvsDb
}

//TODO: Finish
// This function runs through every key in the database
// and re checks what shard it belongs to
func shuffleKvsData() {
	kvsDb.Lock()
	defer kvsDb.Unlock()

	// for key, _ := range kvsDb.Data {
	// }
}

func main() {

	localShardId = -1

	router := gin.Default()

	// View Routes
	router.GET("/view", getView)
	router.PUT("/view", putView)
	router.DELETE("/view", deleteView)

	// kvs Routes
	router.GET("/kvs/:key", getKey)
	router.PUT("/kvs/:key", putKey)
	router.DELETE("/kvs/:key", deleteKey)

	// shard routes
	router.GET("/shard/ids", getShardIds)
	router.GET("/shard/node-shard-id", getNodeShardId)
	router.GET("/shard/members/:id", getShardMembers)
	router.GET("/shard/key-count/:id", getShardKeyCount)
	router.PUT("/shard/add-member/:id", addNodeToShard)
	router.PUT("/shard/reshard", putReshard)

	// kvs Routes
	router.PUT("/rep/kvs", repPutKey)
	router.DELETE("/rep/kvs", repDeleteKey)

	router.PUT("/rep/shard/add-member", repAddNodeToShard)
	router.PUT("/rep/shard/reshard", repReshard)
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
