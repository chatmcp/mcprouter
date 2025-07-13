package model

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Tool struct {
	UUID        string    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   time.Time `json:"-"`
	Name        string    `json:"name"`
	ServerKey   string    `json:"server_key"`
	Description string    `json:"description"`
	InputSchema string    `json:"input_schema" gorm:"column:input_schema"`
	Raw         string    `json:"-"`
}

func (s *Tool) TableName() string {
	return "tools"
}

func CreateTool(tool *Tool) error {
	return db().Create(tool).Error
}

func UpdateTool(tool *Tool) error {
	if tool == nil || tool.UUID == "" {
		return errors.New("invalid tool")
	}

	return db().Where("uuid = ?", tool.UUID).Updates(tool).Error
}

func FindTool(name, serverKey string) (*Tool, error) {
	tool := &Tool{}

	err := db().Where("name = ?", name).
		Where("server_key = ?", serverKey).
		First(tool).Error

	return tool, err
}

func FindToolByUUID(uuid string) (*Tool, error) {
	tool := &Tool{}

	err := db().Where("uuid = ?", uuid).
		First(tool).Error

	return tool, err
}

func GetServerTools(serverKey string) ([]*Tool, error) {
	tools := []*Tool{}

	err := db().Where("server_key = ?", serverKey).
		Find(&tools).Error

	return tools, err
}

// GetServerToolsWithContext get server tools with context
func GetServerToolsWithContext(ctx context.Context, serverKey string) ([]*Tool, error) {
	tools := []*Tool{}

	err := db().WithContext(ctx).Where("server_key = ?", serverKey).
		Find(&tools).Error

	return tools, err
}

func GetTools(page int, limit int) ([]*Tool, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 30
	}

	tools := []*Tool{}

	err := db().Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&tools).Error

	return tools, err
}

// UpdateServerTools create or update server tools
func UpdateServerTools(serverKey string, tools []*Tool) error {
	return db().Transaction(func(tx *gorm.DB) error {
		// Delete existing tools for the server
		if err := tx.Where("server_key = ?", serverKey).Delete(&Tool{}).Error; err != nil {
			return err
		}

		// Batch insert new tools
		if len(tools) > 0 {
			if err := tx.Create(&tools).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&Server{}).Where("server_key = ?", serverKey).Update("updated_at", time.Now()).Error; err != nil {
			return err
		}

		return nil
	})
}
