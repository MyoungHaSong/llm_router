package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/liushuangls/go-anthropic/v2"
	"github.com/llm-router/internal/models"
)

type anthropicProvider struct {
	client *anthropic.Client
}

func NewAntropicProvider(apiKey string) Provider {
	client := anthropic.NewClient(apiKey)
	return &anthropicProvider{
		client: client,
	}
}

func (p *anthropicProvider) ChatCompletion(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	var anthropicMessages []anthropic.Message
	var systemMessage string
	var maxTokens int

	for _, msg := range req.Messages {
		switch msg.Role {
		case "user":
			anthropicMessages = append(anthropicMessages, anthropic.NewUserTextMessage(msg.Content))
		case "assistant":
			anthropicMessages = append(anthropicMessages, anthropic.NewAssistantTextMessage(msg.Content))
		case "system":
			systemMessage = msg.Content

		default:
			return nil, fmt.Errorf("unsupported message role: %s", msg.Role)
		}
	}

	if req.MaxTokens != nil {
		maxTokens = int(*req.MaxTokens)
	} else {
		maxTokens = 1024
	}

	messagesRequest := anthropic.MessagesRequest{
		Model:         anthropic.Model(req.Model),
		Messages:      anthropicMessages,
		MaxTokens:     maxTokens,
		System:        systemMessage,
		StopSequences: req.Stop,
	}
	if req.Temperature != nil {
		temp := float32(*req.Temperature)
		messagesRequest.Temperature = &temp
	}

	if req.TopP != nil {
		topP := float32(*req.TopP)
		messagesRequest.TopP = &topP
	}

	anthropicResp, err := p.client.CreateMessages(ctx, messagesRequest)
	if err != nil {
		return nil, err
	}

	var responseText string
	if len(anthropicResp.Content) > 0 && anthropicResp.Content[0].Text != nil {
		responseText = *anthropicResp.Content[0].Text
	}

	modelName := string(anthropicResp.Model)
	role := string(anthropicResp.Role)

	response := &models.ChatCompletionResponse{
		ID:      anthropicResp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   modelName,
		Choices: []models.ChatCompletionChoice{
			{
				Index: 0,
				Message: models.ChatMessage{
					Role:    role,
					Content: responseText,
				},
			},
		},
	}

	if anthropicResp.StopReason != "" {
		response.Choices[0].FinishReason = string(anthropicResp.StopReason)
	} else {
		response.Choices[0].FinishReason = "stop"
	}

	response.Usage = models.Usage{
		PromptTokens:     int64(anthropicResp.Usage.InputTokens),
		CompletionTokens: int64(anthropicResp.Usage.OutputTokens),
		TotalTokens:      int64(anthropicResp.Usage.InputTokens) + int64(anthropicResp.Usage.OutputTokens),
	}

	return response, nil
}

func (p *anthropicProvider) ChatCompletionStream(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionChunk, <-chan error) {
	chunkChan := make(chan *models.ChatCompletionChunk)
	errorChan := make(chan error, 1) // 버퍼를 주어 오류 발생 시 즉시 반환 가능하도록

	var anthropicMessages []anthropic.Message
	var systemMessage string

	var maxTokens int
	if req.MaxTokens != nil {
		maxTokens = int(*req.MaxTokens)
	} else {
		maxTokens = 1024
	}

	for _, msg := range req.Messages {
		switch msg.Role {
		case "user":
			anthropicMessages = append(anthropicMessages, anthropic.NewUserTextMessage(msg.Content))
		case "assistant":
			anthropicMessages = append(anthropicMessages, anthropic.NewAssistantTextMessage(msg.Content))
		case "system":
			systemMessage = msg.Content
		default:
			errorChan <- fmt.Errorf("unsupported message role: %s", msg.Role)
			close(chunkChan)
			return chunkChan, errorChan
		}
	}

	baseRequest := anthropic.MessagesRequest{
		Model:         anthropic.Model(req.Model),
		Messages:      anthropicMessages,
		MaxTokens:     maxTokens,
		System:        systemMessage,
		StopSequences: req.Stop,
		Stream:        true, // 스트리밍 요청임을 명시
	}

	if req.Temperature != nil {
		temp := float32(*req.Temperature)
		baseRequest.Temperature = &temp
	}
	if req.TopP != nil {
		topP := float32(*req.TopP)
		baseRequest.TopP = &topP
	}

	streamRequest := anthropic.MessagesStreamRequest{
		MessagesRequest: baseRequest,
	}

	// 스트림 ID와 모델명을 저장하기 위한 변수 (OnMessageStart에서 설정)
	var streamID string
	var streamModel string

	go func() {
		defer close(chunkChan)

		streamRequest.OnMessageStart = func(data anthropic.MessagesEventMessageStartData) {
			streamID = data.Message.ID
			streamModel = string(data.Message.Model)

		}

		streamRequest.OnContentBlockDelta = func(data anthropic.MessagesEventContentBlockDeltaData) {
			if *data.Delta.Text != "" {
				chunk := &models.ChatCompletionChunk{
					ID:      streamID,
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   streamModel,
					Choices: []models.ChatCompletionChunkChoice{
						{
							Index: data.Index,
							Delta: models.ChatMessage{
								Role:    "assistant",
								Content: *data.Delta.Text,
							},
						},
					},
				}
				chunkChan <- chunk
			}
		}

		streamRequest.OnMessageDelta = func(data anthropic.MessagesEventMessageDeltaData) {
			finishReason := string(data.Delta.StopReason)
			if finishReason == "" {
				chunk := &models.ChatCompletionChunk{
					ID:      streamID,
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   streamModel,
					Choices: []models.ChatCompletionChunkChoice{
						{
							Index:        0,
							FinishReason: &finishReason,
							Delta:        models.ChatMessage{Role: "assistant"},
						},
					},
				}
				chunkChan <- chunk
			}
		}

		streamRequest.OnMessageStop = func(data anthropic.MessagesEventMessageStopData) {
		}

		streamRequest.OnError = func(errResp anthropic.ErrorResponse) {
			if errResp.Error != nil {
				select {
				case errorChan <- fmt.Errorf("anthropic stream callback error: type=%s, message=%s", errResp.Error.Type, errResp.Error.Message):
				default:
				}
			}
		}

		_, err := p.client.CreateMessagesStream(ctx, streamRequest)
		if err != nil {
			select {
			case errorChan <- fmt.Errorf("failed to complete messages stream: %w", err):
			default:

			}
			return
		}
	}()

	return chunkChan, errorChan
}
