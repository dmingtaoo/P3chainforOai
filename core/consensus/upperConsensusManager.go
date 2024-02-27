package consensus

import (
	"p3Chain/core/separator"
)

type UpperConsensusManager interface {
	Install(cp *ConsensusPromoter)
	ProcessMessage(message *separator.Message)
	Start() error
	Stop()
}
