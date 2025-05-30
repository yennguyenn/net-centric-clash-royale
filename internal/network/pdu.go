package network

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

// PDU đại diện cho một tin nhắn truyền qua mạng
type PDU struct {
	Type    string `json:"type"`
	Payload string `json:"payload"` // Nội dung cụ thể
}

// EncodePDU chuyển PDU thành []byte để gửi đi
func EncodePDU(pdu PDU) ([]byte, error) {
	return json.Marshal(pdu)
}

// DecodePDU chuyển []byte thành PDU (sau khi nhận được)
func DecodePDU(data []byte) (PDU, error) {
	var pdu PDU
	err := json.Unmarshal(data, &pdu)
	if err != nil {
		return PDU{}, fmt.Errorf("failed to decode PDU: %w", err)
	}
	return pdu, nil
}
func SendPDU(conn net.Conn, pduType, payload string) error {
	pdu := PDU{Type: pduType, Payload: payload}
	data, err := EncodePDU(pdu)
	if err != nil {
		return err
	}
	data = append(data, '\n') // để client đọc được dòng
	_, err = conn.Write(data)
	return err
}

func ReadPDU(conn net.Conn) (PDU, error) {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return PDU{}, err
	}
	return DecodePDU([]byte(strings.TrimSpace(line)))
}
