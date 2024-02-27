package setting

import (
	loglogrus "p3Chain/log_logrus"
	"time"

	"github.com/go-ini/ini"
)

type Server struct {
	RunMode      string
	IP           string
	HttpPort     int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

var ServerSetting = &Server{} // 大写，作为全局配置对象

func Setup() {
	Cfg, err := ini.Load("./settings/extended.ini")
	if err != nil {
		loglogrus.Log.Error("Fail to parse './settings/http.ini': %v\n", err)
	}

	err = Cfg.Section("Http").MapTo(ServerSetting)
	if err != nil {
		loglogrus.Log.Error("Cfg.MapTo ServerSetting err: %v\n", err)
	}

	ServerSetting.ReadTimeout = ServerSetting.ReadTimeout * time.Second   // 特殊项再赋值
	ServerSetting.WriteTimeout = ServerSetting.WriteTimeout * time.Second // 特殊项再赋值

}
