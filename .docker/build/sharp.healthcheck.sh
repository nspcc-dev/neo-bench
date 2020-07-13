#!/bin/bash

maxDelayBlocks=10
export port=`jq -r '.PluginConfiguration.Port' < /neo-cli/Plugins/RpcServer/config.json`
export host=127.0.0.1
export addr=${host}:${port}
echo curl -s -X POST http://${addr} -H 'Content-Type: application/json' -d '{ "jsonrpc": "2.0", "id": 5, "method": "getblockcount", "params": [] }'
curBlock=`curl -s -X POST http://${addr} -H 'Content-Type: application/json' -d '{ "jsonrpc": "2.0", "id": 5, "method": "getblockcount", "params": [] }'| jq '.result'`

if [ "$curBlock" == "" ]
then
    curl -X POST http://$addr -H 'Content-Type: application/json' -d '{ "jsonrpc": "2.0", "id": 5, "method": "getblockcount", "params": [] }'
    echo "${addr} => '${host} : ${port}'"
    echo "NODE IS DOWN"
    exit 503
fi

nodes=`jq -r .ProtocolConfiguration.SeedList[] < /neo-cli/protocol.json | sed 's/:20/:30/`

for node in "${nodes[@]}"  
do  
    block=`curl -s -X POST http://$node -H 'Content-Type: application/json' -d '{ "jsonrpc": "2.0", "id": 5, "method": "getblockcount", "params": [] }'| jq '.result'`

    if [ "$block" == "" ]
    then
        block=0
    fi

    syncDelay=`expr $block - $curBlock`

    if [ "$syncDelay" -gt "$maxDelayBlocks" ]
    then 
        echo "NODE OUT OF SYNC"
        exit 408
    fi
done

echo "ALL OK - NODE SYNCED"
