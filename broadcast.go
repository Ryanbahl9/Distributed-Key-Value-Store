package main

import (
	"bytes"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// func broadcastKVSPut(key string, value interface{}, metadata map[string]int, sender string, view *View) {
// dataMap := make(map[string]interface{})
// dataMap["key"] = key
// dataMap["value"] = value
// dataMap["causal-metadata"] = metadata
// dataMap["sender"] = localAddress

// go sendBroadcastMsg("PUT", dataMap, "/rep/kvs")
// }

// func broadcastKVSDelete(key string, metadata map[string]int, sender string, view *View) {
// 	dataMap := make(map[string]interface{})
// 	dataMap["key"] = key
// 	dataMap["causal-metadata"] = metadata
// 	dataMap["sender"] = localAddress

// 	go sendBroadcastMsg("PUT", dataMap, "/rep/kvs")
// }

func sendServiceUnavailable(c *gin.Context) {
	println("sending service unavailable")
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Causal dependencies not satisfied; try again later", "causal-metadata": localVC})
}

// sends a single message to the node specified and returns the responce
// If the responce code is 503 (Service Unavailable) is retries until
// a different status code is returned or timeout
func sendSingleMsg(node string, endpoint string, method string, contentType string, data []byte) (*http.Response, error) {
	nodeUrl := "http://" + node + endpoint

	// Loop on doing request until responce or timeout
	for {
		// Build request
		req, err := http.NewRequest(method, nodeUrl, bytes.NewReader(data))
		if err != nil {
			return &http.Response{}, err
		}
		req.Header.Set("Content-Type", "application/json")

		// Create netClient with timeout set at 1 second
		var netClient = &http.Client{
			Timeout: DEFAULT_TIMEOUT,
		}

		// send request
		resp, err := netClient.Do(req) // if timeout or error, delete
		if err != nil {
			deleteNode(node)
			return resp, err
		}

		// if status code anything but 503, break
		if resp.StatusCode != http.StatusServiceUnavailable {
			return resp, nil
		}
	}
}

// This function takes in a map of nodes and message data
// and attempts to send the message to the first node that will respond
// In other words, it picks a node, and if that node doesn't respond it moves to the
// next node on the list
func sendMsgToGroup(nodes map[string]struct{}, endpoint string, method string, contentType string, data []byte) (*http.Response, error) {
	// Send msg to each node in list until responce
	for node, _ := range nodes {
		res, err := sendSingleMsg(node, endpoint, method, contentType, data)
		if err != nil {
			return res, nil
		}
	}

	// If the program gets here, that means all the
	// nodes in the list are either down or unresponsive
	return nil, errors.New("all nodes specified are down or unresponsive")
}

// Sends the specified message data to all nodes listed
func sendBroadcastMsg(nodes map[string]struct{}, endpoint string, method string, contentType string, data []byte) {
	for node := range nodes {
		go sendSingleMsg(node, endpoint, method, contentType, data)
	}
}
