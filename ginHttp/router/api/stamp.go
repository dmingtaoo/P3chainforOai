package router

import (
	"encoding/json"
	"net/http"
	"p3Chain/api"
	"p3Chain/ginHttp/pkg/app"
	e "p3Chain/ginHttp/pkg/error"
	"p3Chain/ginHttp/pkg/msgtype"

	"github.com/gin-gonic/gin"
)

func BackCredit(ct *api.ContractService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		contractName := "NFT_DEMO "
		functionName := "DEMO::FUNCTION::getCredit "
		account := ct.DperService.CurrentAccount()
		//account := "account"
		args := account

		commandStr := "call " + contractName + " " + functionName + " -args " + args

		data, err := ct.DigtalStamp.BackCredit(commandStr)

		if err != nil {
			appG.Response(http.StatusOK, e.ERROR, err)
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, data)
		}

	}
}

func BackStampList(ct *api.ContractService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		contractName := "NFT_DEMO "
		functionName := "DEMO::FUNCTION::balanceofAll "
		NftSymbol := "Stamp "
		account := ct.DperService.CurrentAccount()
		args := NftSymbol + " " + account

		commandStr := "call " + contractName + " " + functionName + " -args " + args

		data, err := ct.DigtalStamp.BackStampList(commandStr)
		value := []msgtype.StampList{}
		json.Unmarshal([]byte(data[0]), &value)
		if err != nil {
			appG.Response(http.StatusOK, e.ERROR, err)
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, value)
		}

	}
}

// tokenId：邮票名称
// base64：图片文件
func MintNewStamp(ct *api.ContractService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		contractName := "NFT_DEMO "
		functionName := "DEMO::FUNCTION::mint "
		NftSymbol := "Stamp "
		to := ct.DperService.CurrentAccount()
		tokenId := c.PostForm("tokenId")
		base64 := c.PostForm("base64")
		args := NftSymbol + " " + to + " " + tokenId + " " + base64

		commandStr := "invoke " + contractName + " " + functionName + " -args " + args

		recipt, err := ct.DigtalStamp.MintNewStamp(commandStr)
		if err != nil {
			appG.Response(http.StatusOK, e.ERROR, err)
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, recipt)
		}
	}
}

// to：受赠地址
// tokenId：邮票名称
func TransStamp(ct *api.ContractService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		contractName := "NFT_DEMO "
		functionName := "DEMO::FUNCTION::transFrom "
		NftSymbol := "Stamp "
		from := ct.DperService.CurrentAccount()
		to := c.PostForm("to")
		tokenId := c.PostForm("tokenId")
		args := NftSymbol + " " + from + " " + to + " " + tokenId

		commandStr := "invoke " + contractName + " " + functionName + " -args " + args

		recipt, err := ct.DigtalStamp.TransStamp(commandStr)
		if err != nil {
			appG.Response(http.StatusOK, e.ERROR, err)
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, recipt)
		}
	}
}
