package controller

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sethvargo/go-password/password"
	"gorm.io/gorm"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"x-ui/database/model"
	"x-ui/logger"
	"x-ui/web/entity"
	"x-ui/web/global"
	"x-ui/web/service"
)

const NGINX_CONFIG = "/etc/nginx/conf.d/mahsaaminivpn.conf"

var directCodeMap = map[string]bool{
	"vmess":     true,
	"vless":     true,
	"trojan":    true,
	"vmess_cdn": false,
	"vless_cdn": false,
}

type APIController struct {
	inboundService service.InboundService
	xrayService    service.XrayService
	userService    service.UserService
	settingService service.SettingService
	nginxService   service.NginxService
}

func NewAPIController(g *gin.RouterGroup) *APIController {
	a := &APIController{}
	if g != nil {
		a.initRouter(g)
		a.startTask()
	}
	return a
}

func (a *APIController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/api")

	g.GET("/list_users", a.listUsers)
	g.GET("/num_users", a.numUsers)
	g.POST("/add_user", a.addUser)
	g.POST("/all_codes_cdn", a.allCodesCDN)
	g.POST("/delete_user", a.deleteUser)
	g.POST("/remaining_quota", a.remainingQuota)
	g.POST("/add_quota", a.addQuota)
	g.POST("/update_xray_template", a.updateXrayTemplate)
	g.POST("/update_config", a.updateConfig)
}

