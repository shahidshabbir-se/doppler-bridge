package doppler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Webhook represents the Doppler webhook payload
type Webhook struct {
	Type      string        `json:"type"`
	Config    ConfigInfo    `json:"config"`
	Project   ProjectInfo   `json:"project"`
	Workplace WorkplaceInfo `json:"workplace"`
	Diff      SecretsDiff   `json:"diff"`
}

type ConfigInfo struct {
	Name        string `json:"name"`
	Environment string `json:"environment"`
	Project     string `json:"project"`
}

type ProjectInfo struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type WorkplaceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SecretsDiff struct {
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
	Updated []string `json:"updated"`
}

// Client is the Doppler API client
type Client struct {
	token      string
	httpClient *http.Client
}

// NewClient creates a new Doppler API client
func NewClient(token string) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchSecrets fetches all secrets from Doppler
func (c *Client) FetchSecrets(project, config string) (map[string]string, error) {
	url := fmt.Sprintf("https://api.doppler.com/v3/configs/config/secrets/download?project=%s&config=%s&format=json", project, config)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch secrets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("doppler API returned %d: %s", resp.StatusCode, string(body))
	}

	var secrets map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&secrets); err != nil {
		return nil, fmt.Errorf("failed to decode secrets: %w", err)
	}

	return secrets, nil
}

// VerifySignature verifies the Doppler webhook signature
func VerifySignature(secret string, body []byte, signature string) bool {
	// Doppler signature format: "sha256=<hex>"
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	expectedSig := strings.TrimPrefix(signature, "sha256=")

	// Calculate HMAC
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	calculatedSig := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(calculatedSig), []byte(expectedSig))
}

// SecretsToEnvString converts secrets map to .env format
func SecretsToEnvString(secrets map[string]string) string {
	var lines []string
	for key, value := range secrets {
		// Escape and quote values if necessary
		if strings.Contains(value, " ") || strings.Contains(value, "\n") || strings.Contains(value, "\"") {
			value = strings.ReplaceAll(value, "\\", "\\\\")
			value = strings.ReplaceAll(value, "\"", "\\\"")
			lines = append(lines, fmt.Sprintf("%s=\"%s\"", key, value))
		} else {
			lines = append(lines, fmt.Sprintf("%s=%s", key, value))
		}
	}
	return strings.Join(lines, "\n")
}
