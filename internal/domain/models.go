package domain

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

type Summary struct {
}

type MemoryChunk struct {
	Id        string
	ChatID    int64
	Content   string
	Embedding []float32
}
