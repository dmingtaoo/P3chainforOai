package router

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"p3Chain/api"
	"p3Chain/common"
	"p3Chain/core/consensus"
	"p3Chain/core/eles"
	"p3Chain/crypto"
	"p3Chain/dper/transactionCheck"
	"p3Chain/ginHttp/pkg/app"
	e "p3Chain/ginHttp/pkg/error"
	loglogrus "p3Chain/log_logrus"
	"reflect"
	"strings"
	"sync"

	"github.com/astaxie/beego/validation"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var messages []string
var mu sync.Mutex

type VCrequest struct {
	Identifier     string `json:"identifier"` // VC ID
	Subject        string `json:"subject"`    // SP DID
	Issuer         string `json:"issuer"`     // UE DID
	Validity       string `json:"validity"`   // Valid time
	Purpose        string `json:"purpose"`    // 用途
	SubjectAddress string `json:"address"`    // subject address
}

type Transfer struct {
	Recipient        string `json:"recipient"`        // SP2 DID
	RecipientPurpose string `json:"purpose"`          // 用途
	RecipientAddress string `json:"recipientaddress"` // subject address
	DonorSignature   string `json:"signature"`        // UE Signature
	Timestamp        string `json:"timestamp"`        // time

}

func removeSpaces(str string) string {
	return strings.ReplaceAll(str, " ", "")
}

func ExtractAddress(input string) (string, error) {
	prefix := "address:"
	startIndex := strings.Index(input, prefix)
	if startIndex == -1 {
		return "", fmt.Errorf("address prefix not found")
	}

	// 提取并返回地址
	address := input[startIndex+len(prefix):]
	return address, nil
}

type VC struct {
	Identifier     string   `json:"identifier"`     // VC ID
	Subject        string   `json:"subject"`        // SP DID
	Issuer         string   `json:"issuer"`         // UE DID
	Validity       string   `json:"validity"`       // Valid time
	Purpose        string   `json:"purpose"`        // 用途
	Signature      string   `json:"signature"`      // UE Signature
	Reassign       string   `json:"reassign"`       // 是否允许二次转让
	IssuerAddress  string   `json:"IssuerAddress"`  // issuer address
	SubjectAddress string   `json:"SubjectAddress"` // subject address
	Transfer       Transfer `json:"transfer"`       // subject address
}

type DataStruct struct {
	DID    string `json:"did"`
	Name   string `json:"name"`
	Gender string `json:"gender"`
	Age    string `json:"age"`
}

// FindDataInFile 从指定的 JSON 文件中查找特定的 DID 并返回对应的数据
func FindDataInFile(did string) (string, error) {
	filename := "/Users/dengmingtao/Documents/GitHub/dpchain/ginHttp/router/api/data.json" // 假设文件在当前目录
	var data []DataStruct

	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("error reading file: %v", err)
	}

	err = json.Unmarshal(file, &data)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling data: %v", err)
	}

	for _, d := range data {
		if d.DID == did {
			result, err := json.Marshal(d)
			if err != nil {
				return "", fmt.Errorf("error marshalling data: %v", err)
			}
			return string(result), nil
		}
	}
	return "", fmt.Errorf("DID not found")
}
func ResultToRecipientAddress(byteSlices [][]byte) (string, error) {
	var resultString string
	for _, b := range byteSlices {
		resultString += string(b)
	}

	// 解析 JSON 字符串
	var data map[string]interface{}
	err := json.Unmarshal([]byte(resultString), &data)
	if err != nil {
		return "failed", err
	}

	// 提取 RecipientAddress
	// 注意这里的字段名称要与 JSON 字符串中的一致
	recipientAddress, ok := data["RecipientAddress"].(string)
	if !ok {
		return "recipientaddress not found", nil
	}

	return recipientAddress, nil
}

func ResultToSubjectAddress(byteSlices [][]byte) (string, error) {
	var resultString string
	for _, b := range byteSlices {
		resultString += string(b)
	}
	// 解析 JSON 字符串
	var data map[string]interface{}

	err := json.Unmarshal([]byte(resultString), &data)
	if err != nil {
		return "failed", err
	}
	// 提取 SubjectAddress
	subjectAddress, ok := data["SubjectAddress"].(string)
	if !ok {
		return "subjectaddress not found", nil
	}
	resultString = subjectAddress

	return resultString, nil
}

