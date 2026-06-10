package storage

import (
	"ai-bot/internal/domain"
	"context"
	"fmt"

	chromem "github.com/philippgille/chromem-go"
)

type ChromemDB struct {
	client     *chromem.DB
	collection *chromem.Collection
}

func NewChromemDB() (*ChromemDB, error) {
	db, err := chromem.NewPersistentDB("./rag_data", false)
	if err != nil {
		return nil, err
	}

	collection, err := db.GetOrCreateCollection("memories", nil, nil)
	if err != nil {
		return nil, err
	}

	return &ChromemDB{client: db, collection: collection}, nil
}

func (b *ChromemDB) SaveChunk(ctx context.Context, chunk domain.MemoryChunk) error {
	doc := chromem.Document{
		ID:        chunk.Id,
		Embedding: chunk.Embedding,
		Content:   chunk.Content,
		Metadata: map[string]string{
			"chat_id": fmt.Sprintf("%d", chunk.ChatID),
		},
	}

	return b.collection.AddDocument(ctx, doc)
}

func (b *ChromemDB) SearchSimilar(ctx context.Context, chatID int64, embedding []float32, limit int) ([]domain.MemoryChunk, error) {
	results, err := b.collection.QueryEmbedding(ctx, embedding, limit, map[string]string{"chat_id": fmt.Sprintf("%d", chatID)}, nil)
	if err != nil {
		return nil, err
	}
	var chunks []domain.MemoryChunk

	for _, res := range results {
		chunk := domain.MemoryChunk{
			Id:        res.ID,
			ChatID:    chatID,
			Content:   res.Content,
			Embedding: res.Embedding,
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}
