package contract

import (
	"fmt"
	"p3Chain/common"
	"p3Chain/core/eles"
	"p3Chain/core/worldstate"
	loglogrus "p3Chain/log_logrus"
	"strconv"
	"sync"
)

const (
	domain_TypeCodeLength = 1 // the type code to infer a domain type (contract or storage), which is 8 bits and 1 byte
	domain_ContractCode   = byte(0x1)
	domain_StorageCode    = byte(0x2)

	contract_ValueTypeCodeLength = 1         // type code is to specify the type of a inner address in contract, which is one byte, 8 bits
	contract_FunctionCode        = byte(0x1) // type of function
	contract_BytesCode           = byte(0x2) // type of a bytes value
	contract_Int64Code           = byte(0x3) // type of an int64 value
)

// 0x00 of inner address is to specify the domain address
var domainInfoAddress = common.Address{}

type PrecompileContractEngine struct {
	useMux sync.Mutex

	stateManager worldstate.StateManager
	valueBox     *sandBox
	funcCpl      functionCompiler
	checkIpt     CheckElementInterpreter
}

type sandBox struct {
	accessMu sync.Mutex
	readSet  map[common.WsKey]worldStateValueType
	writeSet map[common.WsKey]worldStateValueType
}

// empty reset the sandbox to handle a new round of transaction executions
func (sb *sandBox) empty() {
	sb.accessMu.Lock()
	defer sb.accessMu.Unlock()
	sb.readSet = make(map[common.WsKey]worldStateValueType)
	sb.writeSet = make(map[common.WsKey]worldStateValueType)
}

// add try to add a world state key value pair in to the sandbox
func (sb *sandBox) add(wsKey common.WsKey, value worldStateValueType) error {
	sb.accessMu.Lock()
	defer sb.accessMu.Unlock()
	if _, ok := sb.readSet[wsKey]; ok {
		return fmt.Errorf("value has already existed!")
	}
	sb.readSet[wsKey] = value
	return nil
}

// get try to get the value in sandbox, if not have return nil and false
func (sb *sandBox) get(wsKey common.WsKey) (worldStateValueType, bool) {
	sb.accessMu.Lock()
	defer sb.accessMu.Unlock()
	if ele, ok := sb.readSet[wsKey]; ok {
		return ele, true
	} else {
		return nil, false
	}
}

// set try to set the value of the write set of sandbox, also updates the
// read set value
func (sb *sandBox) set(wsKey common.WsKey, value worldStateValueType) {
	sb.accessMu.Lock()
	defer sb.accessMu.Unlock()
	sb.writeSet[wsKey] = value
	sb.readSet[wsKey] = value
}

func (sb *sandBox) exportWriteSet() []eles.WriteEle {
	sb.accessMu.Lock()
	defer sb.accessMu.Unlock()
	writeSet := make([]eles.WriteEle, 0)
	for wsKey, ele := range sb.writeSet {
		we := eles.WriteEle{
			ValueAddress: wsKey,
			Value:        ele.toBytes(),
		}
		writeSet = append(writeSet, we)
	}
	writeSet = SortWriteEles(writeSet)
	return writeSet
}

// create a new precompile contract engine using the base function
// compiler.
func NewPrecompileCE(sm worldstate.StateManager, fc functionCompiler, ci CheckElementInterpreter) *PrecompileContractEngine {
	sb := &sandBox{
		readSet:  make(map[common.WsKey]worldStateValueType),
		writeSet: make(map[common.WsKey]worldStateValueType),
	}
	ce := &PrecompileContractEngine{
		stateManager: sm,
		valueBox:     sb,
		funcCpl:      fc,
		checkIpt:     ci,
	}
	return ce
}

