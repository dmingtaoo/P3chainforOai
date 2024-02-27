package contract

import (
	"p3Chain/common"
)

const (
	GET_OK    = 0
	GET_ERROR = 1
)

type GET_return_type struct {
	Code    byte
	Payload string
}

type SET_args struct {
	WsKey []byte
	Value []byte
}
type SET_LIST_args struct {
	WsKey [][]byte
	Value [][]byte
}
type Function_invoke_type struct {
	Args         [][]byte
	ContractName []byte
	FunctionName []byte
	Self         common.Address //who invoke the function
}
type Function_return_type struct {
	Result [][]byte
	Err    error
}

type Contract_install_type struct {
	InstallKeys   []common.WsKey
	InstallValues [][]byte
}

type Pipe_type struct {
	Request_type string
	payload      string
}
