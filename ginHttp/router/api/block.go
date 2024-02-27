package router

import (
	"fmt"
	"net/http"
	"p3Chain/api"
	"p3Chain/common"
	"p3Chain/core/eles"
	"p3Chain/ginHttp/pkg/app"
	e "p3Chain/ginHttp/pkg/error"
	"p3Chain/ginHttp/pkg/msgtype"
	loglogrus "p3Chain/log_logrus"
	"reflect"
	"time"

	"github.com/astaxie/beego/validation"
	"github.com/gin-gonic/gin"
	"github.com/unknwon/com"
)

func ParseBlockReceiptToStr(blockReceipt *eles.BlockReceipt) msgtype.BlockReceipt {
	receiptStr := new(msgtype.BlockReceipt)

	receiptStr.ReceiptID = fmt.Sprintf("%x", blockReceipt.ReceiptID)
	for _, txReceipt := range blockReceipt.TxReceipt {
		txStr := msgtype.TransactionReceipt{}
		txStr.Valid = txReceipt.Valid
		for _, result := range txReceipt.Result {
			txStr.Result = append(txStr.Result, string(result))
		}
		receiptStr.TxReceipt = append(receiptStr.TxReceipt, txStr)
	}
	for _, writeSet := range blockReceipt.WriteSet {
		writeSetStr := msgtype.WriteEle{}
		writeSetStr.Value = string(writeSet.Value)
		receiptStr.WriteSet = append(receiptStr.WriteSet, writeSetStr)
	}
	return *receiptStr
}

func ParesTransactionReceiptToStr(txReceipt *eles.TransactionReceipt) msgtype.TransactionReceipt {
	txStr := msgtype.TransactionReceipt{}
	txStr.Valid = txReceipt.Valid
	for _, result := range txReceipt.Result {
		txStr.Result = append(txStr.Result, string(result))
	}
	return txStr
}

func GetCurrentBlockNumber(bs *api.BlockService) func(*gin.Context) {

	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- GetCurrentBlockNumber\n")

		count := bs.GetCurrentBlockNumber()
		appG := app.Gin{c}
		appG.Response(http.StatusOK, e.SUCCESS, count)

		loglogrus.Log.Infof("Http: Reply to user request -- GetCurrentBlockNumber  succeed!\n")
	}
}

func GetCurrentBlockHash(bs *api.BlockService) func(*gin.Context) {

	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- GetCurrentBlockHash\n")
		hash := bs.GetCurrentBlockHash()
		appG := app.Gin{c}
		appG.Response(http.StatusOK, e.SUCCESS, fmt.Sprintf("%x", hash))
		loglogrus.Log.Infof("Http: Reply to user request -- GetCurrentBlockHash  succeed!\n")
	}
}

