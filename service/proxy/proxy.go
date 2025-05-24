package proxy

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/chatmcp/mcprouter/model"
	"github.com/chatmcp/mcprouter/util"
)

// ProxyInfo is the info for the proxy
type ProxyInfo struct {
	JSONRPCVersion     string      `json:"jsonrpc_version"`
	ProtocolVersion    string      `json:"protocol_version"`
	ConnectionTime     time.Time   `json:"connection_time"`
	ClientName         string      `json:"client_name"`
	ClientVersion      string      `json:"client_version"`
	RequestMethod      string      `json:"request_method"`
	RequestParams      interface{} `json:"request_params"`
	RequestID          interface{} `json:"request_id"`
	RequestTime        time.Time   `json:"request_time"`
	RequestFrom        string      `json:"request_from"`
	SessionID          string      `json:"session_id"`
	ServerUUID         string      `json:"server_uuid"`
	ServerKey          string      `json:"server_key"`
	ServerConfigName   string      `json:"server_config_name"`
	ServerShareProcess bool        `json:"server_share_process"`
	ServerType         string      `json:"server_type"`
	ServerURL          string      `json:"server_url"`
	ServerCommand      string      `json:"server_command"`
	ServerCommandHash  string      `json:"server_command_hash"`
	ServerName         string      `json:"server_name"`
	ServerVersion      string      `json:"server_version"`
	ResponseTime       time.Time   `json:"response_time"`
	ResponseResult     interface{} `json:"response_result"`
	ResponseError      string      `json:"response_error"`
	CostTime           int64       `json:"cost_time"`
}

// GetSessionID returns the session ID for the proxy info
func (p *ProxyInfo) GetSessionID() string {
	return util.MD5(p.ServerKey)
}

// ToServerLog converts a ProxyInfo to a ServerLog
func (p *ProxyInfo) ToServerLog() *model.ServerLog {
	sl := &model.ServerLog{
		JSONRPCVersion:     p.JSONRPCVersion,
		ProtocolVersion:    p.ProtocolVersion,
		ConnectionTime:     p.ConnectionTime,
		ClientName:         p.ClientName,
		ClientVersion:      p.ClientVersion,
		RequestMethod:      p.RequestMethod,
		RequestTime:        p.RequestTime,
		RequestFrom:        p.RequestFrom,
		SessionID:          p.SessionID,
		ServerUUID:         p.ServerUUID,
		ServerKey:          p.ServerKey,
		ServerConfigName:   p.ServerConfigName,
		ServerShareProcess: p.ServerShareProcess,
		ServerType:         p.ServerType,
		ServerURL:          p.ServerURL,
		ServerCommand:      p.ServerCommand,
		ServerCommandHash:  p.ServerCommandHash,
		ServerName:         p.ServerName,
		ServerVersion:      p.ServerVersion,
		ResponseTime:       p.ResponseTime,
		ResponseError:      p.ResponseError,
		CostTime:           p.CostTime,
	}

	if p.RequestID != nil {
		sl.RequestID = fmt.Sprintf("%v", p.RequestID)
	}

	if p.RequestParams != nil {
		if b, err := json.Marshal(p.RequestParams); err == nil {
			sl.RequestParams = string(b)
		}
	}

	if p.ResponseResult != nil {
		if b, err := json.Marshal(p.ResponseResult); err == nil {
			sl.ResponseResult = string(b)
		}
	}

	return sl
}
