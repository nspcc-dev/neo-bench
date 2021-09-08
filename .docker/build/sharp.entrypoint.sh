#!/bin/bash

if [ -z "$BIN" ]; then
  BIN=/neo-cli/neo-cli
fi

if [ -n "$NEOBENCH_TC" ]; then
  # shellcheck disable=SC2086 # Intended splitting of $NEOBENCH_TC (may be "100ms 10ms distribution normal")
  if tc qdisc add dev eth0 root netem $NEOBENCH_TC; then
    echo "Set qdisc to netem $NEOBENCH_TC"
  else
    echo "Can't set qdisc to netem $NEOBENCH_TC"
    exit 1
  fi
fi

${BIN} "$@"
