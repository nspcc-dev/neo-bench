#!/bin/bash

source .env

OUTPUT=""
ARGS=()
FILES=()
MODE=""
TARGET_RPS=""
# Count of workers
WORKERS_COUNT="30"
IR_TYPE=go
RPC_TYPE=
RPC_ADDR=()
EXTERNAL_NETWORK=false
export NEOBENCH_LOGGER=${NEOBENCH_LOGGER:-none}
export NEOBENCH_TYPE=${NEOBENCH_TYPE:-NEO}
export NEOBENCH_FROM_COUNT=${NEOBENCH_FROM_COUNT:-1}
export NEOBENCH_TC=${NEOBENCH_TC:-}
export NEOBENCH_TO_COUNT=${NEOBENCH_TO_COUNT:-1}
export NEOBENCH_VALIDATOR_COUNT=${NEOBENCH_VALIDATOR_COUNT:-4}

show_help() {
	echo "Usage of benchmark runner:"
	echo "   -v, --validators                 Consensus node count."
	echo "                                    Possible values: 1, 4 (default), 7."
	echo "   -n, --nodes                      Consensus node type."
	echo "                                    Possible values: go (default), mixed, sharp."
	echo "   -r, --rpc                        RPC node type. Default is the same as --nodes."
	echo "   -h, --help                       Show usage message."
	echo "   -b, --benchmark                  Benchmark type."
	echo "                                    Possible values: NEO (default) or GAS"
	echo "       --from                       Number of tx senders (default: 1)"
	echo "       --to                         Number of fund receivers (default: 1)"
	echo "       --vote                       Whether or not candidates should be voted for before the bench."
	echo "   -d                               Benchmark description."
	echo "   -m                               Benchmark mode. Possible values: rate, wrk. In rate mode, -q and -w flags should be specified. In wrk mode, only -w flag should be specified."
	echo "                                    Example: -m wrk -m rate"
	echo "   -w                               Number of used workers."
	echo "                                    Example: -w 10 -w 15 -w 40"
	echo "   -z                               The time limit when an application can send requests."
	echo "                                    When the time limit is reached, application stops send requests and wait for parsing transactions."
	echo "                                    Examples: -z 10s -z 3m"
	echo "   -q                               QPS - queries per second, rate limit"
	echo "   -c                               Number of used cpu cores."
	echo "                                    Example: -c 4"
	echo "   -a                               RPC addresses for RPC calls to test nodes."
	echo "                                    You can specify multiple addresses."
	echo "                                    Example -a 127.0.0.1:80 -a 127.0.0.2:8080"
	echo "   -t                               Request timeout."
	echo "                                    Used for RPC requests."
	echo "                                    Example: -t 30s"
	echo "   -l, --log                        Container logging facility. Default value is none."
	echo "                                    Example: -l journald -l syslog -l json-file"
	echo "       --tc                         Arguments to pass to 'tc qdisc netem' inside the container."
	echo "                                    Example: 'delay 100ms'"
	echo "       --msPerBlock                 Protocol setting specifying the minimal (and targeted for) time interval between blocks. Must be an integer number of milliseconds."
	echo "                                    The default value is set in configuration templates and is 1s and 5s for single node and multinode setup respectively."
	echo "                                    Example: --msPerBlock 1000"
	echo "   -e, --external                   Use external network for benchmarking. Default is false. -a flag should be used to specify RPC addresses."
	exit 0
}

fatal() {
	echo "$1"
	exit 1
}

if [ $# == 0 ]; then
	show_help
fi

while test $# -gt 0; do
	_opt=$1
	shift

	case $_opt in
	-h | --help) show_help ;;
	-e|--external)
        EXTERNAL_NETWORK=true
        ;;
	-l | --log)
		if [[ $# -gt 0 && ${1:0:1} != "-" ]]; then
			case "$1" in
			"syslog" | "journald" | "json-file" | "none")
				export NEOBENCH_LOGGER="$1"
				shift
				;;
			*)
				fatal "unknown logger specified: $1"
				;;
			esac
		else
		  export NEOBENCH_LOGGER="none"
		fi
		;;

	--vote) export NEOBENCH_VOTE=1 ;;

	-v | --validators)
		test $# -gt 0 || fatal "Amount must be specified for --validators."
		NEOBENCH_VALIDATOR_COUNT=$1
		shift
		;;

	--from)
		test $# -gt 0 || fatal "Amount must be specified for --from."
		NEOBENCH_FROM_COUNT=$1
		shift
		;;

	--to)
		test $# -gt 0 || fatal "Amount must be specified for --to."
		NEOBENCH_TO_COUNT=$1
		shift
		;;

	-n | --nodes)
		test $# -gt 0 || fatal "Nodes type must be specified."
		IR_TYPE=$1
		shift
		;;

	-r | --rpc)
		test $# -gt 0 || fatal "RPC node type must be specified."
		RPC_TYPE=$1
		shift
		;;

	-b | --benchmark)
		test $# -gt 0 || fatal "benchmark type must be specified"
		export NEOBENCH_TYPE="$1"
		shift
		;;

	-m)
		test $# -gt 0 || fatal "benchmark mode should be specified"
		case "$1" in
		"rate" | "wrk")
			ARGS+=(-m "$1")
			MODE="$1"
			;;
		*)
			fatal "unknown benchmark mode specified: $1"
			;;
		esac
		shift
		;;

	-d)
		test $# -gt 0 || fatal "benchmark description should be specified"
		ARGS+=(-d "$1")
		OUTPUT="$1"
		shift
		;;

	-w)
		test $# -gt 0 || fatal "workers count should be specified"
		ARGS+=(-w "$1")
		WORKERS_COUNT="$1"
		shift
		;;

	-z)
		test $# -gt 0 || fatal "benchmark time limit should be specified"
		ARGS+=(-z "$1")
		shift
		;;

	-q)
		test $# -gt 0 || fatal "benchmark rate limit should be specified"
		ARGS+=(-q "$1")
		TARGET_RPS="$1"
		shift
		;;

	-c)
		test $# -gt 0 || fatal "number of used CPU cores should be specified"
		ARGS+=(-c "$1")
		shift
		;;

	-a)
		test $# -gt 0 || fatal "RPC address should be specified"
		RPC_ADDR+=(-a "$1")
		shift
		;;

	-t)
		test $# -gt 0 || fatal "request timeout should be specified"
		ARGS+=(-t "$1")
		shift
		;;

	--tc)
		test $# -gt 0 || fatal "tc arguments should be specified"
		export NEOBENCH_TC="$1"
		shift
		;;

	--msPerBlock)
		test $# -gt 0 || fatal "milliseconds per block should be specified"
		export MS_PER_BLOCK="$1"
		shift
		;;

	*) fatal "Unknown option: $_opt" ;;
	esac
