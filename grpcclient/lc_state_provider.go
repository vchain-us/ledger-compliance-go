/*
Copyright 2019-2023 vChain, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grpcclient

import (
	"context"

	immuschema "github.com/codenotary/immudb/pkg/api/schema"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/vchain-us/ledger-compliance-go/schema"
	"google.golang.org/grpc"
)

type LcStateProvider struct {
	client schema.LcServiceClient
}

func NewLcStateProvider(client schema.LcServiceClient) *LcStateProvider {
	return &LcStateProvider{client}
}

func (r LcStateProvider) CurrentState(ctx context.Context) (*immuschema.ImmutableState, error) {
	var metadata runtime.ServerMetadata
	var protoReq empty.Empty
	return r.client.CurrentState(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
}
