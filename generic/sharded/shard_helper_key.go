package sharded

import "sort"

// keyShardHelper is used to group keys by the shards they target. This is used
// in MGet.
type keyShardHelper[K comparable] struct {
	keys         []K
	shardIndices []int
}

// newKeyShardHelper is used to group keys for MGet
func newKeyShardHelper[K comparable](keys []K, fIx func(K) int) *keyShardHelper[K] {
	keyCopy := make([]K, len(keys))
	shardIndices := make([]int, len(keys))
	for i := 0; i < len(keys); i++ {
		keyCopy[i] = keys[i]
		shardIndices[i] = fIx(keys[i])
	}
	retval := &keyShardHelper[K]{keyCopy, shardIndices}
	sort.Sort(retval)
	return retval
}

// Len gets the length. This is part of sort.Interface
func (s *keyShardHelper[K]) Len() int { return len(s.keys) }

// Less compares the target shards of two keys by index. This is part of
// sort.Interface
func (s *keyShardHelper[K]) Less(i, j int) bool {
	return s.shardIndices[i] < s.shardIndices[j]
}

// Swap is used to switch the positions of two keys by index. This is part of
// sort.Interface
func (s *keyShardHelper[K]) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
	s.shardIndices[i], s.shardIndices[j] = s.shardIndices[j], s.shardIndices[i]
}

// TakeGroup returns the first set of keys that have the same target shard, and
// the index of that target shard. If there are no more groups left, the target
// shard index will be negative.
func (s *keyShardHelper[K]) TakeGroup() (int, []K) {
	if len(s.keys) == 0 {
		return -1, nil
	}
	shardix := s.shardIndices[0]
	for e := 1; e < len(s.keys); e++ {
		if s.shardIndices[e] != shardix {
			retval := s.keys[0:e]
			s.keys = s.keys[e:]
			s.shardIndices = s.shardIndices[e:]
			return shardix, retval
		}
	}
	retval := s.keys
	s.keys = nil
	s.shardIndices = nil
	return shardix, retval
}
