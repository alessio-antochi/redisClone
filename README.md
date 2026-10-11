# redisClone

A small Redis-compatible server written in Go, built from raw TCP sockets. It uses the real RESP protocol, so the standard `redis-cli` and `redis-benchmark` tools work against it unmodified.

No Redis libraries and no RESP parsing libraries: the protocol parser, command handling, expiry, and pub/sub are all written from scratch.

## Supported commands

| Command | Notes |
| --- | --- |
| `PING [msg]`, `ECHO msg` | |
| `SET key value`, `GET key` | No `SET` flags were implemented |
| `DEL key [key ...]`, `EXISTS key [key ...]` | |
| `EXPIRE key seconds`, `TTL key` | `TTL` returns -1 (no expiry) or -2 (missing key), like Redis |
| `SUBSCRIBE`, `UNSUBSCRIBE`, `PUBLISH` | `PUBLISH` payloads may contain spaces, quote them when using `redis-cli` |

## Run it

Requires the Go version listed in `go.mod`.

```bash
go run .            # listens on :8080
redis-cli -p 8080 PING
```

Pub/sub across two terminals:

```bash
redis-cli -p 8080 SUBSCRIBE news                 # terminal 1: prints messages as they arrive
redis-cli -p 8080 PUBLISH news "hello world"     # terminal 2: replies with the receiver count
```

Run the tests (the `-race` flag enables Go's data race detector):

```bash
go test -race ./...
```

## Architecture

```
redis-cli ──TCP──> listener ──> one goroutine per connection
                                        │
                  read bytes ──> RESP parser ──> run command
                                        │            │
                                        │            ├──> Store (map + mutex, lazy TTL)
                                        │            └──> Hub (channel -> subscribers)
                                        ▼                           │
                      Client: outgoing buffer (64 messages)  <──────┘  (pushed messages)
                        + one writer goroutine ──> socket
```

- **Protocol:** hand-written RESP parser and encoder (`parsing.go`).
- **Store:** `map[string]Entry` behind a single mutex (`storing.go`). Each entry holds a value and an expiry date (the zero time means no expiry).
- **Pub/sub:** a `Hub` maps channels to subscribed clients (`pubsub.go`). Each client has a buffer of up to 64 outgoing messages and one writer goroutine.

## Design decisions

1. **Absolute expiry timestamps with lazy expiry.** Keys are checked and deleted when accessed. There is no background sweep, so keys that are set and never read again stay in memory (see limitations).
2. **`Mutex` instead of `RWMutex`.** Lazy expiry means any read can turn into a delete, so every read access might also need write access. `RWMutex` would add complexity with no parallelism gain.
3. **One writer goroutine per client.** Only that goroutine writes to the socket, so command replies and pushed pub/sub messages can't overlap. `Publish` never writes to a socket directly.
4. **Slow subscribers are disconnected, not waited on.** If a client's buffer is full because it isn't reading fast enough, `Publish` closes that client instead of blocking every other publisher.

## Bugs found and fixed

- **Lost update on lazy expiry.** The expiry check read an entry, released the lock, then deleted based on the stale copy. A concurrent `SET` could be silently erased. Fixed by making check-and-delete one atomic operation under a single lock. Regression test: `TestNoLostUpdateOnExpiryRace`.
- **`EXPIRE` overwrote concurrent writes.** It read the entry, released the lock, then wrote the old value back with a new deadline. Fixed by doing the read, the expiry check and the write in one lock.
- **Double close on client disconnect.** Found by `TestPubSubConcurrent`: a disconnect, a failed write and a publisher could all close the same client at once, panicking with "close of closed channel". Fixed by making `Close()` safe for multiple calls with `sync.Once`.
- **Inverted expiry check.** After replacing the "no expiry" default time with a zero time, an inverted condition deleted every non-expiring key on first read, so `GET` right after `SET` returned nil. Found by hand with `redis-cli`; fixed, and guarded by `TestSetThenGet`.

## Testing

- Unit tests for the RESP parser (valid and malformed input).
- Store concurrency tests, run with `-race`.
- Pub/sub: a concurrent stress test (clients connecting, disconnecting and publishing at the same time, checking that nothing crashes, hangs or stays subscribed afterwards).
- Manual end-to-end checks against the real `redis-cli`.
- The concurrency tests are probabilistic: they make regressions likely to be caught, not impossible.

## Benchmarks

Machine: CPU: *i7-13650HX*, OS: *Arch Linux x86_64*.
Command: `redis-benchmark -p 8080 -t SET,GET -n 100000 -c 50`, results as a range over 5 runs.

| | Throughput | p50 | p99 |
| --- | --- | --- | --- |
| SET | 230k-250k rps | 0.103 ms | 0.207-0.343 ms |
| GET | 242k-253k rps | 0.103 ms | 0.207-0.223 ms |

Figures are the range over 5 runs. `redis-benchmark` measures speed, not correctness; correctness under concurrent access is covered by the tests above.

## Limitations

- **One `Read` is treated as one command.** There is no stream-based framing, so a command split across TCP packets, or several pipelined commands in one packet, will not parse correctly.
- Commands larger than the 1024-byte read buffer are not supported.
- No active expiry sweep, so expired keys that are never read again stay in memory.
- A single global lock caps scalability. A double-checked `RWMutex` pattern or sharding could restore read parallelism.
- No persistence: all data is lost on restart.
- Subscribed-mode restrictions are not enforced: a subscribed connection still accepts any command.

## What I'd do next

- Parse from a buffered stream (`bufio.Reader`) instead of one `Read` per command, which also enables pipelining.
- Add an active expiry sweep that samples a random subset of keys per pass, as real Redis does.
- Add snapshot persistence: periodically write the store to disk and reload it on startup.
