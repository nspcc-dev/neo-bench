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

.PHONY: build push build.node.go build.node.sharp stop start deps gen config

# Build all images
build: build.node.bench build.node.go build.node.sharp

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
	@docker build -t $(HUB)-sharp:$(TAG) -f $(DF_SHARP) $(BUILD_DIR)

# Test local benchmark (go run) with Neo single node
test: single.go deps
	@echo "=> Test Single node"
	@set -x \
		&& cd cmd/ \
		&& go run ./bench -o ../single.log -i ../dump.txs -d "SingleNode" -m rate -q 1000 -z 1m -t 30s -a localhost:20331
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

# Generate `dump.txs` (run it before any benchmarks)
gen: deps
	@echo "=> Generate transactions dump"
	@set -x \
		&& cd cmd/ \
		&& go run ./gen -out ../dump.txs

dump.single: deps config
	@echo "=> Generate block dump for the single node network"
	@set -x \
		&& cd cmd/ \
		&& go run ./dump -single -out ../$(BUILD_DIR)/single.acc

# Generate `dump.acc` for the 4-node network
dump: deps config
	@echo "=> Generate block dump for the 4-node network"
	@set -x \
		&& cd cmd/ \
		&& go run ./dump -out ../$(BUILD_DIR)/dump.acc

# Generate configurations for single-node and four-nodes networks from templates
config: deps
	@echo "=> Generate configurations for single-node and four-nodes networks from templates"
	@set -x \
		&& cd ./cmd \
		&& go run ./config/ --go-template go.protocol.template.yml --sharp-template sharp.protocol.template.yml


# Run benchmark (uncomment needed)
start: stop
## GoSingle:
#	## Workers:
#	.make/runner.sh -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "GoSingle" -m wrk -w 10 -z 5m -t 30s -a node:20331
#	make stop
#	@.make/runner.sh -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "GoSingle" -m wrk -w 30 -z 5m -t 30s -a node:20331
#	@make stop
#	@.make/runner.sh -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "GoSingle" -m wrk -w 100 -z 5m -t 30s -a node:20331
#	@make stop

#	## Rate:
#	.make/runner.sh -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "GoSingle" -m rate -q 25 -z 5m -t 30s -a node:20331
#	make stop
#	.make/runner.sh -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "GoSingle" -m rate -q 50 -z 5m -t 30s -a node:20331
#	make stop
#	.make/runner.sh -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "GoSingle" -m rate -q 60 -z 5m -t 30s -a node:20331
#	make stop
#	.make/runner.sh -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "GoSingle" -m rate -q 300 -z 5m -t 30s -a node:20331
#	make stop
#	.make/runner.sh -f $(DC_GO_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "GoSingle" -m rate -q 1000 -z 5m -t 30s -a node:20331
#	make stop

## Go x 4 + GoRPC:
#	## Workers:
#	.make/runner.sh -f $(DC_GO_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Go4x1" -m wrk -w 10 -z 5m -t 30s -a go-node:20331
#	make stop
#	.make/runner.sh -f $(DC_GO_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Go4x1" -m wrk -w 30 -z 5m -t 30s -a go-node:20331
#	make stop
#	.make/runner.sh -f $(DC_GO_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Go4x1" -m wrk -w 100 -z 5m -t 30s -a go-node:20331
#	make stop

#	## Rate:
#	.make/runner.sh -f $(DC_GO_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Go4x1" -m rate -q 25 -z 5m -t 30s -a go-node:20331
#	make stop
#	.make/runner.sh -f $(DC_GO_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Go4x1" -m rate -q 50 -z 5m -t 30s -a go-node:20331
#	make stop
#	.make/runner.sh -f $(DC_GO_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Go4x1" -m rate -q 60 -z 5m -t 30s -a go-node:20331
#	make stop
#	.make/runner.sh -f $(DC_GO_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Go4x1" -m rate -q 300 -z 5m -t 30s -a go-node:20331
#	make stop
#	.make/runner.sh -f $(DC_GO_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Go4x1" -m rate -q 1000 -z 5m -t 30s -a go-node:20331
#	make stop

## SharpSingle:
#	## Workers:
#	.make/runner.sh -f $(DC_SHARP_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "SharpSingle" -m wrk -w 10 -z 5m -t 30s -a node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "SharpSingle" -m wrk -w 30 -z 5m -t 30s -a node:20331
#	make stop

#	## Rate:
#	.make/runner.sh -f $(DC_SHARP_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "SharpSingle" -m rate -q 25 -z 5m -t 30s -a node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "SharpSingle" -m rate -q 50 -z 5m -t 30s -a node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "SharpSingle" -m rate -q 60 -z 5m -t 30s -a node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "SharpSingle" -m rate -q 300 -z 5m -t 30s -a node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR_SINGLE) -f $(DC_SINGLE) -i /dump.txs -d "SharpSingle" -m rate -q 1000 -z 5m -t 30s -a node:20331
#	make stop

## Sharp x 4 + SharpRPC:
#	## Workers:
#	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -i /dump.txs -d "Sharp4x_SharpRPC" -m wrk -w 10 -z 5m -t 30s -a sharp-node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -i /dump.txs -d "Sharp4x_SharpRPC" -m wrk -w 30 -z 5m -t 30s -a sharp-node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -i /dump.txs -d "Sharp4x_SharpRPC" -m wrk -w 50 -z 5m -t 30s -a sharp-node:20331
#	make stop

#	## Rate:
#	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -i /dump.txs -d "Sharp4x_SharpRPC" -m rate -q 25 -z 5m -t 30s -a sharp-node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -i /dump.txs -d "Sharp4x_SharpRPC" -m rate -q 50 -z 5m -t 30s -a sharp-node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -i /dump.txs -d "Sharp4x_SharpRPC" -m rate -q 60 -z 5m -t 30s -a sharp-node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -i /dump.txs -d "Sharp4x_SharpRPC" -m rate -q 300 -z 5m -t 30s -a sharp-node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_SHARP_RPC) -i /dump.txs -d "Sharp4x_SharpRPC" -m rate -q 1000 -z 5m -t 30s -a sharp-node:20331
#	make stop

## Sharp x 4 + GoRPC:
#	## Workers:
#	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Sharp4x_GoRPC" -m wrk -w 10 -z 5m -t 30s -a go-node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Sharp4x_GoRPC" -m wrk -w 30 -z 5m -t 30s -a go-node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Sharp4x_GoRPC" -m wrk -w 100 -z 5m -t 30s -a go-node:20331
#	make stop

#	## Rate:
#	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Sharp4x_GoRPC" -m rate -q 25 -z 5m -t 30s -a go-node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Sharp4x_GoRPC" -m rate -q 50 -z 5m -t 30s -a go-node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Sharp4x_GoRPC" -m rate -q 60 -z 5m -t 30s -a go-node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Sharp4x_GoRPC" -m rate -q 300 -z 5m -t 30s -a go-node:20331
#	make stop
#	.make/runner.sh -f $(DC_SHARP_IR) -f $(DC_GO_RPC) -i /dump.txs -d "Sharp4x_GoRPC" -m rate -q 1000 -z 5m -t 30s -a go-node:20331
#	make stop