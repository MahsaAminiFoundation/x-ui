package controller

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sethvargo/go-password/password"
	"gorm.io/gorm"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"x-ui/database/model"
	"x-ui/logger"
	"x-ui/web/entity"
	"x-ui/web/global"
	"x-ui/web/service"
)

type APIController struct {
	inboundService service.InboundService
	xrayService    service.XrayService
	userService    service.UserService
	settingService service.SettingService
}

func NewAPIController(g *gin.RouterGroup) *APIController {
	a := &APIController{}
	a.initRouter(g)
	a.startTask()
	return a
}

func (a *APIController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/api")

	g.GET("/list_users", a.listUsers)
	g.GET("/num_users", a.numUsers)
	g.POST("/add_user", a.addUser)
	g.POST("/delete_user", a.deleteUser)
	g.POST("/remaining_quota", a.remainingQuota)
	g.POST("/add_quota", a.addQuota)

}

func (a *APIController) startTask() {
	webServer := global.GetWebServer()
	c := webServer.GetCron()
	c.AddFunc("@every 10s", func() {
		if a.xrayService.IsNeedRestartAndSetFalse() {
			err := a.xrayService.RestartXray(false)
			if err != nil {
				logger.Error("restart xray failed:", err)
			}
		}
	})
}

func (a *APIController) numUsers(c *gin.Context) {
	inbounds, err := a.inboundService.GetAllInbounds()
	if err != nil {
		jsonMsg(c, "获取", err)
		return
	}

	m := entity.Msg{
		Obj:     nil,
		Success: true,
		Msg:     strconv.Itoa(len(inbounds)),
	}
	c.JSON(http.StatusOK, m)
}

func (a *APIController) listUsers(c *gin.Context) {
	inbounds, err := a.inboundService.GetAllInbounds()
	if err != nil {
		jsonMsg(c, "获取", err)
		return
	}
	jsonObj(c, inbounds, nil)
}

type VlessObject struct {
	Version string `json:"v"`
	Ps      string `json:"ps"`
	Address string `json:"add"`
	Port    int    `json:"port"`
	UUID    string `json:"id"`
	AlterId int64  `json:"aid"`
	Net     string `json:"net"`
	Type    string `json:"type"`
	Host    string `json:"host"`
	Path    string `json:"path"`
	TLS     string `json:"tls"`
}

func (a *APIController) remainingQuota(c *gin.Context) {
	inbound := &model.Inbound{}
	err := c.ShouldBind(inbound)
	if err != nil {
		jsonMsg(c, "添加", err)
		return
	}

	inbounds, err := a.inboundService.GetInboundsWithRemark(inbound.Remark)
	if err != nil || len(inbounds) == 0 {
		jsonMsg(c, "获取", gorm.ErrRecordNotFound)
		return
	}

	protocols := make([]entity.ProtocolBandWidth, len(inbounds))
	for index, _ := range protocols {
		protocols[index].Protocol = string(inbounds[index].Protocol)
		protocols[index].TotalBandwidth = int(inbounds[index].Total / 1000 / 1000)
		protocols[index].RemainingBandwidth = int((inbounds[index].Total - inbounds[index].Up - inbounds[index].Down) / 1000 / 1000)
	}

	m := entity.QuotaResp{
		Success:   true,
		Protocols: protocols,
	}
	c.JSON(http.StatusOK, m)
}

