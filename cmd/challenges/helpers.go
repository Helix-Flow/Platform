package main

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

func splitToken(token string) []string {
	return strings.Split(token, ".")
}

func decodeBase64JSON(seg string) map[string]interface{} {
	// JWT base64url may need padding
	padding := 4 - len(seg)%4
	if padding != 4 {
		seg += strings.Repeat("=", padding)
	}
	data, err := base64.URLEncoding.DecodeString(seg)
	if err != nil {
		return nil
	}
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result
}
