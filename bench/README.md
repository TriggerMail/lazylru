# LazyLRU Benchmarking

Because this implementation is designed for groups of keys that come in waves, a simple [testing benchmark ](https://golang.org/pkg/testing/#hdr-Benchmarks) that reads and writes random keys would not be an accurate representation of this library. For those kinds of general loads, [hashicorp/golang-lru](https://github.com/hashicorp/golang-lru) is just as good as LazyLRU when the cache is >25% full. So these benchmarks try to fill that gap.

Benchmarking independently is interesting, but not as instructive. The candidates for all the tests were:

* **null**: Do nothing. Don't save anything. All `get` operations are misses.
* **mapcache.{hour|50ms}**: A map of `key => {value, expiration}`. If the map is full, items are dropped at random. The time indicates the expiration -- _50ms_ for expiring frequently relative to read/write operations, _hour_ for exprining infrequently relative to read/write operations.
* **lazylru.{hour|50ms}**: The thing in this repo. The one we're here to test.
* **hashicorp.lru**: This is the default implementation in the [hashicorp/golang-lru](https://pkg.go.dev/github.com/hashicorp/golang-lru?utm_source=godoc) package. This is the implementation based on [groupcache](https://github.com/golang/groupcache/blob/master/lru/lru.go). _This implementation does not support expiration._
* **hashicorp.exp_{hour|50ms}**: This is the hashicorp.lru, but instead of storing raw values, we store `key => {value, expiration} ` like we did in the mapcache above. Expiry is checked on read and stale values are discarded.
* **hashicorp.arc**: hashicorp's implementation of the [Adaptive Relay Cache](https://www.usenix.org/legacy/event/fast03/tech/full_papers/megiddo/megiddo.pdf). _This implementation does not support expiration._
* **hashicorp.2Q**: hashicorp's implementation of the [multi-queue replacement algorithm](https://static.usenix.org/event/usenix01/full_papers/zhou/zhou.pdf). 

These tests define sets of keys, then rotate through those sets. This is meant to simulate the waves of requests for a set of keys that would come as marketing sends run through a day. Tests have the following parameters: 

* **algorithm**: What we are testing
* **ranges**: How many ranges of keys are in the test
* **keys/range**: How big each range is
* **cycles/range**: How many times each range is read
* **threads**: The number of concurrent reader/writer workers
* **size**: Capacity of the cache under test
* **work_time_µs**: On each operation, spin-wait to alleviate lock contention while not releasing the CPU
* **sleep_time_µs**: On each operation, sleep to allevaite lock contention while yielding
* **cycles**: How may read or write operations are in the test
* **duration_ms**: How long the test took
* **rate_kHz**: Cycles/duration
* **hit_rate_%**: How efficient the cache was

I ran 253 variations of these tests on a Google Cloud [n1-standard-8](https://cloud.google.com/compute/docs/machine-types#n1_machine_types) (8-core) VM running Go 1.16.4. I've included the [raw results](results.csv.gz) in this repo.

* **Test A**: 5 ranges of 1000 keys, 1000000 cycles/range, size 10000, 1 thread, 0 work, 0 sleep.
* **Test B**: 5 ranges of 1000 keys, 1000000 cycles/range, size 10000, 64 thread, 0 work, 0 sleep
* **Test C**: 1 ranges of 20000 keys, 1000000 cycles/range, size 10000, 64 thread, 0 work, 0 sleep

| Algorithm          | rate (kHz) | hit rate (%) | rate (kHz) | hit rate (%) | rate (kHz) | hit rate (%) |
| ------------------ | ---------: | -----------: | ---------: | -----------: | ---------: | -----------: |
|                    | **Test A** |   **Test A** | **Test B** |   **Test B** | **Test C** |   **Test C** |
| null               |   18532.44 |         0.00 |    3755.14 |         0.00 |    3131.50 |         0.00 |
| mapcache.hour      |    3177.00 |        99.90 |    1854.16 |        99.90 |     881.83 |        10.00 |
| mapcache.50ms      |    3108.52 |        96.83 |    1654.61 |        94.20 |     865.27 |         8.20 |
| lazylru.hour       |    4811.95 |        99.90 |    2719.89 |        99.90 |     462.65 |         9.96 |
| lazylru.50ms       |    3977.11 |        97.50 |    1466.69 |        93.29 |     458.97 |         9.94 |
| hashicorp.lru      |    3796.49 |        99.90 |    1457.54 |        99.90 |     696.47 |        10.00 |
| hashicorp.exp_hour |    2627.22 |        99.90 |    1343.52 |        99.90 |     591.02 |         9.97 |
| hashicorp.exp_50ms |    2616.85 |        99.90 |    1342.45 |        99.90 |     587.46 |        10.00 |
| hashicorp.arc      |    3496.26 |        99.90 |    1455.67 |        99.90 |     338.93 |         9.95 |
| hashicorp.2Q       |    3804.71 |        99.90 |    1476.98 |        99.90 |     357.44 |         9.97 |

The reason that HashiCorp's implementations were used as the reference is that they are well done. For general purpose caching needs, they are hard to beat. However, the Tests A and B are a reasonable facsimilie of what we see in real life. And in that environment, it performs very well. In Test C, where the cache is undersized, LazyLRU worse than the regular LRU algorithm. The two "smart" algorithms, ARC and 2Q, shouldn't be expected to perform well in this test because of the random request pattern.