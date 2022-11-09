package controller

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sethvargo/go-password/password"
	"gorm.io/gorm"
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
	g.POST("/remaining_quota", a.remainingQuota)

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

	dbInbound, err := a.inboundService.GetInboundWithRemark(inbound.Remark)
	if err != nil || dbInbound.Id == 0 {
		jsonMsg(c, "获取", gorm.ErrRecordNotFound)
		return
	}

	m := entity.UserAddResp{
		Obj:                nil,
		Success:            true,
		TotalBandwidth:     int(dbInbound.Total / 1000 / 1000),
		RemainingBandwidth: int((dbInbound.Total - dbInbound.Up - dbInbound.Down) / 1000 / 1000),
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
	}

	inbound.Port = 20000 + rand.Intn(30000) /*port between 20,000 to 50,000*/
	var url string

	inbound.UserId = user.Id
	inbound.Total = inbound.Total * 1024 * 1024 * 1024
	inbound.Enable = true
	inbound.Tag = fmt.Sprintf("inbound-%v", inbound.Port)
	err = a.inboundService.AddInbound(inbound)

	if err != nil && strings.HasPrefix(err.Error(), "ALREADY_EXISTS") {
		dbInbound, err := a.inboundService.GetInboundWithRemark(inbound.Remark)
		if err != nil {
			jsonMsg(c, "添加", err)
			return
		}

		if requestedProtocol != dbInbound.Protocol {
			a.inboundService.DelInbound(dbInbound.Id)
			err = a.inboundService.AddInbound(inbound)
			if err != nil {
				jsonMsg(c, "添加", err)
				return
			}
		} else {
			//Inbound exists with the right protocol
			inbound = dbInbound
			var settings map[string]any
			json.Unmarshal([]byte(inbound.Settings), &settings)

			clients := settings["clients"].([]any)
			client := clients[0].(map[string]any)
			if inbound.Protocol == "vmess" {
				userUUIDstring = client["id"].(string)
			} else if inbound.Protocol == "trojan" {
				password = client["password"].(string)
			}

		}
	}

	hostname := a.getHostname(c)

	if inbound.Protocol == "vmess" {
		url, err = a.getVmessURL(inbound, userUUIDstring, hostname)
		if err != nil {
			jsonMsg(c, "添加", err)
			return
		}

	} else if inbound.Protocol == "trojan" {
		url = a.getTrojanURL(inbound, password, hostname)
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

func (a *APIController) getHostname(c *gin.Context) string {
	hostname, err := a.settingService.GetServerName()
	if (err != nil) || (hostname == "localhost") {
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

func (a *APIController) getVmessURL(inbound *model.Inbound, userUUIDstring string, hostname string) (string, error) {
	obj := VlessObject{
		Version: "2",
		Ps:      inbound.Remark,
		Address: hostname,
		Port:    inbound.Port,
		UUID:    string(userUUIDstring),
		AlterId: 0,
		Net:     "ws",
		Type:    "none",
		Host:    "",
		Path:    "/",
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
