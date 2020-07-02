#!/bin/bash

BIN=/neo-cli/neo-cli

if [ -z "$ACC" ]; then
  ACC=6000-privnet-blocks.acc.gz
fi

if test -f ${ACC}; then
    gunzip -c /${ACC} > /neo-cli/chain.acc
fi

[[ -p node.log ]] || mkfifo node.log

trap 'echo "Exit"; exit 1' 1 2 3 15

screen -dmS node -L node.log ${BIN} "$@"

tail -n+1 -f node.log
