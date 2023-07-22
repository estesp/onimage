package util

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type NoConfigSectionError struct{}
type NoConfigEntryError struct{}

func RunCommand(workdir string, command []string) (string, error) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", workdir))
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func DatetimeFromDir(dir string) string {
	parts := strings.Split(dir, "/")
	timestamp := parts[len(parts)-1]
	datestr := parts[len(parts)-2]

	return fmt.Sprintf("%s @ %s:%s", datestr, timestamp[:2], timestamp[2:4])
}

func GetDateString() string {
	return time.Now().Format("2006-01-02")
}

func GetStringFromConfig(config map[string]interface{}, key string) (string, error) {
	val, err := getValueFromConfig(config, key)
	if err != nil {
		return "", err
	}
	valStr, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("config item %s must be of type string", key)
	}
	return valStr, nil
}

func GetBoolFromConfig(config map[string]interface{}, key string) (bool, error) {
	val, err := getValueFromConfig(config, key)
	if err != nil {
		return false, err
	}
	valB, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("config item %s must be of type boolean", key)
	}
	return valB, nil
}

func GetIntFromConfig(config map[string]interface{}, key string) (int64, error) {
	val, err := getValueFromConfig(config, key)
	if err != nil {
		return 0, err
	}
	valInt, ok := val.(int64)
	if !ok {
		return 0, fmt.Errorf("config item %s must be an integer type", key)
	}
	return valInt, nil
}

func getValueFromConfig(config map[string]interface{}, key string) (interface{}, error) {
	parts := strings.Split(key, ".")
	if len(parts) == 1 {
		if val, ok := config[key]; ok {
			return val, nil
		}
		return nil, &NoConfigEntryError{}
	} else {
		// get section
		configSecInterface, ok := config[parts[0]]
		if !ok {
			return nil, &NoConfigSectionError{}
		}
		// if this is a section it should validate as a map[string]interface{}
		configSec, ok := configSecInterface.(map[string]interface{})
		if !ok {
			return nil, &NoConfigSectionError{}
		}
		if val, ok := configSec[parts[1]]; ok {
			return val, nil
		}
		return nil, &NoConfigEntryError{}
	}
}

func (n *NoConfigSectionError) Error() string {
	return "no such config section exists"
}

func (m *NoConfigEntryError) Error() string {
	return "no such config entry exists"
}
