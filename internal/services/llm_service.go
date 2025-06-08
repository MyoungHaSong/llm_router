package services

import (
	"context"
	"fmt"

	"github.com/llm-router/internal/models"
	"github.com/llm-router/internal/providers"
)

type LLMService interface {
	ChatCompletion(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error)
	ChatCompletionStream(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionChunk, <-chan error)
}

type LLMServiceImpl struct {
	providerFactory *providers.ProviderFactory
}

func NewLLMService(providerFactory *providers.ProviderFactory) LLMService {
	return &LLMServiceImpl{providerFactory: providerFactory}
}

func (s *LLMServiceImpl) ChatCompletion(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	provider, err := s.providerFactory.GetProvider(req.Model)
	if err != nil {
		return nil, fmt.Errorf("provider not found for model %s: %w", req.Model, err)
	}

	response, err := provider.ChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM service error: %w", err)
	}
	return response, nil
}

func (s *LLMServiceImpl) ChatCompletionStream(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionChunk, <-chan error) {
	errCh := make(chan error, 1)

	provider, err := s.providerFactory.GetProvider(req.Model)
	if err != nil {
		errCh <- fmt.Errorf("provider not found for model %s: %w", req.Model, err)
		close(errCh)
		return nil, errCh
	}

	chunkCh, providerErrCh := provider.ChatCompletionStream(ctx, req)

	return chunkCh, providerErrCh
}
