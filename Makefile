SHELL := /bin/bash

DC_GO_IR=.docker/ir/docker-compose.go.yml
DC_GO_RPC=.docker/rpc/docker-compose.go.yml

DC_SINGLE=.docker/rpc/docker-compose.single.yml

DC_GO_IR_SINGLE=.docker/ir/docker-compose.single.go.yml
DC_SHARP_IR_SINGLE=.docker/ir/docker-compose.single.sharp.yml

DC_SHARP_IR=.docker/ir/docker-compose.sharp.yml
DC_SHARP_RPC=.docker/rpc/docker-compose.sharp.yml

DF_GO=.docker/build/Dockerfile.golang
DF_BENCH=.docker/build/Dockerfile.bench
DF_SHARP=.docker/build/Dockerfile.sharp
#DF_SHARP=.docker/build/Dockerfile.sharp.sources

TAG=bench
HUB=nspccdev/neo-node
HUB=registry.nspcc.ru/neo-bench/neo
BUILD_DIR=.docker/build

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

.PHONY: build push build.node.go build.node.sharp stop start deps config \
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
build: dumps gen build.node.bench build.node.go build.node.sharp

# Push all images to registry
push:
	docker push $(HUB)-bench:$(TAG)
	docker push $(HUB)-go:$(TAG)
	docker push $(HUB)-sharp:$(TAG)

# IGNORE
deps:
	@echo "=> Fetch deps"
	@set -x \
		&& cd cmd/ \
		&& go mod tidy -v \
		&& go mod vendor

# IGNORE: Build Benchmark image
build.node.bench: deps
	@echo "=> Building Bench image $(HUB)-bench:$(TAG)"
	@docker build -q -t $(HUB)-bench:$(TAG) -f $(DF_BENCH) cmd/

# IGNORE: Build NeoGo node image
build.node.go: deps
	@echo "=> Building Go Node image $(HUB)-go:$(TAG)"
	@docker build -q -t $(HUB)-go:$(TAG) -f $(DF_GO) $(BUILD_DIR)

# IGNORE: Build NeoSharp node image
build.node.sharp:
	@echo "=> Building Sharp Node image $(HUB)-sharp:$(TAG)"
	@docker build -q -t $(HUB)-sharp:$(TAG) -f $(DF_SHARP) $(BUILD_DIR)

# Test local benchmark (go run) with Neo single node
test: single.go deps
	@echo "=> Test Single node"
	@set -x \
		&& cd cmd/ \
		&& go run ./bench -o ../single.log -i ../$(BUILD_DIR)/dump.txs -d "SingleNode" -m rate -q 1000 -z 1m -t 30s -a localhost:20331
	@make stop

# Bootup NeoGo single node
single.go: stop
	@echo "=> Up Golang single node"
	@docker-compose -f $(DC_GO_IR_SINGLE) up -d healthy

# Bootup NeoSharp single node
single.sharp: stop
	@echo "=> Up Sharp single node"
	@docker-compose -f $(DC_SHARP_IR_SINGLE) up -d healthy

# Stop all containers
stop:
	@echo "=> Stop environment"
	@docker-compose -f $(DC_GO_IR) -f $(DC_GO_RPC) -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -f $(DC_SHARP_IR_SINGLE) kill &> /dev/null
	@docker-compose -f $(DC_GO_IR) -f $(DC_GO_RPC) -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -f $(DC_SHARP_IR_SINGLE) down --remove-orphans &> /dev/null

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

# Generate `dump.txs`
gen: $(BUILD_DIR)/dump.txs

# IGNORE: create transactions dump
$(BUILD_DIR)/dump.txs: deps cmd/gen/main.go
	@echo "=> Generate transactions dump"
	@set -x \
		&& cd cmd/ \
		&& go run ./gen -out ../$@

# Generate both block dumps used for tests.
dumps: ../$(BUILD_DIR)/single.acc ../$(BUILD_DIR)/dump.acc

# Generate `single.acc` for single-node network
dump.single: ../$(BUILD_DIR)/single.acc

../$(BUILD_DIR)/single.acc: deps config cmd/dump/main.go cmd/dump/chain.go
	@echo "=> Generate block dump for the single node network"
	@set -x \
		&& cd cmd/ \
		&& go run ./dump -single -out $@

# Generate `dump.acc` for the 4-node network
dump: ../$(BUILD_DIR)/dump.acc

