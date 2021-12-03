package main

import (
	"encoding/json"
	"errors"

	"github.com/gin-gonic/gin"
)

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
