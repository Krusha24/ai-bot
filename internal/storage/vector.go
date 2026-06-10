package storage

import (
	"ai-bot/internal/domain"
	"context"
	"fmt"
	"time"

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
			"chat_id":    fmt.Sprintf("%d", chunk.ChatID),
			"role":       chunk.Role,
			"created_at": chunk.CreatedAt.Format(time.RFC3339Nano),
		},
	}

	return b.collection.AddDocument(ctx, doc)
}

func (b *ChromemDB) SearchSimilar(ctx context.Context, chatID int64, embedding []float32, limit int) ([]domain.MemoryChunk, error) {
	if limit <= 0 || len(embedding) == 0 {
		return nil, nil
	}
	var chunks []domain.MemoryChunk
	countOfChunks := b.collection.Count()
	if countOfChunks <= 0 {
		return chunks, nil
	} else if countOfChunks < limit {
		limit = countOfChunks
	}

	results, err := b.collection.QueryEmbedding(ctx, embedding, limit, map[string]string{"chat_id": fmt.Sprintf("%d", chatID)}, nil)
	if err != nil {
		return nil, err
	}

	for _, res := range results {
		createdAt, err := time.Parse(time.RFC3339Nano, res.Metadata["created_at"])
		if err != nil {
			createdAt = time.Time{}
		}

		chunk := domain.MemoryChunk{
			Id:        res.ID,
			ChatID:    chatID,
			Role:      res.Metadata["role"],
			Content:   res.Content,
			Embedding: res.Embedding,
			CreatedAt: createdAt,
		}

		chunks = append(chunks, chunk)
	}
	return chunks, nil

}
