# Sharded LazyLRU

The `LazyLRU` works hard to reduce blocking by reducing reorder operations, but there are limits. Another common mechanism to reduce blocking is to shard the cache. Two keys that target separate shards will not conflict, at least not from a memory-safety perspective. If lock contention is a problem you have in your application, sharding may be for you! There are volumes of literature on this subject and this README is here to tell you how you _can_ shard your cache, not whether or not you should.

The implementation here is a wrapper over the `lazylru.LazyLRU` cache. There are no attempts to spread out reaper threads or assist with memory locality. As a result, high shard counts will hurt performance. Considering that lock contention should reduce directly by the shard count, there are no advantages to high shard counts.

## Dependencies

`lazylru.LazyLRU` has no external dependencies beyond the standard library. However, `sharded.HashingSharder` relies on [github.com/zeebo/xxh3](https://github.com/zeebo/xxh3).

## Usage

```go
import (
    "time"

    lazylru "github.com/TriggerMail/lazylru/generic"
    sharded "github.com/TriggerMail/lazylru/generic/sharded"
)

func main() {
    regularCache := lazylru.NewT[string, string](10, time.Minute)
    shardedCache := sharded.NewT[string, string](10, time.Minute, 10, sharded.StringSharder)
}
```

In the example above, we are creating a flat cache and a sharded cache. The flat cache will hold 10 items. The sharded cache will hold up to 10 items in each of 10 shards, so up to 100 items. There is no mechanism to limit the total size of the sharded cache other than limiting the size of each shard. `sharded.LazyLRU` exposes the same interface as `lazylru.LazyLRU`, so it should be a drop-in replacement.

## Sharding

The sharding function should return integers, uniformly-distributed over the key space. This does _not_ need to be limited to values less than the shard count -- the library takes care of that.

The sharded cache in the example above is using a helper function, `sharded.StringSharder`, rather than implementing a custom sharder function. There is also a `shared.BytesSharder` for byte-slice keys.

### Custom types using the `HashingSharder`

Caching with custom key types is also possible. Without access to some of the compiler guts, we can't do [what Go does for hashmaps](https://dave.cheney.net/2018/05/29/how-the-go-runtime-implements-maps-efficiently-without-generics). There is some compiler-time rewriting to use magic in the `runtime` package as well as hash functions implemented outside of Go that aren't available to us. We could also try to create a universal hasher, such as [mitchellh/hashstructure](https://github.com/mitchellh/hashstructure), but the reliance on reflection makes that a non-starter from a performance perspective. The same for an encoding-based solution like `encoding/gob`, which also relies on reflection. It might be possible to inspect the type of `K` at start time and use reflection to generate a hash function, but that's beyond the scope of what I'm trying to do here.

Instead, we can use generics to allow the caller to define a hash function. There is another helper to assist in the creation of those sharder functions. These go through a hash function to get the necessary `uint64` value. The `HashingSharder` is a zero-allocation implementation based on `XXH3`, so as long as you don't make any allocations yourself, the resulting sharder should be pretty cheap.

```go
import (
    "time"

    lazylru "github.com/TriggerMail/lazylru/generic"
    sharded "github.com/TriggerMail/lazylru/generic/sharded"
)

type CustomKeyType struct {
    Field0 string
    Field1 []byte
    Field2 int
}

func main() {
    shardedCache := sharded.NewT[CustomKeyType, string](
        10,
        time.Minute,
        10,
        sharded.HashingSharder(func(k CustomKeyType, h sharded.H) {
            h.WriteString(k.Field0)
            h.Write(k.Field1)
            h.WriteUint64(uint64(k.Field2))
        },
    )
}
```

On my laptop, the `HashingSharder` and its children the `StringSharder` and the `BytesSharder` proved to be fast and cheap with various key sizes. The `GobSharder`, which used the same pool of `XXH3` hashers, was 28x slower and allocated significant memory on each cycle. I have deleted the `GobSharder` for this reason.

```text
$ go test -bench . -benchmem
goos: darwin
goarch: arm64
pkg: github.com/TriggerMail/lazylru/generic/sharded
BenchmarkStringSharder/1-8     56328306       21.19 ns/op      0 B/op    0 allocs/op
BenchmarkStringSharder/4-8     57913196       20.57 ns/op      0 B/op    0 allocs/op
BenchmarkStringSharder/16-8    59431563       19.92 ns/op      0 B/op    0 allocs/op
BenchmarkStringSharder/64-8    54227484       21.80 ns/op      0 B/op    0 allocs/op
BenchmarkStringSharder/256-8   26014356       45.72 ns/op      0 B/op    0 allocs/op
BenchmarkBytesSharder/1-8      55908974       21.32 ns/op      0 B/op    0 allocs/op
BenchmarkBytesSharder/4-8      57933231       20.60 ns/op      0 B/op    0 allocs/op
BenchmarkBytesSharder/16-8     59253040       19.96 ns/op      0 B/op    0 allocs/op
BenchmarkBytesSharder/64-8     54259972       21.80 ns/op      0 B/op    0 allocs/op
BenchmarkBytesSharder/256-8    26069034       45.88 ns/op      0 B/op    0 allocs/op
BenchmarkCustomSharder-8       23089836       50.88 ns/op      0 B/op    0 allocs/op
BenchmarkGobSharder-8            794709     1412 ns/op      1128 B/op   24 allocs/op
```

### Custom sharder

If the `HashingSharder` doesn't do what you need, any function that returns a `uint64` value will do. Integer-like keys are a great example of when a custom sharder may be appropriate. Just be aware something like "cast to `uint64`" means that any bias in common factors between your keyspace and the number of shards in the cache can result in uneven loading of the shards.

## Conclusions about implementing sharding with Go generics

While there is some real-world value to sharding a cache as we've done here, it's not something I need in any current project. The sharding here was intended to be an experiment on the use of generics in Go for something where the standard library couldn't help me.

Just as the generic `lazylru.LazyLRU` implementation relies on a generic version of `containers/heap` that was copied from the standard library, sharding required a generic hasher to distribute keys among the shards. Total victory would have been to create a generic sharder that doesn't require any user-generated code to handle different key types, but has performance reasonably close to a custom, hand-tuned hasher for the given key type. I don't think we acheived that.

Go's implementation for `map` is really the gold standard here. While `map` doesn't use generics, it does some compile-time tricks that are basically unavailable to library authors. Generics cover some of that ground, but there's a big gap left. It's conceivable that we could bridge that gap, but it would take a lot of code.

Instead, I think we got about halfway there. The `HashingSharder` creates a mechanism for arbitrary key types to be used to shard, albeit with some end-user intervention. The value of generics there is that no casting or boxing is required to use the `HashingSharder`, and I'd argue it is a lot easier to use correctly than an analogous function written without generics. The performance is also excellent due to the power of `sync.Pool` to avoid allocations and the fantastic speed of [github.com/zeebo/xxh3](https://github.com/zeebo/xxh3).

The other lesson here is that generics can spread across your API. In other languages with generics, this doesn't feel weird, but it does in Go. As an example, when .NET 2.0 brought generics to C#, it quickly became common to see generics everywhere, largely because the standard library in .NET 2.0 made broad use of generics. In Go, the compatibility promise means that the standard library is not going to be retrofit to support generics, nor should it. At least for now, it seems like generic code in Go is going to feel a little out-of-place, especially when the types cannot be inferred.

Generics are not as powerful as C++ templates or generics in some other languages. Go's lack of polymorphism is enough to ensure that it probably never will be. The hasher here highlights that. Instead, it seems like Go generics are great when you put in things of type `T` and get things out of type `T`. Putting in `T` and getting out anything else, even a fixed type like `uint64` as we do here, is going to be a little messy unless there is an interface constraint you can rely on.

## Performance

The reason to shard is to reduce lock contention. However, that assumes that lock contention is the problem. The purpose of LazyLRU was to reduce exclusive locks, and thus lock contention. Benchmarks were run on my [laptop (8-core MacBook Pro M1 14" 2021)](benchmark_results_macbook_pro_m1_8.txt) and on a [Google Cloud N2 server (N2 "Ice Lake" 64 cores at 2.60GHz)](benchmark_results_n2-highcpu-64_64.txt). If we compare the unsharded vs. 64-way sharded performance with 1 thread and 64 threads, we should get some sense of the trade-offs. We will also compare oversized (guaranteed evictions), undersized (no evictions, no shuffles), and equal-sized (no evictions, shuffles) caches to see how the "lazyness" may save us some writes. Because the sharded caches use the regular LazyLRU behind the scenes, the capacity of each shard is `total/shard_count` so we aren't unfairly advantaging the sharded versions.

All times in nanoseconds/operation, so lower is better. Mac testing was not done on a clean environment, so some variability is to be expected. The first set of numbers are from the Mac. The numbers after the gutter are from the server -- tables in markdown aren't great.

### 100% writes

| capacity | Shards | threads | unsharded | 16 shards | 64 shards | | unsharded | 16 shards | 64 shards |
| -------- | -----: | ------: | --------: | --------: | --------: |-| --------: | --------: | --------: |
| over     |      1 |       1 |     103.0 |     126.6 |     108.9 | |     159.2 |     204.9 |     191.9 |
| over     |      1 |     256 |     381.8 |     143.1 |     93.01 | |     228.8 |     89.90 |     35.65 |
| over     |      1 |   65536 |     573.6 |     169.8 |     112.7 | |     223.5 |     121.5 |     40.94 |
| under    |      1 |       1 |     318.4 |     294.0 |     214.9 | |     477.2 |     451.4 |     345.8 |
| under    |      1 |     256 |     470.2 |     228.6 |     139.0 | |     581.4 |     154.8 |     49.18 |
| under    |      1 |   65536 |     620.5 |     316.6 |     146.0 | |     606.8 |     157.4 |     61.02 |
| equal    |      1 |       1 |     129.4 |     159.9 |     166.1 | |     210.4 |     260.2 |     269.5 |
| equal    |      1 |     256 |     322.2 |     140.4 |     99.66 | |     367.0 |     81.49 |     26.67 |
| equal    |      1 |   65536 |     553.8 |     229.6 |     122.9 | |     283.8 |     107.2 |     45.99 |

### 1% writes, 99% reads

| capacity | Shards | threads | unsharded | 16 shards | 64 shards | | unsharded | 16 shards | 64 shards |
| -------- | -----: | ------: | --------: | --------: | --------: |-| --------: | --------: | --------: |
| over     |      1 |       1 |     72.66 |     104.9 |     89.66 | |     109.6 |     146.3 |     143.1 |
| over     |      1 |     256 |     282.0 |     73.93 |     43.52 | |     185.8 |     39.78 |     17.44 |
| over     |      1 |   65536 |     275.8 |     114.2 |     67.85 | |     134.4 |     43.88 |     20.08 |
| under    |      1 |       1 |     63.50 |     84.80 |     65.50 | |     85.43 |     117.3 |     102.9 |
| under    |      1 |     256 |     179.2 |     65.58 |     35.47 | |     176.0 |     30.77 |     10.58 |
| under    |      1 |   65536 |     246.8 |     110.5 |     62.39 | |     144.3 |     37.23 |     25.61 |
| equal    |      1 |       1 |     107.1 |     134.2 |     141.3 | |     167.0 |     203.0 |     206.7 |
| equal    |      1 |     256 |     231.1 |     135.5 |     74.56 | |     422.5 |     73.29 |     30.92 |
| equal    |      1 |   65536 |     302.9 |     228.0 |     138.0 | |     219.0 |     82.13 |     36.20 |

This shows that sharding is somewhat effective when there are lots of exclusive locks due to writes and an abundance of threads that might block on those locks. It even looks like we could do even better with more shards. As expected, the 64-core server sees a bigger win from sharding than the 8-core Mac, though the Mac is 50% faster at running the hashing step.

An additional learning from this testing was that the default source from [math/rand](https://pkg.go.dev/math/rand) is made thread-safe by locking on every call, which is its own source of contention. This overwhelmed the actual cache performance on the 64-core server, but was visible even on the 8-core Mac. Creating a new source for each worker thread eliminated that contention. As always, `go tool pprof` was awesome.
