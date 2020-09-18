# Neo Blockchain benchmark (Neo 3 version)

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
   
       build     Build all images
       config    Generate configurations for single-node and four-nodes networks from templates
       dump      Generate `dump.acc` for the 4-node network
       gen       Generate `dump.txs` (run it before any benchmarks)
       help      Show this help prompt
       prepare   Generate transactions, dump and nodes configurations for four-nodes network
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

## Build options

By default, neo-bench uses released versions of Neo nodes to build Docker images.
However, you can easily test non-released branches or even separate commits for both Go and C# Neo nodes.

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

To test non-released version of C# node:

1. Set `DF_SHARP` variable of [neo-bench Makefile](https://github.com/nspcc-dev/neo-bench/blob/master/Makefile#L18)
to `.docker/build/Dockerfile.sharp.sources`. It is necessary because neo-bench have two separate Dockerfiles to build C#
Neo node image from release and from sources.

2. Set `CLIBRANCH`, `MODULESBRANCH`, `NEOVMBRANCH` and `NEOBRANCH` variables of 
[C# node-sources Dockerfile](https://github.com/nspcc-dev/neo-bench/blob/master/.docker/build/Dockerfile.sharp.sources#L12)
to the desired branch, tag or commit from the corresponding repositories. Refer the following table for the variables meaning:

    | Variable | Purpose | Example |
    | --- | --- | --- |
    | [`CLIBRANCH`](https://github.com/nspcc-dev/neo-bench/blob/master/.docker/build/Dockerfile.sharp.sources#L12) | Branch, tag or commit from [C# neo-node github repository](https://github.com/neo-project/neo-node) to build neo-cli from the source code | `ENV CLIBRANCH="v3.0.0-preview3"` |
    | [`MODULESBRANCH`](https://github.com/nspcc-dev/neo-bench/blob/master/.docker/build/Dockerfile.sharp.sources#L24) | Branch, tag or commit from [C# neo-modules github repository](https://github.com/neo-project/neo-modules) to build node Plugins from the source code | `ENV MODULESBRANCH="v3.0.0-preview3-00"` |
    | [`NEOVMBRANCH`](https://github.com/nspcc-dev/neo-bench/blob/master/.docker/build/Dockerfile.sharp.sources#L46) | Branch, tag or commit from [C# neo-vm github repository](https://github.com/neo-project/neo-vm) to build neo VM (Neo.VM.dll) from the source code | `ENV NEOVMBRANCH="v3.0.0-preview3"` |
    | [`NEOBRANCH`](https://github.com/nspcc-dev/neo-bench/blob/master/.docker/build/Dockerfile.sharp.sources#L54) | Branch, tag or commit from [C# neo github repository](https://github.com/neo-project/neo/) to build neo itself (Neo.dll) from the source code | `ENV NEOBRANCH="v3.0.0-preview3"` |

3. C# node image includes `LevellDBStore`, `BadgerDBStore` and `RpcServer` Plugins by default. If you need to install other 
Plugins, add desired Plugin name to [`MODULES`](https://github.com/nspcc-dev/neo-bench/blob/master/.docker/build/Dockerfile.sharp.sources#L25) 
variable of C# node-sources Dockerfile. This will use `dotnet build` command to build the specified plugin without dependencies.
If you need to build plugin with dependant .dll-s, refer to [this section](https://github.com/nspcc-dev/neo-bench/blob/master/.docker/build/Dockerfile.sharp.sources#L35-L43)
of C# node-sources Dockerfile.

4. ***Please, be careful while choosing branch, tag or commit on step 2.*** It is possible that one of the `neo-cli`, `neo-modules`, `neo` or `neo-vm` versions
 is incompatible with the others. For example, some method required for `neo.dll` from master-branch is missing in `neo-vm.dll` from v3.0.0-preview3-branch. In this
 case you either have to provide the compatible `neo-vm` branch via [`NEOVMBRANCH`](https://github.com/nspcc-dev/neo-bench/blob/master/.docker/build/Dockerfile.sharp.sources#L46)
 variable *or* provide `neo-vm.dll` from the dependencies of built `neo` (not from built `neo-vm`) by editing 
 [these sections](https://github.com/nspcc-dev/neo-bench/blob/master/.docker/build/Dockerfile.sharp.sources#L89-L105) of C# node-sources Dockerfile.

5. Build C# Neo node image with the following command:
   ```
   $   make build
   ```

