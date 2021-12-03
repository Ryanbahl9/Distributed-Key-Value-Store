package main

import (
	"time"

	"github.com/gin-gonic/gin"
)

const DEFAULT_TIMEOUT = time.Second

var kvsDb *KeyValStoreDatabase
var view *View
var ring *Ring
var localShardId int
var localAddress string

func main() {

	// Parse Environment Variables
	localAdd, initialView, initialShardCount, shardCountExists := parseEnvironmentVariables()
	localAddress = localAdd

	// Set Up Node
	if shardCountExists {
		initPrimaryNode(initialView, initialShardCount)
	} else {
		initTertiaryNode(initialView)
	}

	// Set Up Router
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
	router.PUT("/rep/shard/kvs", repPutKeyNoChecks)
	router.GET("/rep/shard", repCloneRing)
	router.GET("/rep/clone-shard-data", repCloneShardData)

	router.GET("/test", testDataDump)

	router.Run("0.0.0.0:8090")
}
