package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func sendServiceUnavailable(c *gin.Context) {
	println("sending service unavailable")
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Causal dependencies not satisfied; try again later"})
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
	for node := range nodes {
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

// Wrapper for sendBroadcastMsg for put kvs
func broadcastKvsPut(key string, value interface{}, metadata map[string]int, sender string) {
	dataMap := make(map[string]interface{})
	dataMap["key"] = key
	dataMap["value"] = value
	dataMap["causal-metadata"] = metadata
	dataMap["sender"] = localAddress

	// turn body data into string JSON
	jsonData, _ := json.Marshal(dataMap)

	// send broadcast messages on new thread
	sendBroadcastMsg(
		removeLocalAddressFromMap(view.Nodes),
		"/rep/kvs",
		http.MethodPut,
		"application/json",
		jsonData)
}

// Wrapper for sendBroadcastMsg for delete kvs
func broadcastKvsDelete(key string, metadata map[string]int, sender string) {
	dataMap := make(map[string]interface{})
	dataMap["key"] = key
	dataMap["causal-metadata"] = metadata
	dataMap["sender"] = localAddress

	// turn body data into string JSON
	jsonData, _ := json.Marshal(dataMap)

	// send broadcast messages on new thread
	sendBroadcastMsg(
		removeLocalAddressFromMap(view.Nodes),
		"/rep/kvs",
		http.MethodDelete,
		"application/json",
		jsonData)
}

// Wrapper for sendBroadcastMsg for adding node to shard
func broadcastAddNodeToShard(nodeAddress string, shardId int) {
	// build responce to broadcast
	dataMap := make(map[string]interface{})
	dataMap["socket-address"] = nodeAddress
	dataMap["shard-id"] = shardId

	// turn body data into string JSON
	jsonData, _ := json.Marshal(dataMap)

	// send broadcast messages on new thread
	sendBroadcastMsg(
		removeLocalAddressFromMap(view.Nodes),
		"/rep/shard/add-member",
		http.MethodPut,
		"application/json",
		jsonData)
}

// Wrapper for sendBroadcastMsg for resharding
func broadcastReshard(r Ring) {
	// build responce to broadcast
	dataMap := make(map[string]interface{})
	dataMap["ring"] = r

	// turn body data into string JSON
	jsonData, _ := json.Marshal(dataMap)

	sendBroadcastMsg(
		removeLocalAddressFromMap(view.Nodes),
		"/rep/shard/reshard",
		http.MethodPut,
		"application/json",
		jsonData)

}

// Wrapper for sendBroadcastMsg for put View
func broadcastPutView(node string) {
	// build responce to broadcast
	dataMap := make(map[string]interface{})
	dataMap["socket-address"] = node

	// turn body data into string JSON
	jsonData, _ := json.Marshal(dataMap)

	sendBroadcastMsg(
		removeLocalAddressFromMap(view.Nodes),
		"/view",
		http.MethodPut,
		"application/json",
		jsonData)
}

// Wrapper for sendBroadcastMsg for Delete View
func broadcastDeleteView(node string) {
	// build responce to broadcast
	dataMap := make(map[string]interface{})
	dataMap["socket-address"] = node

	// turn body data into string JSON
	jsonData, _ := json.Marshal(dataMap)

	sendBroadcastMsg(
		removeLocalAddressFromMap(view.Nodes),
		"/view",
		http.MethodDelete,
		"application/json",
		jsonData)
}

// asks the other nodes in the view for ring data
func getRingData() Ring {
	res, err := sendMsgToGroup(
		removeLocalAddressFromMap(view.Nodes),
		"/rep/shard",
		http.MethodGet,
		"application/json",
		make([]byte, 0),
	)
	if err != nil {
		log.Fatal(err)
	}

	var newRing Ring
	reqBody, _ := io.ReadAll(res.Body)
	json.Unmarshal(reqBody, &newRing)
	ring = newRing

	return newRing

}

func sendKeyValNoChecks(key string, val interface{}, node string) {
	// build responce to broadcast
	dataMap := make(map[string]interface{})
	dataMap["key"] = key
	dataMap["value"] = val

	// turn body data into string JSON
	jsonData, _ := json.Marshal(dataMap)

	go sendSingleMsg(
		node,
		"/rep/shard/kvs",
		http.MethodPut,
		"application/json",
		jsonData)
}

func broadcastKeyValNoChecks(key string, val interface{}, nodes map[string]struct{}) {
	for node := range nodes {
		sendKeyValNoChecks(key, val, node)
	}
}
