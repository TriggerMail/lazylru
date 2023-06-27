package sharded_test

import (
	"testing"

	lazylru "github.com/TriggerMail/lazylru"
	"github.com/stretchr/testify/require"
)

type ExpectedStats struct {
	KeysWritten      *uint32
	KeysReadOK       *uint32
	KeysReadNotFound *uint32
	KeysReadExpired  *uint32
	Shuffles         *uint32
	Evictions        *uint32
	KeysReaped       *uint32
	ReaperCycles     *uint32
}

func (es ExpectedStats) WithKeysWritten(v uint32) ExpectedStats {
	es.KeysWritten = &v
	return es
}

func (es ExpectedStats) WithKeysReadOK(v uint32) ExpectedStats {
	es.KeysReadOK = &v
	return es
}

func (es ExpectedStats) WithKeysReadNotFound(v uint32) ExpectedStats {
	es.KeysReadNotFound = &v
	return es
}

func (es ExpectedStats) WithKeysReadExpired(v uint32) ExpectedStats {
	es.KeysReadExpired = &v
	return es
}

func (es ExpectedStats) WithShuffles(v uint32) ExpectedStats {
	es.Shuffles = &v
	return es
}

func (es ExpectedStats) WithEvictions(v uint32) ExpectedStats {
	es.Evictions = &v
	return es
}

func (es ExpectedStats) WithKeysReaped(v uint32) ExpectedStats {
	es.KeysReaped = &v
	return es
}

func (es ExpectedStats) WithReaperCycles(v uint32) ExpectedStats {
	es.ReaperCycles = &v
	return es
}

func (es ExpectedStats) Test(t *testing.T, stats lazylru.Stats) {
	if es.KeysWritten != nil {
		require.Equal(t, int(*es.KeysWritten), int(stats.KeysWritten), "keys written")
	}

	if es.KeysReadOK != nil {
		require.Equal(t, int(*es.KeysReadOK), int(stats.KeysReadOK), "keys read ok")
	}

	if es.KeysReadNotFound != nil {
		require.Equal(t, int(*es.KeysReadNotFound), int(stats.KeysReadNotFound), "keys read not found")
	}

	if es.KeysReadExpired != nil {
		require.Equal(t, int(*es.KeysReadExpired), int(stats.KeysReadExpired), "keys read expired")
	}

	if es.Shuffles != nil {
		require.Equal(t, int(*es.Shuffles), int(stats.Shuffles), "shuffles")
	}

	if es.Evictions != nil {
		require.Equal(t, int(*es.Evictions), int(stats.Evictions), "evictions")
	}

	if es.KeysReaped != nil {
		require.Equal(t, int(*es.KeysReaped), int(stats.KeysReaped), "keys reaped")
	}

	if es.ReaperCycles != nil {
		require.Equal(t, int(*es.ReaperCycles), int(stats.ReaperCycles), "reaper cycles")
	}
}
