LOCAL_BIN := $(CURDIR)/bin
EASYP_BIN := $(LOCAL_BIN)/easyp
GOIMPORTS_BIN := $(LOCAL_BIN)/goimports
PROTOC_DOWNLOAD_LINK="https://github.com/protocolbuffers/protobuf/releases"
PROTOC_VERSION=29.2
UNAME_S := $(shell uname -s)
UNAME_P := $(shell uname -p)

ARCH :=

ifeq ($(UNAME_S),Linux)
    INSTALL_CMD = sudo apt install -y protobuf-compiler
    ARCH = linux-x86_64
endif

ifeq ($(UNAME_S),Darwin)
    ifeq ($(UNAME_P),arm)
        INSTALL_CMD = brew install protobuf
        ARCH = osx-universal_binary
    else
        INSTALL_CMD = sudo apt install -y protobuf-compiler
        ARCH = linux-x86_64
    endif
endif

.install-protoc:
	$(INSTALL_CMD)

build:
	go build -o ./bin/library ./cmd/library/

bin-deps: .bin-deps

.bin-deps: export GOBIN := $(LOCAL_BIN)
.bin-deps: .create-bin .install-protoc
	go install github.com/easyp-tech/easyp/cmd/easyp@0.7.8 && \
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.18.1 && \
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.18.1 && \
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.1 && \
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0 && \
	go install golang.org/x/tools/cmd/goimports@v0.19.0 && \
	go install github.com/envoyproxy/protoc-gen-validate@v1.2.1

.create-bin:
	rm -rf ./bin
	mkdir -p ./bin

generate: bin-deps .generate
fast-generate: .generate

.generate:
	$(info Generating code...)

	rm -rf ./generated
	mkdir ./generated

	rm -rf ./docs/spec
	mkdir -p ./docs/spec

	rm -rf ~/.easyp/

	(PATH="$(PATH):$(LOCAL_BIN)" && $(EASYP_BIN) mod download && $(EASYP_BIN) generate)

	$(GOIMPORTS_BIN) -w .
