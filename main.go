package main

import (
	kvs_pkg "cse138/assignment_4/kvs"
	ring_pkg "cse138/assignment_4/sharding"
	view_pkg "cse138/assignment_4/view"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

var kvsDb kvs_pkg.KeyValStoreDatabase
var view view_pkg.View
var ring ring_pkg.Ring
var shardId int
var localShardId int
var localAddress string

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

func getNodeShardId(c *gin.Context) {
	if localShardId != -1 {
		c.JSON(http.StatusOK, gin.H{"node-shard-id": localShardId})
	} else {

	}
}

func main() {

	localAddress := os.Getenv("SOCKET_ADDRESS")

	// initialView := os.Getenv("VIEW")

	kvsDb = kvs_pkg.KeyValStoreDatabase{
		Data:         make(map[string]interface{}),
		Metadata:     make(map[string]int),
		LocalAddress: localAddress,
	}

	localShardId = -1

	router := gin.Default()

	router.GET("/shard/ids", getShardIds)
	router.GET("/shard/node-shard-id", getNodeShardId)

	router.PUT("/rep/shard/kvs/:key")
	router.PUT("/rep/shard/kvs/:key")

	router.Run("localhost:8080")
}
