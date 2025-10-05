package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type AlertLevel int

const (
	Info AlertLevel = iota
	Warning
	Critical
	Emergency
)

type Alert struct {
	ID         string
	Level      AlertLevel
	Title      string
	Message    string
	Source     string
	Timestamp  time.Time
	Resolved   bool
	ResolvedAt *time.Time
	Metadata   map[string]interface{}
}

type AlertRule struct {
	ID            string
	Name          string
	Condition     func(metrics map[string]float64) bool
	Level         AlertLevel
	Cooldown      time.Duration
	LastTriggered time.Time
	Enabled       bool
}

type AlertManager struct {
	alerts   map[string]*Alert
	rules    map[string]*AlertRule
	channels []AlertChannel
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

type AlertChannel interface {
	Send(alert *Alert) error
}

type ConsoleAlertChannel struct{}

func (c *ConsoleAlertChannel) Send(alert *Alert) error {
	fmt.Printf("[%s] %s: %s\n", alert.Level.String(), alert.Title, alert.Message)
	return nil
}

func (al AlertLevel) String() string {
	switch al {
	case Info:
		return "INFO"
	case Warning:
		return "WARNING"
	case Critical:
		return "CRITICAL"
	case Emergency:
		return "EMERGENCY"
	default:
		return "UNKNOWN"
	}
}

func NewAlertManager() *AlertManager {
	ctx, cancel := context.WithCancel(context.Background())
	am := &AlertManager{
		alerts:   make(map[string]*Alert),
		rules:    make(map[string]*AlertRule),
		channels: []AlertChannel{&ConsoleAlertChannel{}},
		ctx:      ctx,
		cancel:   cancel,
	}

	go am.startMonitoring()
	return am
}

func (am *AlertManager) AddRule(rule *AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.rules[rule.ID] = rule
}

func (am *AlertManager) RemoveRule(ruleID string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.rules, ruleID)
}

func (am *AlertManager) AddChannel(channel AlertChannel) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.channels = append(am.channels, channel)
}

func (am *AlertManager) TriggerAlert(alert *Alert) {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert.ID = fmt.Sprintf("alert_%d", time.Now().UnixNano())
	alert.Timestamp = time.Now()
	am.alerts[alert.ID] = alert

	for _, channel := range am.channels {
		go func(ch AlertChannel) {
			if err := ch.Send(alert); err != nil {
				fmt.Printf("Failed to send alert: %v\n", err)
			}
		}(channel)
	}
}

func (am *AlertManager) ResolveAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	now := time.Now()
	alert.Resolved = true
	alert.ResolvedAt = &now

	return nil
}

func (am *AlertManager) GetAlerts(level AlertLevel, resolved bool) []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var alerts []*Alert
	for _, alert := range am.alerts {
		if alert.Level == level && alert.Resolved == resolved {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

func (am *AlertManager) startMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-am.ctx.Done():
			return
		case <-ticker.C:
			am.checkRules()
		}
	}
}

func (am *AlertManager) checkRules() {
	am.mu.RLock()
	rules := make([]*AlertRule, 0, len(am.rules))
	for _, rule := range am.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	am.mu.RUnlock()

	metrics := GetSystemMetrics()

	for _, rule := range rules {
		if time.Since(rule.LastTriggered) < rule.Cooldown {
			continue
		}

		if rule.Condition(metrics) {
			rule.LastTriggered = time.Now()
			alert := &Alert{
				Level:   rule.Level,
				Title:   rule.Name,
				Message: fmt.Sprintf("Rule %s triggered", rule.Name),
				Source:  "monitoring",
				Metadata: map[string]interface{}{
					"rule_id": rule.ID,
					"metrics": metrics,
				},
			}
			am.TriggerAlert(alert)
		}
	}
}

func (am *AlertManager) Close() {
	am.cancel()
}

type EmailAlertChannel struct {
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	From     string
	To       []string
}

func (e *EmailAlertChannel) Send(alert *Alert) error {
	return nil
}

type SlackAlertChannel struct {
	WebhookURL string
}

func (s *SlackAlertChannel) Send(alert *Alert) error {
	return nil
}

type PagerDutyAlertChannel struct {
	IntegrationKey string
}

func (p *PagerDutyAlertChannel) Send(alert *Alert) error {
	return nil
}
