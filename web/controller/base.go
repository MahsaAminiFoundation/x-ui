package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"x-ui/web/session"
)

type BaseController struct {
}

func (a *BaseController) checkLogin(c *gin.Context) {

	ipWhitelist := []string{"::1", "127.0.0.1", "192.184.149.170" /*R*/, "76.246.10.212" /*D*/, "143.198.72.88" /*bot*/}

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
