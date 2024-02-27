package blockOrdering

type SyncProto interface {
	UseNormalMode()
	UseLeaderMode()
}
