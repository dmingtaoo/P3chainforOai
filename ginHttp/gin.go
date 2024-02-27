package ginHttp

import (
	"fmt"
	"net/http"
	"p3Chain/api"
	"p3Chain/ginHttp/pkg/setting"
	router "p3Chain/ginHttp/router/api"
)

func NewGinRouter(bs *api.BlockService, ds *api.DperService, ns *api.NetWorkService, ct *api.ContractService) {

	setting.Setup()

	router := router.InitRouter(bs, ds, ns, ct) //返回一个gin路由器

	s := &http.Server{
		Addr:           setting.ServerSetting.IP + fmt.Sprintf(":%d", setting.ServerSetting.HttpPort),
		Handler:        router,
		ReadTimeout:    setting.ServerSetting.ReadTimeout,
		WriteTimeout:   setting.ServerSetting.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	s.ListenAndServe()
}