../$(BUILD_DIR)/dump.acc: deps config cmd/dump/main.go cmd/dump/chain.go
	@echo "=> Generate block dump for the 4-node network"
	@set -x \
		&& cd cmd/ \
		&& go run ./dump -out $@

# Generate configurations for single-node and four-nodes networks from templates
config: deps
	@echo "=> Generate configurations for single-node and four-nodes networks from templates"
	@set -x \
		&& cd ./cmd \
		&& go run ./config/ --go-template go.protocol.template.yml --go-db leveldb --sharp-template sharp.protocol.template.yml --sharp-db LevelDBStore


# Generate transactions, dump and nodes configurations for four-nodes network
prepare: stop gen dump

# Generate transactions, dump and nodes configurations fore single-node network
prepare.single: stop gen dump.single

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
start.GoSingle10wrk: prepare.single
	.make/runner.sh -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "GoSingle" -m wrk -w 10 -z 5m -t 30s -a node:20331
	make stop

start.GoSingle30wrk: prepare.single
	.make/runner.sh -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "GoSingle" -m wrk -w 30 -z 5m -t 30s -a node:20331
	make stop

start.GoSingle100wrk: prepare.single
	.make/runner.sh -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "GoSingle" -m wrk -w 100 -z 5m -t 30s -a node:20331
	make stop

#	## Rate:
start.GoSingle25rate: prepare.single
	.make/runner.sh -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "GoSingle" -m rate -q 25 -z 5m -t 30s -a node:20331
	make stop

start.GoSingle50rate: prepare.single
	.make/runner.sh -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "GoSingle" -m rate -q 50 -z 5m -t 30s -a node:20331
	make stop

start.GoSingle60rate: prepare.single
	.make/runner.sh -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "GoSingle" -m rate -q 60 -z 5m -t 30s -a node:20331
	make stop

start.GoSingle300rate: prepare.single
	.make/runner.sh -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "GoSingle" -m rate -q 300 -z 5m -t 30s -a node:20331
	make stop

start.GoSingle1000rate: prepare.single
	.make/runner.sh -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "GoSingle" -m rate -q 1000 -z 5m -t 30s -a node:20331
	make stop

