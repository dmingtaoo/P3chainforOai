package chaincodesupport

// import (
// 	"p3Chain/common"
// 	"p3Chain/core/contract"
// 	"p3Chain/rlp"
// 	"fmt"
// 	"net"
// 	"net/rpc"
// )

// // DperService prvide worldstatus for chaincode
// type DperService struct {
// 	Dper_ip             string
// 	ContractStorageAddr common.Address
// 	Self 				common.Address
// }
// // remote_ip is depr ip
// func (d *DperService) GetStatus(key []byte) ([]byte, error) {
// 	client, err := rpc.Dial("tcp", d.Dper_ip)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var reply string
// 	innerAddr := contract.BytesToHashAddress(key)
// 	wsKey := common.AddressesCatenateWsKey(d.ContractStorageAddr, innerAddr)
// 	err = client.Call("RpcChainCodeAPI"+d.Dper_ip+".Get", string(wsKey[:]), &reply)
// 	if err != nil {
// 		return nil, err
// 	}
// 	receive := new(contract.GET_return_type)
// 	tmp := []byte(reply)
// 	rlp.DecodeBytes(tmp, receive)
// 	if receive.Code != contract.GET_OK {
// 		return nil, fmt.Errorf("receive.code not ok")
// 	}
// 	return []byte(receive.Payload), nil
// }

// func (d *DperService) UpdateStatus(key []byte, value []byte) error {
// 	client, err := rpc.Dial("tcp", d.Dper_ip)
// 	if err != nil {
// 		return err
// 	}
// 	innerAddr := contract.BytesToHashAddress(key)
// 	wsKey := common.AddressesCatenateWsKey(d.ContractStorageAddr, innerAddr)
// 	set_data := &contract.SET_args{
// 		WsKey: wsKey[:],
// 		Value: value,
// 	}
// 	tmp, _ := rlp.EncodeToBytes(set_data)
// 	var reply string
// 	err = client.Call("RpcChainCodeAPI"+d.Dper_ip+".Set", string(tmp[:]), &reply)
// 	if err != nil {
// 		return err
// 	}
// 	if (reply) != "ok" {
// 		return fmt.Errorf("chainCode reply error")
// 	}
// 	return nil
// }
// func (d *DperService) UpdateStatusList(key [][]byte, value [][]byte) error {
// 	client, err := rpc.Dial("tcp", d.Dper_ip)
// 	if err != nil {
// 		return err
// 	}
// 	var set_data  contract.SET_LIST_args
// 	for k,v := range key{
// 		innerAddr := contract.BytesToHashAddress(v)
// 		wsKey := common.AddressesCatenateWsKey(d.ContractStorageAddr, innerAddr)
// 		set_data.WsKey = append(set_data.WsKey, wsKey[:])
// 		set_data.Value = append(set_data.Value, value[k])
// 	}

// 	tmp, _ := rlp.EncodeToBytes(set_data)
// 	var reply string
// 	err = client.Call("RpcChainCodeAPI"+d.Dper_ip+".SetList", string(tmp), &reply)
// 	if err != nil {
// 		return err
// 	}
// 	if (reply) != "ok" {
// 		return fmt.Errorf("chainCode reply error")
// 	}
// 	return nil
// }
// func (d *DperService) BytesTo20thAddress(aim []byte) []byte{
// 	return contract.BytesToHashAddress(aim).Bytes()
// }

// type ContractFunc func(args [][]byte,ds DperService) ([][]byte, error)

// type ChainCodeEngine struct {
// 	dper_ip     string
// 	funcMap map[string]ContractFunc
// }

// func (s *ChainCodeEngine) Call_func(request string, reply *string) error {

// 	request_data := new(contract.Function_invoke_type)
// 	rlp.DecodeBytes([]byte(request), request_data)
// 	reply_data := new(contract.Function_return_type)

// 	ds :=&DperService{
// 		Dper_ip: s.dper_ip,
// 		ContractStorageAddr: contract.StringToStorageAddress(string(request_data.ContractName)),
// 		Self: request_data.Self,
// 	}

// 	fn := s.funcMap[string(request_data.FunctionName)]
// 	if fn == nil{
// 		reply_data.Err = fmt.Errorf("function not exist")
// 		tmp, _ := rlp.EncodeToBytes(reply_data)
// 		*reply = string(tmp)
// 		return nil
// 	}
// 	reply_data.Result, reply_data.Err = fn(request_data.Args,*ds)
// 	tmp, _ := rlp.EncodeToBytes(reply_data)
// 	*reply = string(tmp)
// 	return nil
// }
// func ContractExecute(Dper_ip string,local_ip string,funcMap map[string]ContractFunc) error {
// 	CCE := new(ChainCodeEngine)
// 	CCE.dper_ip = Dper_ip
// 	CCE.funcMap = funcMap
// 	rpc.RegisterName("ChainCodeEngine", CCE)
// 	listener, err := net.Listen("tcp", local_ip)
// 	if err != nil {
// 		return err
// 	}
// 	for {
// 		conn, err := listener.Accept()
// 		if err != nil {
// 			return err
// 		}
// 		go rpc.ServeConn(conn)
// 	}
// }
// func InstallContract(contractName string,funcMap map[string]ContractFunc, dper_ip string) error {

// 	var installKeys []common.WsKey
// 	var installValues [][]byte

// 	contract_addr := contract.StringToContractAddress(contractName)
// 	contract_wskey := common.AddressesCatenateWsKey(contract_addr, common.Address{})
// 	installKeys = append(installKeys, contract_wskey)
// 	installValues = append(installValues, []byte(contractName))
// 	for key := range funcMap {
// 		tmp := contract.StringToFunctionAddress(key)
// 		installKeys =append(installKeys, common.AddressesCatenateWsKey(contract_addr,tmp))
// 		installValues =append(installValues, []byte(key))
// 	}
// 	install_data := &contract.Contract_install_type{
// 		InstallKeys:   installKeys,
// 		InstallValues: installValues,
// 	}
// 	tmp, _ := rlp.EncodeToBytes(install_data)
// 	var reply string
// 	client, err := rpc.Dial("tcp", dper_ip)
// 	if err != nil {
// 		return err
// 	}
// 	err = client.Call("RpcChainCodeAPI"+dper_ip+".InstallContract", string(tmp[:]), &reply)
// 	if err != nil {
// 		return err
// 	}

// 	if (reply) != "install succeed" {
// 		return fmt.Errorf(reply)
// 	}
// 	return nil
// }
