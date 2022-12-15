package controller

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
	"x-ui/web/session"
)

type BaseController struct {
	ipWhitelist []string
}

func (a *BaseController) checkLogin(c *gin.Context) {
	var ipWhitelist []string
	if a.ipWhitelist == nil || len(a.ipWhitelist) == 0 {
		log.Print("calling loadConfigFromS3")
		a.ipWhitelist = loadConfigFromS3()

		log.Printf("ipWhitelist: %@, clientIP: %@", ipWhitelist)
	}
	ipWhitelist = a.ipWhitelist

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

func loadConfigFromS3() []string {

	url := "https://nofiltervpn.s3.eu-central-1.amazonaws.com/mahsa_amini.vpn.config"

	httpClient := http.Client{
		Timeout: time.Second * 10, // Timeout after 2 seconds
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		log.Print(getErr)
		return []string{}
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Print(getErr)
		return []string{}
	}

	var data map[string]interface{}
	jsonErr := json.Unmarshal(body, &data)
	if jsonErr != nil {
		log.Print(getErr)
		return []string{}
	}

	whitelists, ok := data["whitelist"]
	if !ok {
		log.Print("There is no whitelist in config")
		return []string{}
	}

	whitelistsArr := whitelists.([]any)

	whitelistsStrArr := make([]string, len(whitelistsArr))
	for i := 0; i < len(whitelistsArr); i++ {
		strElem, ok := whitelistsArr[i].(string)
		if !ok {
			log.Print("got slice with non Stringer elements")
		} else {
			whitelistsStrArr[i] = strElem
		}
	}

	log.Printf("whitelistsArr: %s", whitelistsArr)
	return whitelistsStrArr
}
