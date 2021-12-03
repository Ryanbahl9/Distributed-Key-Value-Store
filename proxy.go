package main

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func proxyToShard(c *gin.Context, endpoint string, shardId int) {

	// Get body data from context
	readCLoser, _ := c.Request.GetBody()
	reqData, _ := io.ReadAll(readCLoser)

	// send client request to shard and get responce
	res, err := sendMsgToGroup(
		removeLocalAddressFromMap(ring.Shards[shardId].Replicas),
		endpoint,
		c.Request.Method,
		c.ContentType(),
		reqData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	// send the responce from the shard back to the OG client
	resData, _ := io.ReadAll(res.Body)
	c.Data(res.StatusCode, res.Header.Get("Content-Type"), resData)
}