func (a *APIController) startTask() {
	webServer := global.GetWebServer()
	c := webServer.GetCron()
	c.AddFunc("@every 10s", func() {
		if a.nginxService.IsNeedRestartAndSetFalse() {
			err := a.nginxService.RestartNginx(false)
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

type SmallInbound struct {
	Remark string `json:"remark" form:"remark"`
}

func (a *APIController) listUsers(c *gin.Context) {
	inbounds, err := a.inboundService.GetAllInbounds()
	if err != nil {
		jsonMsg(c, "获取", err)
		return
	}

	outputInbounds := make([]SmallInbound, len(inbounds))
	for index, inbound := range inbounds {
		outputInbounds[index].Remark = inbound.Remark
	}

	jsonObj(c, outputInbounds, nil)
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
	Sni     string `json:"sni,omitempty"`
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

type AllCodesCDNResp struct {
	Success bool     `json:"success"`
	Codes   []string `json:"codes"`
}

func (a *APIController) allCodesCDN(c *gin.Context) {
	inbound := &model.Inbound{}
	err := c.ShouldBind(inbound)
	if err != nil {
		jsonMsg(c, "添加", err)
		return
	}

	fakeServerNames, err := a.settingService.GetFakeServerName()
	if err != nil {
		jsonMsg(c, "Could not find config for fakeServerName", err)
	}

	fakeNamesSlice := strings.Split(fakeServerNames, ",")
	codes := make([]string, 2*len(fakeNamesSlice))

	for index, fakeName := range fakeNamesSlice {
		inbound.Id = 0
		inbound.Protocol = "vmess_cdn"
		hostname := a.getHostname(c, string(inbound.Protocol))
		code_vmess, error_string, error_object := a.addInbound(inbound, hostname, fakeName)
		if error_object != nil {
			jsonMsg(c, error_string, error_object)
			return
		}

		inbound.Protocol = "vless_cdn"
		hostname = a.getHostname(c, string(inbound.Protocol))
		code_vless, error_string, error_object := a.addInbound(inbound, hostname, fakeName)
		if error_object != nil {
			jsonMsg(c, error_string, error_object)
			return
		}

		codes[2*index+0] = code_vmess
		codes[2*index+1] = code_vless
	}

	m := AllCodesCDNResp{
		Success: true,
		Codes:   codes,
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

	hostname := a.getHostname(c, string(inbound.Protocol))

	fakeServerName := ""
	if inbound.Protocol == "vmess_cdn" || inbound.Protocol == "vless_cdn" {
		fakeServerNames, err := a.settingService.GetFakeServerName()
		if err != nil {
			jsonMsg(c, "Could not find config for fakeServerName", err)
			return
		}
		fakeNamesSlice := strings.Split(fakeServerNames, ",")
		rand.Seed(time.Now().Unix())
		n := rand.Int() % len(fakeNamesSlice)
		fakeServerName = fakeNamesSlice[n]
	}

	url_string, error_string, error_object := a.addInbound(inbound, hostname, fakeServerName)
	if error_object != nil {
		jsonMsg(c, error_string, error_object)
		return
	}

	m := entity.UserAddResp{
		Obj:                nil,
		Success:            true,
		Msg:                url_string,
		TotalBandwidth:     int(inbound.Total / 1000 / 1000),
		RemainingBandwidth: int((inbound.Total - inbound.Up - inbound.Down) / 1000 / 1000),
	}
	c.JSON(http.StatusOK, m)
}

func (a *APIController) isCDNServer() bool {
	port, err := a.settingService.GetPort()
	if err != nil {
		return false
	}
	if port == 8443 {
		return true
	} else {
		return false
	}
}

// returns url_string, error_string, error_object
func (a *APIController) addInbound(inbound *model.Inbound, hostname string, fakeServerName string) (string, string, error) {
	user, err := a.userService.GetFirstUser()
	if err != nil {
		return "", "GetFirstUser failed", err
	}

	var trojanPassword string
	var userUUIDstring string
	requestedProtocol := inbound.Protocol

	weeklyQuotaGB, err := a.settingService.GetWeeklyQuota()
	if err != nil {
		return "", "Could not fetch WeeklyQuota", err
	}

	inbound.Total = int64(weeklyQuotaGB) * 1024 * 1024 * 1024
	for true {
		inbound.Port = 20000 + rand.Intn(30000) /*port between 20,000 to 50,000*/
		exists, err := a.inboundService.CheckPortExist(inbound.Port, 0)
		if err != nil {
			return "", "Could not check for port in DB", err
		}
		if !exists {
			break
		}
	}

	if a.isCDNServer() && directCodeMap[string(requestedProtocol)] {
		inbound.Remark = inbound.Remark + "_direct"
	}

	if requestedProtocol == "vmess" {
		userUUIDstring = a.setVmessSettingsForInbound(inbound)

	} else if requestedProtocol == "vless" {
		userUUIDstring, err = a.setVlessSettingsForInbound(inbound)
		if err != nil {
			return "", "VlessSettingsForInbound failed", err
		}

	} else if requestedProtocol == "trojan" {
		logger.Info("Setting protocol as trojan")
		trojanPassword, err = a.setTrojanSettingsForInbound(inbound)
		if err != nil {
			return "", "TrojanSettingsForInbound failed", err
		}

	} else if requestedProtocol == "vmess_cdn" {
		vmessHostname, _ := a.settingService.GetServerName()
		userUUIDstring = a.setVmessCDNSettingsForInbound(inbound, vmessHostname)
		inbound.Protocol = "vmess"

	} else if requestedProtocol == "vless_cdn" {
		userUUIDstring = a.setVlessCDNSettingsForInbound(inbound, hostname)
		inbound.Protocol = "vless"
	}

	var url string

	inbound.UserId = user.Id
	inbound.Enable = true

	randomTag, err := password.Generate(10, 4, 0, true, true)
	if err != nil {
		return "", "password.Generate failed", err
	}
	inbound.Tag = fmt.Sprintf("inbound-%v", randomTag)

	err = a.inboundService.AddInbound(inbound)

	if err != nil && strings.HasPrefix(err.Error(), "ALREADY_EXISTS") {
		// To make sure the assumed settings for the protocol is in-sync with the XRAY/DB
		rowsCount, err := a.inboundService.UpdateStreamSettings(inbound.Remark, string(inbound.Protocol), inbound.StreamSettings)
		if err != nil || rowsCount != 1 {
			return "", "UpdateStreamSettings failed", err
		}

		dbInbound, err := a.inboundService.GetInboundWithRemarkProtocol(inbound.Remark, string(inbound.Protocol))
		if err != nil {
			return "", "GetInboundWithRemarkProtocol failed", err
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
			trojanPassword = client["password"].(string)
		}
	} else if err != nil {
		return "", "Could not add user", err
	} else {
		a.xrayService.SetToNeedRestart()

		if requestedProtocol == "vmess_cdn" || requestedProtocol == "vless_cdn" {
			err = a.updateNginxConfig(hostname, true)
			if err != nil {
				return "", "updateNginxConfig failed", err
			}
		}
	}

	if requestedProtocol == "vmess" {
		url, err = a.getVmessURL(inbound, userUUIDstring, hostname)
		if err != nil {
			return "", "getVmessURL failed", err
		}
	} else if requestedProtocol == "trojan" {
		url = a.getTrojanURL(inbound, trojanPassword, hostname)

	} else if requestedProtocol == "vless" {
		url = a.getVlessURL(inbound, userUUIDstring, hostname)

	} else if requestedProtocol == "vmess_cdn" {
		url, err = a.getVmessCDNURL(inbound, userUUIDstring, hostname, fakeServerName)
		if err != nil {
			return "", "getVmessCDNURL failed", err
		}

	} else if requestedProtocol == "vless_cdn" {
		url = a.getVlessCDNURL(inbound, userUUIDstring, hostname, fakeServerName)

	}

	return url, "", nil
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

	if a.doesNginxConfigExist() {
		hostname, err := a.settingService.GetServerName()
		if err != nil {
			jsonMsg(c, "hostname could not be found", err)
			return
		}
		err = a.updateNginxConfig(hostname, false)
		if err != nil {
			jsonMsg(c, "deleteFromNginx", err)
			return
		}
	}

	jsonMsg(c, "delete", err)
	a.xrayService.SetToNeedRestart()
}

func (a *APIController) getPrefixForProtocol(protocol string) string {
	if protocol == "vmess" {
		return "r"
	} else if protocol == "vless" {
		return "v"
	} else {
		return "x"
	}
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

	log.Printf("current total: %d, newTotal: %d, enable: %t", inbound.Total, inbound.Total+addSize, inbound.Enable)

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

	if protocol == "vmess" {
		hostname, err = a.settingService.GetServerIP()
	} else {
		if protocol == "vless" || protocol == "trojan" {
			directHostname, err := a.settingService.GetDirectServerName()
			if err == nil && directHostname != "" {
				return directHostname
			}
		}

		// vmess_cdn, vless_cdn (or vless/trojan when direct is not defined)
		hostname, err = a.settingService.GetServerName()
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
           "type": "http",
           "request": {
             "method": "GET",
             "path": [
               "/"
             ],
             "headers": {}
           },
           "response": {
             "version": "1.1",
             "status": "200",
             "reason": "OK",
             "headers": {}
           }
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

func (a *APIController) setVmessCDNSettingsForInbound(inbound *model.Inbound, serverName string) string {
	userUUID := uuid.New()
	inbound.Settings = fmt.Sprintf(
		`{
          "clients": [
            {
              "id": "%s",
              "alterId": 0
            }
          ],
          "disableInsecureEncryption": false
        }`,
		userUUID)
	userUUIDstring := userUUID.String()

	inbound.StreamSettings = fmt.Sprintf(`{
      "network": "ws",
      "security": "none",
      "wsSettings": {
        "acceptProxyProtocol": false,
        "path": "/r%s",
        "headers": {
              "Host": "%s"
        }        
      }
    }`, inbound.Remark, serverName)

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

func (a *APIController) setVlessSettingsForInbound(inbound *model.Inbound) (string, error) {
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

	return userUUIDstring, nil
}

func (a *APIController) setVlessCDNSettingsForInbound(inbound *model.Inbound, serverName string) string {
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

	inbound.StreamSettings = fmt.Sprintf(`{
      "network": "ws",
      "security": "none",
      "wsSettings": {
        "acceptProxyProtocol": false,
        "path": "/v%s",
        "headers": {
              "Host": "%s"
        }
      }
    }`, inbound.Remark, serverName)

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
	var obj VlessObject
	obj = VlessObject{
		Version: "2",
		Ps:      inbound.Remark,
		Address: hostname,
		Port:    inbound.Port,
		UUID:    string(userUUIDstring),
		AlterId: 0,
		Net:     "tcp",
		Type:    "http",
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

func (a *APIController) getVlessURL(inbound *model.Inbound, userUUIDstring string, hostname string) string {
	return fmt.Sprintf("vless://%s@%s:%d?type=tcp&security=xtls&flow=xtls-rprx-direct#%s",
		userUUIDstring, hostname, inbound.Port, inbound.Remark)
}

func (a *APIController) getVmessCDNURL(inbound *model.Inbound, userUUIDstring string, hostname string, fakeServerName string) (string, error) {
	obj := VlessObject{
		Version: "2",
		Ps:      inbound.Remark,
		Address: fakeServerName,
		Port:    443,
		UUID:    string(userUUIDstring),
		AlterId: 0,
		Net:     "ws",
		Type:    "none",
		Host:    hostname,
		Sni:     hostname,
		Path:    fmt.Sprintf("/r%s", inbound.Remark),
		TLS:     "tls",
	}

	objStr, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	objStrBase64 := b64.StdEncoding.EncodeToString(objStr)
	vmessURL := "vmess://" + objStrBase64

	return vmessURL, nil
}

func (a *APIController) getVlessCDNURL(inbound *model.Inbound, userUUIDstring string, hostname string, fakeServerName string) string {
	return fmt.Sprintf("vless://%s@%s:443?type=ws&encryption=none&security=tls&path=%%2Fv%s&sni=%s&host=%s#%s",
		userUUIDstring, fakeServerName, inbound.Remark, hostname, hostname, inbound.Remark)
}

func (a *APIController) updateNginxConfig(serverName string, restartServer bool) error {
	var sb strings.Builder

	allServerName := serverName

	oldServerNames, err := a.settingService.GetOldServerNames()
	if err == nil && oldServerNames != "" {
		allServerName = fmt.Sprintf("%s %s", oldServerNames, serverName)
	}

	sb.WriteString(fmt.Sprintf(`
    server {
    	listen 80;

    	# The host name to respond to
        server_name %s;
        
    	index index.html;
        root /var/www/html/2048;

    `, allServerName))

	inbounds, err := a.inboundService.GetAllInbounds()
	if err != nil {
		return err
	}

	locationMap := make(map[string]int)

	for _, inbound := range inbounds {
		prefix := a.getPrefixForProtocol(string(inbound.Protocol))
		inbound_path := fmt.Sprintf("/%s%s", prefix, inbound.Remark)
		if locationMap[inbound_path] == 1 {
			//The location is already added, continue
			continue
		}

		locationMap[inbound_path] = 1

		sb.WriteString(fmt.Sprintf(`
        location %s {
            proxy_pass  http://127.0.0.1:%d%s;

            proxy_set_header Host $host;

            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";	
        }
        
        `, inbound_path, inbound.Port, inbound_path))
	}

	sb.WriteString(`
    }
`)

	err = os.WriteFile(NGINX_CONFIG, []byte(sb.String()), 0644)
	if err != nil {
		logger.Error("unable to write nginx config:", err)
		return err
	}

	if restartServer {
		a.nginxService.SetToNeedRestart()
	}

	return nil
}

func (a *APIController) doesNginxConfigExist() bool {
	if _, err := os.Stat(NGINX_CONFIG); errors.Is(err, os.ErrNotExist) {
		// Not an nginx-based server
		return false
	}
	return true
}

type XrayConfig struct {
	XrayTemplateConfig string `json:"xrayTemplateConfig" binding:"required"`
}

func (a *APIController) updateXrayTemplate(c *gin.Context) {
	var xrayConfig XrayConfig
	err := c.ShouldBind(&xrayConfig)
	if err != nil {
		jsonMsg(c, "添加", err)
		return
	}
	xrayTemplateConfig := xrayConfig.XrayTemplateConfig

	if !json.Valid([]byte(xrayTemplateConfig)) {
		jsonMsg(c, "Not a valid json string, can not set this as a template", errors.New("INVALID JSON"))
		return
	}
	err = a.settingService.SetXrayConfigTemplate(xrayTemplateConfig)
	if err != nil {
		jsonMsg(c, "Can not set the xray template string", err)
	}

	fmt.Printf("The XrayConfig->XrayTemplateConfig is updated!")
	jsonMsg(c, "success", nil)
}

type ConfigUpdate struct {
	Key   string `json:"key" binding:"required" form:"key"`
	Value string `json:"value" binding:"required" form:"value"`
}

func (a *APIController) updateConfig(c *gin.Context) {
	var configUpdate ConfigUpdate
	err := c.ShouldBind(&configUpdate)
	if err != nil {
		jsonMsg(c, "Could not find config", err)
		return
	}

	fmt.Printf("Setting config %s = %s\n", configUpdate.Key, configUpdate.Value)
	switch configUpdate.Key {
	case "weeklyQuota":
		fmt.Printf("weeklyQuota\n")
		intVar, err := strconv.Atoi(configUpdate.Value)
		if err == nil {
			err = a.settingService.SetWeeklyQuota(intVar)
		}
	case "fakeServerName":
		err = a.settingService.SetFakeServerName(configUpdate.Value)
	case "serverName":
		currHostname, err := a.settingService.GetServerName()
		if err != nil {
			jsonMsg(c, "hostname could not be found", err)
			return
		}
		if currHostname == configUpdate.Value {
			jsonMsg(c, "hostname is already at the requested value", err)
			return
		}

		oldServerNames, err := a.settingService.GetOldServerNames()
		newOldServerNames := currHostname
		if err == nil && oldServerNames != "" {
			newOldServerNames = fmt.Sprintf("%s %s", oldServerNames, currHostname)
		}
		err = a.settingService.SetOldServerNames(newOldServerNames)
		if err != nil {
			jsonMsg(c, "hostname could not be found", err)
			return
		}

		err = a.settingService.SetServerName(configUpdate.Value)
		if err != nil {
			jsonMsg(c, "Can not set the config", err)
			return
		}

		err = a.updateNginxConfig(configUpdate.Value, true)
		if err != nil {
			jsonMsg(c, "deleteFromNginx", err)
			return
		}
	default:
		fmt.Printf("Undefined key\n")
		err = gorm.ErrRecordNotFound
	}

	if err != nil {
		jsonMsg(c, "Can not set the config", err)
		return
	}

	jsonMsg(c, "success", nil)
}
