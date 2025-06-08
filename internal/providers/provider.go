package providers

import (
	"context"

	"github.com/llm-router/internal/models"
)

type Provider interface {
	ChatCompletion(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error)
	ChatCompletionStream(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionChunk, <-chan error)
}
