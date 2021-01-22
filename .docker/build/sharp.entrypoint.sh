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

${BIN} "$@"
