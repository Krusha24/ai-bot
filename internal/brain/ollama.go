package brain

import (
	"ai-bot/internal/domain"
	"context"
	"encoding/json"

	ollama "github.com/ollama/ollama/api"
)

type Brain struct {
	model      *ollama.Client
	model_name string
}

func NewOllamaClient(model_name string) (*Brain, error) {
	client, err := ollama.ClientFromEnvironment()
	if err != nil {
		return nil, err
	}
	return &Brain{model: client, model_name: model_name}, nil
}

func (b *Brain) Think(messages []ollama.Message) (domain.Action, error) {
	stream := false
	req := &ollama.ChatRequest{
		Model:    b.model_name,
		Messages: messages,
		//Prompt: fmt.Sprintf("Ты — ИИ-собеседник. Пользователь написал тебе: '%s'. Выбери одно из действий. Доступные значения для поля 'type_action': 'reply' (если хочешь ответить), 'ignore' (если хочешь промолчать). Не придумывай свои типы! Поле Text - тег 'text'", prompt),
		Format: json.RawMessage(`"json"`),
		Stream: &stream,
	}
	ctx := context.Background()
	var action domain.Action
	respFunc := func(resp ollama.ChatResponse) error {
		json.Unmarshal([]byte(resp.Message.Content), &action)
		return nil
	}
	err := b.model.Chat(ctx, req, respFunc)
	if err != nil {
		return action, err
	}
	return action, nil
}

func (b *Brain) GetEmbedding(text string) ([]float64, error) {
	req := &ollama.EmbeddingRequest{
		Model:  b.model_name,
		Prompt: text,
	}
	ctx := context.Background()

	resp, err := b.model.Embeddings(ctx, req)
	if err != nil {

		return nil, err
	}
	return resp.Embedding, nil
}

func (b *Brain) Summarize(messages []ollama.Message) (string, error) {
	stream := false
	var Summarize []ollama.Message
	Summarize = append(Summarize, ollama.Message{Role: "system", Content: "Ты — биограф и менеджер памяти. Твоя задача — составить краткую выжимку диалога. Напиши ключевые факты, которые нужно запомнить (имена, договоренности, темы). Будь максимально лаконичен. Используй не более 3-4 предложений."})
	Summarize = append(Summarize, messages...)

	req := &ollama.ChatRequest{
		Model:    b.model_name,
		Messages: messages,
		Stream:   &stream,
	}
	ctx := context.Background()
	var summary string
	respFunc := func(resp ollama.ChatResponse) error {
		summary = resp.Message.Content
		return nil
	}
	err := b.model.Chat(ctx, req, respFunc)
	if err != nil {
		return summary, err
	}
	return summary, nil
}
