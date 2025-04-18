package services

import (
	"fmt"

	"github.com/estesp/onimage/pkg/plugins"
	"github.com/estesp/onimage/pkg/util"
	"github.com/sirupsen/logrus"
)

type Monitor interface {
	StartPing()
	SendFailure(string) error
}

func NewMonitorService(config map[string]interface{}, errChan chan error) (Monitor, error) {

	var (
		monitor Monitor
		err     error
	)
	enabledHP, err := util.GetBoolFromConfig(config, "hyperping.enabled")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve 'hyperping.enabled' from config: %w", err)
	}
	enabledCR, err := util.GetBoolFromConfig(config, "cronitor.enabled")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve 'cronitor.enabled' from config: %w", err)
	}

	if enabledCR && enabledHP {
		logrus.Warn("Can't have both monitoring services enabled; using cronitor by default")
	}
	if enabledCR {
		return plugins.InitCronitor(config, errChan)
	}
	if enabledHP {
		return plugins.InitHyperping(config, errChan)
	}
	// no monitor enabled
	return monitor, nil
}
