package monitoring

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// AlertManager handles alerting and notifications
type AlertManager struct {
	config       HealthConfig
	activeAlerts map[string]*ActiveAlert
}

// ActiveAlert represents an active alert
type ActiveAlert struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	Timestamp   time.Time              `json:"timestamp"`
	Component   string                 `json:"component"`
	Status      string                 `json:"status"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SlackMessage represents a Slack webhook message
type SlackMessage struct {
	Text        string       `json:"text"`
	Username    string       `json:"username,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment represents a Slack attachment
type Attachment struct {
	Color     string  `json:"color"`
	Title     string  `json:"title"`
	Text      string  `json:"text"`
	Timestamp int64   `json:"ts"`
	Fields    []Field `json:"fields,omitempty"`
}

// Field represents a Slack field
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NewAlertManager creates a new alert manager
func NewAlertManager(config HealthConfig) *AlertManager {
	return &AlertManager{
		config:       config,
		activeAlerts: make(map[string]*ActiveAlert),
	}
}

// TriggerAlert triggers an alert through configured channels
func (am *AlertManager) TriggerAlert(alert ActiveAlert) {
	logrus.Warnf("Alert triggered: %s - %s", alert.Name, alert.Description)

	// Store active alert
	am.activeAlerts[alert.Name] = &alert

	// Send to Slack if configured
	if am.config.SlackWebhookURL != "" {
		am.sendSlackAlert(alert)
	}

	// Send to PagerDuty if configured
	if am.config.PagerDutyKey != "" {
		am.sendPagerDutyAlert(alert)
	}
}

// sendSlackAlert sends an alert to Slack
func (am *AlertManager) sendSlackAlert(alert ActiveAlert) {
	color := "danger"
	emoji := "🚨"

	switch alert.Severity {
	case "warning":
		color = "warning"
		emoji = "⚠️"
	case "info":
		color = "good"
		emoji = "ℹ️"
	}

	attachment := Attachment{
		Color:     color,
		Title:     fmt.Sprintf("%s %s", emoji, alert.Name),
		Text:      alert.Description,
		Timestamp: alert.Timestamp.Unix(),
		Fields: []Field{
			{
				Title: "Component",
				Value: alert.Component,
				Short: true,
			},
			{
				Title: "Severity",
				Value: alert.Severity,
				Short: true,
			},
			{
				Title: "Time",
				Value: alert.Timestamp.Format("2006-01-02 15:04:05 UTC"),
				Short: true,
			},
		},
	}

	message := SlackMessage{
		Text:        fmt.Sprintf("MimirInsights Alert: %s", alert.Name),
		Username:    "MimirInsights",
		IconEmoji:   ":warning:",
		Attachments: []Attachment{attachment},
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		logrus.Errorf("Failed to marshal Slack message: %v", err)
		return
	}

	resp, err := http.Post(am.config.SlackWebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logrus.Errorf("Failed to send Slack alert: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.Errorf("Slack webhook returned status %d", resp.StatusCode)
	} else {
		logrus.Infof("Slack alert sent successfully for: %s", alert.Name)
	}
}

// sendPagerDutyAlert sends an alert to PagerDuty
func (am *AlertManager) sendPagerDutyAlert(alert ActiveAlert) {
	// PagerDuty Events API v2 payload
	payload := map[string]interface{}{
		"routing_key":  am.config.PagerDutyKey,
		"event_action": "trigger",
		"dedup_key":    fmt.Sprintf("mimir-insights-%s", alert.Component),
		"payload": map[string]interface{}{
			"summary":        alert.Name,
			"source":         "mimir-insights",
			"severity":       alert.Severity,
			"component":      alert.Component,
			"group":          "mimir-insights",
			"class":          "health-check",
			"custom_details": alert.Metadata,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		logrus.Errorf("Failed to marshal PagerDuty payload: %v", err)
		return
	}

	resp, err := http.Post("https://events.pagerduty.com/v2/enqueue", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logrus.Errorf("Failed to send PagerDuty alert: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		logrus.Errorf("PagerDuty API returned status %d", resp.StatusCode)
	} else {
		logrus.Infof("PagerDuty alert sent successfully for: %s", alert.Name)
	}
}

// ResolveAlert resolves an active alert
func (am *AlertManager) ResolveAlert(alertName string) {
	if alert, exists := am.activeAlerts[alertName]; exists {
		alert.Status = "resolved"
		delete(am.activeAlerts, alertName)

		logrus.Infof("Alert resolved: %s", alertName)

		// Send resolution to PagerDuty if configured
		if am.config.PagerDutyKey != "" {
			am.sendPagerDutyResolution(alertName, alert.Component)
		}
	}
}

// sendPagerDutyResolution sends a resolution to PagerDuty
func (am *AlertManager) sendPagerDutyResolution(alertName, component string) {
	payload := map[string]interface{}{
		"routing_key":  am.config.PagerDutyKey,
		"event_action": "resolve",
		"dedup_key":    fmt.Sprintf("mimir-insights-%s", component),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		logrus.Errorf("Failed to marshal PagerDuty resolution: %v", err)
		return
	}

	resp, err := http.Post("https://events.pagerduty.com/v2/enqueue", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logrus.Errorf("Failed to send PagerDuty resolution: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		logrus.Errorf("PagerDuty resolution API returned status %d", resp.StatusCode)
	} else {
		logrus.Infof("PagerDuty resolution sent for: %s", alertName)
	}
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []*ActiveAlert {
	var alerts []*ActiveAlert
	for _, alert := range am.activeAlerts {
		alerts = append(alerts, alert)
	}
	return alerts
}

// GetAlertHistory returns alert history (placeholder)
func (am *AlertManager) GetAlertHistory(hours int) []ActiveAlert {
	// In production, this would query a database or log store
	// For now, return empty slice
	return []ActiveAlert{}
}
