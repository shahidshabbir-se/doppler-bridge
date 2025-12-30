package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/craetivohq/doppler-bridge/internal/config"
	"github.com/craetivohq/doppler-bridge/internal/handler"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting Doppler Bridge")
	log.Printf("  Port: %d", cfg.Port)
	log.Printf("  Dokploy Host: %s", cfg.DokployHost)
	log.Printf("  Service Type: %s", cfg.DokployServiceType)
	log.Printf("  Service ID: %s", cfg.DokployApplicationID)
	if cfg.DopplerSecret != "" {
		log.Printf("  Doppler Signature Verification: ENABLED")
	} else {
		log.Printf("  Doppler Signature Verification: DISABLED")
	}
	log.Printf("  Webhook Authentication: ENABLED")
	if cfg.CFAccessClientID != "" && cfg.CFAccessClientSecret != "" {
		log.Printf("  Cloudflare Zero Trust: ENABLED")
	} else {
		log.Printf("  Cloudflare Zero Trust: DISABLED")
	}

	// Create handler
	h := handler.NewHandler(cfg)

	// Setup routes
	http.HandleFunc("/webhook", h.AuthMiddleware(h.WebhookHandler))
	http.HandleFunc("/health", h.HealthHandler)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Server listening on %s", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
