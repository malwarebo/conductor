package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type WebhookEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"created_at"`
	Source    string                 `json:"source"`
}

type WebhookEndpoint struct {
	ID            string     `json:"id"`
	URL           string     `json:"url"`
	Events        []string   `json:"events"`
	Secret        string     `json:"-"`
	IsActive      bool       `json:"is_active"`
	RetryCount    int        `json:"retry_count"`
	Timeout       int        `json:"timeout"`
	CreatedAt     time.Time  `json:"created_at"`
	LastTriggered *time.Time `json:"last_triggered,omitempty"`
}

type WebhookManager struct {
	endpoints map[string]*WebhookEndpoint
	events    []WebhookEvent
}

func CreateWebhookManager() *WebhookManager {
	return &WebhookManager{
		endpoints: make(map[string]*WebhookEndpoint),
		events:    make([]WebhookEvent, 0),
	}
}

func (wm *WebhookManager) RegisterEndpoint(endpoint *WebhookEndpoint) error {
	if endpoint.ID == "" {
		endpoint.ID = fmt.Sprintf("webhook_%d", time.Now().Unix())
	}

	if endpoint.Secret == "" {
		secret, err := wm.generateSecret()
		if err != nil {
			return err
		}
		endpoint.Secret = secret
	}

	endpoint.CreatedAt = time.Now()
	wm.endpoints[endpoint.ID] = endpoint

	return nil
}

func (wm *WebhookManager) TriggerEvent(ctx context.Context, eventType string, data map[string]interface{}, source string) error {
	event := WebhookEvent{
		ID:        fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		Type:      eventType,
		Data:      data,
		CreatedAt: time.Now(),
		Source:    source,
	}

	wm.events = append(wm.events, event)

	for _, endpoint := range wm.endpoints {
		if !endpoint.IsActive {
			continue
		}

		if wm.shouldTrigger(endpoint, eventType) {
			go wm.sendWebhook(ctx, endpoint, event)
		}
	}

	return nil
}

func (wm *WebhookManager) shouldTrigger(endpoint *WebhookEndpoint, eventType string) bool {
	for _, event := range endpoint.Events {
		if event == eventType || event == "*" {
			return true
		}
	}
	return false
}

func (wm *WebhookManager) sendWebhook(ctx context.Context, endpoint *WebhookEndpoint, event WebhookEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	signature := wm.generateSignature(payload, endpoint.Secret)

	now := time.Now()
	endpoint.LastTriggered = &now

	for attempt := 0; attempt <= endpoint.RetryCount; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		success := wm.deliverWebhook(ctx, endpoint.URL, payload, signature)
		if success {
			return
		}
	}
}

func (wm *WebhookManager) deliverWebhook(ctx context.Context, url string, payload []byte, signature string) bool {
	timeout := time.Duration(30) * time.Second
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	req, err := wm.createRequest(url, payload, signature)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (wm *WebhookManager) createRequest(url string, payload []byte, signature string) (*http.Request, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))

	return req, nil
}

func (wm *WebhookManager) generateSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

func (wm *WebhookManager) generateSecret() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (wm *WebhookManager) GetEndpoints() map[string]*WebhookEndpoint {
	return wm.endpoints
}

func (wm *WebhookManager) GetEndpoint(id string) (*WebhookEndpoint, bool) {
	endpoint, exists := wm.endpoints[id]
	return endpoint, exists
}

func (wm *WebhookManager) UpdateEndpoint(id string, endpoint *WebhookEndpoint) error {
	if _, exists := wm.endpoints[id]; !exists {
		return fmt.Errorf("endpoint not found")
	}

	wm.endpoints[id] = endpoint
	return nil
}

func (wm *WebhookManager) DeleteEndpoint(id string) error {
	if _, exists := wm.endpoints[id]; !exists {
		return fmt.Errorf("endpoint not found")
	}

	delete(wm.endpoints, id)
	return nil
}

func (wm *WebhookManager) GetEvents(limit int) []WebhookEvent {
	if limit <= 0 || limit > len(wm.events) {
		limit = len(wm.events)
	}

	start := len(wm.events) - limit
	if start < 0 {
		start = 0
	}

	return wm.events[start:]
}

func (wm *WebhookManager) GetStats() WebhookStats {
	stats := WebhookStats{
		TotalEndpoints:  len(wm.endpoints),
		ActiveEndpoints: 0,
		TotalEvents:     len(wm.events),
	}

	for _, endpoint := range wm.endpoints {
		if endpoint.IsActive {
			stats.ActiveEndpoints++
		}
	}

	return stats
}

type WebhookStats struct {
	TotalEndpoints  int `json:"total_endpoints"`
	ActiveEndpoints int `json:"active_endpoints"`
	TotalEvents     int `json:"total_events"`
}
