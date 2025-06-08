package server

import (
	"github.com/gin-gonic/gin"
	"github.com/llm-router/internal/handlers"
)

func RegisterRoutes(engine *gin.Engine, llmHandler *handlers.LLMHandler) {

	// Register health check route
	engine.GET("/_health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// Register LLM chat completion route
	engine.POST("/v1/chat/completions", llmHandler.HandleChatCompletion)
}
