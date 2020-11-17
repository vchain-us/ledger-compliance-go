export GO111MODULE=on

SHELL=/bin/bash -o pipefail

.PHONY: build/codegen
build/codegen:
	protoc -I schema/ schema/lc.proto  \
	-I${GOPATH}/pkg/mod \
	-I${GOPATH}/pkg/mod/github.com/codenotary/immudb@v0.8.1-0.20201118111522-48be275a4f8f/pkg/api/schema \
	-I${GOPATH}/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.14.4/third_party/googleapis \
	-I${GOPATH}/pkg/mod/github.com/dgraph-io/badger/v2@v2.0.0-20200408100755-2e708d968e94 \
	-I${GOPATH}/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.14.4 \
	--go_out=schema --go-grpc_out=schema \
    --go_opt=paths=source_relative \
    --go-grpc_opt=paths=source_relative
