#!/bin/bash

source .env

OUTPUT=""
ARGS=(-i "/dump.txs")
FILES=()
MODE=""
COUNT=""
SINGLE=
IR_TYPE=go
RPC_TYPE=
RPC_ADDR=()
NEOBENCH_LOGGER=${NEOBENCH_LOGGER:-none}
export NEOBENCH_TYPE=${NEOBENCH_TYPE:-NEO}

show_help() {
echo "Usage of benchmark runner:"
echo "   -s, --single                     Use single consensus node."
echo "   -n, --nodes                      Consensus node type."
echo "                                    Possible values: go (default), mixed, sharp."
echo "   -r, --rpc                        RPC node type. Default is the same as --nodes."
echo "   -h, --help                       Show usage message."
echo "   -b, --benchmark                  Benchmark type."
echo "                                    Possible values: NEO (default) or GAS"
echo "   -d                               Benchmark description."
echo "   -m                               Benchmark mode."
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
echo "   -l, --log                        Enable logging on consensus nodes."
echo "       --tc                         Arguments to pass to 'tc qdisc netem' inside the container."
echo "                                    Example: 'delay 100ms'"
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
    -h|--help) show_help ;;
    -s|--single) SINGLE=1 ;;
    -l|--log) export NEOBENCH_LOGGER=journald ;;

    -n|--nodes)
      test $# -gt 0 || fatal "Nodes type must be specified."
      IR_TYPE=$1
      shift
      ;;

    -r|--rpc)
      test $# -gt 0 || fatal "RPC node type must be specified."
      RPC_TYPE=$1
      shift
      ;;

    -b|--benchmark)
      test $# -gt 0 || fatal "benchmark type must be specified"
      export NEOBENCH_TYPE="$1"
      shift
      ;;

    -m)
      test $# -gt 0 || fatal "benchmark mode should be specified"
      case "$1" in
        "rate"|"wrk")
          ARGS+=(-m "$1")
          MODE="$1"
          ;;
        *)
          fatal "unknown benchmark mode specified: $1"
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
      COUNT="$1"
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
      COUNT="$1"
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
      ARGS+=( -t "$1")
      shift
      ;;

    --tc)
      test $# -gt 0 || fatal "tc arguments should be specified"
      export NEOBENCH_TC="$1"
      shift
      ;;

    *) fatal "Unknown option: $1" ;;
  esac
done

RPC_TYPE=${RPC_TYPE:-$IR_TYPE}

if [ -z "$SINGLE" ]; then
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

  if [ "$RPC_TYPE" = go ]; then
    FILES+=(-f "$DC_GO_RPC")
    DEFAULT_RPC_ADDR=(-a "go-node:20331")
  else
    FILES+=(-f "$DC_SHARP_RPC")
    DEFAULT_RPC_ADDR=(-a "sharp-node:20331")
  fi
else
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
fi

OUTPUT="/out/${OUTPUT}_${MODE}_${COUNT}.log"
if [ ${#RPC_ADDR[@]} -eq 0 ]; then
  ARGS+=("${DEFAULT_RPC_ADDR[@]}")
else
  ARGS+=("${RPC_ADDR[@]}")
fi

if [ -z "$SINGLE" ]; then
  make prepare
else
  make prepare.single
fi

docker-compose "${FILES[@]}" run bench neo-bench -o "$OUTPUT" "${ARGS[@]}"

make stop