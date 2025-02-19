LOCAL_BIN := $(CURDIR)/bin
EASYP_BIN := $(LOCAL_BIN)/easyp
GOIMPORTS_BIN := $(LOCAL_BIN)/goimports
PROTOC_DOWNLOAD_LINK="https://github.com/protocolbuffers/protobuf/releases"
PROTOC_VERSION=29.2
GO_TEST_ARGS="-race -v ./..."
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

all: generate lint test

.PHONY: lint
lint:
	echo 'Running linter on files...'
	$(GOLANGCI_BIN) run \
	--config=.golangci.yaml \
	--sort-results \
	--max-issues-per-linter=0 \
	--max-same-issues=0

.PHONY: test
test:
	echo 'Running tests...'
	${GO_TEST} "${GO_TEST_ARGS}"

.PHONY: update
update:
	@if [ -n "$(shell git status --untracked-files=no --porcelain)" ]; then \
		echo 'You have some changes. Please commit, checkout or stash them.'; \
		exit 1; \
	fi
	@current_branch=$$(git branch --show-current); \
	echo "Current branch: $$current_branch"; \
	git checkout main; \
	git pull; \
	for branch in $$(git branch | sed 's/^[* ]*//'); do \
		git checkout $$branch; \
		if ! git rev-parse --symbolic-full-name @{u} >/dev/null 2>&1; then \
			branch_exists=$$(git ls-remote --heads origin $$branch); \
			if [ -n "$$branch_exists" ]; then \
				echo "Upstream exists for $$branch. Setting upstream to origin/$$branch."; \
				git branch --set-upstream-to=origin/$$branch; \
			else \
				echo "Upstream not found for $$branch. Pushing and setting upstream to origin/$$branch."; \
				git push --set-upstream origin $$branch; \
				git branch --set-upstream-to=origin/$$branch; \
			fi; \
		fi; \
		git pull --rebase; \
		git push -f; \
		git rebase main; \
		git push -f; \
	done; \
	git checkout $$current_branch; \
	echo 'Successfully updated'

.install-protoc:
	$(INSTALL_CMD)

build:
	go build -o ./bin/library ./cmd/library/

bin-deps: .bin-deps

.bin-deps: export GOBIN := $(LOCAL_BIN)
.bin-deps: .create-bin .install-protoc
	go install github.com/easyp-tech/easyp/cmd/easyp@latest && \
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

build:
	go build -o ./bin/library ./cmd/library/
