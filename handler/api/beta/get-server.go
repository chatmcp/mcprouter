package beta

import (
	"time"

	"github.com/chatmcp/mcprouter/model"
	"github.com/chatmcp/mcprouter/service"
	"github.com/chatmcp/mcprouter/service/api"
	"github.com/labstack/echo/v4"
)

type GetServerRequest struct {
	Name       string `json:"name" validate:"required"`
	AuthorName string `json:"author_name" validate:"required"`
}

func GetServer(c echo.Context) error {
	ctx := api.GetAPIContext(c)

	req := &GetServerRequest{}
	if err := ctx.Valid(req); err != nil {
		return ctx.RespErr(err)
	}

	server, err := model.FindServer(req.Name, req.AuthorName)
	if err != nil {
		return ctx.RespErr(err)
	}

	tools, err := model.GetServerTools(server.ServerKey)
	if err != nil {
		return ctx.RespErr(err)
	}

	if len(tools) == 0 || server.UpdatedAt.Before(time.Now().Add(-time.Minute*10)) {
		// get server tools
		client, err := ctx.Connect(server.ServerKey)
		if err != nil {
			return ctx.RespErr(err)
		}
		defer client.Close()

		result, err := client.ListTools()
		if err != nil {
			return ctx.RespErr(err)
		}

		service.SaveServerTools(server.ServerKey, result.Tools)

		tools, err = model.GetServerTools(server.ServerKey)
		if err != nil {
			return ctx.RespErr(err)
		}
	}

	server.Tools = tools

	return ctx.RespData(server)
}
