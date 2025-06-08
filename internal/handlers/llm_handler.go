package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/llm-router/internal/models"
	"github.com/llm-router/internal/services"
)

type LLMHandler struct {
	llmService services.LLMService
}

func NewLLMHandler(llmService services.LLMService) (*LLMHandler, error) {
	return &LLMHandler{
		llmService: llmService,
	}, nil
}

func (h *LLMHandler) HandleChatCompletion(c *gin.Context) {
	var req models.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	if err := validateChatCompletionRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %s", err.Error()),
		})
		return
	}

	if req.Stream {
		h.handleStreamChatCompletion(c, &req)
	} else {
		h.handleNormalChatCompletion(c, &req)
	}
}

func (h *LLMHandler) handleNormalChatCompletion(c *gin.Context, req *models.ChatCompletionRequest) {
	response, err := h.llmService.ChatCompletion(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process chat completion",
		})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *LLMHandler) handleStreamChatCompletion(c *gin.Context, req *models.ChatCompletionRequest) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	streamCh, errCh := h.llmService.ChatCompletionStream(c.Request.Context(), req)

	clientClosed := c.Writer.CloseNotify()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-clientClosed:
			return false
		case chunk, ok := <-streamCh:
			if !ok {
				c.SSEvent("data", "[DONE]")
				return false
			}

			jsonData, err := json.Marshal(chunk)
			if err != nil {
				fmt.Printf("Error marshalling chunk to JSON: %v\n", err)
				c.SSEvent("error", "Failed to process chat completion chunk")
				return false
			}
			c.SSEvent("data", string(jsonData))
			return true

		case err, ok := <-errCh:
			if !ok {
				return false
			}
			if err != nil {
				errorData, _ := json.Marshal(gin.H{"error": err.Error()})
				c.SSEvent("error", string(errorData))
				fmt.Printf("Error from stream service: %v\n", err)
				return false
			}
			return true
		}
	})
}

func validateChatCompletionRequest(req *models.ChatCompletionRequest) error {
	if req.Model == "" {
		return fmt.Errorf("model is required")
	}
	if len(req.Messages) == 0 {
		return fmt.Errorf("messages cannot be empty")
	}
	return nil
}
