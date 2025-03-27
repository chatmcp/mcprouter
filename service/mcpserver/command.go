package mcpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
)

type Config struct {
	CMD          string `json:"cmd"`
	ShareProcess bool   `json:"share_process"`
}

// GetConfig returns the command for the given key
func GetConfig(key string) *Config {
	var config Config
	configKey := fmt.Sprintf("mcp_server_configs.%s", key)

	if err := viper.UnmarshalKey(configKey, &config); err != nil {
		fmt.Printf("failed to unmarshal config for key %s: %v\n", key, err)
		return getRemoteServerConfig(key)
	}

	if config.CMD == "" {
		return getRemoteServerConfig(key)
	}

	return &config
}

// GetCommand returns the command for the given key
func GetCommand(key string) string {
	var config Config
	configKey := fmt.Sprintf("mcp_server_configs.%s", key)

	if err := viper.UnmarshalKey(configKey, &config); err != nil {
		fmt.Printf("failed to unmarshal config for key %s: %v\n", key, err)
		return getRemoteCommand(key)
	}

	if config.CMD == "" {
		return getRemoteCommand(key)
	}

	return config.CMD
}

func CanShareProcess(key string) bool {
	var config Config
	configKey := fmt.Sprintf("mcp_server_configs.%s", key)

	if err := viper.UnmarshalKey(configKey, &config); err != nil {
		fmt.Printf("failed to unmarshal config for key %s: %v\n", key, err)
		return getRemoteShareConfig(key)
	}

	if config.CMD == "" {
		return getRemoteShareConfig(key)
	}

	return config.ShareProcess
}

// getRemoteCommand returns the command for the given key from the remote API
func getRemoteCommand(key string) string {
	if config := getRemoteServerConfig(key); nil != config {
		return config.CMD
	}
	return ""
}

// getRemoteShareConfig returns the share process config for the given key from the remote API
func getRemoteShareConfig(key string) bool {
	if config := getRemoteServerConfig(key); nil != config {
		return config.ShareProcess
	}
	return true
}

// getRemoteServerConfig returns the command for the given key from the remote API
func getRemoteServerConfig(key string) *Config {
	apiUrl := viper.GetString("remote_apis.get_server_command")

	fmt.Printf("get remote command from %s, with key: %s\n", apiUrl, key)

	params := map[string]string{
		"server_key": key,
	}

	jsonData, err := json.Marshal(params)
	if err != nil {
		fmt.Printf("failed to marshal json: %v\n", err)
		return nil
	}

	response, err := http.Post(apiUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("failed to post request: %v\n", err)
		return nil
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("failed to read response body: %v\n", err)
		return nil
	}

	data := gjson.ParseBytes(body)
	command := data.Get("data.server_command").String()
	shareProcess := data.Get("data.share_process").Bool()

	return &Config{
		CMD:          command,
		ShareProcess: shareProcess,
	}
}
