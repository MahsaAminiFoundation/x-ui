package controller

import (
	"github.com/stretchr/testify/assert"
	"net"
	"net/url"
	"os"
	"testing"
	"x-ui/database"
	"x-ui/database/model"
)

func setupTest(t *testing.T) func() {
	err := database.InitDB("/tmp/x-ui.db")
	if err != nil {
		t.Fatal(err)
	}

	// tear down later
	return func() {
		os.Remove("/tmp/x-ui.db")
	}
}

func TestMethod(t *testing.T) {
	// asserts, ensures, requires... here
}

/// / TestHelloName calls greetings.Hello with a name, checking
// // for a valid return value.
// func TestNewUserVmess(t *testing.T) {
//     defer setupTest(t)()
//
//     apiController := NewAPIController(nil)
//
//     expected_code := "vmess://eyJ2IjoiMiIsInBzIjoiMTIzIiwiYWRkIjoibWFoc2EuY29tIiwicG9ydCI6Mzc2MjgsImlkIjoiMGU4YTNmOTEtYTgyMS00ZjIzLTg4MDMtNGU4NWUzMDA5ZjI2IiwiYWlkIjowLCJuZXQiOiJ0Y3AiLCJ0eXBlIjoiaHR0cCIsImhvc3QiOiIiLCJwYXRoIjoiLyIsInRscyI6Im5vbmUifQ=="
//
//     inbound := model.Inbound{Total: 5, Remark: "123", Protocol: "vmess"}
//     code_vmess, error_string, error_object := apiController.addInbound(&inbound, "mahsa.com", "fake.com")
//
//     if error_object != nil {
//         t.Fatalf(`Expected error to be nil, but it is not: %v`, error_object)
//     }
//
//     if error_string != "" {
//         t.Fatalf(`Expected error to be nil, but it is not: %v`, error_string)
//     }
//
//     if code_vmess != expected_code {
//         t.Fatalf(`Vmess code is different than expectation: %v != %v`, code_vmess, expected_code)
//     }
// }

func TestNewUserVlessDirect(t *testing.T) {
	err := database.InitDB("/tmp/x-ui.db")
	if err != nil {
		t.Fatal(err)
	}

	apiController := NewAPIController(nil)
	apiController.settingService.SetPort(8080) //mimic a direct server

	// example code "vless://ba33d66d-ff1d-4d1b-a033-ed60d648163b@mahsa.com:35543?type=tcp&security=xtls&flow=xtls-rprx-direct#123"

	inbound := model.Inbound{Total: 5, Remark: "123", Protocol: "vless"}
	code_vless, error_string, error_object := apiController.addInbound(&inbound, "mahsa.com", "fake.com")

	if error_object != nil {
		t.Fatalf(`Expected error to be nil, but it is not: %v`, error_object)
	}

	if error_string != "" {
		t.Fatalf(`Expected error to be nil, but it is not: %v`, error_string)
	}

	t.Logf(`URL: %v`, code_vless)

	u, err := url.Parse(code_vless)
	if err != nil {
		t.Fatalf(`Could not parse URL`)
	}

	assert.Equal(t, u.Scheme, "vless", "Schema should be vless")
	assert.Equal(t, len(u.User.Username()), 36, "User should be UUID")

	host, _, _ := net.SplitHostPort(u.Host)
	assert.Equal(t, host, "mahsa.com", "Host should be correct")

	m, _ := url.ParseQuery(u.RawQuery)
	assert.Equal(t, m["type"][0], "tcp", "Type of the code should be tcp")
	assert.Equal(t, m["security"][0], "xtls", "Security should be XLTS")
}

func TestNewUserVlessCDN(t *testing.T) {
	err := database.InitDB("/tmp/x-ui.db")
	if err != nil {
		t.Fatal(err)
	}

	apiController := NewAPIController(nil)
	apiController.settingService.SetPort(8443) //mimic a cdn server

	// example code "vless://ba33d66d-ff1d-4d1b-a033-ed60d648163b@mahsa.com:35543?type=tcp&security=xtls&flow=xtls-rprx-direct#123"

	inbound := model.Inbound{Total: 5, Remark: "123", Protocol: "vless"}
	code_vless, error_string, error_object := apiController.addInbound(&inbound, "mahsa.com", "fake.com")

	if error_object != nil {
		t.Fatalf(`Expected error to be nil, but it is not: %v`, error_object)
	}

	if error_string != "" {
		t.Fatalf(`Expected error to be nil, but it is not: %v`, error_string)
	}

	t.Logf(`URL: %v`, code_vless)

	u, err := url.Parse(code_vless)
	if err != nil {
		t.Fatalf(`Could not parse URL`)
	}

	assert.Equal(t, u.Scheme, "vless", "Schema should be vless")
	assert.Equal(t, len(u.User.Username()), 36, "User should be UUID")

	host, _, _ := net.SplitHostPort(u.Host)
	assert.Equal(t, host, "mahsa.com", "Host should be correct")

	m, _ := url.ParseQuery(u.RawQuery)
	assert.Equal(t, m["type"][0], "tcp", "Type of the code should be tcp")
	assert.Equal(t, m["security"][0], "xtls", "Security should be XLTS")
}
