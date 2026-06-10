package domain

import "time"

type Action struct {
	Type string `json:"type_action"`
	Text string `json:"text"`
}

type Event struct {
	Type      string `json:"type_event"`
	Payload   string `json:"payload"`
	ChatID    int64  `json:"chat_id"`
	InContext bool   `json:"in_context"`
}

type StoredMessage struct {
	Id        int64
	ChatID    int64
	Role      string
	Content   string
	InContext bool
	CreatedAt time.Time
}

type MemoryChunk struct {
	Id        string
	ChatID    int64
	Content   string
	Embedding []float32
	Role      string
	CreatedAt time.Time
}
