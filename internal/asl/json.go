package asl

import (
	"github.com/buger/jsonparser"
	"strconv"
)

func JsonExtractString(json []byte, key string) (string, error) {
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

func JsonExtractRefPath(json []byte, key string) (Path, error) {
	value, _, _, err := jsonparser.Get(json, key)
	if err != nil {
		return "", err
	}
	path, errO := NewReferencePath(string(value))
	if errO != nil {
		return "", err
	}
	return path, nil
}

func JsonExtractRefPathOrDefault(json []byte, key string, def Path) Path {
	value, _, _, err := jsonparser.Get(json, key)
	if err != nil {
		return def
	}
	path, errO := NewReferencePath(string(value))
	if errO != nil {
		return def
	}
	return path
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

func JsonNumberOfKeys(json []byte) int {
	num := 0

	_ = jsonparser.ObjectEach(json, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		num++
		return nil
	})
	return num
}
