#!/bin/bash

if [ -z "$BIN" ]; then
  BIN=/neo-cli/neo-cli
fi

if [ -z "$ACC" ]; then
  ACC=single.acc
fi

if test -f /"$ACC"; then
  cp /${ACC} /neo-cli/chain.acc
fi

[[ -p node.log ]] || mkfifo node.log

trap 'echo "Exit"; exit 1' 1 2 3 15

screen -dmS node -L node.log ${BIN} "$@"

tail -n+1 -f node.log