func ResultToAddress(result [][]byte) (string, error) {
	// 将 result ([][]byte) 转换为一个字符串
	var resultString string
	for _, b := range result {
		resultString += string(b)
	}

	// 打印结果字符串以进行调试
	fmt.Println("Result string:", resultString)

	// 定义地址前缀
	addressPrefix := "address:"
	// 寻找地址的开始位置
	startIndex := strings.Index(resultString, addressPrefix)
	if startIndex == -1 {
		return "", fmt.Errorf("address prefix not found in result")
	}

	// 获取地址开始的位置（跳过前缀）
	startIndex += len(addressPrefix)
	// 获取地址结束的位置（假设地址结束于字符串结束或闭合方括号前）
	endIndex := len(resultString)
	if endBracket := strings.Index(resultString[startIndex:], "]"); endBracket != -1 {
		endIndex = startIndex + endBracket
	}

	// 提取地址
	address := resultString[startIndex:endIndex]
	return address, nil
}

func ResultToIssuer(byteSlices [][]byte) (string, error) {
	var resultString string
	for _, b := range byteSlices {
		resultString += string(b)
	}
	// 解析 JSON 字符串
	var data map[string]interface{}

	err := json.Unmarshal([]byte(resultString), &data)
	if err != nil {
		return "failed", err
	}
	// 提取 SubjectAddress
	issuer, ok := data["issuer"].(string)
	if !ok {
		return "issuer not found", nil
	}
	resultString = issuer

	return resultString, nil
}

func ResultToVC(byteSlices [][]byte) (string, error) {
	// 将字节切片合并成一个字符串
	var resultString string
	for _, b := range byteSlices {
		resultString += string(b)
	}

	// 解码 JSON 字符串
	var data map[string]interface{}
	err := json.Unmarshal([]byte(resultString), &data)
	if err != nil {
		return "", err // 如果解码失败，则返回错误
	}

	// 从 JSON 数据中提取所需的字段
	identifier, _ := data["identifier"].(string)
	subject, _ := data["subject"].(string)
	issuer, _ := data["issuer"].(string)
	validity, _ := data["validity"].(string)
	purpose, _ := data["purpose"].(string)
	signature, _ := data["signature"].(string)
	reassign, _ := data["reassign"].(string)
	issuerAddress, _ := data["IssuerAddress"].(string)
	subjectAddress, _ := data["SubjectAddress"].(string)

	// 格式化提取的字段到一个字符串
	formattedString := fmt.Sprintf("Identifier: %s, Subject: %s, Issuer: %s, Validity: %s, Purpose: %s, Signature: %s, Reassign: %s, IssuerAddress: %s, SubjectAddress: %s",
		identifier, subject, issuer, validity, purpose, signature, reassign, issuerAddress, subjectAddress)

	return formattedString, nil
}

func BackAccountList(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- BackAccountList\n")
		appG := app.Gin{c}
		data, err := ds.BackListAccounts()
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- BackAccountList  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, data)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- BackAccountList  succeed!\n")
	}
}

func CreateNewAccount(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- CreateNewAccount\n")
		appG := app.Gin{c}
		password := c.PostForm("password")

		commandStr := "createNewAccount"

		if password != "" {
			commandStr += (" -p " + password)
		}

		account, err := ds.CreateNewAccount(commandStr)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- CreateNewAccount  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, fmt.Sprintf("%x", account))
		}
		loglogrus.Log.Infof("Http: Reply to user request -- CreateNewAccount  succeed!\n")
	}

}

// 十六进制字符串格式的account
func UseAccount(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		account := c.PostForm("account")
		password := c.PostForm("password")
		valid := validation.Validation{}
		valid.Required(account, "account").Message("账户Hash不能为空")
		if valid.HasErrors() {
			app.MarkErrors("", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- UseAccount  account:%s\n", account)

		commandStr := "useAccount " + account

		if password != "" {
			commandStr += (" -p " + password)
		}

		err := ds.UseAccount(commandStr)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- UseAccount  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, nil)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- UseAccount  succeed!\n")
	}

}

func CurrentAccount(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- CurrentAccount\n")
		appG := app.Gin{c}
		infoStr := ds.CurrentAccount()

		appG.Response(http.StatusOK, e.SUCCESS, infoStr)
		loglogrus.Log.Infof("Http: Reply to user request -- CurrentAccount  succeed!\n")
	}

}

