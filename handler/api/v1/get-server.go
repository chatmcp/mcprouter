package v1

import (
	"github.com/chatmcp/mcprouter/model"
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

	return ctx.RespData(server)
}
