package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/craetivohq/doppler-bridge/internal/config"
	"github.com/craetivohq/doppler-bridge/internal/dokploy"
	"github.com/craetivohq/doppler-bridge/internal/doppler"
)

type Handler struct {
	cfg            *config.Config
	dokployClient  *dokploy.Client
	dopplerClients map[string]*doppler.Client // Cache Doppler clients by path
}

func NewHandler(cfg *config.Config) *Handler {
	h := &Handler{
		cfg:            cfg,
		dokployClient:  dokploy.NewClient(cfg.DokployHost, cfg.DokployAPIToken, cfg.CFAccessClientID, cfg.CFAccessClientSecret),
		dopplerClients: make(map[string]*doppler.Client),
	}

	// Create Doppler client for each service
	for _, svc := range cfg.Services {
		h.dopplerClients[svc.Path] = doppler.NewClient(svc.DopplerToken)
		log.Printf("Created Doppler client for path: %s (token: %s...)", svc.Path, svc.DopplerToken[:10])
	}

	return h
}

// AuthMiddleware validates the webhook secret
func (h *Handler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		// Support both "Bearer <token>" and just "<token>"
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

		if token != h.cfg.WebhookSecret {
			log.Println("Unauthorized: Invalid webhook secret")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// WebhookHandler handles Doppler webhook requests with path-based routing
func (h *Handler) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract path from URL (e.g., /webhook/meilisearch)
	path := strings.TrimPrefix(r.URL.Path, "/webhook")
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		log.Println("Missing service path in webhook URL")
		http.Error(w, "Bad request: missing service path", http.StatusBadRequest)
		return
	}

	// Look up service configuration by path
	serviceConfig := h.cfg.GetServiceByPath(path)
	if serviceConfig == nil {
		log.Printf("Service not found for path: %s", path)
		http.Error(w, "Not found: unknown service path", http.StatusNotFound)
		return
	}

	log.Printf("Routing webhook to service: path=%s, serviceId=%s, type=%s",
		path, serviceConfig.ServiceID, serviceConfig.ServiceType)

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify signature if secret is configured
	if h.cfg.DopplerSecret != "" {
		signature := r.Header.Get("X-Doppler-Signature")
		if signature == "" {
			log.Println("Missing Doppler signature header")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !doppler.VerifySignature(h.cfg.DopplerSecret, body, signature) {
			log.Println("Invalid Doppler signature")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		log.Println("Doppler signature verified")
	}

	// Parse webhook payload
	var webhook doppler.Webhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		log.Printf("Failed to parse webhook: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	log.Printf("Received webhook: type=%s, project=%s, config=%s, path=%s, added=%d, removed=%d, updated=%d",
		webhook.Type, webhook.Project.Name, webhook.Config.Name, path,
		len(webhook.Diff.Added), len(webhook.Diff.Removed), len(webhook.Diff.Updated))

	// Respond immediately and process in background
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

	// Process webhook asynchronously with service config
	go h.processWebhookWithService(webhook, serviceConfig)
}

// processWebhookWithService processes the webhook for a specific service
func (h *Handler) processWebhookWithService(webhook doppler.Webhook, serviceConfig *config.ServiceConfig) {
	// Get Doppler client for this service
	dopplerClient, ok := h.dopplerClients[serviceConfig.Path]
	if !ok {
		log.Printf("No Doppler client found for path: %s", serviceConfig.Path)
		return
	}

	// Fetch secrets from Doppler using the service-specific token
	secrets, err := dopplerClient.FetchSecrets(webhook.Config.Project, webhook.Config.Name)
	if err != nil {
		log.Printf("Failed to fetch secrets for %s: %v", serviceConfig.Path, err)
		return
	}

	log.Printf("Fetched %d secrets from Doppler for %s", len(secrets), serviceConfig.Path)

	// Convert to env string
	envString := doppler.SecretsToEnvString(secrets)

	// Update environment in Dokploy
	if err := h.dokployClient.SaveEnvironment(serviceConfig.ServiceID, envString, serviceConfig.ServiceType); err != nil {
		log.Printf("Failed to update environment for %s: %v", serviceConfig.Path, err)
		return
	}

	log.Printf("Successfully updated environment for %s (serviceId: %s, type: %s)",
		serviceConfig.Path, serviceConfig.ServiceID, serviceConfig.ServiceType)

	// Trigger redeploy
	if err := h.dokployClient.Redeploy(serviceConfig.ServiceID, serviceConfig.ServiceType); err != nil {
		log.Printf("Failed to redeploy %s: %v", serviceConfig.Path, err)
		return
	}

	log.Printf("Successfully triggered redeploy for %s (serviceId: %s, type: %s)",
		serviceConfig.Path, serviceConfig.ServiceID, serviceConfig.ServiceType)
}

// HealthHandler handles health check requests
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
