package domain

import (
	"context"
)

type IVectorStorage interface {
	SaveChunk(ctx context.Context, chunk MemoryChunk) error

	SearchSimilar(ctx context.Context, chatID int64, embedding []float32, limit int) (MemoryChunk, error)
}
