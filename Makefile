SHELL := /bin/bash

include .env
export

TAG=bench
HUB=nspccdev/neo-node
HUB=registry.nspcc.ru/neo-bench/neo
BUILD_DIR=.docker/build
NEOBENCH_TYPE ?= NEO
NEOBENCH_FROM_COUNT ?= 1
NEOBENCH_TO_COUNT ?= 1
MS_PER_BLOCK ?= 0

.PHONY: help

# Show this help prompt
help:
	@echo '  Usage:'
	@echo ''
	@echo '    make <target>'
	@echo ''
	@echo '  Targets:'
	@echo ''
	@awk '/^#/{ comment = substr($$0,3) } comment && /^[a-zA-Z][a-zA-Z0-9_-]+ ?:/{ print "   ", $$1, comment }' $(MAKEFILE_LIST) | column -t -s ':' | grep -v 'IGNORE' | sort | uniq

.PHONY: build prepare push gen build.node.go build.node.sharp build.bench stop start config \
	start.GoSingle10wrk start.GoSingle30wrk start.GoSingle100wrk \
	start.GoSingle25rate start.GoSingle50rate start.GoSingle60rate start.GoSingle300rate start.GoSingle1000rate \
	start.GoFourNodes10wrk start.GoFourNodes30wrk start.GoFourNodes100wrk \
	start.GoFourNodes25rate start.GoFourNodes50rate start.GoFourNodes60rate start.GoFourNodes300rate start.GoFourNodes1000rate \
	start.SharpSingle10wrk start.SharpSingle30wrk start.SharpSingle100wrk \
	start.SharpSingle25rate start.SharpSingle50rate start.SharpSingle60rate start.SharpSingle300rate start.SharpSingle1000rate \
	start.SharpFourNodes10wrk start.SharpFourNodes30wrk start.SharpFourNodes100wrk \
	start.SharpFourNodes25rate start.SharpFourNodes50rate start.SharpFourNodes60rate start.SharpFourNodes300rate start.SharpFourNodes1000rate \
	start.SharpFourNodesGoRPC10wrk start.SharpFourNodesGoRPC30wrk start.SharpFourNodesGoRPC100wrk \
	start.SharpFourNodesGoRPC25rate start.SharpFourNodesGoRPC50rate start.SharpFourNodesGoRPC60rate start.SharpFourNodesGoRPC300rate start.SharpFourNodesGoRPC1000rate

# Build all images
build: gen build.node.bench build.node.go build.node.sharp build.bench

# Build Benchmark binary file
build.bench:
	@echo "=> Building Bench binary file"
	@set -x \
		&& export GOGC=off \
		&& export CGO_ENABLED=0 \
		&& go build -C cmd -v -o bin/bench -trimpath ./bench

# Push all images to registry
push:
	docker push $(HUB)-bench:$(TAG)
	docker push $(HUB)-go:$(TAG)
	docker push $(HUB)-sharp:$(TAG)

# IGNORE: Build Benchmark image
build.node.bench:
	@echo "=> Building Bench image $(HUB)-bench:$(TAG)"
	@docker build -q -t $(HUB)-bench:$(TAG) -f $(DF_BENCH) cmd/

# IGNORE: Build NeoGo node image
build.node.go:
	@echo "=> Building Go Node image $(HUB)-go:$(TAG)"
	@docker build -q -t $(HUB)-go:$(TAG) -f $(DF_GO) $(BUILD_DIR)

# IGNORE: Build NeoSharp node image
build.node.sharp:
	@echo "=> Building Sharp Node image $(HUB)-sharp:$(TAG)"
	@docker build -q -t $(HUB)-sharp:$(TAG) -f $(DF_SHARP) $(BUILD_DIR)

# Test local benchmark (go run) with Neo single node
test: start.GoSingle30wrk

# Bootup NeoGo single node
single.go: stop
	@echo "=> Up Golang single node"
	@docker compose -f $(DC_GO_IR_SINGLE) up -d healthy

# Bootup NeoSharp single node
single.sharp: stop
	@echo "=> Up Sharp single node"
	@docker compose -f $(DC_SHARP_IR_SINGLE) up -d healthy

