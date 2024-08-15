package utils

import (
	"github.com/buger/jsonparser"
	"strconv"
)

func JsonExtract(json []byte, key string) (string, error) {
	value, _, _, err := jsonparser.Get(json, key)
	if err != nil {
		return "", err
	}
	return string(value), nil
}

func JsonExtractStringOrDefault(json []byte, key string, def string) string {
	value, _, _, err := jsonparser.Get(json, key)
	if err != nil {
		return def
	}
	return string(value)
}

func JsonExtractObjectOrDefault(json []byte, key string, def interface{}) interface{} {
	value, _, _, err := jsonparser.Get(json, key)
	if err != nil {
		return def
	}
	return value
}

func JsonExtractIntOrDefault(json []byte, key string, def int) int {
	value, _, _, err := jsonparser.Get(json, key)
	if err != nil {
		return def
	}
	i, err := strconv.Atoi(string(value))
	if err != nil {
		return def
	}
	return i
}

func JsonExtractOrNil(json []byte, key string) interface{} {
	value, _, _, err := jsonparser.Get(json, key)
	if err != nil {
		return nil
	}
	return string(value)
}

// JsonExtractBool extracts a boolean value with the specified key. If key does not exist, returns false
func JsonExtractBool(json []byte, key string) bool {
	value, err := jsonparser.GetBoolean(json, key)
	if err != nil {
		return false
	}
	return value
}
