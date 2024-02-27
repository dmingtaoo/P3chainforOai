package api

import "p3Chain/dper/transactionCheck"

type DigtalStamp struct {
	DperService DperService
}

func NewDigtalStamp(dperService *DperService) *DigtalStamp {
	return &DigtalStamp{
		DperService: *dperService,
	}
}

func (ds *DigtalStamp) BackCredit(commandStr string) ([]string, error) {

	arrs := ParseString(commandStr)
	return ds.DperService.dperCommand.SoftCall(arrs)
}

func (ds *DigtalStamp) BackStampList(commandStr string) ([]string, error) {

	arrs := ParseString(commandStr)
	return ds.DperService.dperCommand.SoftCall(arrs)
}

func (ds *DigtalStamp) MintNewStamp(commandStr string) (transactionCheck.CheckResult, error) {

	arrs := ParseString(commandStr)
	return ds.DperService.dperCommand.SoftInvoke(arrs)
}

func (ds *DigtalStamp) TransStamp(commandStr string) (transactionCheck.CheckResult, error) {
	arrs := ParseString(commandStr)
	return ds.DperService.dperCommand.SoftInvoke(arrs)
}
