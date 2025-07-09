package beta

import (
	"github.com/chatmcp/mcprouter/model"
	"github.com/chatmcp/mcprouter/service/api"
	"github.com/chatmcp/mcprouter/util"
	"github.com/labstack/echo/v4"
)

type AddServerRequest struct {
	Name        string `json:"name" validate:"required"`
	AuthorName  string `json:"author_name" validate:"required"`
	Title       string `json:"title" validate:"required"`
	Description string `json:"description"`
	Content     string `json:"content"`
	ServerKey   string `json:"server_key" validate:"required"`
	ServerURL   string `json:"server_url" validate:"required"`
	ConfigName  string `json:"config_name" validate:"required"`
}

func AddServer(c echo.Context) error {
	ctx := api.GetAPIContext(c)

	req := &AddServerRequest{}

	if err := ctx.Valid(req); err != nil {
		return ctx.RespErr(err)
	}

	server := &model.Server{
		UUID:        util.GenUUID(),
		Name:        req.Name,
		AuthorName:  req.AuthorName,
		Title:       req.Title,
		Description: req.Description,
		Content:     req.Content,
		ServerKey:   req.ServerKey,
		ServerURL:   req.ServerURL,
		ConfigName:  req.ConfigName,
	}

	if err := model.CreateServer(server); err != nil {
		return ctx.RespErr(err)
	}

	return ctx.RespData(server)
}
