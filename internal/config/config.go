package config

import (
	"flag"
	"fmt"
	"os"
)

type ServiceType string

const (
	ServiceTypeApplication ServiceType = "application"
	ServiceTypeCompose     ServiceType = "compose"
)

type Config struct {
	// Server settings
	Port int

	// Dokploy settings
	DokployHost          string
	DokployAPIToken      string
	DokployApplicationID string
	DokployServiceType   ServiceType

	// Doppler settings
	DopplerToken  string
	DopplerSecret string // For webhook HMAC signature verification

	// Webhook authentication
	WebhookSecret string // Bearer token for webhook endpoint

	// Cloudflare Zero Trust
	CFAccessClientID     string // Cloudflare Access Client ID
	CFAccessClientSecret string // Cloudflare Access Client Secret
}

func Load() (*Config, error) {
	cfg := &Config{}

	// Define flags
	flag.IntVar(&cfg.Port, "port", getEnvInt("PORT", 3000), "Port to listen on")
	flag.StringVar(&cfg.DokployHost, "dokploy-host", os.Getenv("DOKPLOY_HOST"), "Dokploy instance URL")
	flag.StringVar(&cfg.DokployAPIToken, "dokploy-api-token", os.Getenv("DOKPLOY_API_TOKEN"), "Dokploy API token")
	flag.StringVar(&cfg.DokployApplicationID, "dokploy-application-id", os.Getenv("DOKPLOY_APPLICATION_ID"), "Dokploy application or compose ID")

	serviceType := flag.String("dokploy-service-type", getEnv("DOKPLOY_SERVICE_TYPE", "application"), "Service type: application or compose")

	flag.StringVar(&cfg.DopplerToken, "doppler-token", os.Getenv("DOPPLER_TOKEN"), "Doppler service token")
	flag.StringVar(&cfg.DopplerSecret, "doppler-secret", os.Getenv("DOPPLER_SECRET"), "Doppler webhook signing secret (for HMAC verification)")
	flag.StringVar(&cfg.WebhookSecret, "webhook-secret", os.Getenv("WEBHOOK_SECRET"), "Secret token for webhook authentication")

	flag.StringVar(&cfg.CFAccessClientID, "cf-access-client-id", os.Getenv("CF_ACCESS_CLIENT_ID"), "Cloudflare Access Client ID")
	flag.StringVar(&cfg.CFAccessClientSecret, "cf-access-client-secret", os.Getenv("CF_ACCESS_CLIENT_SECRET"), "Cloudflare Access Client Secret")

	flag.Parse()

	// Set service type
	switch *serviceType {
	case "application":
		cfg.DokployServiceType = ServiceTypeApplication
	case "compose":
		cfg.DokployServiceType = ServiceTypeCompose
	default:
		return nil, fmt.Errorf("invalid service type: %s (must be 'application' or 'compose')", *serviceType)
	}

	// Validate required fields
	if cfg.DokployHost == "" {
		return nil, fmt.Errorf("DOKPLOY_HOST is required")
	}
	if cfg.DokployAPIToken == "" {
		return nil, fmt.Errorf("DOKPLOY_API_TOKEN is required")
	}
	if cfg.DokployApplicationID == "" {
		return nil, fmt.Errorf("DOKPLOY_APPLICATION_ID is required")
	}
	if cfg.DopplerToken == "" {
		return nil, fmt.Errorf("DOPPLER_TOKEN is required")
	}
	if cfg.WebhookSecret == "" {
		return nil, fmt.Errorf("WEBHOOK_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}
