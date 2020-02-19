#!/bin/bash

OUTPUT=""
ARGS=""
FILES=""
MODE=""
COUNT=""

show_help() {
echo "Usage of benchmark runner:"
echo "   -h, --help                       Show usage message."
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
exit 0
}

if [ $# == 0 ]; then
  show_help
fi

while test $# -gt 0; do
  case $1 in
    -h|--help)
      show_help
      ;;

    -f)
      shift
      if test $# -gt 0; then
        FILES="${FILES} -f $1"
      else
        echo "docker-compose file should be specified"
        exit 1
      fi
      shift
      ;;

    -d)
      shift
      if test $# -gt 0; then
        ARGS="${ARGS} -d \"$1\""
        OUTPUT="$1"
      else
        echo "benchmark description should be specified"
        exit 1
      fi
      shift
      ;;

    -m)
      shift
      if test $# -gt 0; then
        case "$1" in
          "rate"|"wrk")
            ARGS="${ARGS} -m $1"
            MODE="$1"
            ;;
          *)
            echo "unknown benchmark mode specified: $1"
            exit 2
        esac
      else
        echo "benchmark mode should be specified"
        exit 1
      fi
      shift
      ;;

    -w)
      shift
      if test $# -gt 0; then
        ARGS="${ARGS} -w $1"
        COUNT="$1"
      else
        echo "workers count should be specified"
        exit 1
      fi
      shift
      ;;

    -z)
      shift
      if test $# -gt 0; then
        ARGS="${ARGS} -z $1"
      else
        echo "benchmark time limit should be specified"
        exit 1
      fi
      shift
      ;;

    -q)
      shift
      if test $# -gt 0; then
        ARGS="${ARGS} -q $1"
        COUNT="$1"
      else
        echo "benchmark rate limit should be specified"
        exit 1
      fi
      shift
      ;;

    -c)
      shift
      if test $# -gt 0; then
        ARGS="${ARGS} -c $1"
      else
        echo "number of used CPU cores should be specified"
        exit 1
      fi
      shift
      ;;

    -i)
      shift
      if test $# -gt 0; then
        ARGS="${ARGS} -i $1"
      else
        echo "path to file with transactions dump should be specified"
        exit 1
      fi
      shift
      ;;

    -a)
      shift
      if test $# -gt 0; then
        ARGS="${ARGS} -a $1"
      else
        echo "RPC address should be specified"
        exit 1
      fi
      shift
      ;;

    -t)
      shift
      if test $# -gt 0; then
        ARGS="${ARGS} -t $1"
      else
        echo "request timeout should be specified"
        exit 1
      fi
      shift
      ;;

    *)
      echo "Unknown option: $1"
      exit 2
      ;;
  esac
done

OUTPUT="/out/${OUTPUT}_${MODE}_${COUNT}.log"

docker-compose ${FILES} run bench neo-bench -o $OUTPUT $ARGS