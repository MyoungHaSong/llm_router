package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/llm-router/internal/config"
	"github.com/llm-router/internal/handlers"
	"github.com/llm-router/internal/providers"
	"github.com/llm-router/internal/services"
)

type Server struct {
	engine     *gin.Engine
	httpServer *http.Server
	cfg        *config.Config
}

func NewServer() (*Server, error) {
	cfg := config.LoadConfig()

	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.Default()

	providerFactory := providers.NewProviderFactory(
		cfg.APIKeys,
	)
	llmService := services.NewLLMService(providerFactory)

	llmHandler, err := handlers.NewLLMHandler(llmService)
	if err != nil {
		return nil, err
	}

	RegisterRoutes(engine, llmHandler)
	return &Server{
		engine: engine,
		cfg:    cfg,
	}, nil
}

func (s *Server) Run() {
	port := s.cfg.Port
	addr := ":" + port

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.engine,
	}
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic("Failed to start server: " + err.Error())
		}
	}()
}

func (s *Server) Shutdown() {
	// Implement graceful shutdown logic if needed
	// For example, you can use s.engine.Shutdown(context.Background())
	// to gracefully shut down the server.
	if s.httpServer != nil {
		ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancle()

		if err := s.httpServer.Shutdown(ctx); err != nil {
			panic("Failed to shutdown server: " + err.Error())
		}
	}
}
