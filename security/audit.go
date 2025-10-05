package security

import (
	"encoding/json"
	"fmt"
	"time"
)

type AuditEvent struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	IP        string                 `json:"ip"`
	UserAgent string                 `json:"user_agent"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
}

type AuditLogger struct {
	events []AuditEvent
}

func CreateAuditLogger() *AuditLogger {
	return &AuditLogger{
		events: make([]AuditEvent, 0),
	}
}

func (a *AuditLogger) LogEvent(userID, action, resource, ip, userAgent string, metadata map[string]interface{}, success bool, err error) {
	event := AuditEvent{
		ID:        fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		IP:        ip,
		UserAgent: userAgent,
		Metadata:  metadata,
		Timestamp: time.Now(),
		Success:   success,
	}

	if err != nil {
		event.Error = err.Error()
	}

	a.events = append(a.events, event)
}

func (a *AuditLogger) LogAuth(userID, action, ip, userAgent string, success bool, err error) {
	a.LogEvent(userID, action, "auth", ip, userAgent, nil, success, err)
}

func (a *AuditLogger) LogPayment(userID, action, paymentID, ip, userAgent string, amount float64, currency string, success bool, err error) {
	metadata := map[string]interface{}{
		"payment_id": paymentID,
		"amount":     amount,
		"currency":   currency,
	}
	a.LogEvent(userID, action, "payment", ip, userAgent, metadata, success, err)
}

func (a *AuditLogger) LogAPI(userID, action, endpoint, ip, userAgent string, method string, success bool, err error) {
	metadata := map[string]interface{}{
		"endpoint": endpoint,
		"method":   method,
	}
	a.LogEvent(userID, action, "api", ip, userAgent, metadata, success, err)
}

func (a *AuditLogger) LogSecurity(userID, action, ip, userAgent string, threat string, success bool, err error) {
	metadata := map[string]interface{}{
		"threat": threat,
	}
	a.LogEvent(userID, action, "security", ip, userAgent, metadata, success, err)
}

func (a *AuditLogger) GetEvents(userID string, limit int) []AuditEvent {
	var userEvents []AuditEvent
	count := 0

	for i := len(a.events) - 1; i >= 0 && count < limit; i-- {
		if a.events[i].UserID == userID {
			userEvents = append(userEvents, a.events[i])
			count++
		}
	}

	return userEvents
}

func (a *AuditLogger) GetEventsByAction(action string, limit int) []AuditEvent {
	var actionEvents []AuditEvent
	count := 0

	for i := len(a.events) - 1; i >= 0 && count < limit; i-- {
		if a.events[i].Action == action {
			actionEvents = append(actionEvents, a.events[i])
			count++
		}
	}

	return actionEvents
}

func (a *AuditLogger) GetSecurityEvents(limit int) []AuditEvent {
	var securityEvents []AuditEvent
	count := 0

	for i := len(a.events) - 1; i >= 0 && count < limit; i-- {
		if a.events[i].Resource == "security" {
			securityEvents = append(securityEvents, a.events[i])
			count++
		}
	}

	return securityEvents
}

func (a *AuditLogger) ExportJSON() ([]byte, error) {
	return json.Marshal(a.events)
}

func (a *AuditLogger) GetStats() AuditStats {
	stats := AuditStats{
		TotalEvents:    len(a.events),
		SuccessCount:   0,
		ErrorCount:     0,
		SecurityEvents: 0,
	}

	for _, event := range a.events {
		if event.Success {
			stats.SuccessCount++
		} else {
			stats.ErrorCount++
		}

		if event.Resource == "security" {
			stats.SecurityEvents++
		}
	}

	return stats
}

type AuditStats struct {
	TotalEvents    int `json:"total_events"`
	SuccessCount   int `json:"success_count"`
	ErrorCount     int `json:"error_count"`
	SecurityEvents int `json:"security_events"`
}
