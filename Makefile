export GO111MODULE=on

SHELL=/bin/bash -o pipefail

.PHONY: build/codegen
build/codegen:
	protoc -I schema/ schema/lc.proto  \
	-I${GOPATH}/pkg/mod \
	-I${GOPATH}/pkg/mod/github.com/codenotary/immudb@v0.9.1-0.20210112171619-0097fbe2c012/pkg/api/schema \
	-I${GOPATH}/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.14.4/third_party/googleapis \
	-I${GOPATH}/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.14.4 \
	--go_out=schema --go-grpc_out=schema \
    --go_opt=paths=source_relative \
    --go-grpc_opt=paths=source_relative
