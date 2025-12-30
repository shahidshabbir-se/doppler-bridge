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
	cfg           *config.Config
	dopplerClient *doppler.Client
	dokployClient *dokploy.Client
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		cfg:           cfg,
		dopplerClient: doppler.NewClient(cfg.DopplerToken),
		dokployClient: dokploy.NewClient(cfg.DokployHost, cfg.DokployAPIToken, cfg.CFAccessClientID, cfg.CFAccessClientSecret),
	}
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

// WebhookHandler handles Doppler webhook requests
func (h *Handler) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	log.Printf("Received webhook: type=%s, project=%s, config=%s, added=%d, removed=%d, updated=%d",
		webhook.Type, webhook.Project.Name, webhook.Config.Name,
		len(webhook.Diff.Added), len(webhook.Diff.Removed), len(webhook.Diff.Updated))

	// Respond immediately and process in background
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

	// Process webhook asynchronously
	go h.processWebhook(webhook)
}

// processWebhook handles the webhook processing in the background
func (h *Handler) processWebhook(webhook doppler.Webhook) {
	// Fetch secrets from Doppler
	secrets, err := h.dopplerClient.FetchSecrets(webhook.Config.Project, webhook.Config.Name)
	if err != nil {
		log.Printf("Failed to fetch secrets: %v", err)
		return
	}

	log.Printf("Fetched %d secrets from Doppler", len(secrets))

	// Convert to env string
	envString := doppler.SecretsToEnvString(secrets)

	// Update environment in Dokploy
	if err := h.dokployClient.SaveEnvironment(h.cfg.DokployApplicationID, envString, h.cfg.DokployServiceType); err != nil {
		log.Printf("Failed to update environment: %v", err)
		return
	}

	log.Printf("Successfully updated environment for %s: %s", h.cfg.DokployServiceType, h.cfg.DokployApplicationID)

	// Trigger redeploy
	if err := h.dokployClient.Redeploy(h.cfg.DokployApplicationID, h.cfg.DokployServiceType); err != nil {
		log.Printf("Failed to redeploy: %v", err)
		return
	}

	log.Printf("Successfully triggered redeploy for %s: %s", h.cfg.DokployServiceType, h.cfg.DokployApplicationID)
}

// HealthHandler handles health check requests
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