done

RPC_TYPE=${RPC_TYPE:-$IR_TYPE}

if [ "$NEOBENCH_VALIDATOR_COUNT" -eq 4 ]; then
	case "$IR_TYPE" in
	go)
		FILES+=(-f "$DC_GO_IR")
		;;
	sharp)
		FILES+=(-f "$DC_SHARP_IR")
		;;
	mixed)
		FILES+=(-f "$DC_MIXED_IR")
		;;
	*)
		echo "Unknown node type: $IR_TYPE"
		exit 2
		;;
	esac

	if [ "$RPC_TYPE" = go ] || [ "$RPC_TYPE" = mixed ]; then
		FILES+=(-f "$DC_GO_RPC")
		DEFAULT_RPC_ADDR=(-a "go-node:20331")
	else
		FILES+=(-f "$DC_SHARP_RPC")
		DEFAULT_RPC_ADDR=(-a "sharp-node:20331")
	fi
elif [ "$NEOBENCH_VALIDATOR_COUNT" -eq 7 ]; then
	case "$IR_TYPE" in
	go) FILES+=(-f "$DC_GO_7_IR") ;;
	sharp) FILES+=(-f "$DC_SHARP_7_IR") ;;
	mixed) FILES+=(-f "$DC_MIXED_7_IR") ;;
	*) fatal "Unknown node type: $IR_TYPE" ;;
	esac

	if [ "$RPC_TYPE" = go ] || [ "$RPC_TYPE" = mixed ]; then
		FILES+=(-f "$DC_GO_7_RPC")
		DEFAULT_RPC_ADDR=(-a "go-node:20331" -a "go-node-2:20331")
	else
		FILES+=(-f "$DC_SHARP_7_RPC")
		DEFAULT_RPC_ADDR=(-a "sharp-node:20331" -a "sharp-node-2:20331")
	fi
elif [ "$NEOBENCH_VALIDATOR_COUNT" -eq 1 ]; then
	case "$IR_TYPE" in
	go)
		FILES+=(-f "$DC_GO_IR_SINGLE" -f "$DC_SINGLE")
		;;
	sharp)
		FILES+=(-f "$DC_SHARP_IR_SINGLE" -f "$DC_SINGLE")
		;;
	*)
		echo "Unknown single node type: $IR_TYPE"
		exit 2
		;;
	esac

	DEFAULT_RPC_ADDR=(-a "node:20331")
else
	fatal "Invalid validator count: $NEOBENCH_VALIDATOR_COUNT"
fi

if [ "rate" = "$MODE" ]; then
  OUTPUT="/out/${OUTPUT}_${MODE}_${TARGET_RPS}_workers_${WORKERS_COUNT}.log"
else
  OUTPUT="/out/${OUTPUT}_${MODE}_${WORKERS_COUNT}.log"
fi

if [ ${#RPC_ADDR[@]} -eq 0 ]; then
	ARGS+=("${DEFAULT_RPC_ADDR[@]}")
else
	ARGS+=("${RPC_ADDR[@]}")
fi

if [ -n "$NEOBENCH_VOTE" ]; then
	ARGS+=(--vote)
fi

make prepare
if [ "$EXTERNAL_NETWORK" = true ]; then
      ARGS+=(-i "./.docker/build/dump.$NEOBENCH_TYPE.$NEOBENCH_FROM_COUNT.$NEOBENCH_TO_COUNT.txs" --disable-stats)
      ./cmd/bin/bench -o "$OUTPUT" "${ARGS[@]}"&
      pid=$!
      wait $pid
else
    ARGS+=(-i "/dump.txs")
    docker compose "${FILES[@]}" run bench neo-bench -o "$OUTPUT" "${ARGS[@]}"
fi
make stop
