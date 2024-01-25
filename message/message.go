package message

import (
	"bytes"
)

type Payload []byte

type Message struct {
	UUID string

	Metadata Metadata

	Payload Payload
}

func NewMessage(uuid string, payload Payload) *Message {
	return &Message{
		UUID:     uuid,
		Metadata: make(map[string]string),
		Payload:  payload,
	}
}

func (m *Message) Equals(toCompare *Message) bool {
	if m.UUID != toCompare.UUID {
		return false
	}
	if len(m.Metadata) != len(toCompare.Metadata) {
		return false
	}
	for key, value := range m.Metadata {
		if value != toCompare.Metadata[key] {
			return false
		}
	}
	return bytes.Equal(m.Payload, toCompare.Payload)
}

func (m *Message) Copy() *Message {
	msg := NewMessage(m.UUID, m.Payload)
	for k, v := range m.Metadata {
		msg.Metadata.Set(k, v)
	}
	return msg
}
