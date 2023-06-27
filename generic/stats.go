package lazylru

// Stats represends counts of actions against the cache.
//
// Deprecated: The "github.com/TriggerMail/lazylru/generic" package has been
// deprecated. Please point all references to "github.com/TriggerMail/lazylru",
// which now includes the generic API.
type Stats struct {
	KeysWritten      uint32
	KeysReadOK       uint32
	KeysReadNotFound uint32
	KeysReadExpired  uint32
	Shuffles         uint32
	Evictions        uint32
	KeysReaped       uint32
	ReaperCycles     uint32
}