// on:开启  off:关闭
func OpenTxCheckMode(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		mode := c.PostForm("mode")
		valid := validation.Validation{}
		valid.Required(mode, "commandStr").Message("on:开启交易上链检查 off:关闭交易上链检查")
		if valid.HasErrors() {
			app.MarkErrors("OpenTxCheckMode", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- OpenTxCheckMode  mode:%s\n", mode)
		commandStr := "txCheckMode " + mode

		err := ds.OpenTxCheckMode(commandStr)
		if err != nil {
			loglogrus.Log.Infof("Http: Reply to user request -- OpenTxCheckMode  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, nil)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- OpenTxCheckMode  succeed!\n")
	}

}

// contractAddr:合约地址
// functionAddr:参数地址
// args: 参数列表,以空格分割
func SolidInvoke(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		contractAddr := c.PostForm("contractAddr")
		functionAddr := c.PostForm("functionAddr")

		args := c.PostForm("args")
		valid := validation.Validation{}
		valid.Required(contractAddr, "contractAddr").Message("合约地址不能为空")
		valid.Required(functionAddr, "functionAddr").Message("函数地址不能为空")
		valid.Required(args, "args").Message("参数不能为空")
		if valid.HasErrors() {
			app.MarkErrors("SolidInvoke", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- SolidInvoke  contractAddr:%s , functionAddr:%s , args:%s\n", contractAddr, functionAddr, args)
		commandStr := "solidInvoke " + contractAddr + " " + functionAddr + " -args " + args

		receipt, err := ds.SolidInvoke(commandStr)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- SolidInvoke  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {

			if reflect.DeepEqual(receipt, transactionCheck.CheckResult{}) {
				appG.Response(http.StatusOK, e.SUCCESS, nil)
			} else {
				type ResponseReceipt struct {
					TxID   string `json:"Transaction ID"`
					Valid  bool   `json:"Valid"`
					Result string `json:"Transaction Results"`
					Delay  string `json:"Consensus Delay"`
				}
				var r ResponseReceipt = ResponseReceipt{
					TxID:   fmt.Sprintf("%x", receipt.TransactionID),
					Valid:  receipt.Valid,
					Result: fmt.Sprintf("%s", receipt.Result),
					Delay:  fmt.Sprintf("%d ms", receipt.Interval),
				}
				appG.Response(http.StatusOK, e.SUCCESS, r)
			}

		}
		loglogrus.Log.Infof("Http: Reply to user request -- SolidInvoke  succeed!\n")
	}

}

// contractAddr:合约地址
// functionAddr:参数地址
// args: 参数列表,以空格分割
func SolidCall(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		contractAddr := c.PostForm("contractAddr")
		functionAddr := c.PostForm("functionAddr")
		args := c.PostForm("args")
		valid := validation.Validation{}
		valid.Required(contractAddr, "contractAddr").Message("合约地址不能为空")
		valid.Required(functionAddr, "functionAddr").Message("函数地址不能为空")
		valid.Required(args, "args").Message("参数不能为空")

		if valid.HasErrors() {
			app.MarkErrors("SolidCall", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- SolidCall  contractAddr:%s , functionAddr:%s , args:%s\n", contractAddr, functionAddr, args)
		commandStr := "solidCall " + contractAddr + " " + functionAddr + " -args " + args

		data, err := ds.SolidCall(commandStr)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- SolidCall  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, data)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- SolidCall  succeed!\n")
	}

}

// contractName:合约名
// functionName:参数名
// args: 参数列表,以空格分割
func SoftInvoke(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		contractName := c.PostForm("contractName")
		functionName := c.PostForm("functionName")
		args := c.PostForm("args")
		valid := validation.Validation{}
		valid.Required(contractName, "contractName").Message("合约名不能为空")
		valid.Required(functionName, "functionName").Message("函数名不能为空")
		valid.Required(args, "args").Message("参数不能为空")

		if valid.HasErrors() {
			app.MarkErrors("SoftInvoke", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}

		loglogrus.Log.Infof("Http: Get user request -- SoftInvoke  contractName:%s , functionName:%s , args:%s\n", contractName, functionName, args)
		fmt.Printf("Http: Get user request -- SoftInvoke  contractName:%s , functionName:%s , args:%s\n", contractName, functionName, args)

		commandStr := "invoke " + contractName + " " + functionName + " -args " + args

		receipt, err := ds.SoftInvoke(commandStr)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- SoftInvoke  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			if reflect.DeepEqual(receipt, transactionCheck.CheckResult{}) {
				appG.Response(http.StatusOK, e.SUCCESS, nil)
			} else {
				type ResponseReceipt struct {
					TxID   string `json:"Transaction ID"`
					Valid  bool   `json:"Valid"`
					Result string `json:"Transaction Results"`
					Delay  string `json:"Consensus Delay"`
				}
				var r ResponseReceipt = ResponseReceipt{
					TxID:   fmt.Sprintf("%x", receipt.TransactionID),
					Valid:  receipt.Valid,
					Result: fmt.Sprintf("%s", receipt.Result),
					Delay:  fmt.Sprintf("%d ms", receipt.Interval.Milliseconds()),
				}
				appG.Response(http.StatusOK, e.SUCCESS, r)
			}
		}
		loglogrus.Log.Infof("Http: Reply to user request -- SoftInvoke  succeed!\n")
		fmt.Printf("Http: Reply to user request -- SoftInvoke  succeed!\n")
	}

}

func SoftInvokeQuery(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		contractName := c.PostForm("contractName")
		functionName := c.PostForm("functionName")
		args := c.PostForm("args")
		valid := validation.Validation{}
		valid.Required(contractName, "contractName").Message("合约名不能为空")
		valid.Required(functionName, "functionName").Message("函数名不能为空")
		valid.Required(args, "args").Message("参数不能为空")

		if valid.HasErrors() {
			app.MarkErrors("SoftInvoke", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- SoftInvokeQuery contractName:%s , functionName:%s , args:%s\n", contractName, functionName, args)
		fmt.Printf("Http: Get user request -- SoftInvokeQuery contractName:%s , functionName:%s , args:%s\n", contractName, functionName, args)

		commandStr := "invoke " + contractName + " " + functionName + " -args " + args

		var wg sync.WaitGroup
		ds.SoftInvokeQuery(commandStr, wg)
		fmt.Printf("Http: Reply to user request -- SoftInvokeQuery  succeed!\n")
	}
}

func PublishTx(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		// 1.解析 []tx 编码得到的byte流
		appG := app.Gin{c}
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			loglogrus.Log.Warnf("Http: PublishTx函数无法成功读取http请求,err:%v\n", err)
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		} else {
			loglogrus.Log.Warnf("Http: PublishTx函数成功读取http请求,body比特流长度:%v\n", len(body))
		}
		// Parse函数
		txList, err := consensus.DeserializeTransactionsSeries(body)
		if err != nil {
			loglogrus.Log.Warnf("Http: PublishTx函数无法解析http请求内容,err:%v\n", err)
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		} else {
			loglogrus.Log.Warnf("Http: PublishTx函数解析获得的交易数量:%v\n", len(txList))
		}

		// 2. 直接上传给consensusPromoter
		txListPtr := make([]*eles.Transaction, 0)
		for _, tx := range txList {
			txPtr := new(eles.Transaction)
			copyTransaction(tx, txPtr)
			txListPtr = append(txListPtr, txPtr)
		}

		err = ds.PublishTx(txListPtr)
		if err != nil {
			loglogrus.Log.Warnf("Http: PublishTx函数无法上传交易,err:%v\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		}
	}
}

func copyTransaction(src eles.Transaction, dst *eles.Transaction) {
	dst.TxID = src.TxID
	dst.Sender = src.Sender
	dst.Nonce = src.Nonce
	dst.Version = src.Version
	dst.LifeTime = src.LifeTime
	dst.Signature = append(dst.Signature, src.Signature...)
	dst.Contract = src.Contract
	dst.Function = src.Function
	dst.Args = append(dst.Args, src.Args...)
	dst.CheckList = append(dst.CheckList, src.CheckList...)
}

// contractName:合约名
// functionName:参数名
// args: 参数列表,以空格分割
func SoftCall(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		contractName := c.PostForm("contractName")
		functionName := c.PostForm("functionName")
		args := c.PostForm("args")
		valid := validation.Validation{}
		valid.Required(contractName, "contractName").Message("合约名不能为空")
		valid.Required(functionName, "functionName").Message("函数名不能为空")
		valid.Required(args, "args").Message("参数不能为空")

		if valid.HasErrors() {
			app.MarkErrors("SoftCall", valid.Errors)
			appG.Response(http.StatusOK, e.INVALID_PARAMS, nil)
			return
		}
		loglogrus.Log.Infof("Http: Get user request -- SoftCall  contractName:%s , functionName:%s , args:%s\n", contractName, functionName, args)

		commandStr := "call " + contractName + " " + functionName + " -args " + args

		data, err := ds.SoftCall(commandStr)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- SoftCall  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, data)
		}

		loglogrus.Log.Infof("Http: Reply to user request -- SoftCall  succeed!\n")
	}

}

