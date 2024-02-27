package FLAPI

type JsonRPCPayload struct {
	Method  string `json:"method"`
	Params  []any  `json:"params"`
	JsonRPC string `json:"jsonrpc"`
	Id      int    `json:"id"`
}

type JsonRPCResult struct {
	Result  string `json:"result"`
	Id      int    `json:"id"`
	JsonRPC string `json:"jsonrpc"`
}

type AggregateRequest struct {
	Models []string `json:"models"`
}

type ModelTrainRequest struct {
	Rank   int      `json:"rank"`
	Models []string `json:"models"`
}

type EvaluateRequest struct {
	Rank  int    `json:"rank"`
	Model string `json:"model"`
}
