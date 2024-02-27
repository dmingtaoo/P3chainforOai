package router

import (
	"net/http"
	"p3Chain/api"
	"p3Chain/ginHttp/pkg/setting"

	"github.com/gin-gonic/gin"
)

func Cors() gin.HandlerFunc {
	return func(context *gin.Context) {
		method := context.Request.Method

		origin := context.Request.Header.Get("Origin")

		if origin != "" {
			context.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			context.Header("Access-Control-Allow-Origin", "*")
		}

		context.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Token")
		context.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, DELETE, PATCH, PUT")
		context.Header("Access-Control-Expose-Headers", "Access-Control-Allow-Headers, Token")
		context.Header("Access-Control-Allow-Credentials", "true")

		if method == "OPTIONS" {
			context.AbortWithStatus(http.StatusNoContent)
		}
	}
}

func InitRouter(bs *api.BlockService, ds *api.DperService, ns *api.NetWorkService, ct *api.ContractService) *gin.Engine {

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.Use(Cors())

	gin.SetMode(setting.ServerSetting.RunMode)

	block := r.Group("block")
	network := r.Group("network")
	dper := r.Group("dper")

	contract := r.Group("contract")

	if bs != nil {
		block.GET("/blockNumber", GetCurrentBlockNumber(bs))
		block.GET("/blockHash", GetCurrentBlockHash(bs))
		block.GET("/getBlockByHash/:hash", GetBlockByHash(bs))
		block.GET("/getBlockByNumber/:number", GetBlockByNumber(bs))
		block.GET("/getBlockHashByNumber/:number", GetBlockHashByNumber(bs))
		block.GET("/getRecentBlocks/:count", GetRecentBlocks(bs))
		block.GET("/getAllBlocks", GetAllBlocks(bs))

		block.GET("/getTransactionNumber", GetTransactionNumber(bs))
		block.GET("/getTransactionByHash/:hash", GetTransactionByHash(bs))
		block.GET("/getTransactionByBlockHashAndIndex", GetTransactionByBlockHashAndIndex(bs))
		block.GET("/getTransactionsByBlockHash/:blockHash", GetTransactionsByBlockHash(bs))
		block.GET("/getTransactionsByBlockNumber/:blockNumber", GetTransactionsByBlockNumber(bs))
		block.GET("/getRecentTransactions/:count", GetRecentTransactions(bs))
		block.GET("/getAllTransactions", GetAllTransactions(bs))

		block.GET("/getAvgValidTxRateInRecentBlocks/:count", GetAvgValidTxRateInRecentBlocks(bs))
		block.GET("/getAvgBlockInterval/:count", GetAvgBlockInterval(bs))
		block.GET("/getRecentBlocksTPS/:count", GetRecentBlocksTPS(bs))

	}

	if ds != nil {
		dper.GET("/accountsList", BackAccountList(ds))
		dper.POST("/newAccount", CreateNewAccount(ds))
		dper.POST("/useAccount", UseAccount(ds))
		dper.GET("/currentAccount", CurrentAccount(ds))
		dper.POST("/txCheck", OpenTxCheckMode(ds))
		dper.POST("/solidInvoke", SolidInvoke(ds))
		dper.POST("/solidCall", SolidCall(ds))
		dper.POST("/softInvoke", SoftInvoke(ds))
		
		dper.POST("/signaturereturn", SignatureReturn(ds))
		dper.POST("/signaturereturn2", SignatureReturn2(ds))
		dper.POST("/signvalid", SignValid(ds))
		dper.POST("/vcreceive", VCReceive(ds))
		dper.POST("/vcreceive2", VCReceive2(ds))
		dper.POST("/vcvalid", VCValid(ds))
		dper.POST("/datasend", DataSend(ds))
		dper.POST("/datasend2", DataSend(ds))
		dper.POST("/getaddress", GetAddress(ds))
		dper.POST("/transvc", TransVC(ds))
		dper.POST("/setdid", SetDID(ds))
		dper.POST("/sendvcrequest/:destinationHost/:destinationPort", SendVCrequest(ds))
		dper.POST("/sendvc/:destinationHost/:destinationPort", SendVC(ds))
		dper.POST("/sendrandom/:destinationHost/:destinationPort", SendRandom(ds))
		dper.POST("/transvcrequest/:destinationHost/:destinationPort", TransVCrequest(ds))
		dper.POST("/datarequest/:destinationHost/:destinationPort", DataRequest(ds))
		dper.POST("/datarequest2/:destinationHost/:destinationPort", DataRequest2(ds))

		dper.POST("/AMFsend/:destinationHost/:destinationPort", AMFsend(ds))
		dper.POST("/SMFreceive", SMFreceive(ds))
	}

	if ns != nil {
		network.GET("/networkInfo", BackDPNetWork(ns))
		network.GET("/allConsensusNode", BackAllConsensusNode(ns))
		network.GET("/selfNodeInfo", BackNodeInfoSelf(ns))
		network.POST("/nodeInfoByNodeID", BackNodeInfoByNodeID(ns))

		network.GET("/groupCount", BackGroupCount(ns))
		network.GET("/allGroupName", BackAllGroupName(ns))
		network.GET("/upperNet", BackUpperNetNodeList(ns))
		network.GET("/allBooters", BackAllBooters(ns))
		network.GET("/allLeaders", BackAllLeaders(ns))
		network.POST("/subNetNodeID", BackNodeListByGroupName(ns))
		network.POST("/subNetLeaderID", BackLeaderNodeIDByGroupName(ns))
		network.POST("/subNetInfo", BackSubNetByGroupName(ns))

	}

	if ct != nil {
		contract.GET("/credit", BackCredit(ct))
		contract.GET("/stampList", BackStampList(ct))
		contract.POST("/mintNewStamp", MintNewStamp(ct))
		contract.POST("/transStamp", TransStamp(ct))
	}

	return r

}
