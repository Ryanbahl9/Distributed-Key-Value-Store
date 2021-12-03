package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func broadcastKVSPut(key string, value interface{}, metadata map[string]int, sender string, view *View) {
	dataMap := make(map[string]interface{})
	dataMap["key"] = key
	dataMap["value"] = value
	dataMap["causal-metadata"] = metadata
	dataMap["sender"] = localAddress

	go sendBroadcastMsg("PUT", dataMap, "/rep/kvs")
}

func broadcastKVSDelete(key string, metadata map[string]int, sender string, view *View) {
	dataMap := make(map[string]interface{})
	dataMap["key"] = key
	dataMap["causal-metadata"] = metadata
	dataMap["sender"] = localAddress

	go sendBroadcastMsg("PUT", dataMap, "/rep/kvs")
}

func sendServiceUnavailable(c *gin.Context) {
	println("sending service unavailable")
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Causal dependencies not satisfied; try again later", "causal-metadata": localVC})
}

func sendSingleMsg(method string, replica string, messageData map[string]interface{}, endpoint string) bool {
	//
	replicaUrl := "http://" + replica + endpoint

	// turn body data into string JSON
	stringJsonBody, err := json.Marshal(messageData)
	if err != nil {
		return false
	}

	// Loop on doing request until responce or timeout
	for {
		// Build request
		var req *http.Request
		switch method {
		case "PUT":
			req, err = http.NewRequest(http.MethodPut, replicaUrl, bytes.NewBuffer(stringJsonBody))
		case "DELETE":
			req, err = http.NewRequest(http.MethodDelete, replicaUrl, bytes.NewBuffer(stringJsonBody))
		}
		if err != nil {
			return false
		}
		req.Header.Set("Content-Type", "application/json")

		var netClient = &http.Client{
			Timeout: time.Second,
		}

		resp, err := netClient.Do(req) // if timeout or error, delete
		if err != nil {
			deleteReplicaAndBroadcast(replica)
			return false
		}

		// if status code anything but 503, break
		if resp.StatusCode != http.StatusServiceUnavailable {
			return true
		}
	}
}

// Method should equal:
// 		"PUT" for put
//		"DELETE" for delete
func sendBroadcastMsg(method string, messageData map[string]interface{}, endpoint string) {

	for replica := range localView {
		// start thread to send message
		if replica != localAddress {
			go sendSingleMsg(method, replica, messageData, endpoint)
		}
	}
}
