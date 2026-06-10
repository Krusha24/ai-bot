package brain

import (
	"ai-bot/internal/domain"
	"context"
	"encoding/json"

	ollama "github.com/ollama/ollama/api"
)

type Brain struct {
	client     *ollama.Client
	chatModel  string
	embedModel string
}

func NewBrain(chatModel, embedModel string) (*Brain, error) {
	client, err := ollama.ClientFromEnvironment()
	if err != nil {
		return nil, err
	}
	return &Brain{client: client, chatModel: chatModel, embedModel: embedModel}, nil
}

func (b *Brain) Think(messages []ollama.Message) (domain.Action, error) {
	stream := false
	req := &ollama.ChatRequest{
		Model:    b.chatModel,
		Messages: messages,
		Format:   json.RawMessage(`"json"`),
		Stream:   &stream,
	}
	ctx := context.Background()
	var action domain.Action
	respFunc := func(resp ollama.ChatResponse) error {
		json.Unmarshal([]byte(resp.Message.Content), &action)
		return nil
	}
	err := b.client.Chat(ctx, req, respFunc)
	if err != nil {
		return action, err
	}
	return action, nil
}

func (b *Brain) GetEmbedding(text string) ([]float32, error) {
	req := &ollama.EmbeddingRequest{
		Model:  b.embedModel,
		Prompt: text,
	}
	ctx := context.Background()

	resp, err := b.client.Embeddings(ctx, req)
	if err != nil {

		return nil, err
	}
	emb32 := make([]float32, len(resp.Embedding))
	for i, v := range resp.Embedding {
		emb32[i] = float32(v)
	}
	return emb32, nil
}

func (b *Brain) Summarize(messages []ollama.Message) (string, error) {
	stream := false
	var Summarize []ollama.Message
	Summarize = append(Summarize, ollama.Message{Role: "system", Content: "Ты — биограф и менеджер памяти. Твоя задача — составить краткую выжимку диалога. Напиши ключевые факты, которые нужно запомнить (имена, договоренности, темы). Будь максимально лаконичен. Используй не более 3-4 предложений."})
	Summarize = append(Summarize, messages...)

	req := &ollama.ChatRequest{
		Model:    b.chatModel,
		Messages: Summarize,
		Stream:   &stream,
	}
	ctx := context.Background()
	var summary string
	respFunc := func(resp ollama.ChatResponse) error {
		summary = resp.Message.Content
		return nil
	}
	err := b.client.Chat(ctx, req, respFunc)
	if err != nil {
		return summary, err
	}
	return summary, nil
}
