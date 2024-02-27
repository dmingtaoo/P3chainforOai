package FLAPI

func ParseJsonRPCPayload(method string, id int, params ...any) *JsonRPCPayload {
	temp_params := make([]any, 0)
	temp_params = append(temp_params, params...)
	return &JsonRPCPayload{
		Method:  method,
		Params:  temp_params,
		JsonRPC: "2.0",
		Id:      id,
	}
}