func (a *APIController) addUser(c *gin.Context) {
	inbound := &model.Inbound{}
	err := c.ShouldBind(inbound)
	if err != nil {
		jsonMsg(c, "添加", err)
		return
	}
	user, err := a.userService.GetFirstUser()
	if err != nil {
		jsonMsg(c, "添加", err)
		return
	}

	var password string
	var userUUIDstring string
	requestedProtocol := inbound.Protocol
	if requestedProtocol == "vmess" {
		userUUIDstring = a.setVmessSettingsForInbound(inbound)

	} else if requestedProtocol == "trojan" {
		logger.Info("Setting protocol as trojan")
		password, err = a.setTrojanSettingsForInbound(inbound)
		if err != nil {
			jsonMsg(c, "添加", err)
			return
		}

	} else if requestedProtocol == "vless" {
		logger.Info("Setting protocol as vless")
		userUUIDstring = a.setVlessSettingsForInbound(inbound)
		if err != nil {
			jsonMsg(c, "添加", err)
			return
		}
	}

	inbound.Port = 20000 + rand.Intn(30000) /*port between 20,000 to 50,000*/
	var url string

	inbound.UserId = user.Id
	inbound.Total = inbound.Total * 1024 * 1024 * 1024
	inbound.Enable = true
	inbound.Tag = fmt.Sprintf("inbound-%v", inbound.Port)
	err = a.inboundService.AddInbound(inbound)

	if err != nil && strings.HasPrefix(err.Error(), "ALREADY_EXISTS") {
		dbInbound, err := a.inboundService.GetInboundWithRemarkProtocol(inbound.Remark, string(inbound.Protocol))
		if err != nil {
			jsonMsg(c, "添加", err)
			return
		}

		//Inbound exists with the right protocol
		inbound = dbInbound
		var settings map[string]any
		json.Unmarshal([]byte(inbound.Settings), &settings)

		clients := settings["clients"].([]any)
		client := clients[0].(map[string]any)
		if inbound.Protocol == "vmess" || inbound.Protocol == "vless" {
			userUUIDstring = client["id"].(string)
		} else if inbound.Protocol == "trojan" {
			password = client["password"].(string)
		}
	}

	hostname := a.getHostname(c, string(inbound.Protocol))

	if inbound.Protocol == "vmess" {
		url, err = a.getVmessURL(inbound, userUUIDstring, hostname)
		if err != nil {
			jsonMsg(c, "添加", err)
			return
		}

	} else if inbound.Protocol == "trojan" {
		url = a.getTrojanURL(inbound, password, hostname)

	} else if inbound.Protocol == "vless" {
		url = a.getVlessURL(inbound, userUUIDstring, hostname)

	}

	m := entity.UserAddResp{
		Obj:                nil,
		Success:            true,
		Msg:                url,
		TotalBandwidth:     int(inbound.Total / 1000 / 1000),
		RemainingBandwidth: int((inbound.Total - inbound.Up - inbound.Down) / 1000 / 1000),
	}
	c.JSON(http.StatusOK, m)

	a.xrayService.SetToNeedRestart()
}

func (a *APIController) deleteUser(c *gin.Context) {
	inbound := &model.Inbound{}
	err := c.ShouldBind(inbound)
	if err != nil {
		jsonMsg(c, "添加", err)
		return
	}

	inbounds, err := a.inboundService.GetInboundsWithRemark(inbound.Remark)
	if err != nil || len(inbounds) == 0 {
		jsonMsg(c, "Not found", gorm.ErrRecordNotFound)
		return
	}

	for _, inbound := range inbounds {
		err = a.inboundService.DelInbound(inbound.Id)
		if err != nil {
			jsonMsg(c, "delete", err)
			return
		}
	}

	jsonMsg(c, "delete", err)
	a.xrayService.SetToNeedRestart()
}

func (a *APIController) addQuota(c *gin.Context) {
	inputInbound := &model.Inbound{}
	err := c.ShouldBind(inputInbound)
	if err != nil {
		jsonMsg(c, "添加", err)
		return
	}

	addSize := inputInbound.Total * 1024 * 1024 * 1024
	inbound, err := a.inboundService.GetInboundWithRemarkProtocol(inputInbound.Remark, string(inputInbound.Protocol))
	if err != nil {
		jsonMsg(c, "获取", gorm.ErrRecordNotFound)
		return
	}

	log.Printf("current total: %@, newTotal: %@, enable: %@", inbound.Total, inbound.Total+addSize, inbound.Enable)

	inbound.Total += addSize
	inbound.Enable = true
	a.inboundService.UpdateInbound(inbound)

	inbounds, err := a.inboundService.GetInboundsWithRemark(inbound.Remark)
	if err != nil || len(inbounds) == 0 {
		jsonMsg(c, "获取", gorm.ErrRecordNotFound)
		return
	}

	protocols := make([]entity.ProtocolBandWidth, len(inbounds))
	for index, _ := range protocols {
		protocols[index].Protocol = string(inbounds[index].Protocol)
		protocols[index].TotalBandwidth = int(inbounds[index].Total / 1000 / 1000)
		protocols[index].RemainingBandwidth = int((inbounds[index].Total - inbounds[index].Up - inbounds[index].Down) / 1000 / 1000)
	}

	m := entity.QuotaResp{
		Success:   true,
		Protocols: protocols,
	}

	c.JSON(http.StatusOK, m)
}

