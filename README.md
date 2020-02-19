# Neo Blockchain benchmark

## Requirements

- make
- docker
- docker-compose 1.24+
- golang (for development / tests)

## Repository structure

- .docker - contains docker files
    - build - build folder for docker images
    - ir - contains IR docker files
    - rpc - contains RPC / Benchmark docker files
- .make - contains makefile specific files
- cmd - contains Benchmark source code
    - bench - Benchmark command source code
    - gen - Transaction generator source code 
    - internal - common code, that used in bench tool and generator
    - vendor - golang vendor folder
    - go.mod - golang modules file
    - go.sum - golang modules summary file
- .gitignore
- Makefile 
- README.md

## Usage example (local benchmark + NeoGo single node)

```
$ make gen
=> Fetch deps
=> Generate transactions dump
2020/02/18 18:47:55 Generate 1000000 txs
2020/02/18 18:49:15 Done: 1m19.397232138s

$ make test
=> Stop environment
=> Up Golang single node
Creating network "ir_default" with the default driver
Creating ir_node_1 ... done
=> Fetch deps
=> Test Golang single node
2020/02/18 18:49:51 Used [localhost:20331] rpc addresses
2020/02/18 18:49:52 fetch current block count
2020/02/18 18:49:52 Started test from block = 1604 at unix time = 1582040992
2020/02/18 18:49:52 Read 1000000 txs from ../dump.txs
2020/02/18 18:49:55 CPU: 4.669, Mem: 66.719: <nil>
2020/02/18 18:49:55 Done 3.534899462s
2020/02/18 18:49:55 Init worker with 150 QPS / 1m0s time limit (1000000 txs will try to send)
2020/02/18 18:49:58 CPU: 7.425, Mem: 76.426: <nil>
...
2020/02/18 18:50:56 (#1661/150) 151 Tx's in 1 secs 150.000000 tps
2020/02/18 18:50:56 (#1662/206) 207 Tx's in 1 secs 206.000000 tps
2020/02/18 18:50:56 (#1663/236) 237 Tx's in 1 secs 236.000000 tps
2020/02/18 18:50:56 (#1664/150) 151 Tx's in 1 secs 150.000000 tps
2020/02/18 18:50:56 (#1665/150) 151 Tx's in 1 secs 150.000000 tps
2020/02/18 18:50:56 try to write profile
2020/02/18 18:50:56 Sended 9000 txs for 1m0.49485746s
2020/02/18 18:50:56 RPS: 148.773
2020/02/18 18:50:56 All transactions were sent

$ cat single.log

GoSingle / 150 rate / 1m0s

RPS ≈ 151.258

TPS ≈ 150.578

CPU, Mem
4.669, 66.719
7.425, 76.426
6.877, 77.047
...

TPS
37.500
150.000
150.000
150.000
...
```

## Benchmark usage

````
  -h, --help                       Show usage message.
  -d, --desc string                Benchmark description. (default "unknown benchmark")
  -o, --out string                 Path where report would be written. (default "report.log")
  -m, --mode                       Benchmark mode.
                                   Example: -m wrk --mode rate (default "rate")
  -w, --workers int                Number of used workers.
                                   Example: -w 10 -w 15 -w 40 (default 30)
  -z, --timeLimit duration         The time limit when an application can send requests.
                                   When the time limit is reached, application stops send requests and wait for parsing transactions.
                                   Examples: -z 10s -z 3m (default 30s)
  -q, --rateLimit int              QPS - queries per second, rate limit (default 1000)
  -c, --concurrent int             Number of used cpu cores.Example: -c 4 --concurrent 8 (default 4)
  -a, --rpcAddress                 RPC addresses for RPC calls to test nodes.
                                   You can specify multiple addresses.
                                   Example -a 127.0.0.1:80 -a 127.0.0.2:8080 (default [127.0.0.1:20331])
  -t, --request_timeout duration   Request timeout.
                                   Used for RPC requests.
                                   Example: -t 30s --request_timeout 15s (default 30s)
  -i, --in                         Path to input file to load transactions.
                                   Example: -i ./dump.txs --in /path/to/import/transactions
````

## Makefile usage

```
  make <target>

  Targets:

    build    Build all images
    gen      Generate `dump.txs` (run it before any benchmarks)
    help     Show this help prompt
    pull     Pull images from registry
    push     Push all images to registry
    single   Bootup NeoGo single node
    start    Run benchmark (uncomment needed)
    stop     Stop all containers
    test     Test local benchmark (go run) with NeoGo single node
```

## Runner usage (`.make/runner.sh`)

```
   -h, --help                       Show usage message.
   -d                               Benchmark description.
   -m                               Benchmark mode.
                                    Example: -m wrk -m rate
   -w                               Number of used workers.
                                    Example: -w 10 -w 15 -w 40
   -z                               The time limit when an application can send requests.
                                    When the time limit is reached, application stops send requests and wait for parsing transactions.
                                    Examples: -z 10s -z 3m
   -q                               QPS - queries per second, rate limit
   -c                               Number of used cpu cores.
                                    Example: -c 4
   -a                               RPC addresses for RPC calls to test nodes.
                                    You can specify multiple addresses.
                                    Example -a 127.0.0.1:80 -a 127.0.0.2:8080
   -t                               Request timeout.
                                    Used for RPC requests.
                                    Example: -t 30s
```
