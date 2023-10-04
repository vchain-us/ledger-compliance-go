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
	"fmt"

	"github.com/codenotary/immudb/pkg/server"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/vchain-us/ledger-compliance-go/schema"
	"google.golang.org/grpc"
)

// ErrNoServerUuid ...
var ErrNoServerUuid = fmt.Errorf(
	"!IMPORTANT WARNING: %s header is not published by the immudb server; "+
		"this client MUST NOT be used to connect to different immudb servers!",
	server.SERVER_UUID_HEADER)

type LcUuidProvider struct {
	client schema.LcServiceClient
}

func NewLcUUIDProvider(client schema.LcServiceClient) *LcUuidProvider {
	return &LcUuidProvider{client}
}

// CurrentUuid issues a Health command to the server, then parses and returns
// the server UUID from the response metadata
func (r LcUuidProvider) CurrentUUID(ctx context.Context) (string, error) {
	var metadata runtime.ServerMetadata
	if _, err := r.client.Health(
		ctx,
		new(empty.Empty),
		grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD),
	); err != nil {
		return "", err
	}
	var serverUuid string
	if len(metadata.HeaderMD.Get(server.SERVER_UUID_HEADER)) > 0 {
		serverUuid = metadata.HeaderMD.Get(server.SERVER_UUID_HEADER)[0]
	}
	if serverUuid == "" {
		return "", ErrNoServerUuid
	}
	return serverUuid, nil
}