func (a *APIController) getHostname(c *gin.Context, protocol string) string {
	var hostname string
	var err error

	if protocol == "trojan" {
		hostname, err = a.settingService.GetServerName()
	} else {
		hostname, err = a.settingService.GetServerIP()
	}

	if (err != nil) || (hostname == "localhost" || hostname == "127.0.0.1") {
		res1 := strings.Split(c.Request.Host, ":")
		hostname = res1[0]
	}

	return hostname
}

func (a *APIController) setVmessSettingsForInbound(inbound *model.Inbound) string {
	userUUID := uuid.New()
	inbound.Settings = fmt.Sprintf(
		`{
            "clients":[{
                "id":"%s",
                "flow":"xtls-rprx-direct"
            }],
            "decryption":"none",
            "fallbacks":[]
        }`,
		userUUID)
	userUUIDstring := userUUID.String()

	inbound.StreamSettings = `{
      "network": "tcp",
      "security": "none",
      "tcpSettings": {
        "acceptProxyProtocol": false,
        "header": {
          "type": "none"
        }
      }
    }`

	inbound.Sniffing = `{
        "enabled":true,
        "destOverride":[
            "http",
            "tls"
        ]
    }`

	return userUUIDstring
}

func (a *APIController) setTrojanSettingsForInbound(inbound *model.Inbound) (string, error) {
	password, err := password.Generate(10, 4, 0, false, true)
	if err != nil {
		return "", err
	}
	inbound.Settings = fmt.Sprintf(`{
                          "clients": [
                            {
                              "password": "%s",
                              "flow": "xtls-rprx-direct"
                            }
                          ],
                          "fallbacks": []
                        }`, password)

	certificateFile, err := a.settingService.GetCertFile()
	if err != nil {
		return "", err
	}

	keyFile, err := a.settingService.GetKeyFile()
	if err != nil {
		return "", err
	}

	inbound.StreamSettings = fmt.Sprintf(`{
      "network": "tcp",
      "security": "xtls",
      "xtlsSettings": {
        "serverName": "",
        "certificates": [
          {
            "certificateFile": "%s",
            "keyFile": "%s"
          }
        ],
        "alpn": []
      },
      "tcpSettings": {
        "acceptProxyProtocol": false,
        "header": {
          "type": "none"
        }
      }
    }`, certificateFile, keyFile)

	inbound.Sniffing = `{
        "enabled":true,
        "destOverride":[
            "http",
            "tls"
        ]
    }`

	return password, nil
}

func (a *APIController) setVlessSettingsForInbound(inbound *model.Inbound) string {
	userUUID := uuid.New()
	inbound.Settings = fmt.Sprintf(
		`{
            "clients": [
            {
              "id": "%s",
              "flow": "xtls-rprx-direct"
            }
          ],
          "decryption": "none",
          "fallbacks": []
        }`,
		userUUID)
	userUUIDstring := userUUID.String()

	inbound.StreamSettings = `{
        "network":"ws",
        "security":"none",
        "wsSettings":{
            "acceptProxyProtocol":false,
            "path":"/",
            "headers":{}
        }
    }`

	inbound.Sniffing = `{
        "enabled":true,
        "destOverride":[
            "http",
            "tls"
        ]
    }`

	return userUUIDstring
}

func (a *APIController) getVmessURL(inbound *model.Inbound, userUUIDstring string, hostname string) (string, error) {
	obj := VlessObject{
		Version: "2",
		Ps:      inbound.Remark,
		Address: hostname,
		Port:    inbound.Port,
		UUID:    string(userUUIDstring),
		AlterId: 0,
		Net:     "tcp",
		Type:    "none",
		Host:    "",
		Path:    "",
		TLS:     "none",
	}

	objStr, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	objStrBase64 := b64.StdEncoding.EncodeToString(objStr)
	vmessURL := "vmess://" + objStrBase64

	return vmessURL, nil
}

func (a *APIController) getTrojanURL(inbound *model.Inbound, password string, hostname string) string {
	return fmt.Sprintf("trojan://%s@%s:%d#%s",
		password, hostname, inbound.Port, inbound.Remark)
}

func (a *APIController) getVlessURL(inbound *model.Inbound, userUUIDstring string, hostname string) string {
	return fmt.Sprintf("vless://%s@%s:%d?type=ws&security=none&path=%%2F#%s",
		userUUIDstring, hostname, inbound.Port, inbound.Remark)

}
