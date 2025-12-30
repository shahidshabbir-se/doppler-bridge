package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type ServiceType string

const (
	ServiceTypeApplication ServiceType = "application"
	ServiceTypeCompose     ServiceType = "compose"
)

type ServiceConfig struct {
	Path         string
	ServiceID    string
	ServiceType  ServiceType
	DopplerToken string
}

type Config struct {
	Port                 int
	DokployHost          string
	DokployAPIToken      string
	DopplerToken         string // Global token (fallback)
	DopplerSecret        string
	WebhookSecret        string
	CFAccessClientID     string
	CFAccessClientSecret string
	Services             []ServiceConfig
}

func Load() (*Config, error) {
	cfg := &Config{}

	flag.IntVar(&cfg.Port, "port", getEnvInt("PORT", 3000), "Port to listen on")
	flag.StringVar(&cfg.DokployHost, "dokploy-host", os.Getenv("DOKPLOY_HOST"), "Dokploy instance URL")
	flag.StringVar(&cfg.DokployAPIToken, "dokploy-api-token", os.Getenv("DOKPLOY_API_TOKEN"), "Dokploy API token")
	flag.StringVar(&cfg.DopplerToken, "doppler-token", os.Getenv("DOPPLER_TOKEN"), "Doppler service token (global/fallback)")
	flag.StringVar(&cfg.DopplerSecret, "doppler-secret", os.Getenv("DOPPLER_SECRET"), "Doppler webhook signing secret")
	flag.StringVar(&cfg.WebhookSecret, "webhook-secret", os.Getenv("WEBHOOK_SECRET"), "Secret token for webhook authentication")
	flag.StringVar(&cfg.CFAccessClientID, "cf-access-client-id", os.Getenv("CF_ACCESS_CLIENT_ID"), "Cloudflare Access Client ID")
	flag.StringVar(&cfg.CFAccessClientSecret, "cf-access-client-secret", os.Getenv("CF_ACCESS_CLIENT_SECRET"), "Cloudflare Access Client Secret")

	servicesEnv := os.Getenv("SERVICES")
	flag.StringVar(&servicesEnv, "services", servicesEnv, "Comma-separated: path:serviceId:serviceType:dopplerToken")

	flag.Parse()

	if servicesEnv != "" {
		cfg.Services = parseServices(servicesEnv)
	}

	if cfg.DokployHost == "" {
		return nil, fmt.Errorf("DOKPLOY_HOST is required")
	}
	if cfg.DokployAPIToken == "" {
		return nil, fmt.Errorf("DOKPLOY_API_TOKEN is required")
	}
	if cfg.WebhookSecret == "" {
		return nil, fmt.Errorf("WEBHOOK_SECRET is required")
	}
	if len(cfg.Services) == 0 {
		return nil, fmt.Errorf("SERVICES is required (format: path:serviceId:serviceType:dopplerToken,path2:serviceId2:serviceType2:token2)")
	}

	return cfg, nil
}

func parseServices(servicesEnv string) []ServiceConfig {
	var services []ServiceConfig
	entries := strings.Split(servicesEnv, ",")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		// Split by colon to get parts
		parts := strings.Split(entry, ":")
		if len(parts) != 4 {
			fmt.Printf("Warning: Invalid service config: %s (expected format: path:serviceId:serviceType:dopplerToken)\n", entry)
			continue
		}

		path := strings.TrimSpace(parts[0])
		serviceID := strings.TrimSpace(parts[1])
		serviceType := ServiceType(strings.TrimSpace(parts[2]))
		dopplerToken := strings.TrimSpace(parts[3])

		if path == "" || serviceID == "" || dopplerToken == "" {
			fmt.Printf("Warning: Empty path/serviceId/token in: %s\n", entry)
			continue
		}

		if serviceType != ServiceTypeApplication && serviceType != ServiceTypeCompose {
			fmt.Printf("Warning: Invalid service type: %s\n", serviceType)
			continue
		}

		services = append(services, ServiceConfig{
			Path:         path,
			ServiceID:    serviceID,
			ServiceType:  serviceType,
			DopplerToken: dopplerToken,
		})
	}
	return services
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

func (c *Config) GetServiceByPath(path string) *ServiceConfig {
	for i := range c.Services {
		if c.Services[i].Path == path {
			return &c.Services[i]
		}
	}
	return nil
}
