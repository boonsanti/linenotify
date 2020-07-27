package main

import (
	"encoding/json"
	"log"
)

type tokenResponse struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	AccessToken string `json:"access_token"`
}

func newTokenResponse(raw []byte) (*tokenResponse, error) {
	ret := &tokenResponse{}
	err := json.Unmarshal(raw, &ret)
	if err != nil {
		log.Println("json unmarshal err:", err)
		return ret, err
	}
	return ret, nil
}