# Stop all containers
stop:
	@echo "=> Stop environment"
	@docker compose -f $(DC_GO_IR) -f $(DC_GO_7_IR) -f $(DC_GO_RPC) -f $(DC_GO_7_RPC) \
		-f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -f $(DC_SHARP_IR) -f $(DC_SHARP_7_IR) \
		-f $(DC_SHARP_RPC) -f $(DC_SHARP_7_RPC) -f $(DC_SHARP_IR_SINGLE) kill &> /dev/null
	@docker compose -f $(DC_GO_IR) -f $(DC_GO_7_IR) -f $(DC_GO_RPC) -f $(DC_GO_7_RPC) \
		-f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -f $(DC_SHARP_IR) -f $(DC_SHARP_7_IR) \
		-f $(DC_SHARP_RPC) -f $(DC_SHARP_7_RPC) -f $(DC_SHARP_IR_SINGLE) down --remove-orphans &> /dev/null
	@echo "=> Stop Bench process"
    +       @killall -w -v -INT bench > /dev/null 2>&1 || :

# Check that all images were built
check.images:
	@echo "=> Check that images created"
	@docker image ls -q $(HUB)-go:$(TAG) >> /dev/null || exit 2
	@docker image ls -q $(HUB)-bench:$(TAG) >> /dev/null || exit 2
	@docker image ls -q $(HUB)-sharp:$(TAG) >> /dev/null || exit 2

# Pull images from registry
pull:
	@docker pull $(HUB)-bench:$(TAG)
	@docker pull $(HUB)-go:$(TAG)
	@docker pull $(HUB)-sharp:$(TAG)

gen: $(BUILD_DIR)/dump.NEO.$(NEOBENCH_FROM_COUNT).$(NEOBENCH_TO_COUNT).txs
gen: $(BUILD_DIR)/dump.GAS.$(NEOBENCH_FROM_COUNT).$(NEOBENCH_TO_COUNT).txs
gen: $(BUILD_DIR)/dump.NEP17.$(NEOBENCH_FROM_COUNT).$(NEOBENCH_TO_COUNT).txs

# Generate `dump.txs`
$(BUILD_DIR)/dump.%.$(NEOBENCH_FROM_COUNT).$(NEOBENCH_TO_COUNT).txs: cmd/gen/main.go
	@echo "=> Generate transactions dump"
	@set -x \
		&& cd cmd/ \
		&& go run ./gen -cnt 3000000 -type $* -from $(NEOBENCH_FROM_COUNT) \
			-to $(NEOBENCH_TO_COUNT) -out ../$@

# Generate configurations for single-node and four-nodes networks from templates
config:
	@echo "=> Generate configurations for single-node and four-nodes networks from templates"
	@set -x \
		&& cd ./cmd \
		&& go run ./config/ --go-template go.protocol.template.yml --go-db leveldb --sharp-template sharp.protocol.template.yml --sharp-db LevelDBStore --msPerBlock $(MS_PER_BLOCK)


# Generate transactions, dump and nodes configurations for four-nodes network
prepare: stop config $(BUILD_DIR)/dump.$(NEOBENCH_TYPE).$(NEOBENCH_FROM_COUNT).$(NEOBENCH_TO_COUNT).txs

