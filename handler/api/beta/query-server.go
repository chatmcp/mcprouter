package beta

import (
	"github.com/chatmcp/mcprouter/model"
	"github.com/chatmcp/mcprouter/service/api"
	"github.com/labstack/echo/v4"
)

type QueryServerRequest struct {
	Name       string `json:"name" validate:"required"`
	AuthorName string `json:"author_name" validate:"required"`
}

func QueryServer(c echo.Context) error {
	ctx := api.GetAPIContext(c)

	req := &QueryServerRequest{}
	if err := ctx.Valid(req); err != nil {
		return ctx.RespErr(err)
	}

	server, err := model.FindServer(req.Name, req.AuthorName)
	if err != nil {
		return ctx.RespErr(err)
	}

	return ctx.RespData(server)
}
