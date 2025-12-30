package dokploy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/craetivohq/doppler-bridge/internal/config"
)

// Client is the Dokploy API client
type Client struct {
	host                 string
	apiToken             string
	httpClient           *http.Client
	cfAccessClientID     string
	cfAccessClientSecret string
}

// NewClient creates a new Dokploy API client
func NewClient(host, apiToken, cfClientID, cfClientSecret string) *Client {
	return &Client{
		host:                 strings.TrimSuffix(host, "/"),
		apiToken:             apiToken,
		cfAccessClientID:     cfClientID,
		cfAccessClientSecret: cfClientSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// addHeaders adds common headers including Cloudflare Access if configured
func (c *Client) addHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiToken)

	// Add Cloudflare Access headers if configured
	if c.cfAccessClientID != "" && c.cfAccessClientSecret != "" {
		req.Header.Set("CF-Access-Client-Id", c.cfAccessClientID)
		req.Header.Set("CF-Access-Client-Secret", c.cfAccessClientSecret)
	}
}

// SaveEnvironment updates environment variables for an application or compose
func (c *Client) SaveEnvironment(serviceID string, envString string, serviceType config.ServiceType) error {
	var endpoint string
	var payload map[string]interface{}

	if serviceType == config.ServiceTypeCompose {
		endpoint = "/api/compose.update"
		payload = map[string]interface{}{
			"composeId": serviceID,
			"env":       envString,
		}
	} else {
		endpoint = "/api/application.saveEnvironment"
		payload = map[string]interface{}{
			"applicationId": serviceID,
			"env":           envString,
		}
	}

	url := c.host + endpoint
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to save environment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dokploy API returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Redeploy triggers a redeployment of an application or compose
func (c *Client) Redeploy(serviceID string, serviceType config.ServiceType) error {
	var endpoint string
	var payload map[string]interface{}

	if serviceType == config.ServiceTypeCompose {
		endpoint = "/api/compose.redeploy"
		payload = map[string]interface{}{
			"composeId": serviceID,
		}
	} else {
		endpoint = "/api/application.redeploy"
		payload = map[string]interface{}{
			"applicationId": serviceID,
		}
	}

	url := c.host + endpoint
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.addHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to redeploy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dokploy API returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
