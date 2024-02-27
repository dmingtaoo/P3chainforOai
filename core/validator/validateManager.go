package validator

import (
	"p3Chain/core/eles"
	loglogrus "p3Chain/log_logrus"
	"sync"
)

type ValidateManager struct {
	updateMux        sync.RWMutex
	currentValidator *CommonValidator

	filterMux sync.RWMutex
	txFilter  *eles.TransactionFilter
}

func (vm *ValidateManager) Update(vlt *CommonValidator) {
	vm.updateMux.Lock()
	defer vm.updateMux.Unlock()
	vm.currentValidator = vlt
}

func (vm *ValidateManager) InjectTransactionFilter(txFilter *eles.TransactionFilter) {
	vm.filterMux.Lock()
	defer vm.filterMux.Unlock()

	vm.txFilter = txFilter
}

func (vm *ValidateManager) IsRepeatTransaction(tx eles.Transaction) bool {
	if vm.txFilter == nil {
		return false
	}
	vm.filterMux.RLock()
	defer vm.filterMux.RUnlock()
	return !vm.txFilter.IsTxUnique(tx)
}

func (vm *ValidateManager) HasRepeatTransaction(block *eles.Block) bool {
	if vm.txFilter == nil {
		loglogrus.Log.Debugf("txFilter is none............")
		return false
	}
	vm.filterMux.RLock()
	defer vm.filterMux.RUnlock()

	loglogrus.Log.Debugf("Transaction Number of block:%x is :%d", block.BlockID, len(block.Transactions))
	return vm.txFilter.HasRepeatTxs(block.Transactions)
}

func (vm *ValidateManager) LowerValidate(block *eles.Block) bool {
	vm.updateMux.RLock()
	defer vm.updateMux.RUnlock()

	return vm.currentValidator.LowerValidate(block)
}

func (vm *ValidateManager) UpperValidate(block *eles.Block) bool {
	vm.updateMux.RLock()
	defer vm.updateMux.RUnlock()
	return vm.currentValidator.UpperValidate(block)
}

func (vm *ValidateManager) SetDefaultValidThreshold() {
	vm.updateMux.Lock()
	defer vm.updateMux.Unlock()
	vm.currentValidator.SetDefaultThreshold()
}

func (vm *ValidateManager) SetLowerValidRate(rate float64) {
	vm.updateMux.Lock()
	defer vm.updateMux.Unlock()
	vm.currentValidator.SetLowerValidThreshold_UseRatio(rate)
}

func (vm *ValidateManager) SetUpperValidRate(rate float64) {
	vm.updateMux.Lock()
	defer vm.updateMux.Unlock()
	vm.currentValidator.SetUpperValidThreshold_UseRatio(rate)
}

func (vm *ValidateManager) SetLowerValidFactor(factor int) {
	vm.updateMux.Lock()
	defer vm.updateMux.Unlock()
	vm.currentValidator.SetLowerValidThreshold_UsePBFTFactor(factor)
}

func (vm *ValidateManager) SetUpperValidFactor(factor int) {
	vm.updateMux.Lock()
	defer vm.updateMux.Unlock()
	vm.currentValidator.SetUpperValidThreshold_UsePBFTFactor(factor)
}