# Runs benchmark for all default single-node and four-nodes C# and Go networks. Use `make start.<option>` to run tests separately
start: start.GoSingle10wrk start.GoSingle30wrk start.GoSingle100wrk \
	start.GoSingle25rate start.GoSingle50rate start.GoSingle60rate start.GoSingle300rate start.GoSingle1000rate \
	start.GoFourNodes10wrk start.GoFourNodes30wrk start.GoFourNodes100wrk \
	start.GoFourNodes25rate start.GoFourNodes50rate start.GoFourNodes60rate start.GoFourNodes300rate start.GoFourNodes1000rate \
	start.SharpSingle10wrk start.SharpSingle30wrk start.SharpSingle100wrk \
	start.SharpSingle25rate start.SharpSingle50rate start.SharpSingle60rate start.SharpSingle300rate start.SharpSingle1000rate \
	start.SharpFourNodes10wrk start.SharpFourNodes30wrk start.SharpFourNodes100wrk \
	start.SharpFourNodes25rate start.SharpFourNodes50rate start.SharpFourNodes60rate start.SharpFourNodes300rate start.SharpFourNodes1000rate \
	start.SharpFourNodesGoRPC10wrk start.SharpFourNodesGoRPC30wrk start.SharpFourNodesGoRPC100wrk \
	start.SharpFourNodesGoRPC25rate start.SharpFourNodesGoRPC50rate start.SharpFourNodesGoRPC60rate start.SharpFourNodesGoRPC300rate start.SharpFourNodesGoRPC1000rate

## GoSingle:
#	## Workers:
start.GoSingle10wrk:
	./runner.sh --validators 1 -d "GoSingle" -m wrk -w 10 -z 5m -t 30s

start.GoSingle30wrk:
	./runner.sh --validators 1 -d "GoSingle" -m wrk -w 30 -z 5m -t 30s

start.GoSingle100wrk:
	./runner.sh --validators 1 -d "GoSingle" -m wrk -w 100 -z 5m -t 30s

#	## Rate:
start.GoSingle25rate:
	./runner.sh --validators 1 -d "GoSingle" -m rate -q 25 -z 5m -t 30s

start.GoSingle50rate:
	./runner.sh --validators 1 -d "GoSingle" -m rate -q 50 -z 5m -t 30s

start.GoSingle60rate:
	./runner.sh --validators 1 -d "GoSingle" -m rate -q 60 -z 5m -t 30s

start.GoSingle300rate:
	./runner.sh --validators 1 -d "GoSingle" -m rate -q 300 -z 5m -t 30s

start.GoSingle1000rate:
	./runner.sh --validators 1 -d "GoSingle" -m rate -q 1000 -z 5m -t 30s

## Go x 4 + GoRPC:
#	## Workers:
start.GoFourNodes10wrk:
	./runner.sh -d "Go4x1" -m wrk -w 10 -z 5m -t 30s

start.GoFourNodes30wrk:
	./runner.sh -d "Go4x1" -m wrk -w 30 -z 5m -t 30s

start.GoFourNodes100wrk:
	./runner.sh -d "Go4x1" -m wrk -w 100 -z 5m -t 30s

#	## Rate:
start.GoFourNodes25rate:
	./runner.sh -d "Go4x1" -m rate -q 25 -z 5m -t 30s

start.GoFourNodes50rate:
	./runner.sh -d "Go4x1" -m rate -q 50 -z 5m -t 30s

start.GoFourNodes60rate:
	./runner.sh -d "Go4x1" -m rate -q 60 -z 5m -t 30s

start.GoFourNodes300rate:
	./runner.sh -d "Go4x1" -m rate -q 300 -z 5m -t 30s

start.GoFourNodes1000rate:
	./runner.sh -d "Go4x1" -m rate -q 1000 -z 5m -t 30s

## Go×4 + SharpRPC
#
start.GoFourNodesSharpRpc10wrk:
	./runner.sh --rpc sharp -d "GoSharpRPC4x1" -m wrk -w 10 -z 5m -t 30s

## SharpSingle:
#	## Workers:
start.SharpSingle10wrk:
	./runner.sh --validators 1 --nodes sharp -d "SharpSingle" -m wrk -w 10 -z 5m -t 30s

start.SharpSingle30wrk:
	./runner.sh --validators 1 --nodes sharp -d "SharpSingle" -m wrk -w 30 -z 5m -t 30s

start.SharpSingle100wrk:
	./runner.sh --validators 1 --nodes sharp -d "SharpSingle" -m wrk -w 100 -z 5m -t 30s

#	## Rate:
start.SharpSingle25rate:
	./runner.sh --validators 1 --nodes sharp -d "SharpSingle" -m rate -q 25 -z 5m -t 30s