func (pce *PrecompileContractEngine) ExecuteTransactions(txs []eles.Transaction) ([]eles.TransactionReceipt, []eles.WriteEle) {
	pce.useMux.Lock()
	defer pce.useMux.Unlock()

	pce.valueBox.empty()
	receipts := make([]eles.TransactionReceipt, 0)

	currentVersion := pce.stateManager.CurrentVersion()

	for _, tx := range txs {
		if pce.stateManager.StaleCheck(tx.Version, currentVersion, tx.LifeTime) {
			loglogrus.Log.Warnf("[Contract Engine] Execute Transaction(%x) is failed, Can't pass StaleCheck\n", tx.TxID)
			receipts = append(receipts, eles.CreateInvalidTransactionReceipt())
			continue
		}

		if !pce.ValidateCheckElements(tx.CheckList) {
			loglogrus.Log.Warnf("[Contract Engine] Execute Transaction(%x) is failed, Can't pass ValidateCheckElements\n", tx.TxID)
			receipts = append(receipts, eles.CreateInvalidTransactionReceipt())
			continue
		}
		funcWsKey := common.AddressesCatenateWsKey(tx.Contract, tx.Function)
		vt, err := pce.Get(funcWsKey)
		if err != nil {
			loglogrus.Log.Warnf("[Contract Engine] Contract failed: Get function (%x) is failed, You could Check the function Address (%x) whether is correct!\n", funcWsKey, tx.Function)
			receipts = append(receipts, eles.CreateInvalidTransactionReceipt())
			continue
		}
		fn, ok := vt.(*contractFunction)
		if !ok {
			loglogrus.Log.Warnf("[Contract Engine] Contract failed: Wskey (%x) is not a function!\n", funcWsKey)
			receipts = append(receipts, eles.CreateInvalidTransactionReceipt())
			continue
		}
		res, err := fn.do(tx.Args, tx.Sender)
		if err != nil {
			loglogrus.Log.Warnf("[Contract Engine] Contract failed: Wskey (%x) fail in execution!\n", funcWsKey)
			receipts = append(receipts, eles.CreateInvalidTransactionReceipt())
			continue
		}
		loglogrus.Log.Infof("[Contract Engine] Contract succeed: Transaction(%x) excute successfully! \n", tx.TxID)
		receipts = append(receipts, eles.CreateValidTransactionReceipt(res))
		//loglogrus.Log.Infof("Contract: Transaction (%x) Receipt has been validated!\n", tx.TxID)
	}
	writeSet := pce.valueBox.exportWriteSet()
	return receipts, writeSet
}

func (pce *PrecompileContractEngine) CommitWriteSet(writeSet []eles.WriteEle) error {
	pce.useMux.Lock()
	defer pce.useMux.Unlock()

	keys := make([]common.WsKey, len(writeSet))
	values := make([][]byte, len(writeSet))
	for i := 0; i < len(writeSet); i++ {
		keys[i] = writeSet[i].ValueAddress
		values[i] = writeSet[i].Value
	}
	err := pce.stateManager.WriteWorldStateValues(keys, values)
	return err
}

func (pce *PrecompileContractEngine) ValidateCheckElements(checkEle []eles.CheckElement) bool {
	for i := 0; i < len(checkEle); i++ {
		if !pce.checkIpt.validate(pce, checkEle[i]) {
			return false
		}
	}
	return true
}

// TODO:
func (pce *PrecompileContractEngine) Get(wsKey common.WsKey) (worldStateValueType, error) {
	if val, ok := pce.valueBox.get(wsKey); ok {
		return val, nil
	}
	plainTexts, err := pce.stateManager.ReadWorldStateValues(wsKey)
	if err != nil {
		return nil, err
	}
	plainText := plainTexts[0]
	domainAddr, innerAddr := common.WsKeySplitToAddresses(wsKey)
	switch domainAddr[0] {
	case domain_ContractCode:
		if innerAddr == domainInfoAddress {
			value := &domainInfo{
				plainValue: plainText,
			}
			pce.valueBox.add(wsKey, value)
			return value, nil
		}
		switch innerAddr[0] {
		case contract_FunctionCode:
			fn, err := pce.parseFunction(plainText)
			if err != nil {
				return nil, err
			}
			value := &contractFunction{
				do: fn,
			}
			pce.valueBox.add(wsKey, value)
			return value, nil
		case contract_BytesCode:
			value := &commonBytes{
				value: plainText,
			}
			pce.valueBox.add(wsKey, value)
			return value, nil
		case contract_Int64Code:
			num, err := strconv.Atoi(string(plainText))
			if err != nil {
				return nil, fmt.Errorf("type error: %s cannot convert into int64", string(plainText))
			}
			num64 := int64(num)
			value := &commonInt64{
				value: num64,
			}
			pce.valueBox.add(wsKey, value)
			return value, nil
		default:
			return nil, fmt.Errorf("unknown inner address of a contract")
		}

	case domain_StorageCode:
		if innerAddr == domainInfoAddress {
			value := &domainInfo{
				plainValue: plainText,
			}
			pce.valueBox.add(wsKey, value)
			return value, nil
		}

		value := &commonBytes{
			value: plainText,
		}
		pce.valueBox.add(wsKey, value)
		return value, nil
	default:
		return nil, fmt.Errorf("unknown domain address")
	}
}

func (pce *PrecompileContractEngine) Set(wsKey common.WsKey, value worldStateValueType) {
	pce.valueBox.set(wsKey, value)
}

func (pce *PrecompileContractEngine) parseFunction(plainText []byte) (func(args [][]byte, self common.Address) ([][]byte, error), error) {
	return pce.funcCpl.compileFunc(pce, plainText)
}
