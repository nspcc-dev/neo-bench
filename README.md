# Neo Blockchain benchmark (Neo 3 version)

## Requirements

- make
- any Docker version that includes Compose V2 (for Mac, Windows and Linux desktop versions, see the [installation instructions](https://docs.docker.com/compose/install/?_gl=1*uflyxz*_ga*MTE3MDkzODg4Ny4xNzA5MDM4MDA0*_ga_XJWPQMJYHQ*MTcxMzgwNTA0MC4yMC4xLjE3MTM4MDY5MDQuNjAuMC4w#installation-scenarios))
- Docker Compose V2 plugin (for Linux, if Docker Engine and Docker CLI are installed, see the [installation instructions](https://docs.docker.com/compose/install/?_gl=1*uflyxz*_ga*MTE3MDkzODg4Ny4xNzA5MDM4MDA0*_ga_XJWPQMJYHQ*MTcxMzgwNTA0MC4yMC4xLjE3MTM4MDY5MDQuNjAuMC4w#scenario-two-install-the-compose-plugin))
- Golang 1.20+

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
    - go.mod - golang modules file
    - go.sum - golang modules summary file
- .gitignore
- Makefile 
- README.md

## Usage example (local benchmark + NeoGo single node)

1. Build the benchmark images and binary file with the following command:
```
$ make build
=> Building Bench image registry.nspcc.ru/neo-bench/neo-bench:bench
sha256:b08f9fd42198be6c351d725543ac1e451063d18018a738f2446678a0cdf8ee78
=> Building Go Node image registry.nspcc.ru/neo-bench/neo-go:bench
sha256:2bf655747dfa06b85ced1ad7f0257128e7261e6d16b2c8087bc16fd27fcb3a6d
=> Building Sharp Node image registry.nspcc.ru/neo-bench/neo-sharp:bench
sha256:a6ed753e8f81fedf8a9be556e60c6a41e385dd1ab2c90755ab44e2ceab92bca2
=> Building Bench binary file
+ export GOGC=off
+ GOGC=off
+ export CGO_ENABLED=0
+ CGO_ENABLED=0
+ go -C cmd build -v -o bin/bench -trimpath ./bench
```

2. Run `test` target for a test run:
```
$ make test
./runner.sh --validators 1 -d "GoSingle" -m wrk -w 30 -z 5m -t 30s
make[1]: Entering directory '/home/anna/Documents/GitProjects/nspcc-dev/neo-bench'
=> Stop environment
=> Generate configurations for single-node and four-nodes networks from templates
+ cd ./cmd
+ go run ./config/ --go-template go.protocol.template.yml --go-db leveldb --sharp-template sharp.protocol.template.yml --sharp-db LevelDBStore
creating: 3115231476/go.protocol.template.yml
creating: 3115231476/go.protocol.template.yml
creating: 3115231476/sharp.protocol.template.yml
creating: 3115231476/sharp.protocol.template.yml
make[1]: Leaving directory '/home/anna/Documents/GitProjects/nspcc-dev/neo-bench' 
[+] Creating 3/1
 ✔ Network neo_go_network  Created                                                                                                                                                                            0.1s 
 ✔ Container ir-node-1     Created                                                                                                                                                                            0.1s 
 ✔ Container ir-healthy-1  Created                                                                                                                                                                            0.0s 
[+] Running 2/2
 ✔ Container ir-node-1     Healthy                                                                                                                                                                            5.7s 
 ✔ Container ir-healthy-1  Started                                                                                                                                                                            0.2s 
2024/04/22 17:03:33 Used [node:20331] rpc addresses
2024/04/22 17:03:33 Run benchmark for GoSingle :: NEO-GO
2024/04/22 17:03:34 Read 3000000 txs from /dump.txs
2024/04/22 17:03:36 CPU: 0.025%, Mem: 23.242MB
2024/04/22 17:03:38 CPU: 0.286%, Mem: 23.469MB
2024/04/22 17:03:40 CPU: 0.031%, Mem: 23.781MB
2024/04/22 17:03:41 Done 6.858495874s
2024/04/22 17:03:41 Init 30 workers / 5m0s time limit (3000000 txs will try to send)
2024/04/22 17:03:41 Prepare chain for benchmark
2024/04/22 17:03:41 Determined validators count: 1
2024/04/22 17:03:41 Sending NEO and GAS transfer tx
2024/04/22 17:03:41 Contract hash: ceb508fc02abc2dc27228e21976699047bbbcce0
2024/04/22 17:03:41 Sending contract deploy tx
2024/04/22 17:03:41 Contract was persisted: false
2024/04/22 17:03:42 Contract was persisted: false
2024/04/22 17:03:42 CPU: 0.155%, Mem: 24.777MB
2024/04/22 17:03:42 Contract was persisted: true
2024/04/22 17:03:42 fetch current block count
2024/04/22 17:03:42 Waiting for an empty block to be processed
2024/04/22 17:03:43 Started test from block = 17 at unix time = 1713805423759
2024/04/22 17:03:44 empty block: 17
2024/04/22 17:03:44 CPU: 37.295%, Mem: 38.734MB
2024/04/22 17:03:45 #18: 13690 transactions in 1011 ms - 13541.048467 tps
2024/04/22 17:03:46 CPU: 64.889%, Mem: 131.801MB
2024/04/22 17:03:46 #19: 13440 transactions in 1045 ms - 12861.244019 tps
2024/04/22 17:03:48 #20: 13625 transactions in 1036 ms - 13151.544402 tps
2024/04/22 17:03:48 CPU: 64.515%, Mem: 141.691MB
2024/04/22 17:03:49 #21: 14820 transactions in 1036 ms - 14305.019305 tps
2024/04/22 17:03:50 #22: 12248 transactions in 1042 ms - 11754.318618 tps
2024/04/22 17:03:50 CPU: 55.543%, Mem: 165.152MB

...

2024/04/22 17:08:37 CPU: 84.711%, Mem: 703.641MB
2024/04/22 17:08:38 #287: 10736 transactions in 1013 ms - 10598.223100 tps
2024/04/22 17:08:39 #288: 8970 transactions in 1030 ms - 8708.737864 tps
2024/04/22 17:08:39 CPU: 85.515%, Mem: 734.520MB
2024/04/22 17:08:40 #289: 8333 transactions in 1032 ms - 8074.612403 tps
2024/04/22 17:08:41 #290: 8100 transactions in 1029 ms - 7871.720117 tps
2024/04/22 17:08:41 CPU: 82.350%, Mem: 802.832MB
2024/04/22 17:08:42 #291: 8139 transactions in 1034 ms - 7871.373308 tps
2024/04/22 17:08:43 #292: 6965 transactions in 1036 ms - 6722.972973 tps
2024/04/22 17:08:43 CPU: 86.500%, Mem: 851.633MB
2024/04/22 17:08:43 all request workers stopped
2024/04/22 17:08:43 Sent 2429426 transactions in 5m0.004691002s
2024/04/22 17:08:43 RPS: 8097.960
2024/04/22 17:08:43 All transactions have been sent successfully
2024/04/22 17:08:43 RPC Errors: 0 / 0.000%
2024/04/22 17:08:43 sender worker stopped
2024/04/22 17:08:44 #293: 8023 transactions in 1033 ms - 7766.698935 tps
2024/04/22 17:08:44 #294: 3611 transactions in 1023 ms - 3529.814272 tps
2024/04/22 17:08:44 parser worker stopped
2024/04/22 17:08:44 try to write profile
GoSingle :: NEO-GO / 30 wrk / 5m0s

TXs ≈ 2429426
RPS ≈ 8097.960
RPC Errors  ≈ 0 / 0.000%
TPS ≈ 8079.692
DefaultMSPerBlock = 1000

CPU ≈ 72.766%
Mem ≈ 518.845MB
```

3. Check the test run results:
```
$ cat .docker/rpc/out/GoSingle_wrk_30.log
GoSingle :: NEO-GO / 30 wrk / 5m0s

TXs ≈ 1000000
RPS ≈ 12783.740
RPC Errors  ≈ 0 / 0.000%
TPS ≈ 12631.047
DefaultMSPerBlock = 1000

CPU ≈ 63.366%
Mem ≈ 275.360MB

MillisecondsFromStart, CPU, Mem
2005.115, 0.034%, 26.766MB
4015.739, 0.348%, 27.668MB
6027.083, 47.061%, 52.148MB
8034.887, 72.988%, 187.363MB
...

DeltaTime, TransactionsCount, TPS
1010, 16331, 16169.307
1043, 20012, 19186.961
1038, 18998, 18302.505
1052, 19018, 18077.947
...
```

4. Explore and run different benchmark configurations via the set of `make` boilerplate targets:
```
$ make start.GoFourNodes100wrk
$ make start.GoFourNodes300rate
$ make start.GoSingle30wrk
$ make start.SharpFourNodes50rate
$ make start.SharpFourNodesGoRPC30wrk
$ make start.MixedFourNodesGoRPC50rate
...
```
... or use the `runner.sh` script for custom benchmark setup:
```
$ ./runner.sh -h
$ ./runner.sh -d "Go4x1" -m wrk -w 30 -z 5m -t 30s
$ ./runner.sh --validators 1 --nodes sharp -d "SharpSingle" -m rate -q 25 -z 5m -t 30s
$ ./runner.sh --nodes mixed -d "MixedGoRPC4x1" -m rate -q 50 -z 5m -t 30s
...
```
## Usage example (local benchmark + external cluster)

NeoBench can be run in a stand-alone mode (loader only) to benchmark some external network. 
NeoBench expects external network to be launched with the known set of validators and committee (with wallets from `./docker/ir/`)
and not contain any transactions in the network (to successfully perform initial NEO/GAS transfers). In this setup the 
loader will be launched as a system process, without Docker container. You have to provide RPS address(-es) of some 
node from the external network to the loader instance on start. The loader will use the provided RPC to send 
transactions to the network.

1. Build the benchmark binary file with the following command:
```
$ make build
=> Building Bench image registry.nspcc.ru/neo-bench/neo-bench:bench
sha256:b08f9fd42198be6c351d725543ac1e451063d18018a738f2446678a0cdf8ee78
=> Building Go Node image registry.nspcc.ru/neo-bench/neo-go:bench
sha256:2bf655747dfa06b85ced1ad7f0257128e7261e6d16b2c8087bc16fd27fcb3a6d
=> Building Sharp Node image registry.nspcc.ru/neo-bench/neo-sharp:bench
sha256:a6ed753e8f81fedf8a9be556e60c6a41e385dd1ab2c90755ab44e2ceab92bca2
=> Building Bench binary file
+ export GOGC=off
+ GOGC=off
+ export CGO_ENABLED=0
+ CGO_ENABLED=0
+ go -C cmd build -v -o bin/bench -trimpath ./bench
```

2. Run benchmarks using the `runner.sh` script with RPC address(-es) of the external network and `--external` flag set:
```
$ ./runner.sh -e -d "Go4x1" -m rate -q 1000 -z 5m -t 30s -a 192.168.1.100:20331 -a 192.168.1.101:20331
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
      --vote                       Vote before the bench.
      --disable-stats              Disable memory and CPU usage statistics collection.
````

## Makefile usage

```
  make <target>

   Targets:
   
       build     Build all images
       config    Generate configurations for single-node and four-nodes networks from templates
       gen       Generate `dump.txs` (run it before any benchmarks)
       help      Show this help prompt
       prepare   Generate transactions and nodes configurations for four-nodes network
       pull      Pull images from registry
       push      Push all images to registry
       start     Runs benchmark for all default single-node and four-nodes C# and Go networks. Use `make start.<option>` to run tests separately
       stop      Stop all containers
       test      Test local benchmark (go run) with Neo single node
```

The following default configurations are available:

| Make target | Configuration description |
| --- | --- |
| `start` | Runs benchmark for all default single-node and four-nodes C# and Go networks. |
| `start.GoSingle10wrk` | Runs benchmark for single-node Go privat network under the load of 10 workers. |
| `start.GoSingle30wrk` | Runs benchmark for single-node Go privat network under the load of 30 workers. |
| `start.GoSingle100wrk` | Runs benchmark for single-node Go privat network under the load of 100 workers. |
| `start.GoSingle25rate` | Runs benchmark for single-node Go privat network under the load of 25 requests per second. |
| `start.GoSingle50rate` | Runs benchmark for single-node Go privat network under the load of 50 requests per second. |
| `start.GoSingle60rate` | Runs benchmark for single-node Go privat network under the load of 60 requests per second. |
| `start.GoSingle300rate` | Runs benchmark for single-node Go privat network under the load of 300 requests per second. |
| `start.GoSingle1000rate` | Runs benchmark for single-node Go privat network under the load of 1000 requests per second. |
| `start.GoFourNodes10wrk` | Runs benchmark for four-nodes Go privat network with Go RPC node under the load of 10 workers. |
| `start.GoFourNodes30wrk` | Runs benchmark for four-nodes Go privat network with Go RPC node under the load of 30 workers. |
| `start.GoFourNodes100wrk` | Runs benchmark for four-nodes Go privat network with Go RPC node under the load of 100 workers. |
| `start.GoFourNodes25rate` | Runs benchmark for four-nodes Go privat network with Go RPC node under the load of 25 requests per second. |
| `start.GoFourNodes50rate` | Runs benchmark for four-nodes Go privat network with Go RPC node under the load of 50 requests per second. |
| `start.GoFoutNodes60rate` | Runs benchmark for four-nodes Go privat network with Go RPC node under the load of 60 requests per second. |
| `start.GoFoutNodes300rate` | Runs benchmark for four-nodes Go privat network with Go RPC node under the load of 300 requests per second. |
| `start.GoFoutNodes1000rate` | Runs benchmark for four-nodes Go privat network with Go RPC node under the load of 1000 requests per second. |
| `start.SharpSingle10wrk` | Runs benchmark for single-node C# privat network under the load of 10 workers. |
| `start.SharpSingle30wrk` | Runs benchmark for single-node C# privat network under the load of 30 workers. |
| `start.SharpSingle100wrk` | Runs benchmark for single-node C# privat network under the load of 100 workers. |
| `start.SharpSingle25rate` | Runs benchmark for single-node C# privat network under the load of 25 requests per second. |
| `start.SharpSingle50rate` | Runs benchmark for single-node C# privat network under the load of 50 requests per second. |
| `start.SharpSingle60rate` | Runs benchmark for single-node C# privat network under the load of 60 requests per second. |
| `start.SharpSingle300rate` | Runs benchmark for single-node C# privat network under the load of 300 requests per second. |
| `start.SharpSingle1000rate` | Runs benchmark for single-node C# privat network under the load of 1000 requests per second. |
| `start.SharpFourNodes10wrk` | Runs benchmark for four-nodes C# privat network with C# RPC node under the load of 10 workers. |
| `start.SharpFourNodes30wrk` | Runs benchmark for four-nodes C# privat network with C# RPC node under the load of 30 workers. |
| `start.SharpFourNodes100wrk` | Runs benchmark for four-nodes C# privat network with C# RPC node under the load of 100 workers. |
| `start.SharpFourNodes25rate` | Runs benchmark for four-nodes C# privat network with C# RPC node under the load of 25 requests per second. |
| `start.SharpFourNodes50rate` | Runs benchmark for four-nodes C# privat network with C# RPC node under the load of 50 requests per second. |
| `start.SharpFoutNodes60rate` | Runs benchmark for four-nodes C# privat network with C# RPC node under the load of 60 requests per second. |
| `start.SharpFoutNodes300rate` | Runs benchmark for four-nodes C# privat network with C# RPC node under the load of 300 requests per second. |
| `start.SharpFoutNodes1000rate` | Runs benchmark for four-nodes C# privat network with C# RPC node under the load of 1000 requests per second. |
| `start.SharpFourNodesGoRPC10wrk` | Runs benchmark for four-nodes C# privat network with Go RPC node under the load of 10 workers. |
| `start.SharpFourNodesGoRPC30wrk` | Runs benchmark for four-nodes C# privat network with Go RPC node under the load of 30 workers. |
| `start.SharpFourNodesGoRPC100wrk` | Runs benchmark for four-nodes C# privat network with Go RPC node under the load of 100 workers. |
| `start.SharpFourNodesGoRPC25rate` | Runs benchmark for four-nodes C# privat network with Go RPC node under the load of 25 requests per second. |
| `start.SharpFourNodesGoRPC50rate` | Runs benchmark for four-nodes C# privat network with Go RPC node under the load of 50 requests per second. |
| `start.SharpFoutNodesGoRPC60rate` | Runs benchmark for four-nodes C# privat network with Go RPC node under the load of 60 requests per second. |
| `start.SharpFoutNodesGoRPC300rate` | Runs benchmark for four-nodes C# privat network with Go RPC node under the load of 300 requests per second. |
| `start.SharpFoutNodesGoRPC1000rate` | Runs benchmark for four-nodes C# privat network with Go RPC node under the load of 1000 requests per second. |

## Runner usage (`./runner.sh`)

```
   -v, --validators                 Consensus node count.
                                    Possible values: 1, 4 (default), 7.
   -n, --nodes                      Consensus node type.
                                    Possible values: go (default), mixed, sharp.
   -r, --rpc                        RPC node type. Default is the same as --nodes.
   -h, --help                       Show usage message.
   -b, --benchmark                  Benchmark type.
                                    Possible values: NEO (default) or GAS
       --from                       Number of tx senders (default: 1)
       --to                         Number of fund receivers (default: 1)
       --vote                       Whether or not candidates should be voted for before the bench.
   -d                               Benchmark description.
   -m                               Benchmark mode. Possible values: rate, wrk. In rate mode, -q and -w flags should be specified. In wrk mode, only -w flag should be specified.
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
   -l, --log                        Container logging facility. Default value is none.
                                    Example: -l journald -l syslog -l json-file
       --tc                         Arguments to pass to 'tc qdisc netem' inside the container.
                                    Example: 'delay 100ms'
       --msPerBlock                 Protocol setting specifying the minimal (and targeted for) time interval between blocks. Must be an integer number of milliseconds.
                                    The default value is set in configuration templates and is 1s and 5s for single node and multinode setup respectively.
                                    Example: --msPerBlock 3000
   -e, --external                   Use external network for benchmarking. Default is false. -a flag should be used to specify RPC addresses.

```

## Build options

By default, neo-bench uses released versions of Neo nodes to build Docker images.
However, you can easily test non-released branches or even separate commits for both Go and C# Neo nodes.
`--external` flag should be used for external network benchmarks to run the loader in a standalone mode without Docker container.

### Build Go node image from sources

To test non-released version of Go Neo node:

1. Set `ARG REV` variable of 
[Go node Dockerfile](https://github.com/nspcc-dev/neo-bench/blob/master/.docker/build/Dockerfile.golang#L9)
to the desired branch, tag or commit from the [neo-go github repository](https://github.com/nspcc-dev/neo-go).
Example:
```
ARG REV="v0.91.0"
```

2. Build Go Neo node image with the following command:
```
$   make build
```

### Build C# node image from sources

Use this way to test non-released version of C# node. Here we build all the neo, neo-vm, neo-node and neo-modules
projects with their MyGet dependencies from the source commit.

1. Set `DF_SHARP` variable of [neo-bench Makefile](https://github.com/nspcc-dev/neo-bench/blob/master/Makefile#L18)
to `.docker/build/Dockerfile.sharp.sources`.

2. Set `REVISION` environmental variable of
[C# node sources Dockerfile](https://github.com/nspcc-dev/neo-bench/blob/master/.docker/build/Dockerfile.sharp.sources#L12)
to the desired branch, tag or commit from the neo-project/neo GitHub repository.

3. C# node image includes `LevelDBStore`, `DBFTPlugin` and `RpcServer` plugins by default. If you need to install other
plugins, add desired plugin name to [`MODULES`](https://github.com/nspcc-dev/neo-bench/blob/master/.docker/build/Dockerfile.sharp.sources#L19)
variable of C# node-sources Dockerfile. This will use `dotnet build` command to build the specified plugin.

4. Build C# Neo node image with the following command:
   ```
   $   make build
   ```
   
## Nodes configurations

Nodes configuration files are built from templates for better maintenance and flexibility.
Template files describe application and protocol settings for single node, four nodes and RPC node which are used to run benchmarks.
[go.protocol.template.yml](https://github.com/nspcc-dev/neo-bench/blob/master/.docker/ir/go.protocol.template.yml)
is a configuration template for Golang Neo nodes, [sharp.protocol.template.yml](https://github.com/nspcc-dev/neo-bench/blob/master/.docker/ir/sharp.protocol.template.yml)
is a template for C# Neo node and [template.data.yml](https://github.com/nspcc-dev/neo-bench/blob/master/.docker/ir/template.data.yml)
contains a set of common data used by both Golang and C# node configurations.

We use [Yaml Templating Tool](https://github.com/k14s/ytt) to generate plain YAML and JSON files from the given templates.
It is quite intuitive, so if you'd like to change any node settings, just edit the corresponding configuration template and re-generate configuration files by using the following make command:
```
   $   make config
```

To add one more node configuration, provide all necessary information to the `node_info` list of [template.data.yml](https://github.com/nspcc-dev/neo-bench/blob/master/.docker/ir/template.data.yml), e.g.:
```
- node_name: five
    node_port: 20337
    node_rpc_port: 30337
    node_monitoring_port: 20005
    node_pprof_port: 30005
    node_prometheus_port: 40005
    validator_hash: "02a7bc55fe8684e0119768d104ba30795bdcc86619e864add26156723ed185cd62"
    wallet_password: "five"
```

## Environment variables

Name|Description| Default |Example
---|---|---------|---
NEOBENCH_LOGGER|Container logging facility| `none`  |`none`, `journald`, `syslog`,`json-file`
NEOBENCH_TC|Parameters passed to the `tc qdisc` (netem discipline) on container startup|         |`delay 100ms`
NEOBENCH_TYPE|Type of the load| `NEO`   |`NEO`, `GAS`
NEOBENCH_FROM_COUNT|Number of tx senders| `1`     | `1`
NEOBENCH_TO_COUNT|Number of fund receivers| `1`     | `1`
NEOBENCH_VALIDATOR_COUNT|Number of validators| `4`     | `1`, `4`, `7`
NEOBENCH_VOTE|Vote for validators before the bench| empty   |`1` or empty

For MacOS NEOBENCH_LOGGER should be set to `json-file` as `journald` and
`syslog` are not supported by this architecture.
## Benchmark results visualisation

There's a Python plotting script available for benchmark data visualisation. 
We are mostly concerned about transactions per second (TPS), transactions per block (TPB), 
milliseconds per block, CPU and Memory dependencies during benchmarking, so these are five types
of plots to be visualised.

### How to plot

1. Check that all benchmark logs are placed into the `.docker/ir/out/` folder (that's a default location for log files).

2. Edit `files_batch` variable in the [plot.py](https://github.com/nspcc-dev/neo-bench/blob/master/plot.py)
python script in order to include desired benchmark logs from the step 1 with the corresponding names.
   
3. Run the command `$    python3 plot.py .docker/ir/out/` where `.docker/ir/out/` is the logs source folder from step 1.

The resulting images will be saved to `./img/` folder.
