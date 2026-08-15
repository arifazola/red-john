package models

import "encoding/json"

type Event struct {
	Message string `json:"Message"`
	Payload json.RawMessage `json:"Data"`
}