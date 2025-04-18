package plugins

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dghubble/sling"
	"github.com/estesp/onimage/pkg/util"
	"github.com/sirupsen/logrus"
)

type Hyperping struct {
	enabled bool
	baseURL string
	urlKey  string
	errChan chan error
}

func InitHyperping(config map[string]interface{}, errChan chan error) (*Hyperping, error) {

	enabled, err := util.GetBoolFromConfig(config, "hyperping.enabled")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve 'hyperping.enabled' from config: %w", err)
	}
	if !enabled {
		return &Hyperping{}, nil
	}

	baseUrl, err := util.GetStringFromConfig(config, "hyperping.base_url")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve 'hyperping.base_url' from config: %w", err)
	}
	urlKey, err := util.GetStringFromConfig(config, "hyperping.key")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve 'hyperping.key' from config: %w", err)
	}

	mon := &Hyperping{
		baseURL: baseUrl,
		urlKey:  urlKey,
		enabled: enabled,
		errChan: errChan,
	}

	return mon, nil
}

func (m *Hyperping) StartPing() {
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

func (m *Hyperping) sendHeartbeat() error {
	if !m.enabled {
		return nil
	}

	client := http.DefaultClient
	urlBase := fmt.Sprintf("%s/%s", m.baseURL, m.urlKey)

	req, err := sling.New().Get(urlBase).Request()
	if err != nil {
		logrus.Errorf("failed creating sling URL: %v", err)
	}
	_, err = client.Do(req)
	if err != nil {
		logrus.Errorf("failed to send hyperping: %v", err)
	}
	return nil
}

func (m *Hyperping) SendFailure(msg string) error {
	return nil
}
