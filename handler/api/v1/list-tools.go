package v1

import (
	"time"

	"github.com/chatmcp/mcprouter/service/api"
	"github.com/chatmcp/mcprouter/service/jsonrpc"
	"github.com/labstack/echo/v4"
)

type ListToolsRequest struct {
	Server string `json:"server" validate:"required"`
}

// ListTools is a handler for the list tools endpoint
func ListTools(c echo.Context) error {
	ctx := api.GetAPIContext(c)

	req := &ListToolsRequest{}

	if err := ctx.Valid(req); err != nil {
		return ctx.RespErr(err)
	}

	client, err := ctx.Connect(req.Server)
	if err != nil {
		return ctx.RespErr(err)
	}
	defer client.Close()

	proxyInfo := ctx.ProxyInfo()
	proxyInfo.RequestMethod = jsonrpc.MethodListTools

	result, err := client.ListTools()
	if err != nil {
		return ctx.RespErr(err)
	}

	proxyInfo.ResponseResult = result

	proxyInfo.ResponseTime = time.Now()
	proxyInfo.CostTime = proxyInfo.ResponseTime.Sub(proxyInfo.RequestTime).Milliseconds()

	// proxyInfoB, _ := json.Marshal(proxyInfo)
	// fmt.Printf("proxyInfo: %s\n", string(proxyInfoB))

	return ctx.RespData(result)
}
