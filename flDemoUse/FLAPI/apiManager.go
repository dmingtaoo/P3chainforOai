package FLAPI

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type FLManager struct {
	Rank   int
	client *http.Client
}

func NewFLManager(rank int) *FLManager {
	return &FLManager{
		Rank:   rank,
		client: &http.Client{},
	}
}

func (fm *FLManager) SendRequest(info []byte) (*JsonRPCResult, error) {
	req, err := http.NewRequest("POST", "http://localhost:4000/jsonrpc", bytes.NewReader(info))
	if err != nil {
		return nil, err
	}

	resp, err := fm.client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var res JsonRPCResult
	json.Unmarshal(body, &res)
	return &res, nil
}

func (fm *FLManager) Aggregate(models ...[]byte) ([]byte, error) {
	paras := make([]string, 0)
	for _, model := range models {
		strpara := base64.StdEncoding.EncodeToString(model)
		paras = append(paras, strpara)
	}
	request := &AggregateRequest{Models: paras}
	payload := ParseJsonRPCPayload("aggregate", 0, request)
	info, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	res, err := fm.SendRequest(info)
	if err != nil {
		return nil, err
	}
	decoderes, err := base64.StdEncoding.DecodeString(res.Result)
	if err != nil {
		return nil, err
	}
	return decoderes, nil
}

func (fm *FLManager) ModelTrain(models ...[]byte) ([]byte, error) {
	paras := make([]string, 0)
	for _, model := range models {
		strpara := base64.StdEncoding.EncodeToString(model)
		paras = append(paras, strpara)
	}
	request := &ModelTrainRequest{
		Rank:   fm.Rank,
		Models: paras,
	}

	payload := ParseJsonRPCPayload("modeltrain", 0, request)
	info, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	res, err := fm.SendRequest(info)
	if err != nil {
		return nil, err
	}
	decoderes, err := base64.StdEncoding.DecodeString(res.Result)
	if err != nil {
		return nil, err
	}
	return decoderes, nil
}

func (fm *FLManager) Evaluate(model []byte) (string, error) {
	strpara := base64.StdEncoding.EncodeToString(model)
	request := &EvaluateRequest{
		Rank:  fm.Rank,
		Model: strpara,
	}
	payload := ParseJsonRPCPayload("evaluate", 0, request)
	info, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	res, err := fm.SendRequest(info)
	if err != nil {
		return "", err
	}
	return res.Result, nil
}