func GetBlockByHash(bs *api.BlockService) func(*gin.Context) {
	return func(c *gin.Context) {

		hash := c.Param("hash")
		valid := validation.Validation{}
		valid.Required(hash, "hash").Message("区块Hash不能为空")
		//valid.Length(hash, 32, "hash").Message("区块Hash必须为32位")

		appG := app.Gin{c}
		if valid.HasErrors() {
			app.MarkErrors("GetBlockByHash", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- GetBlockByHash  blockHash:%s\n", hash)
		//表单验证通过
		block := bs.GetBlockByHash(common.HexToHash(hash))
		if block == nil || (reflect.DeepEqual(*block, eles.WrapBlock{})) { // 结构体的比较必须使用reflect
			loglogrus.Log.Warnf("Http: Reply to user request -- GetBlockByHash  warn:%s", "Block is not exist!\n")
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_BLOCK, nil)
			return
		} else {
			txList := make([]string, 0)
			for _, tx := range block.RawBlock.Transactions {
				txList = append(txList, tx.TxID.Hex())
			}
			data := msgtype.Block{
				Number:    block.Number,
				BlockID:   fmt.Sprintf("%x", block.RawBlock.BlockID),
				Leader:    fmt.Sprintf("%x", block.RawBlock.Leader),
				PrevBlock: fmt.Sprintf("%x", block.RawBlock.PrevBlock),
				SubNet:    string(block.RawBlock.Subnet),
				TxCount:   len(block.RawBlock.Transactions),
				TxList:    txList,
				TimeStamp: block.ReceivedAt,
				Receipt:   ParseBlockReceiptToStr(&block.RawBlock.Receipt),
			}
			appG.Response(http.StatusOK, e.SUCCESS, data)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- GetBlockByHash  succeed!\n")
	}
}

func GetBlockByNumber(bs *api.BlockService) func(*gin.Context) {
	return func(c *gin.Context) {
		num := com.StrTo(c.Param("number")).MustInt()
		valid := validation.Validation{}
		valid.Min(num, 0, "num").Message("区块编号不能低于0")

		appG := app.Gin{c}
		if valid.HasErrors() {
			app.MarkErrors("GetBlockByNumber", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- GetBlockByNumber  blockNumber:%d\n", num)
		//表单验证通过
		block := bs.GetBlockByNumber(uint64(num))

		if block == nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- GetBlockByNumber  warn:%s", "Block is not exist!\n")
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_BLOCK, nil)
			return
		} else {
			txList := make([]string, 0)
			for _, tx := range block.RawBlock.Transactions {
				txList = append(txList, tx.TxID.Hex())
			}
			data := msgtype.Block{
				Number:    block.Number,
				BlockID:   fmt.Sprintf("%x", block.RawBlock.BlockID),
				Leader:    fmt.Sprintf("%x", block.RawBlock.Leader),
				PrevBlock: fmt.Sprintf("%x", block.RawBlock.PrevBlock),
				SubNet:    string(block.RawBlock.Subnet),
				TxCount:   len(block.RawBlock.Transactions),
				TxList:    txList,
				TimeStamp: block.ReceivedAt,
				Receipt:   ParseBlockReceiptToStr(&block.RawBlock.Receipt),
			}
			appG.Response(http.StatusOK, e.SUCCESS, data)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- GetBlockByNumber  succeed!\n")
	}
}

func GetBlockHashByNumber(bs *api.BlockService) func(*gin.Context) {
	return func(c *gin.Context) {
		num := com.StrTo(c.Param("number")).MustInt()
		valid := validation.Validation{}
		valid.Min(num, 0, "num").Message("区块编号不能低于0")

		appG := app.Gin{c}
		if valid.HasErrors() {
			app.MarkErrors("GetBlockHashByNumber", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- GetBlockHashByNumber  blockNumber:%d\n", num)
		//表单验证通过
		blockHash := bs.GetBlockHashByNumber(uint64(num))
		if (blockHash == common.Hash{}) { // 基本数据类型之间的比较可以不使用reflect，而可以直接比较
			loglogrus.Log.Warnf("Http: Reply to user request -- GetBlockHashByNumber  warn:%s", "Block is not exist!\n")
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_BLOCK, nil)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, fmt.Sprintf("%x", blockHash))
		}
		loglogrus.Log.Infof("Http: Reply to user request -- GetBlockHashByNumber  succeed!\n")

	}
}

func GetRecentBlocks(bs *api.BlockService) func(*gin.Context) {
	return func(c *gin.Context) {
		count := com.StrTo(c.Param("count")).MustInt()
		valid := validation.Validation{}
		valid.Min(count, 1, "num").Message("区块数量至少为1个")

		appG := app.Gin{c}
		if valid.HasErrors() {
			app.MarkErrors("GetRecentBlocks", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- GetRecentBlocks  blockCount:%d\n", count)
		//表单验证通过
		blockList := bs.GetRecentBlocks(uint64(count))
		if blockList == nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- GetRecentBlocks  warn:%s", "The number of blocks in the current blockchain is zero!\n")
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_BLOCK, nil)
			return
		} else {
			datas := make([]msgtype.Block, 0)
			for _, block := range blockList {
				txList := make([]string, 0)
				for _, tx := range block.RawBlock.Transactions {
					txList = append(txList, tx.TxID.Hex())
				}
				data := msgtype.Block{
					Number:    block.Number,
					BlockID:   fmt.Sprintf("%x", block.RawBlock.BlockID),
					Leader:    fmt.Sprintf("%x", block.RawBlock.Leader),
					PrevBlock: fmt.Sprintf("%x", block.RawBlock.PrevBlock),
					SubNet:    string(block.RawBlock.Subnet),
					TxCount:   len(block.RawBlock.Transactions),
					TxList:    txList,
					TimeStamp: block.ReceivedAt,
					Receipt:   ParseBlockReceiptToStr(&block.RawBlock.Receipt),
				}
				datas = append(datas, data)
			}
			type recentBlock struct {
				BlockList []msgtype.Block `json:"blockList"`
				ReqCount  int             `json:"reqCount"`
				ResCount  int             `json:"resCount"`
			}
			abk := recentBlock{BlockList: datas, ReqCount: count, ResCount: len(datas)}
			appG.Response(http.StatusOK, e.SUCCESS, abk)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- GetRecentBlocks  succeed!\n")
	}
}

func GetAllBlocks(bs *api.BlockService) func(*gin.Context) {

	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- GetAllBlocks")

		blockList := bs.GetAllBlocks()
		appG := app.Gin{c}
		if blockList == nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- GetAllBlocks  warn:%s", "The number of blocks in the current blockchain is zero\n")
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_BLOCK, nil)
			return
		} else {
			datas := make([]msgtype.Block, 0)
			for _, block := range blockList {
				txList := make([]string, 0)
				for _, tx := range block.RawBlock.Transactions {
					txList = append(txList, tx.TxID.Hex())
				}
				data := msgtype.Block{
					Number:    block.Number,
					BlockID:   fmt.Sprintf("%x", block.RawBlock.BlockID),
					Leader:    fmt.Sprintf("%x", block.RawBlock.Leader),
					PrevBlock: fmt.Sprintf("%x", block.RawBlock.PrevBlock),
					SubNet:    string(block.RawBlock.Subnet),
					TxCount:   len(block.RawBlock.Transactions),
					TxList:    txList,
					TimeStamp: block.ReceivedAt,
					Receipt:   ParseBlockReceiptToStr(&block.RawBlock.Receipt),
				}
				datas = append(datas, data)
			}
			type allBlock struct {
				BlockList  []msgtype.Block `json:"blockList"`
				BlockCount int             `json:"blockCount"`
			}
			abk := allBlock{BlockList: datas, BlockCount: len(datas)}
			appG.Response(http.StatusOK, e.SUCCESS, abk)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- GetAllBlocks  succeed!\n")
	}
}

func GetAvgValidTxRateInRecentBlocks(bs *api.BlockService) func(*gin.Context) {
	return func(c *gin.Context) {
		count := com.StrTo(c.Param("count")).MustInt()
		valid := validation.Validation{}
		valid.Min(count, 1, "num").Message("区块数量至少为1个")

		appG := app.Gin{c}
		if valid.HasErrors() {
			app.MarkErrors("GetAvgValidTxRateInRecentBlocks", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- GetAvgValidTxRateInRecentBlocks  blockCount:%d\n", count)
		//表单验证通过
		blockList := bs.GetRecentBlocks(uint64(count))
		if blockList == nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- GetAvgValidTxRateInRecentBlocks  warn:%s", "The number of blocks in the current blockchain is zero!\n")
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_BLOCK, nil)
			return
		} else {
			validTx := 0
			total := 0
			for _, block := range blockList {
				for _, receipt := range block.RawBlock.Receipt.TxReceipt {
					total += 1
					if receipt.Valid == byte(1) {
						validTx += 1
					}

				}
			}

			rate := float64(validTx) / float64(total)

			appG.Response(http.StatusOK, e.SUCCESS, rate)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- GetAvgValidTxRateInRecentBlocks  succeed!\n")
	}
}

func GetAvgBlockInterval(bs *api.BlockService) func(*gin.Context) {
	return func(c *gin.Context) {
		count := com.StrTo(c.Param("count")).MustInt()
		valid := validation.Validation{}
		valid.Min(count, 2, "num").Message("区块数量至少为2个")

		appG := app.Gin{c}
		if valid.HasErrors() {
			app.MarkErrors("GetAvgBlockInterval", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- GetAvgBlockInterval blockCount:%d\n", count)
		//表单验证通过
		blockList := bs.GetRecentBlocks(uint64(count))
		if blockList == nil || len(blockList) < 2 {
			loglogrus.Log.Warnf("Http: Reply to user request -- GetAvgBlockInterval warn:%s", "The number of blocks in the current blockchain less than 2!\n")
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_BLOCK, nil)
			return
		} else {
			first, _ := time.Parse("2006-01-02 15:04:05", blockList[0].ReceivedAt)
			final, _ := time.Parse("2006-01-02 15:04:05", blockList[len(blockList)-1].ReceivedAt)
			delta := first.Unix() - final.Unix()
			num := len(blockList) - 1

			avgInterval := float64(delta) / float64(num)

			appG.Response(http.StatusOK, e.SUCCESS, avgInterval)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- GetAvgBlockInterval  succeed!\n")
	}
}

func GetTransactionNumber(bs *api.BlockService) func(*gin.Context) {

	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- GetTransactionNumber\n")
		appG := app.Gin{c}
		txSum := bs.GetTransactionNumber()
		appG.Response(http.StatusOK, e.SUCCESS, txSum)
		loglogrus.Log.Infof("Http: Reply to user request -- GetTransactionNumber\n")
	}
}

func GetTransactionByHash(bs *api.BlockService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		txHash := c.Param("hash")
		valid := validation.Validation{}
		valid.Required(txHash, "txHash").Message("交易Hash不能为空")
		if valid.HasErrors() {
			app.MarkErrors("GetTransactionByHash", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- GetTransactionByHash  txHash:%x\n", txHash)
		//表单验证通过
		tx, txReceipt := bs.GetTransactionByHash(common.HexToHash(txHash))
		if tx == nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- GetTransactionByHash  warn:%s", "Transaction is not exist!\n")
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_TRANSACTION, nil)
			return
		} else {
			args := make([]string, 0)
			for _, arg := range tx.Args {
				args = append(args, string(arg))
			}
			data := msgtype.Transaction{
				TxID:      fmt.Sprintf("%x", tx.TxID),
				Sender:    fmt.Sprintf("%x", tx.Sender),
				Version:   fmt.Sprintf("%x", tx.Version),
				Signature: tx.Signature,
				Contract:  fmt.Sprintf("%x", tx.Contract),
				Function:  fmt.Sprintf("%x", tx.Function),
				Args:      args,
				Receipt:   ParesTransactionReceiptToStr(txReceipt),
			}
			appG.Response(http.StatusOK, e.SUCCESS, data)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- GetTransactionByHash  succeed!\n")
	}
}
func GetTransactionByBlockHashAndIndex(bs *api.BlockService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		blockHash := c.Query("hash")
		txIndex := com.StrTo(c.Query("index")).MustInt()

		valid := validation.Validation{}
		valid.Required(blockHash, "txID").Message("区块Hash不能为空")
		valid.Min(txIndex, 0, "txIndex").Message("交易所有不能为负数")

		if valid.HasErrors() {
			app.MarkErrors("GetTransactionByBlockHashAndIndex", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- GetTransactionByBlockHashAndIndex  blockHash:%x,index:%d\n", blockHash, txIndex)
		//表单验证通过
		tx, txReceipt := bs.GetTransactionByBlockHashAndIndex(common.HexToHash(blockHash), txIndex)
		if tx == nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- GetTransactionByBlockHashAndIndex  warn:%s", "Transaction is not exist!\n")
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_TRANSACTION, nil)
			return
		} else {
			args := make([]string, 0)
			for _, arg := range tx.Args {
				args = append(args, string(arg))
			}
			data := msgtype.Transaction{
				TxID:      fmt.Sprintf("%x", tx.TxID),
				Sender:    fmt.Sprintf("%x", tx.Sender),
				Version:   fmt.Sprintf("%x", tx.Version),
				Signature: tx.Signature,
				Contract:  fmt.Sprintf("%x", tx.Contract),
				Function:  fmt.Sprintf("%x", tx.Function),
				Args:      args,
				Receipt:   ParesTransactionReceiptToStr(txReceipt),
			}
			appG.Response(http.StatusOK, e.SUCCESS, data)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- GetTransactionByBlockHashAndIndex  succeed!\n")
	}
}

func GetTransactionsByBlockHash(bs *api.BlockService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		blockHash := c.Param("blockHash")
		valid := validation.Validation{}
		valid.Required(blockHash, "blockHash").Message("区块Hash不能为空")

		if valid.HasErrors() {
			app.MarkErrors("GetTransactionsByBlockHash", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- GetTransactionsByBlockHash  blockHash:%x\n", blockHash)
		//表单验证通过
		txList, txReceiptList := bs.GetTransactionsByBlockHash(common.HexToHash(blockHash))
		if txList == nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- GetTransactionsByBlockHash  warn:%s", "No transaction exists in the specified block!\n")
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_TRANSACTION, nil)
			return
		} else {
			datas := make([]msgtype.Transaction, 0)
			for index, tx := range txList {
				args := make([]string, 0)
				for _, arg := range tx.Args {
					args = append(args, string(arg))
				}
				data := msgtype.Transaction{
					TxID:      fmt.Sprintf("%x", tx.TxID),
					Sender:    fmt.Sprintf("%x", tx.Sender),
					Version:   fmt.Sprintf("%x", tx.Version),
					Signature: tx.Signature,
					Contract:  fmt.Sprintf("%x", tx.Contract),
					Function:  fmt.Sprintf("%x", tx.Function),
					Args:      args,
					Receipt:   ParesTransactionReceiptToStr(&txReceiptList[index]),
				}
				datas = append(datas, data)
			}
			appG.Response(http.StatusOK, e.SUCCESS, datas)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- GetTransactionsByBlockHash  succeed!\n")
	}
}
func GetTransactionsByBlockNumber(bs *api.BlockService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		blockNumber := com.StrTo(c.Param("blockNumber")).MustInt()
		valid := validation.Validation{}
		//valid.Required(blockNumber, "blockNumber").Message("区块编号不能为空")  // 不能使用此项表单检测，因为是Int型，所以等于零此项会产生错误
		valid.Min(blockNumber, 0, "blockNumber").Message("区块编号不能为负数")

		if valid.HasErrors() {
			app.MarkErrors("GetTransactionsByBlockNumber", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- GetTransactionsByBlockNumber  blockNumber:%d\n", blockNumber)

		txList, txReceiptList := bs.GetTransactionsByBlockNumber(uint64(blockNumber))
		if txList == nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- GetTransactionsByBlockNumber  warn:%s", "No transaction exists in the specified block!\n")
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_TRANSACTION, nil)
			return
		} else {
			datas := make([]msgtype.Transaction, 0)
			for index, tx := range txList {
				args := make([]string, 0)
				for _, arg := range tx.Args {
					args = append(args, string(arg))
				}
				data := msgtype.Transaction{
					TxID:      fmt.Sprintf("%x", tx.TxID),
					Sender:    fmt.Sprintf("%x", tx.Sender),
					Version:   fmt.Sprintf("%x", tx.Version),
					Signature: tx.Signature,
					Contract:  fmt.Sprintf("%x", tx.Contract),
					Function:  fmt.Sprintf("%x", tx.Function),
					Args:      args,
					Receipt:   ParesTransactionReceiptToStr(&txReceiptList[index]),
				}
				datas = append(datas, data)
			}
			appG.Response(http.StatusOK, e.SUCCESS, datas)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- GetTransactionsByBlockNumber  succeed!\n")
	}

}

func GetRecentTransactions(bs *api.BlockService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		count := com.StrTo(c.Param("count")).MustInt()
		valid := validation.Validation{}
		valid.Required(count, "count").Message("区块数量不能为空")
		valid.Min(count, 1, "count").Message("区块数量至少为1个") // 0用于获取全部区块

		if valid.HasErrors() {
			app.MarkErrors("GetRecentTransactions", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- GetRecentTransactions  blockCount:%d\n", count)
		txList, txReceiptList := bs.GetRecentTransactions(uint64(count))
		if txList == nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- GetAllTransactions  warn:%s", "The current number of blockchain transactions is zero!\n")
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_TRANSACTION, nil)
			return
		} else {
			datas := make([]msgtype.Transaction, 0)
			for index, tx := range txList {
				args := make([]string, 0)
				for _, arg := range tx.Args {
					args = append(args, string(arg))
				}
				data := msgtype.Transaction{
					TxID:      fmt.Sprintf("%x", tx.TxID),
					Sender:    fmt.Sprintf("%x", tx.Sender),
					Version:   fmt.Sprintf("%x", tx.Version),
					Signature: tx.Signature,
					Contract:  fmt.Sprintf("%x", tx.Contract),
					Function:  fmt.Sprintf("%x", tx.Function),
					Args:      args,
					Receipt:   ParesTransactionReceiptToStr(&txReceiptList[index]),
				}
				datas = append(datas, data)
			}
			type recentTx struct {
				TransactionList []msgtype.Transaction `json:"transactionList"`
				ReqTxCount      int                   `json:"reqTxCount"`
				ResTxCount      int                   `json:"resTxCount"`
			}
			rt := recentTx{TransactionList: datas, ReqTxCount: count, ResTxCount: len(datas)}
			appG.Response(http.StatusOK, e.SUCCESS, rt)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- GetRecentTransactions  succeed!\n")
	}

}

func GetAllTransactions(bs *api.BlockService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- GetAllTransactions")
		appG := app.Gin{c}
		txList, txReceiptList := bs.GetRecentTransactions(0)
		if txList == nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- GetAllTransactions  warn:%s", "The current number of blockchain transactions is zero!\n")
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_TRANSACTION, nil)
			return
		} else {
			datas := make([]msgtype.Transaction, 0)
			for index, tx := range txList {
				args := make([]string, 0)
				for _, arg := range tx.Args {
					args = append(args, string(arg))
				}
				data := msgtype.Transaction{
					TxID:      fmt.Sprintf("%x", tx.TxID),
					Sender:    fmt.Sprintf("%x", tx.Sender),
					Version:   fmt.Sprintf("%x", tx.Version),
					Signature: tx.Signature,
					Contract:  fmt.Sprintf("%x", tx.Contract),
					Function:  fmt.Sprintf("%x", tx.Function),
					Args:      args,
					Receipt:   ParesTransactionReceiptToStr(&txReceiptList[index]),
				}
				datas = append(datas, data)
			}
			type allTx struct {
				TransactionList []msgtype.Transaction `json:"transactionList"`
				TxCount         int                   `json:"txCount"`
			}
			at := allTx{TransactionList: datas, TxCount: len(datas)}
			appG.Response(http.StatusOK, e.SUCCESS, at)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- GetAllTransactions  succeed!\n")
	}
}

func GetRecentBlocksTPS(bs *api.BlockService) func(*gin.Context) {
	return func(c *gin.Context) {
		count := com.StrTo(c.Param("count")).MustInt()
		valid := validation.Validation{}
		valid.Min(count, 2, "num").Message("区块数量至少为2个")

		appG := app.Gin{c}
		if valid.HasErrors() {
			app.MarkErrors("GetRecentBlocksTPS", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- GetRecentBlocksTPS  blockCount:%d\n", count)
		//表单验证通过
		blockList := bs.GetRecentBlocks(uint64(count))
		if blockList == nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- GetRecentBlocksTPS  warn:%s", "The number of blocks in the current blockchain is zero!\n")
			appG.Response(http.StatusOK, e.ERROR_NOT_EXIST_BLOCK, nil)
			return
		} else {
			txNum := 0
			first, _ := time.Parse("2006-01-02 15:04:05", blockList[0].ReceivedAt)
			final, _ := time.Parse("2006-01-02 15:04:05", blockList[len(blockList)-1].ReceivedAt)
			delta := first.Unix() - final.Unix()

			for i := 0; i < len(blockList)-1; i++ {
				txNum += len(blockList[i].RawBlock.Transactions)
			}

			rate := float64(txNum) / float64(delta)
			appG.Response(http.StatusOK, e.SUCCESS, rate)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- GetRecentBlocksTPS succeed!\n")
	}
}
