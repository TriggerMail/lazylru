package sharded

import "sort"

// kvShardHelper is used to group keys and associated values by the shards they
// target. This is used in MSet and MSetTTL.
type kvShardHelper[K comparable, V any] struct {
	keys         []K
	values       []V
	shardIndices []int
}

// newKVShardHelper is used to group keys and values for MSet and MSetTTL
func newKVShardHelper[K comparable, V any](keys []K, values []V, fIx func(K) int) *kvShardHelper[K, V] {
	keyCopy := make([]K, len(keys))
	valCopy := make([]V, len(values))
	shardIndices := make([]int, len(keys))
	for i := 0; i < len(keys); i++ {
		keyCopy[i] = keys[i]
		valCopy[i] = values[i]
		shardIndices[i] = fIx(keys[i])
	}
	retval := &kvShardHelper[K, V]{keyCopy, valCopy, shardIndices}
	sort.Sort(retval)
	return retval
}

// Len gets the length. This is part of sort.Interface
func (s *kvShardHelper[K, V]) Len() int { return len(s.keys) }

// Less compares the target shards of two key/value pairs by index. This is part
// of sort.Interface
func (s *kvShardHelper[K, V]) Less(i, j int) bool {
	return s.shardIndices[i] < s.shardIndices[j]
}

// Swap is used to switch the positions of two key/value pairs by index. This
// is part of sort.Interface
func (s *kvShardHelper[K, V]) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
	s.values[i], s.values[j] = s.values[j], s.values[i]
	s.shardIndices[i], s.shardIndices[j] = s.shardIndices[j], s.shardIndices[i]
}

// TakeGroup returns the first set of keys and values that have the same target
// shard, and the index of that target shard. If there are no more groups left,
// the target shard index will be negative.
func (s *kvShardHelper[K, V]) TakeGroup() (int, []K, []V) {
	if len(s.keys) == 0 {
		return -1, nil, nil
	}
	shardix := s.shardIndices[0]
	for e := 1; e < len(s.keys); e++ {
		if s.shardIndices[e] != shardix {
			retkeys := s.keys[0:e]
			retvals := s.values[0:e]
			s.keys = s.keys[e:]
			s.values = s.values[e:]
			s.shardIndices = s.shardIndices[e:]
			return shardix, retkeys, retvals
		}
	}
	retkeys := s.keys
	retvals := s.values
	s.keys = nil
	s.values = nil
	s.shardIndices = nil
	return shardix, retkeys, retvals
}