func BecomeBooter(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- BecomeBooter")
		appG := app.Gin{c}
		err := ds.BecomeBooter()
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- BecomeBooter  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, nil)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- BecomeBooter  succeed!\n")
	}

}

func BackViewNet(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- BackViewNet")
		appG := app.Gin{c}
		data, err := ds.BackViewNet()
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- BackViewNet  warn:%s\n", err)
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			appG.Response(http.StatusOK, e.SUCCESS, data)
		}
		loglogrus.Log.Infof("Http: Reply to user request -- BackViewNet  succeed!\n")
	}

}

func HelpMenu(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		data := ds.HelpMenu()
		appG.Response(http.StatusOK, e.SUCCESS, data)
	}
}

func Exit(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		loglogrus.Log.Infof("Http: Get user request -- Exit\n")
		appG := app.Gin{c}
		appG.Response(http.StatusOK, e.SUCCESS, "ByeBye")
		defer ds.Exit()
	}
}

func SendVCrequest(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		// 假设 destinationHost 作为路由参数或从配置中传递
		destinationHost := c.Param("destinationHost")
		destinationPort := c.Param("destinationPort")
		SubjectDID := c.PostForm("SubjectDID")
		Identifier := uuid.New().String()

		// 构造目标 URL
		destinationURL := fmt.Sprintf("http://%s:%s/dper/vcreceive", destinationHost, destinationPort)

		// 准备要发送的数据
		formData := url.Values{
			"SubjectDID": {SubjectDID},
			"Identifier": {Identifier},
		}

		// 发送 HTTP POST 请求
		resp, err := http.PostForm(destinationURL, formData)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}
		defer resp.Body.Close()

		// 读取响应内容
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}

		// 解码 JSON 响应
		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}

		c.JSON(http.StatusOK, response)
	}
}

