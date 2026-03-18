package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// cacheKey builds a cache key in the form "service:method:<hash>",
// where the hash is the first 16 hex characters of sha256(json(params)).
func cacheKey(service, method string, params ...interface{}) string {
	data, _ := json.Marshal(params)
	hash := sha256.Sum256(data)
	return service + ":" + method + ":" + hex.EncodeToString(hash[:8])
}
