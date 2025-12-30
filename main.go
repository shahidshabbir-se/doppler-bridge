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

	log.Printf("Starting Doppler Bridge (Multi-Tenant)")
	log.Printf("  Port: %d", cfg.Port)
	log.Printf("  Dokploy Host: %s", cfg.DokployHost)
	log.Printf("  Services: %d configured", len(cfg.Services))
	for _, svc := range cfg.Services {
		log.Printf("    - /webhook/%s -> %s (%s) [token: %s...]", svc.Path, svc.ServiceID, svc.ServiceType, svc.DopplerToken[:15])
	}
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

	// Setup routes (path-based routing)
	http.HandleFunc("/webhook/", h.AuthMiddleware(h.WebhookHandler))
	http.HandleFunc("/health", h.HealthHandler)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Server listening on %s", addr)
	log.Printf("Webhook URLs:")
	for _, svc := range cfg.Services {
		log.Printf("  https://your-domain.com/webhook/%s", svc.Path)
	}

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
