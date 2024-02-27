package coefficient

import "time"

const DEFAULT_COE_MODE = false

// for block
var (
	Block_MaxTransactionNum = 2048
)

// for consensus promoter
var (
	Promoter_UpperChannelUpdateScanCycle = 500 * time.Millisecond
	Promoter_LowerChannelUpdateScanCycle = 500 * time.Millisecond
	Promoter_MaxPBFTRoundLaunchInterval  = 1000 * time.Millisecond
	Promoter_NewPBFTRoundScanCycle       = 200 * time.Millisecond
	Promoter_MaxParallelRoundNum         = 20
	Promoter_IntraWSUpdateScanCycle      = 5000 * time.Millisecond
)

// for block synchronization
var (
	Synchronization_BlockSynchronizationCheckCycle = 500 * time.Millisecond
)

// for block ordering
var (
	BlockOrder_BlockUploadScanCycle = 200 * time.Millisecond

	BlockOrder_BlockExpireDuration = 30 * time.Second

	BlockOrder_AgentBlockHoldTime = 60 * time.Second

	BlockOrder_SignThresholdRate = 0.6
	BlockOrder_DoneThresholdRate = 0.6
	BlockOrder_InitThresholdRate = 1

	BlockOrder_SignMsgMaxWait = 3 * time.Second
	BlockOrder_DoneMsgMaxWait = 2 * time.Second

	BlockOrder_ServerPromoteCycle   = 300 * time.Millisecond
	BlockOrder_MessageProcessCycle  = 300 * time.Millisecond
	BlockOrder_BlockExpireScanCycle = 3 * time.Second
)

// for upperOrderService
// NOTE: Not used anymore
var (
	// for service provider
	ServiceProvider_ServeCheckCycle   = 300 * time.Millisecond
	ServiceProvider_MessageCheckCycle = 300 * time.Millisecond
	ServiceProvider_ExpireCheckCycle  = 10000 * time.Millisecond
	ServiceProvider_MaxHoldThreshold  = 60 * time.Second

	ServiceProvider_BlackListCheckCycle = 30 * time.Second
	ServiceProvider_MinRequireInterval  = 3 * time.Second

	ServiceProvider_ReadyThresholdRate = 0.6
	ServiceProvider_VoteThresholdRate  = 0.6
	ServiceProvider_InitThresholdRate  = 1

	ServiceProvider_MaxFinalWaitTime = 2000 * time.Millisecond

	// for service agent
	ServiceAgent_LocalBlockScanCycle = 300 * time.Millisecond
	ServiceAgent_PrepareScanCycle    = 300 * time.Millisecond
	ServiceAgent_CheckExpireCycle    = 1 * time.Second

	ServiceAgent_MaxPrepareHoldTime = 60 * time.Second

	//about stability
	ServiceAgent_NextMessageRetryTime     = 3
	ServiceAgent_NextMessageRetryInterval = 500 * time.Millisecond

	ServiceAgent_RecoveryTriggerTime = 3 * time.Second
)

// for separator
var (
	Separator_MaxTransactionSeriesNum = 256
)

func init() {
	if DEFAULT_COE_MODE {
		return
	}

	// Promoter_UpperChannelUpdateScanCycle = 100 * time.Millisecond
	// Promoter_LowerChannelUpdateScanCycle = 100 * time.Millisecond

	// Promoter_NewPBFTRoundLaunchCycle = 100 * time.Millisecond

	Promoter_LowerChannelUpdateScanCycle = 100 * time.Millisecond
	Promoter_UpperChannelUpdateScanCycle = 100 * time.Millisecond
	Promoter_MaxPBFTRoundLaunchInterval = 500 * time.Millisecond
	Promoter_NewPBFTRoundScanCycle = 100 * time.Millisecond

	Synchronization_BlockSynchronizationCheckCycle = 300 * time.Millisecond

	BlockOrder_ServerPromoteCycle = 100 * time.Millisecond
	BlockOrder_MessageProcessCycle = 100 * time.Millisecond

	// ServiceProvider_ServeCheckCycle = 100 * time.Millisecond
	// ServiceProvider_MessageCheckCycle = 100 * time.Millisecond

	// ServiceAgent_LocalBlockScanCycle = 100 * time.Millisecond
	// ServiceAgent_PrepareScanCycle = 100 * time.Millisecond

	// Promoter_LowerChannelUpdateScanCycle = 100 * time.Millisecond
	// Promoter_UpperChannelUpdateScanCycle = 100 * time.Millisecond
	// Promoter_NewPBFTRoundLaunchCycle = 100 * time.Millisecond

	// Synchronization_BlockSynchronizationCheckCycle = 100 * time.Millisecond

	// ServiceProvider_ServeCheckCycle = 100 * time.Millisecond
	// ServiceProvider_MessageCheckCycle = 100 * time.Millisecond

	// ServiceAgent_LocalBlockScanCycle = 100 * time.Millisecond
	// ServiceAgent_PrepareScanCycle = 100 * time.Millisecond

}