func VCReceive(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}

		// 从请求中获取 SubjectDID 和 Identifier
		subjectDID := c.PostForm("SubjectDID")
		identifier := c.PostForm("Identifier")
		if subjectDID == "" || identifier == "" {
			appG.Response(http.StatusBadRequest, e.ERROR, "Missing SubjectDID or Identifier")
			return
		}

		// 构造响应
		response := gin.H{
			"Identifier": identifier,
			"SubjectDID": subjectDID,
		}

		// 返回响应给调用方
		c.JSON(http.StatusOK, response)
	}
}

func SendVC(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		// 假设 destinationHost 作为路由参数或从配置中传递
		destinationHost := c.Param("destinationHost")
		destinationPort := c.Param("destinationPort")
		// 从请求中获取数据
		Identifier := c.PostForm("Identifier")
		Issuer := c.PostForm("Issuer")
		Subject := c.PostForm("Subject")
		Validity := c.PostForm("Validity")
		SubjectAddress := c.PostForm("SubjectAddress")
		Signature, _ := ds.SignMessage(Identifier)
		IssuerAddress := ds.CurrentAccount()

		Purpose := "GetData"
		Reassign := "OK"
		// 构造VC
		vc := VC{
			Identifier:     Identifier,
			Subject:        Subject,
			Issuer:         Issuer,
			Validity:       Validity,
			Purpose:        Purpose,
			Signature:      Signature,
			Reassign:       Reassign,
			IssuerAddress:  IssuerAddress,
			SubjectAddress: SubjectAddress,
		}
		//VC转为字符串
		vcString := fmt.Sprintf("Identifier:%s,Subject:%s,Issuer:%s,Validity:%s,Purpose:%s,Signature:%s,Reassign:%s,IssuerAddress:%s,SubjectAddress:%s",
			vc.Identifier, vc.Subject, vc.Issuer, vc.Validity, vc.Purpose, vc.Signature, vc.Reassign, vc.IssuerAddress, vc.SubjectAddress)
		// 构造目标 URL

		// 使用提供的主机和端口构造目标 URL
		destinationURL := fmt.Sprintf("http://%s:%s/dper/vcvalid", destinationHost, destinationPort)

		// 准备要发送的数据
		formData := url.Values{
			"vc":         {vcString},
			"signature":  {Signature},
			"identifier": {Identifier},
			"subject":    {Subject},
		}
		// 发送 HTTP POST 请求
		resp, err := http.PostForm(destinationURL, formData)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}
		defer resp.Body.Close()

		// 读取响应内容
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}

		// 解码 JSON 响应
		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}

		c.JSON(http.StatusOK, response)
	}
}

func VCValid(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}

		// 从请求中获取VC字符串
		vcString := c.PostForm("vc")
		Signature := c.PostForm("signature")
		Identifier := c.PostForm("identifier")
		Subject := c.PostForm("subject")
		if vcString == "" {
			appG.Response(http.StatusBadRequest, e.ERROR, "No VC data received")
			return
		}
		//调用合约
		contractName := "DID::SPECTRUM::TRADE"
		functionName := "VCValid"
		args := vcString
		commandStr := "invoke " + contractName + " " + functionName + " -args " + args

		receipt, err := ds.SoftInvoke(commandStr)
		if err != nil {
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			if reflect.DeepEqual(receipt, transactionCheck.CheckResult{}) {
				appG.Response(http.StatusOK, e.SUCCESS, nil)
			} else {
				response := gin.H{
					"Identifier": Identifier,
					"SubjectDID": Subject,
					"Signature":  Signature,
				}
				// 返回响应给调用方
				c.JSON(http.StatusOK, response)
			}
		}

	}
}

func SignValid(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		// 解析消息内容
		message := c.PostForm("message")
		signature := c.PostForm("signature")
		//address相当于pk
		address := c.PostForm("address")
		// 获取签名和事务ID

		msg := crypto.Sha3Hash([]byte(message))
		sig := common.Hex2Bytes(signature)
		add := common.HexToAddress(address)
		ok, err := crypto.SignatureValid(add, sig, msg)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status": "error",
				"error":  err.Error(),
			})
			return
		} else {
			if ok == true {
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			} else if ok == false {
				c.JSON(http.StatusOK, gin.H{"status": "fail"})
			}
		}

	}

}

