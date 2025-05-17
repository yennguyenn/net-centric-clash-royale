package network

import (
	"encoding/json"
	"fmt"
)

type PDU struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

func EncodePDU(pdu PDU) ([]byte, error) {
	return json.Marshal(pdu)
}

func DecodePDU(data []byte) (*PDU, error) {
	var pdu PDU
	err := json.Unmarshal(data, &pdu)
	if err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}
	return &pdu, nil
}
