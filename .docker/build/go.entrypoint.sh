#!/bin/sh

BIN=/usr/bin/neo-go

if [ -z "$ACC" ]; then
  ACC=single.acc
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

case $@ in
	"node"*)
	echo "=> Try to restore blocks before running node"
	if test -f /"$ACC"; then
		${BIN} db restore -p --config-path /config -i /"$ACC"
	fi
  	;;
esac

${BIN} "$@"
