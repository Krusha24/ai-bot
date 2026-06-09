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
