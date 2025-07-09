package mcpserver

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/chatmcp/mcprouter/model"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
)

// ServerConfig is the config for the remote mcp server
type ServerConfig struct {
	ServerUUID       string `json:"server_uuid,omitempty" mapstructure:"server_uuid,omitempty"`
	ServerName       string `json:"server_name,omitempty" mapstructure:"server_name,omitempty"`
	ServerConfigName string `json:"server_config_name,omitempty" mapstructure:"server_config_name,omitempty"`
	ServerKey        string `json:"server_key,omitempty" mapstructure:"server_key,omitempty"`
	Command          string `json:"command,omitempty" mapstructure:"command,omitempty"`
	CommandHash      string `json:"command_hash,omitempty" mapstructure:"command_hash,omitempty"`
	ShareProcess     bool   `json:"share_process,omitempty" mapstructure:"share_process"`
	ServerType       string `json:"server_type,omitempty" mapstructure:"server_type,omitempty"`
	ServerURL        string `json:"server_url,omitempty" mapstructure:"server_url,omitempty"`
	ServerParams     string `json:"server_params,omitempty" mapstructure:"server_params,omitempty"`
}

// GetServerConfig returns the config for the given key
func GetServerConfig(key string) *ServerConfig {
	config := &ServerConfig{}
	err := viper.UnmarshalKey(fmt.Sprintf("mcp_servers.%s", key), config)
	log.Printf("get server config: %s from local env: %+v, with error: %v\n", key, config, err)

	if (config.Command == "" && config.ServerURL == "") && viper.GetBool("app.use_db") {
		config, err = getDBServerConfig(key)
		if err != nil {
			log.Printf("get db config failed: %v\n", err)
		}
	}

	if config == nil || (config.Command == "" && config.ServerURL == "") {
		log.Printf("get local config failed: %v, try to get remote config\n", err)

		config, err = getRemoteServerConfig(key)
		log.Printf("get remote config: %+v\n", config)
		if err != nil {
			log.Printf("get remote config failed: %v\n", err)
			return nil
		}
	}

	return config
}

// getDBServerConfig returns the config for the given key from the database
func getDBServerConfig(key string) (*ServerConfig, error) {
	// serverkey, err := model.FindServerkeyByServerKey(key)
	// if err != nil {
	// 	return nil, err
	// }

	server, err := model.FindServerByKey(key)
	if err != nil {
		return nil, err
	}

	return &ServerConfig{
		ServerUUID:       server.UUID,
		ServerName:       server.Name,
		ServerConfigName: server.ConfigName,
		ServerKey:        server.ServerKey,
		Command:          "",
		CommandHash:      "",
		ShareProcess:     true,
		ServerType:       "rest",
		ServerURL:        server.ServerURL,
		ServerParams:     "",
	}, nil
}

// getRemoteServerConfig returns the config for the given key from the remote API
func getRemoteServerConfig(key string) (*ServerConfig, error) {
	apiUrl := viper.GetString("remote_apis.get_server_config")

	params := map[string]string{
		"server_key": key,
	}

	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	log.Printf("get remote config from %s, with params: %s\n", apiUrl, jsonData)

	response, err := http.Post(apiUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	data := gjson.ParseBytes(body)
	log.Printf("get remote config with key: %s, response: %s\n", key, data.String())

	if data.Get("code").Int() != 0 {
		return nil, fmt.Errorf("get remote config failed: %s", data.Get("message").String())
	}

	config := &ServerConfig{}
	if err = json.Unmarshal([]byte(data.Get("data").String()), config); err != nil {
		return nil, err
	}

	if config.Command != "" && config.CommandHash == "" {
		config.CommandHash = fmt.Sprintf("%x", md5.Sum([]byte(config.Command)))
	}

	return config, nil
}
