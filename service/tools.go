package service

import (
	"encoding/json"
	"time"

	"github.com/chatmcp/mcprouter/model"
	"github.com/chatmcp/mcprouter/service/jsonrpc"
	"github.com/chatmcp/mcprouter/util"
)

func SaveServerTools(serverKey string, tools []*jsonrpc.Tool) error {
	modelTools := make([]*model.Tool, 0, len(tools))
	now := time.Now()

	for _, newTool := range tools {
		tool := &model.Tool{
			UUID:        util.GenUUID(),
			Name:        newTool.Name,
			ServerKey:   serverKey,
			Description: newTool.Description,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if inputSchemaBytes, err := json.Marshal(newTool.InputSchema); err == nil {
			tool.InputSchema = string(inputSchemaBytes)
		}

		if rawBytes, err := json.Marshal(newTool); err == nil {
			tool.Raw = string(rawBytes)
		}

		modelTools = append(modelTools, tool)
	}

	if err := model.UpdateServerTools(serverKey, modelTools); err != nil {
		return err
	}

	return nil
}
