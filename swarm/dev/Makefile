.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/gclchaineum/go-gclchaineum/cmd/swarm
	go build -o bin/ggcl     github.com/gclchaineum/go-gclchaineum/cmd/ggcl
	go build -o bin/bootnode github.com/gclchaineum/go-gclchaineum/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/gclchaineum/go-gclchaineum/swarm/...
