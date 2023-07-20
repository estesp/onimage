package services

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dghubble/sling"
	"github.com/sirupsen/logrus"
)

type Monitor struct {
	enabled     bool
	cronitorURL string
	cronitorKey string
	cronitorId  string
	environment string
}

type cparams struct {
	Metric      string `url:"metric,omitempty"`
	Environment string `url:"env,omitempty"`
}

type fparams struct {
	State       string `url:"state,omitempty"`
	Message     string `url:"message,omitempty"`
	Environment string `url:"env,omitempty"`
}

func NewMonitorService(config map[string]interface{}) (*Monitor, error) {

	enabled, ok := config["cronitor.enabled"].(bool)
	if !ok {
		return nil, fmt.Errorf("config file has no boolean entry for 'cronitor.enabled'")
	}
	if !enabled {
		return &Monitor{}, nil
	}

	baseUrl, ok := config["cronitor.base_url"].(string)
	if !ok {
		return nil, fmt.Errorf("config file has no string entry for 'cronitor.base_url'")
	}
	appId, ok := config["cronitor.appid"].(string)
	if !ok {
		return nil, fmt.Errorf("config file has no string entry for 'cronitor.appid'")
	}
	cronitorId, ok := config["cronitor.heartbeat_id"].(string)
	if !ok {
		return nil, fmt.Errorf("config file has no string entry for 'cronitor.heartbeat_id'")
	}
	var envStr string
	env, ok := config["cronitor.environment"]
	if ok {
		envStr, ok = env.(string)
		if !ok {
			return nil, fmt.Errorf("config file entry for 'cronitor.environment' must be a string")
		}
	}

	mon := &Monitor{
		cronitorURL: baseUrl,
		cronitorKey: appId,
		cronitorId:  cronitorId,
		environment: envStr,
	}

	return mon, nil
}

func (m *Monitor) StartCronitorPing() {
	if !m.enabled {
		return
	}
	t := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-t.C:
			m.sendHeartbeat()
		}
	}
}

func (m *Monitor) sendHeartbeat() error {

	client := http.DefaultClient
	urlBase := fmt.Sprintf("%s%s/%s", m.cronitorURL, m.cronitorKey, m.cronitorId)

	metricStr := fmt.Sprintf("error_count:%d", 0)
	params := &cparams{Metric: metricStr}
	if m.environment != "" {
		params.Environment = m.environment
	}
	req, err := sling.New().Get(urlBase).QueryStruct(params).Request()
	if err != nil {
		logrus.Errorf("failed creating sling URL: %v", err)
	}
	_, err = client.Do(req)
	if err != nil {
		logrus.Errorf("failed to ping cronitor: %v", err)
	}
	return nil
}

func (m *Monitor) SendFailure(msg string) error {

	client := http.DefaultClient
	urlBase := fmt.Sprintf("%s%s/%s", m.cronitorURL, m.cronitorKey, m.cronitorId)

	params := &fparams{State: "fail", Message: msg}
	if m.environment != "" {
		params.Environment = m.environment
	}
	req, err := sling.New().Get(urlBase).QueryStruct(params).Request()
	if err != nil {
		logrus.Errorf("failed creating sling URL: %v", err)
	}
	_, err = client.Do(req)
	if err != nil {
		logrus.Errorf("failed to ping cronitor: %v", err)
	}
	return nil
}
