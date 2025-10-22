package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	headerSize     = 4
	maxPayloadSize = 10 * 1024 * 1024 // 10MB max payload size
)

// Encode encodes the commandID as the first 4 bytes (big-endian) followed by the payload.
func Encode(commandID uint32, payload []byte) ([]byte, error) {
	if len(payload) > maxPayloadSize {
		return nil, fmt.Errorf("payload size %d exceeds maximum %d bytes", len(payload), maxPayloadSize)
	}

	out := make([]byte, headerSize+len(payload))
	binary.BigEndian.PutUint32(out[:headerSize], commandID)
	copy(out[headerSize:], payload)
	return out, nil
}

// Decode decodes the first 4 bytes as commandID (big-endian) and returns the rest as payload.
// The payload slice references the input data for performance - do not modify it.
func Decode(data []byte) (uint32, []byte, error) {
	if len(data) < headerSize {
		return 0, nil, errors.New("data too short")
	}

	payloadSize := len(data) - headerSize
	if payloadSize > maxPayloadSize {
		return 0, nil, fmt.Errorf("payload size %d exceeds maximum %d bytes", payloadSize, maxPayloadSize)
	}

	cmd := binary.BigEndian.Uint32(data[:headerSize])
	// Use slicing instead of copying for better performance
	// Caller should not modify the payload slice
	payload := data[headerSize:]
	return cmd, payload, nil
}
