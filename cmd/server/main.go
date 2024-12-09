package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pixaverse-studios/websocket-server/internal/config"
	"github.com/pixaverse-studios/websocket-server/internal/websocket"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := config.ValidateConfig(cfg); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create WebSocket handler
	handler := websocket.NewHandler(cfg)

	// Set up HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: handler,
	}

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting server on port %d", cfg.Server.Port)
		var err error
		if cfg.Server.EnableTLS {
			log.Printf("TLS enabled, using certificate: %s", cfg.Server.CertFile)
			err = server.ListenAndServeTLS(cfg.Server.CertFile, cfg.Server.KeyFile)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-stop
	log.Println("Shutting down server...")

	// Perform cleanup
	if err := server.Close(); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}
}