## Go x 4 + GoRPC:
#	## Workers:
start.GoFourNodes10wrk: prepare
	.make/runner.sh -f $(DC_GO_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Go4x1" -m wrk -w 10 -z 5m -t 30s -a go-node:20331
	make stop

start.GoFourNodes30wrk: prepare
	.make/runner.sh -f $(DC_GO_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Go4x1" -m wrk -w 30 -z 5m -t 30s -a go-node:20331
	make stop

start.GoFourNodes100wrk: prepare
	.make/runner.sh -f $(DC_GO_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Go4x1" -m wrk -w 100 -z 5m -t 30s -a go-node:20331
	make stop

#	## Rate:
start.GoFourNodes25rate: prepare
	.make/runner.sh -f $(DC_GO_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Go4x1" -m rate -q 25 -z 5m -t 30s -a go-node:20331
	make stop

start.GoFourNodes50rate: prepare
	.make/runner.sh -f $(DC_GO_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Go4x1" -m rate -q 50 -z 5m -t 30s -a go-node:20331
	make stop

start.GoFourNodes60rate: prepare
	.make/runner.sh -f $(DC_GO_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Go4x1" -m rate -q 60 -z 5m -t 30s -a go-node:20331
	make stop

start.GoFourNodes300rate: prepare
	.make/runner.sh -f $(DC_GO_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Go4x1" -m rate -q 300 -z 5m -t 30s -a go-node:20331
	make stop

start.GoFourNodes1000rate: prepare
	.make/runner.sh -f $(DC_GO_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Go4x1" -m rate -q 1000 -z 5m -t 30s -a go-node:20331
	make stop

## SharpSingle:
#	## Workers:
start.SharpSingle10wrk: prepare.single
	.make/runner.sh -f $(DC_SHARP_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "SharpSingle" -m wrk -w 10 -z 5m -t 30s -a node:20331
	make stop

start.SharpSingle30wrk: prepare.single
	.make/runner.sh -f $(DC_SHARP_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "SharpSingle" -m wrk -w 30 -z 5m -t 30s -a node:20331
	make stop

start.SharpSingle100wrk: prepare.single
	.make/runner.sh -f $(DC_SHARP_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "SharpSingle" -m wrk -w 100 -z 5m -t 30s -a node:20331
	make stop

#	## Rate:
start.SharpSingle25rate: prepare.single
	.make/runner.sh -f $(DC_SHARP_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "SharpSingle" -m rate -q 25 -z 5m -t 30s -a node:20331
	make stop

start.SharpSingle50rate: prepare.single
	.make/runner.sh -f $(DC_SHARP_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "SharpSingle" -m rate -q 50 -z 5m -t 30s -a node:20331
	make stop

start.SharpSingle60rate: prepare.single
	.make/runner.sh -f $(DC_SHARP_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "SharpSingle" -m rate -q 60 -z 5m -t 30s -a node:20331
	make stop

start.SharpSingle300rate: prepare.single
	.make/runner.sh -f $(DC_SHARP_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "SharpSingle" -m rate -q 300 -z 5m -t 30s -a node:20331
	make stop

start.SharpSingle1000rate: prepare.single
	.make/runner.sh -f $(DC_SHARP_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "SharpSingle" -m rate -q 1000 -z 5m -t 30s -a node:20331
	make stop

## Sharp x 4 + SharpRPC:
#	## Workers:
start.SharpFourNodes10wrk: prepare
	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -i /dump.txs -d "Sharp4x_SharpRPC" -m wrk -w 10 -z 5m -t 30s -a sharp-node:20331
	make stop

start.SharpFourNodes30wrk: prepare
	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -i /dump.txs -d "Sharp4x_SharpRPC" -m wrk -w 30 -z 5m -t 30s -a sharp-node:20331
	make stop

start.SharpFourNodes100wrk: prepare
	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -i /dump.txs -d "Sharp4x_SharpRPC" -m wrk -w 100 -z 5m -t 30s -a sharp-node:20331
	make stop

#	## Rate:
start.SharpFourNodes25rate: prepare
	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -i /dump.txs -d "Sharp4x_SharpRPC" -m rate -q 25 -z 5m -t 30s -a sharp-node:20331
	make stop

start.SharpFourNodes50rate: prepare
	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -i /dump.txs -d "Sharp4x_SharpRPC" -m rate -q 50 -z 5m -t 30s -a sharp-node:20331
	make stop

start.SharpFourNodes60rate: prepare
	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -i /dump.txs -d "Sharp4x_SharpRPC" -m rate -q 60 -z 5m -t 30s -a sharp-node:20331
	make stop

start.SharpFourNodes300rate: prepare
	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -i /dump.txs -d "Sharp4x_SharpRPC" -m rate -q 300 -z 5m -t 30s -a sharp-node:20331
	make stop

start.SharpFourNodes1000rate: prepare
	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -i /dump.txs -d "Sharp4x_SharpRPC" -m rate -q 1000 -z 5m -t 30s -a sharp-node:20331
	make stop

## Sharp x 4 + GoRPC:
#	## Workers:
start.SharpFourNodesGoRPC10wrk: prepare
	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Sharp4x_GoRPC" -m wrk -w 10 -z 5m -t 30s -a go-node:20331
	make stop

start.SharpFourNodesGoRPC30wrk: prepare
	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Sharp4x_GoRPC" -m wrk -w 30 -z 5m -t 30s -a go-node:20331
	make stop

start.SharpFourNodesGoRPC100wrk: prepare
	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Sharp4x_GoRPC" -m wrk -w 100 -z 5m -t 30s -a go-node:20331
	make stop

#	## Rate:
start.SharpFourNodesGoRPC25rate: prepare
	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Sharp4x_GoRPC" -m rate -q 25 -z 5m -t 30s -a go-node:20331
	make stop

start.SharpFourNodesGoRPC50rate: prepare
	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Sharp4x_GoRPC" -m rate -q 50 -z 5m -t 30s -a go-node:20331
	make stop

start.SharpFourNodesGoRPC60rate: prepare
	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Sharp4x_GoRPC" -m rate -q 60 -z 5m -t 30s -a go-node:20331
	make stop

start.SharpFourNodesGoRPC300rate: prepare
	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Sharp4x_GoRPC" -m rate -q 300 -z 5m -t 30s -a go-node:20331
	make stop

start.SharpFourNodesGoRPC1000rate: prepare
	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Sharp4x_GoRPC" -m rate -q 1000 -z 5m -t 30s -a go-node:20331
	make stop
