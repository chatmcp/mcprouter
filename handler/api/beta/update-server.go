package beta

import (
	"github.com/chatmcp/mcprouter/model"
	"github.com/chatmcp/mcprouter/service/api"
	"github.com/labstack/echo/v4"
)

type UpdateServerRequest struct {
	AddServerRequest
	UUID string `json:"uuid" validate:"required"`
}

func UpdateServer(c echo.Context) error {
	ctx := api.GetAPIContext(c)

	req := &UpdateServerRequest{}

	if err := ctx.Valid(req); err != nil {
		return ctx.RespErr(err)
	}

	server, err := model.FindServerByUUID(req.UUID)
	if err != nil {
		return ctx.RespErr(err)
	}

	server.Name = req.Name
	server.AuthorName = req.AuthorName
	server.Title = req.Title
	server.Description = req.Description
	server.Content = req.Content
	server.ServerURL = req.ServerURL
	server.ConfigName = req.ConfigName
	server.ServerKey = req.ServerKey

	if err := model.UpdateServer(server); err != nil {
		return ctx.RespErr(err)
	}

	return ctx.RespData(server)
}
