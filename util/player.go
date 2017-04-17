package util

import (
	"encoding/json"
	"net/http"
	"time"
	"log"
)

var httpClient = &http.Client{Timeout: 5 * time.Second}

func GetFromJSONAPI(url string, target interface{}) error {
	log.Printf("[MAPI]: Executing Mojang API...")
	r, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}
