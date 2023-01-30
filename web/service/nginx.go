package service

import (
	"os/exec"
	"sync"
	"x-ui/logger"

	"go.uber.org/atomic"
)

var nlock sync.Mutex
var isNeedNginxRestart atomic.Bool

type NginxService struct {
	settingService SettingService
}

func (s *NginxService) IsNeedRestartAndSetFalse() bool {
	return isNeedNginxRestart.CAS(true, false)
}

func (s *NginxService) SetToNeedRestart() {
	isNeedNginxRestart.Store(true)
}

func (s *NginxService) RestartNginx(isForce bool) error {
	nlock.Lock()
	defer nlock.Unlock()
	logger.Debug("restart nginx, force:", isForce)

	cmd := exec.Command("/etc/init.d/nginx", "restart")
	err := cmd.Run()
	if err != nil {
		logger.Error("unable to restart nginx:", err)
	}

	return nil
}
