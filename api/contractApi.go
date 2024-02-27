package api

type ContractService struct {
	DperService *DperService

	// 用户的自定义合约
	DigtalStamp *DigtalStamp
}

func NewContractService(dperService *DperService) *ContractService {
	return &ContractService{
		DperService: dperService,
		DigtalStamp: NewDigtalStamp(dperService),
	}
}
