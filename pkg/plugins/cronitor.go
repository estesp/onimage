package plugins

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dghubble/sling"
	"github.com/estesp/onimage/pkg/util"
	"github.com/sirupsen/logrus"
)

type Cronitor struct {
	enabled     bool
	cronitorURL string
	cronitorKey string
	cronitorId  string
	environment string
	errChan     chan error
	errCnt      uint
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

func InitCronitor(config map[string]interface{}, errChan chan error) (*Cronitor, error) {

	enabled, err := util.GetBoolFromConfig(config, "cronitor.enabled")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve 'cronitor.enabled' from config: %w", err)
	}
	if !enabled {
		return &Cronitor{}, nil
	}

	baseUrl, err := util.GetStringFromConfig(config, "cronitor.base_url")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve 'cronitor.base_url' from config: %w", err)
	}
	appId, err := util.GetStringFromConfig(config, "cronitor.appid")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve 'cronitor.appid' from config: %w", err)
	}
	cronitorId, err := util.GetStringFromConfig(config, "cronitor.heartbeat_id")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve 'cronitor.heartbeat_id' from config: %w", err)
	}
	// we can ignore any errors as this is not a required field for the configuration
	env, _ := util.GetStringFromConfig(config, "cronitor.environment")

	mon := &Cronitor{
		cronitorURL: baseUrl,
		cronitorKey: appId,
		cronitorId:  cronitorId,
		environment: env,
		enabled:     enabled,
		errChan:     errChan,
	}

	return mon, nil
}

func (m *Cronitor) StartPing() {
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

func (m *Cronitor) sendHeartbeat() error {
	if !m.enabled {
		return nil
	}

	client := http.DefaultClient
	urlBase := fmt.Sprintf("%s%s/%s", m.cronitorURL, m.cronitorKey, m.cronitorId)

	metricStr := fmt.Sprintf("error_count:%d", m.errCnt)
	m.errCnt = 0
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

func (m *Cronitor) SendFailure(msg string) error {
	if !m.enabled {
		return nil
	}
	// # of errors will be reported during each heartbeat ping
	m.errCnt++

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
