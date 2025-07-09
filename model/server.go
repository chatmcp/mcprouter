package model

import (
	"errors"
	"time"
)

type Server struct {
	UUID        string    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   time.Time `json:"deleted_at"`
	Name        string    `json:"name"`
	AuthorName  string    `json:"author_name"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	ServerKey   string    `json:"server_key"`
	ServerURL   string    `json:"-"`
	ConfigName  string    `json:"config_name"`
}

func (s *Server) TableName() string {
	return "servers"
}

func CreateServer(server *Server) error {
	return db().Create(server).Error
}

func UpdateServer(server *Server) error {
	if server == nil || server.UUID == "" {
		return errors.New("invalid server")
	}

	return db().Where("uuid = ?", server.UUID).Updates(server).Error
}

func FindServer(name, authorName string) (*Server, error) {
	server := &Server{}

	err := db().Where("name = ?", name).
		Where("author_name = ?", authorName).
		First(server).Error

	return server, err
}

func FindServerByUUID(uuid string) (*Server, error) {
	server := &Server{}

	err := db().Where("uuid = ?", uuid).
		First(server).Error

	return server, err
}

func FindServerByKey(serverKey string) (*Server, error) {
	server := &Server{}

	err := db().Where("server_key = ?", serverKey).
		First(server).Error

	return server, err
}

func GetServers(page int, limit int) ([]*Server, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 30
	}

	servers := []*Server{}

	err := db().Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&servers).Error

	return servers, err
}
