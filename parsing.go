package main

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
)

func parseKeysFromBody(c *gin.Context, keys ...string) (map[string]interface{}, error) {
	// get the json data from the body
	data, err := parseDataFromBody(c)
	if err != nil {
		return nil, err
	}

	output := make(map[string]interface{})

	for _, key := range keys {
		val, exists := data[key]
		if !exists {
			return nil, errors.New("not all keys present")
		}
		output[key] = val
	}

	return output, nil
}

func parseDataFromBody(c *gin.Context) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// get the value from the body
	rawData, _ := c.GetRawData()

	// Unmarshal data
	err := json.Unmarshal(rawData, &data)
	if err != nil {
		return nil, errors.New("cannot parse values from message body")
	}

	return data, nil
}

func parseShardIdFromURL(c *gin.Context) (int, error) {
	// get ID from URL
	strId := c.Param(("id"))
	var id int

	// convert string to int
	if n, err := strconv.Atoi(strId); err == nil {
		id = n
	} else {
		// respond with error
		return -1, errors.New("cannot parse shard id")
	}

	// Check if id is outside of bounds
	if id < 0 || id >= len(ring.Shards) {
		// respond with error
		return -1, errors.New("cannot parse shard id")
	}

	return id, nil
}

func parseKVSDataFromContext(c *gin.Context) (map[string]int, string, interface{}, string, error) {
	key := c.Param(("key"))

	// Create Data Var
	var data kvsStruct

	// get the value from the body
	rawData, _ := c.GetRawData()

	// Unmarshal data
	err := json.Unmarshal(rawData, &data)
	if err != nil {
		return nil, key, nil, "", errors.New("cannot parse values from message body")
	}

	return data.Metadata, key, data.Value, data.Sender, nil
}