start.SharpSingle50rate:
	./runner.sh --validators 1 --nodes sharp -d "SharpSingle" -m rate -q 50 -z 5m -t 30s

start.SharpSingle60rate:
	./runner.sh --validators 1 --nodes sharp -d "SharpSingle" -m rate -q 60 -z 5m -t 30s

start.SharpSingle300rate:
	./runner.sh --validators 1 --nodes sharp -d "SharpSingle" -m rate -q 300 -z 5m -t 30s

start.SharpSingle1000rate:
	./runner.sh --validators 1 --nodes sharp -d "SharpSingle" -m rate -q 1000 -z 5m -t 30s

## Sharp x 4 + SharpRPC:
#	## Workers:
start.SharpFourNodes10wrk:
	./runner.sh --nodes sharp --rpc sharp -d "Sharp4x_SharpRPC" -m wrk -w 10 -z 5m -t 30s

start.SharpFourNodes30wrk:
	./runner.sh --nodes sharp --rpc sharp -d "Sharp4x_SharpRPC" -m wrk -w 30 -z 5m -t 30s

start.SharpFourNodes100wrk:
	./runner.sh --nodes sharp --rpc sharp -d "Sharp4x_SharpRPC" -m wrk -w 100 -z 5m -t 30s

#	## Rate:
start.SharpFourNodes25rate:
	./runner.sh --nodes sharp --rpc sharp -d "Sharp4x_SharpRPC" -m rate -q 25 -z 5m -t 30s

start.SharpFourNodes50rate:
	./runner.sh --nodes sharp --rpc sharp -d "Sharp4x_SharpRPC" -m rate -q 50 -z 5m -t 30s

start.SharpFourNodes60rate:
	./runner.sh --nodes sharp --rpc sharp -d "Sharp4x_SharpRPC" -m rate -q 60 -z 5m -t 30s

start.SharpFourNodes300rate:
	./runner.sh --nodes sharp --rpc sharp -d "Sharp4x_SharpRPC" -m rate -q 300 -z 5m -t 30s

start.SharpFourNodes1000rate:
	./runner.sh --nodes sharp --rpc sharp -d "Sharp4x_SharpRPC" -m rate -q 1000 -z 5m -t 30s

## Sharp x 4 + GoRPC:
#	## Workers:
start.SharpFourNodesGoRPC10wrk:
	./runner.sh --nodes sharp -d "Sharp4x_GoRPC" -m wrk -w 10 -z 5m -t 30s

start.SharpFourNodesGoRPC30wrk:
	./runner.sh --nodes sharp -d "Sharp4x_GoRPC" -m wrk -w 30 -z 5m -t 30s

start.SharpFourNodesGoRPC100wrk:
	./runner.sh --nodes sharp -d "Sharp4x_GoRPC" -m wrk -w 100 -z 5m -t 30s

#	## Rate:
start.SharpFourNodesGoRPC25rate:
	./runner.sh --nodes sharp -d "Sharp4x_GoRPC" -m rate -q 25 -z 5m -t 30s

start.SharpFourNodesGoRPC50rate:
	./runner.sh --nodes sharp -d "Sharp4x_GoRPC" -m rate -q 50 -z 5m -t 30s

start.SharpFourNodesGoRPC60rate:
	./runner.sh --nodes sharp -d "Sharp4x_GoRPC" -m rate -q 60 -z 5m -t 30s

start.SharpFourNodesGoRPC300rate:
	./runner.sh --nodes sharp -d "Sharp4x_GoRPC" -m rate -q 300 -z 5m -t 30s

start.SharpFourNodesGoRPC1000rate:
	./runner.sh --nodes sharp -d "Sharp4x_GoRPC" -m rate -q 1000 -z 5m -t 30s

## Mixed setup
#
start.MixedFourNodesGoRPC50rate:
	./runner.sh --nodes mixed -d "MixedGoRPC4x1" -m rate -q 50 -z 5m -t 30s

start.MixedFourNodesSharpRPC50rate:
	./runner.sh --nodes mixed --rpc sharp -d "MixedSharpRPC4x1" -m rate -q 50 -z 5m -t 30s
