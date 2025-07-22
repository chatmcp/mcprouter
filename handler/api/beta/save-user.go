package beta

import (
	"github.com/chatmcp/mcprouter/model"
	"github.com/chatmcp/mcprouter/service/api"
	"github.com/labstack/echo/v4"
)

type SaveUserRequest struct {
	Email          string `json:"email"`
	Nickname       string `json:"nickname"`
	AvatarURL      string `json:"avatar_url"`
	SigninType     string `json:"signin_type"`
	SigninProvider string `json:"signin_provider"`
	SigninOpenid   string `json:"signin_openid"`
	SigninIP       string `json:"signin_ip"`
}

func SaveUser(c echo.Context) error {
	ctx := api.GetAPIContext(c)
	req := &SaveUserRequest{}
	if err := ctx.Valid(req); err != nil {
		return ctx.RespErr(err)
	}

	user := &model.User{
		Email:          req.Email,
		Nickname:       req.Nickname,
		AvatarURL:      req.AvatarURL,
		SigninType:     req.SigninType,
		SigninProvider: req.SigninProvider,
		SigninOpenid:   req.SigninOpenid,
		SigninIP:       req.SigninIP,
	}

	if err := model.SaveUser(user); err != nil {
		return ctx.RespErr(err)
	}

	return ctx.RespData(user)
}
