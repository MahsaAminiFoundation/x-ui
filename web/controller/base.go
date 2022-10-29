package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"x-ui/logger"
	"x-ui/web/session"
)

type BaseController struct {
}

func (a *BaseController) checkLogin(c *gin.Context) {

	// TODO: api-lookup
	// TODO: num-users
	ipWhitelist := []string{"::1", "127.0.0.1", "192.184.149.170" /*R*/, "76.246.10.212" /*D*/, "143.198.72.88" /*bot*/}

	logger.Info("checkLogin, fullPath: ", c.FullPath())
	logger.Info("client ip: ", c.ClientIP())
	logger.Info("client ip is whitelisted: ", contains(ipWhitelist, c.ClientIP()))

	if strings.HasPrefix(c.FullPath(), "/xui/api") && contains(ipWhitelist, c.ClientIP()) {
		c.Next()

	} else if !session.IsLogin(c) {
		if isAjax(c) {
			pureJsonMsg(c, false, "登录时效已过，请重新登录")
		} else {
			c.Redirect(http.StatusTemporaryRedirect, c.GetString("base_path"))
		}
		c.Abort()

	} else {
		c.Next()
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
