# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: ggcl android ios ggcl-cross swarm evm all test clean
.PHONY: ggcl-linux ggcl-linux-386 ggcl-linux-amd64 ggcl-linux-mips64 ggcl-linux-mips64le
.PHONY: ggcl-linux-arm ggcl-linux-arm-5 ggcl-linux-arm-6 ggcl-linux-arm-7 ggcl-linux-arm64
.PHONY: ggcl-darwin ggcl-darwin-386 ggcl-darwin-amd64
.PHONY: ggcl-windows ggcl-windows-386 ggcl-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

ggcl:
	build/env.sh go run build/ci.go install ./cmd/ggcl
	@echo "Done building."
	@echo "Run \"$(GOBIN)/ggcl\" to launch ggcl."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/ggcl.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Ggcl.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

lint: ## Run linters.
	build/env.sh go run build/ci.go lint

clean:
	./build/clean_go_build_cache.sh
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

swarm-devtools:
	env GOBIN= go install ./cmd/swarm/mimegen

# Cross Compilation Targets (xgo)

ggcl-cross: ggcl-linux ggcl-darwin ggcl-windows ggcl-android ggcl-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-*

ggcl-linux: ggcl-linux-386 ggcl-linux-amd64 ggcl-linux-arm ggcl-linux-mips64 ggcl-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-linux-*

ggcl-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/ggcl
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-linux-* | grep 386

ggcl-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/ggcl
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-linux-* | grep amd64

ggcl-linux-arm: ggcl-linux-arm-5 ggcl-linux-arm-6 ggcl-linux-arm-7 ggcl-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-linux-* | grep arm

ggcl-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/ggcl
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-linux-* | grep arm-5

ggcl-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/ggcl
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-linux-* | grep arm-6

ggcl-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/ggcl
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-linux-* | grep arm-7

ggcl-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/ggcl
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-linux-* | grep arm64

ggcl-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/ggcl
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-linux-* | grep mips

ggcl-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/ggcl
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-linux-* | grep mipsle

ggcl-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/ggcl
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-linux-* | grep mips64

ggcl-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/ggcl
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-linux-* | grep mips64le

ggcl-darwin: ggcl-darwin-386 ggcl-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-darwin-*

ggcl-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/ggcl
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-darwin-* | grep 386

ggcl-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/ggcl
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-darwin-* | grep amd64

ggcl-windows: ggcl-windows-386 ggcl-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-windows-*

ggcl-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/ggcl
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-windows-* | grep 386

ggcl-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/ggcl
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ggcl-windows-* | grep amd64
