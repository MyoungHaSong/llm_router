package providers

import (
	"context"
	"fmt"

	"github.com/llm-router/internal/models"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type OpenAIProvider struct {
	client openai.Client
}

func NewOpenAIProvider(apiKey string) Provider {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)
	return &OpenAIProvider{
		client: client,
	}
}

func (p *OpenAIProvider) ChatCompletion(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	messages := make([]openai.ChatCompletionMessageParamUnion, len(req.Messages))
	for i, msg := range req.Messages {
		switch msg.Role {
		case "user":
			messages[i] = openai.UserMessage(msg.Content)
		case "assistant":
			messages[i] = openai.AssistantMessage(msg.Content)
		case "system":
			messages[i] = openai.SystemMessage(msg.Content)
		default:
			return nil, fmt.Errorf("unsupported message role: %s", msg.Role)
		}
	}

	chatCompletion, err := p.client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    req.Model,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get chat completion: %w", err)
	}

	choices := make([]models.ChatCompletionChoice, len(chatCompletion.Choices))
	for i, choice := range chatCompletion.Choices {
		choices[i] = models.ChatCompletionChoice{
			Message: models.ChatMessage{
				Role:    string(choice.Message.Role),
				Content: choice.Message.Content,
			},
			FinishReason: string(choice.FinishReason),
		}
	}

	return &models.ChatCompletionResponse{
		ID:      chatCompletion.ID,
		Object:  string(chatCompletion.Object),
		Created: chatCompletion.Created,
		Model:   string(chatCompletion.Model),
		Choices: choices,
		Usage: models.Usage{
			PromptTokens:     chatCompletion.Usage.PromptTokens,
			CompletionTokens: chatCompletion.Usage.CompletionTokens,
			TotalTokens:      chatCompletion.Usage.TotalTokens,
		},
		Error: nil,
	}, nil
}

func (p *OpenAIProvider) ChatCompletionStream(
	ctx context.Context,
	req *models.ChatCompletionRequest,
) (<-chan *models.ChatCompletionChunk, <-chan error) {
	chunkCh := make(chan *models.ChatCompletionChunk, 2)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		messages := make([]openai.ChatCompletionMessageParamUnion, len(req.Messages))
		for i, msg := range req.Messages {
			switch msg.Role {
			case "user":
				messages[i] = openai.UserMessage(msg.Content)
			case "assistant":
				messages[i] = openai.AssistantMessage(msg.Content)
			case "system":
				messages[i] = openai.SystemMessage(msg.Content)
			default:
				errCh <- fmt.Errorf("unsupported role: %s", msg.Role)
				return
			}
		}

		params := openai.ChatCompletionNewParams{
			Model:    req.Model,
			Messages: messages,
		}
		if req.Temperature != nil {
			params.Temperature = openai.Float(*req.Temperature)
		}
		if req.TopP != nil {
			params.TopP = openai.Float(*req.TopP)
		}
		if req.MaxTokens != nil {
			params.MaxTokens = openai.Int(*req.MaxTokens)
		}

		stream := p.client.Chat.Completions.NewStreaming(ctx, params)

		for stream.Next() {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			resp := stream.Current()

			choices := make([]models.ChatCompletionChunkChoice, len(resp.Choices))
			for i, c := range resp.Choices {
				var reason *string
				if c.FinishReason != "" {
					r := string(c.FinishReason)
					reason = &r
				}
				choices[i] = models.ChatCompletionChunkChoice{
					Index:        int(c.Index),
					Delta:        models.ChatMessage{Role: string(c.Delta.Role), Content: c.Delta.Content},
					FinishReason: reason,
				}
			}

			chunk := &models.ChatCompletionChunk{
				ID:      resp.ID,
				Object:  string(resp.Object),
				Created: resp.Created,
				Model:   string(resp.Model),
				Choices: choices,
			}

			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case chunkCh <- chunk:
			}
		}

		if err := stream.Err(); err != nil {
			errCh <- fmt.Errorf("stream iteration error: %w", err)
		}
	}()

	return chunkCh, errCh
}
