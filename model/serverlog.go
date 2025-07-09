package model

import (
	"errors"
	"fmt"
	"time"
)

// ServerLog is the model for the server log
type ServerLog struct {
	JSONRPCVersion     string    `json:"jsonrpc_version" gorm:"column:jsonrpc_version"`
	ProtocolVersion    string    `json:"protocol_version" gorm:"column:protocol_version"`
	ConnectionTime     time.Time `json:"connection_time" gorm:"column:connection_time;type:timestamptz"`
	ClientName         string    `json:"client_name" gorm:"column:client_name"`
	ClientVersion      string    `json:"client_version" gorm:"column:client_version"`
	ClientURL          string    `json:"client_url" gorm:"column:client_url"`
	RequestMethod      string    `json:"request_method" gorm:"column:request_method"`
	RequestParams      string    `json:"request_params" gorm:"column:request_params"`
	RequestID          string    `json:"request_id" gorm:"column:request_id"`
	RequestTime        time.Time `json:"request_time" gorm:"column:request_time;type:timestamptz"`
	RequestFrom        string    `json:"request_from" gorm:"column:request_from"`
	SessionID          string    `json:"session_id" gorm:"column:session_id"`
	ServerUUID         string    `json:"server_uuid" gorm:"column:server_uuid"`
	ServerKey          string    `json:"server_key" gorm:"column:server_key"`
	ServerConfigName   string    `json:"server_config_name" gorm:"column:server_config_name"`
	ServerShareProcess bool      `json:"server_share_process" gorm:"column:server_share_process"`
	ServerType         string    `json:"server_type" gorm:"column:server_type"`
	ServerURL          string    `json:"server_url" gorm:"column:server_url"`
	ServerCommand      string    `json:"server_command" gorm:"column:server_command"`
	ServerCommandHash  string    `json:"server_command_hash" gorm:"column:server_command_hash"`
	ServerName         string    `json:"server_name" gorm:"column:server_name"`
	ServerVersion      string    `json:"server_version" gorm:"column:server_version"`
	ResponseTime       time.Time `json:"response_time" gorm:"column:response_time;type:timestamptz"`
	ResponseResult     string    `json:"response_result" gorm:"column:response_result"`
	ResponseError      string    `json:"response_error" gorm:"column:response_error"`
	CostTime           int64     `json:"cost_time" gorm:"column:cost_time"`
}

// TableName returns the table name for the server log
func (s *ServerLog) TableName() string {
	return "serverlogs"
}

// CreateServerLog creates a new server log
func CreateServerLog(sl *ServerLog) error {
	if sl == nil || sl.RequestMethod == "" {
		return errors.New("invalid server log")
	}

	fmt.Printf("save server log: %+v\n", sl)

	return db().Table(sl.TableName()).Create(sl).Error
}