func DataRequest(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		// 获取目标端口和消息内容
		destinationHost := c.Param("destinationHost")
		destinationPort := c.Param("destinationPort")
		ID := c.PostForm("ID")

		// 签名消息
		signature, err := ds.SignMessage(ID)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}
		// 构造目标 URL
		destinationURL := fmt.Sprintf("http://%s:%s/dper/datasend", destinationHost, destinationPort)

		// 发送 HTTP POST 请求
		resp, err := http.PostForm(destinationURL, url.Values{
			"ID":        {ID},
			"signature": {signature},
		})
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}
		defer resp.Body.Close()

		// 读取响应
		responseBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}

		// 返回响应给调用方
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("ID: %s, response: %s", ID, string(responseBody)),
		})
	}
}

func DataSend(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		ID := c.PostForm("ID")
		signature := c.PostForm("signature")

		// 合约调用逻辑...
		contractName := "DID::SPECTRUM::TRADE"
		functionName := "GetVC"
		args := ID
		commandStr := "invoke " + contractName + " " + functionName + " -args " + args
		receipt, err := ds.SoftInvoke(commandStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
			return
		}

		subjectAddress, _ := ResultToSubjectAddress(receipt.Result)
		issuer, _ := ResultToIssuer(receipt.Result)
		msg := crypto.Sha3Hash([]byte(ID))
		sig := common.Hex2Bytes(signature)
		add := common.HexToAddress(subjectAddress)
		ok, err := crypto.SignatureValid(add, sig, msg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
			return
		}

		if ok {
			data, err := FindDataInFile(issuer)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
				return
			}
			// 直接返回结果给调用方
			c.JSON(http.StatusOK, gin.H{
				"status": "success",
				"data":   data,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{"status": "error", "error": "Signature validation failed"})
		}
	}
}
func DataRequest2(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		// 获取目标端口和消息内容
		destinationHost := c.Param("destinationHost")
		destinationPort := c.Param("destinationPort")
		ID := c.PostForm("ID")

		// 签名消息
		signature, err := ds.SignMessage(ID)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}
		// 构造目标 URL
		destinationURL := fmt.Sprintf("http://%s:%s/dper/vcvalid", destinationHost, destinationPort)

		// 发送 HTTP POST 请求
		resp, err := http.PostForm(destinationURL, url.Values{
			"ID":        {ID},
			"signature": {signature},
		})
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}
		defer resp.Body.Close()

		// 读取响应
		responseBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}

		// 返回响应给调用方
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("ID: %s, response: %s", ID, string(responseBody)),
		})
	}
}
func DataSend2(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		ID := c.PostForm("ID")
		signature := c.PostForm("signature")
		// 合约调用逻辑...
		contractName := "DID::SPECTRUM::TRADE"
		functionName := "GetVC"
		args := ID
		commandStr := "invoke " + contractName + " " + functionName + " -args " + args
		receipt, err := ds.SoftInvoke(commandStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
			return
		}

		subjectAddress, _ := ResultToRecipientAddress(receipt.Result)
		issuer, _ := ResultToIssuer(receipt.Result)
		msg := crypto.Sha3Hash([]byte(ID))
		sig := common.Hex2Bytes(signature)
		add := common.HexToAddress(subjectAddress)
		ok, err := crypto.SignatureValid(add, sig, msg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
			return
		}

		if ok {
			data, err := FindDataInFile(issuer)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
				return
			}
			// 直接返回结果给调用方
			c.JSON(http.StatusOK, gin.H{
				"status": "success",
				"data":   data,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{"status": "error", "error": "Signature validation failed"})
		}
	}
}

func SignatureReturn(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		// 解析消息内容
		message := c.PostForm("message")
		// 获取签名和事务ID
		signature, err := ds.SignMessage(message)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status": "error",
				"error":  err.Error(),
			})
			return
		} else {
			c.JSON(http.StatusOK, gin.H{"signature": signature})
		}

	}
}

func SignatureReturn2(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		// 解析消息内容
		message := c.PostForm("message")
		// 获取签名和事务ID
		address := ds.CurrentAccount()
		signature, err := ds.SignMessage(message)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status": "error",
				"error":  err.Error(),
			})
			return
		} else {
			c.JSON(http.StatusOK, gin.H{
				"signature": signature,
				"address":   address,
			})
		}

	}
}

