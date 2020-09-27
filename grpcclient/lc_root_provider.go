package grpcclient

import (
	"context"
	immuschema "github.com/codenotary/immudb/pkg/api/schema"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/vchain-us/ledger-compliance-go/schema"
	"google.golang.org/grpc"
)

type LcRootProvider struct {
	client schema.LcServiceClient
}

func NewLcRootProvider(client schema.LcServiceClient) *LcRootProvider {
	return &LcRootProvider{client}
}

func (r LcRootProvider) CurrentRoot(ctx context.Context) (*immuschema.Root, error) {
	var metadata runtime.ServerMetadata
	var protoReq empty.Empty
	return r.client.CurrentRoot(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
}
