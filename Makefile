export GO111MODULE=on

SHELL=/bin/bash -o pipefail

.PHONY: build/codegen
build/codegen:
	protoc -I schema/ schema/lc.proto  \
	-I${GOPATH}/pkg/mod \
	-I${GOPATH}/pkg/mod/github.com/codenotary/immudb@v0.9.2-0.20210324115202-e54bda6e1cc3/pkg/api/schema \
	-I${GOPATH}/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.14.4/third_party/googleapis \
	-I${GOPATH}/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.14.4 \
	--go_out=schema --go-grpc_out=schema \
    --go_opt=paths=source_relative \
    --go-grpc_opt=paths=source_relative
