package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/llm-router/internal/server"
)

func main() {

	Server, err := server.NewServer()
	if err != nil {
		panic(err)
	}

	defer Server.Shutdown()

	Server.Run()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
}