func SendRandom(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		// 获取目标端口和消息内容
		// 假设 destinationHost 作为路由参数或从配置中传递
		destinationHost := c.Param("destinationHost")
		destinationPort := c.Param("destinationPort")
		message := c.PostForm("message")
		// 构造目标 URL
		destinationURL := fmt.Sprintf("http://%s:%s/dper/signaturereturn", destinationHost, destinationPort)

		// 发送 HTTP POST 请求
		resp, err := http.PostForm(destinationURL, url.Values{"message": {message}})
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}

		defer resp.Body.Close()

		// 读取响应
		responseBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}
		// 解析 JSON 响应以获取签名
		var responseJSON map[string]string
		err = json.Unmarshal(responseBody, &responseJSON)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}

		signature, ok := responseJSON["signature"]
		if !ok {
			appG.Response(http.StatusInternalServerError, e.ERROR, "Signature not found in response")
			return
		}

		// 返回响应给调用方
		c.JSON(http.StatusOK, gin.H{"signature": signature})

	}
}

func TransVCrequest(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		// 假设 destinationHost 作为路由参数或从配置中传递
		destinationHost := c.Param("destinationHost")
		destinationPort := c.Param("destinationPort")

		// 从请求中获取数据
		VCID := c.PostForm("vcid")
		Recipient := c.PostForm("recipient")
		RecipientPurpose := "GetData"
		RecipientAddress := ds.CurrentAccount()

		//调用合约
		contractName := "DID::SPECTRUM::TRADE"
		functionName := "GetVC"
		args := VCID
		commandStr := "invoke " + contractName + " " + functionName + " -args " + args
		receipt, err := ds.SoftInvoke(commandStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
			return
		}
		oldvcstring, err := ResultToVC(receipt.Result)

		transferString := fmt.Sprintf("%s,VCID:%s,Recipient:%s,RecipientPurpose:%s,RecipientAddress:%s", oldvcstring, VCID, Recipient, RecipientPurpose, RecipientAddress)
		// 构造目标 URL
		destinationURL := fmt.Sprintf("http://%s:%s/dper/transvc", destinationHost, destinationPort)

		// 准备要发送的数据
		formData := url.Values{
			"VCID":             {VCID},
			"Recipient":        {Recipient},
			"RecipientAddress": {RecipientAddress},
			"transferstring":   {transferString},
		}
		// 发送 HTTP POST 请求
		resp, err := http.PostForm(destinationURL, formData)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, "resp wrong")
			return
		}
		defer resp.Body.Close()

		// 读取响应内容
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, "body wrong")
			return
		}

		// 解码 JSON 响应
		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, "json wrong")
			return
		}

		c.JSON(http.StatusOK, response)
	}
}

func TransVC(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		// 从请求中获取VC字符串
		VCID := c.PostForm("VCID")
		Recipient := c.PostForm("Recipient")
		RecipientAddress := c.PostForm("RecipientAddress")
		transferString := c.PostForm("transferstring")
		if transferString == "" {
			appG.Response(http.StatusBadRequest, e.ERROR, "No VC data received")
			return
		}
		signature, _ := ds.SignMessage(VCID)
		newvcString := fmt.Sprintf("%s,DonorSignature:%s", transferString, signature)
		str := removeSpaces(newvcString)
		// 调用合约
		contractName := "DID::SPECTRUM::TRADE"
		functionName := "ResetVC"
		args := str
		commandStr := "invoke " + contractName + " " + functionName + " -args " + args
		receipt, err := ds.SoftInvoke(commandStr)
		if err != nil {
			appG.Response(http.StatusOK, e.ERROR, "receipt wrong")
			return
		} else {
			if reflect.DeepEqual(receipt, transactionCheck.CheckResult{}) {
				appG.Response(http.StatusOK, e.SUCCESS, "receipt wrong2")
			} else {
				// 返回响应给调用方
				response := gin.H{
					"Recipient":        Recipient,
					"RecipientAddress": RecipientAddress,
					"DonorSignature":   signature,
				}
				c.JSON(http.StatusOK, response)
			}
		}
	}
}

func VCReceive2(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}

		// 从请求中获取VC字符串
		vcString := c.PostForm("transferstring")
		if vcString == "" {
			appG.Response(http.StatusBadRequest, e.ERROR, "No VCrequest data received")
			return
		}
		// 打印接收到的VC字符串
		fmt.Printf("Received VC String: %s\n", vcString)

		// 返回响应给调用方
		appG.Response(http.StatusOK, e.SUCCESS, fmt.Sprintf("Received VCrequest: %s", vcString))
	}
}
func StringToMap(str string) (map[string]string, error) {
	resultMap := make(map[string]string)
	pairs := strings.Split(str, ",")
	for _, pair := range pairs {
		// 找到第一个冒号的位置
		idx := strings.Index(pair, ":")
		if idx == -1 {
			return nil, fmt.Errorf("invalid pair (no colon found): %s", pair)
		}

		key := pair[:idx]
		value := pair[idx+1:]
		resultMap[key] = value
	}
	return resultMap, nil
}

