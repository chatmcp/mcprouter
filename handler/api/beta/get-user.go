package beta

import (
	"github.com/chatmcp/mcprouter/model"
	"github.com/chatmcp/mcprouter/service/api"
	"github.com/labstack/echo/v4"
)

type GetUserRequest struct {
	UUID  string `json:"uuid"`
	Email string `json:"email"`
}

func GetUser(c echo.Context) error {
	ctx := api.GetAPIContext(c)

	req := &GetUserRequest{}

	if err := ctx.Valid(req); err != nil {
		return ctx.RespErr(err)
	}

	var err error
	var user *model.User

	if req.UUID != "" {
		user, err = model.FindUserByUUID(req.UUID)
	} else {
		user, err = model.FindUserByEmail(req.Email)
	}

	if err != nil {
		return ctx.RespErr(err)
	}

	return ctx.RespData(user)
}
