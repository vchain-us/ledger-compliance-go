module github.com/vchain-us/ledger-compliance-go

go 1.15

require (
	github.com/codenotary/immudb v0.8.0
	github.com/golang/protobuf v1.4.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/grpc-ecosystem/grpc-gateway v1.14.4
	google.golang.org/grpc v1.29.1
	google.golang.org/protobuf v1.21.0
)

replace github.com/codenotary/immudb v0.8.0 => github.com/codenotary/immudb v0.0.0-20201015224644-7387902c29a9