func SetDID(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		contractName := "DID::SPECTRUM::TRADE"
		functionName := "SetDID"
		args1 := c.PostForm("DID")
		args2 := ds.CurrentAccount()
		args := args1 + " " + "address:" + args2 // 修改这里，添加空格
		commandStr := "invoke " + contractName + " " + functionName + " -args " + args
		receipt, err := ds.SoftInvoke(commandStr)
		if err != nil {
			appG.Response(http.StatusOK, e.ERROR, err)
			return
		} else {
			if reflect.DeepEqual(receipt, transactionCheck.CheckResult{}) {
				appG.Response(http.StatusOK, e.SUCCESS, nil)
			} else {
				type ResponseReceipt struct {
					TxID   string `json:"Transaction ID"`
					Valid  bool   `json:"Valid"`
					Result string `json:"Transaction Results"`
					Delay  string `json:"Consensus Delay"`
				}
				var r ResponseReceipt = ResponseReceipt{
					TxID:   fmt.Sprintf("%x", receipt.TransactionID),
					Valid:  receipt.Valid,
					Result: fmt.Sprintf("%s", receipt.Result),
					Delay:  fmt.Sprintf("%d ms", receipt.Interval.Milliseconds()),
				}
				appG.Response(http.StatusOK, e.SUCCESS, r)
			}
		}
	}
}

func AMFsend(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{c}
		// 解析消息内容
		destinationHost := c.Param("destinationHost")
		destinationPort := c.Param("destinationPort")
		DID := c.PostForm("DID")
		// 获取签名和事务ID
		// 签名消息
		signature, err := ds.SignMessage(DID)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}

		// 构造目标 URL
		destinationURL := fmt.Sprintf("http://%s:%s/dper/SMFreceive", destinationHost, destinationPort)
		log.Println("Destination URL:", destinationURL) // 添加这行来检查URL是否正确构造
		// 发送 HTTP POST 请求
		resp, err := http.PostForm(destinationURL, url.Values{
			"DID":       {DID},
			"signature": {signature},
		})
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}
		defer resp.Body.Close()

		// 读取响应
		responseBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			appG.Response(http.StatusInternalServerError, e.ERROR, err)
			return
		}

		// 返回响应给调用方
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("DID: %s,res: %s", DID, string(responseBody)),
		})
	}

}

func GetAddress(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		DID := c.PostForm("DID")
		// 合约调用逻辑...
		contractName := "DID::SPECTRUM::TRADE"
		functionName := "GetAddress"
		args := DID

		commandStr := "call " + contractName + " " + functionName + " -args " + args
		Address := ""
		data, err := ds.SoftCall(commandStr)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- SoftCall  warn:%s\n", err)
			return
		} else {
			Address = strings.Join(data, "")
			address, err := ExtractAddress(Address)
			if err != nil {
				fmt.Println("Error:", err)
				return
			} else {
				c.JSON(http.StatusOK, gin.H{"address": address})
			}

		}
		// commandStr := "invoke " + contractName + " " + functionName + " -args " + args
		// receipt, err := ds.SoftInvoke(commandStr)
		// if err != nil {
		// 	c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
		// 	return
		// }
		// Address, err := ResultToAddress(receipt.Result)
		// if err != nil {
		// 	c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
		// 	return
		// } else {
		// 	c.JSON(http.StatusOK, gin.H{"address": Address})
		// }
	}
}
func SMFreceive(ds *api.DperService) func(*gin.Context) {
	return func(c *gin.Context) {
		// 解析消息内容

		DID := c.PostForm("DID")
		signature := c.PostForm("signature")
		//getaddress
		contractName := "DID::SPECTRUM::TRADE"
		functionName := "GetAddress"
		args := DID
		commandStr := "call " + contractName + " " + functionName + " -args " + args
		Address := ""
		data, err := ds.SoftCall(commandStr)
		if err != nil {
			loglogrus.Log.Warnf("Http: Reply to user request -- SoftCall  warn:%s\n", err)
			return
		} else {
			Address = strings.Join(data, "")
		}
		//signaturevalid
		address, err := ExtractAddress(Address)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		msg := crypto.Sha3Hash([]byte(DID))
		sig := common.Hex2Bytes(signature)
		add := common.HexToAddress(address)
		ok, err := crypto.SignatureValid(add, sig, msg)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status": "fail",
				"error":  err.Error(),
			})
			return
		} else {
			if ok == true {
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			} else if ok == false {
				c.JSON(http.StatusOK, gin.H{"status": "fail"})
			}
		}

	}

}
