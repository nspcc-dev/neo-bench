#!/bin/bash

maxDelayBlocks=10
# shellcheck disable=SC2155
export port=$(jq -r '.PluginConfiguration.Servers[0].Port' </neo-cli/Plugins/RpcServer/RpcServer.json)
export host=127.0.0.1
export addr=${host}:${port}
echo curl -s -X POST "http://$addr" -H 'Content-Type: application/json' -d '{ "jsonrpc": "2.0", "id": 5, "method": "getblockcount", "params": [] }'
curBlock=$(curl -s -X POST "http://$addr" -H 'Content-Type: application/json' -d '{ "jsonrpc": "2.0", "id": 5, "method": "getblockcount", "params": [] }' | jq '.result')

if [ "$curBlock" == "" ]; then
	curl -X POST "http://$addr" -H 'Content-Type: application/json' -d '{ "jsonrpc": "2.0", "id": 5, "method": "getblockcount", "params": [] }'
	echo "${addr} => '${host} : ${port}'"
	echo "NODE IS DOWN"
	exit 1
fi

nodes=$(jq -r .ProtocolConfiguration.SeedList[] </neo-cli/config.json | sed 's/:20/:30/')

for node in "${nodes[@]}"; do
	block=$(curl -s -X POST "http://$node" -H 'Content-Type: application/json' -d '{ "jsonrpc": "2.0", "id": 5, "method": "getblockcount", "params": [] }' | jq '.result')

	if [ "$block" == "" ]; then
		block=0
	fi

	syncDelay=$((block - curBlock))

	if [ "$syncDelay" -gt "$maxDelayBlocks" ]; then
		echo "NODE OUT OF SYNC"
		exit 2
	fi
done

echo "ALL OK - NODE SYNCED"
