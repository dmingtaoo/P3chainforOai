package separator

// induce whether a transaction
func (m *Message) IsTransaction() bool {
	return m.MsgCode == transactionCode
}

func (m *Message) IsTransactions() bool {
	return m.MsgCode == transactionsCode
}

// induce whether a lower-level consensus message
func (m *Message) IsLowerConsensus() bool {
	return m.MsgCode == lowerConsensusCode
}

// induce whether a lower-level consensus message
func (m *Message) IsUpperConsensus() bool {
	return m.MsgCode == upperConsensusCode
}

// induce whether a control message
func (m *Message) IsControl() bool {
	return m.MsgCode == controlCode
}

func (m *Message) IsIntraWorldStateUpdate() bool {
	return m.MsgCode == intraWSUpdateCode
}
