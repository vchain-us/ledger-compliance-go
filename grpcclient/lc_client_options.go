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
	"crypto/ecdsa"

	immuclient "github.com/codenotary/immudb/pkg/client"
	"github.com/codenotary/immudb/pkg/client/state"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/vchain-us/ledger-compliance-go/schema"
	"google.golang.org/grpc"
)

type LcClientOption func(*LcClient)

func Dir(c string) LcClientOption {
	return func(args *LcClient) {
		args.Dir = c
	}
}

func Host(c string) LcClientOption {
	return func(args *LcClient) {
		args.Host = c
	}
}

func Port(port int) LcClientOption {
	return func(args *LcClient) {
		args.Port = port
	}
}

func ApiKey(apiKey string) LcClientOption {
	return func(args *LcClient) {
		args.ApiKey = apiKey
	}
}

func MetadataPairs(metadataPairs []string) LcClientOption {
	return func(args *LcClient) {
		args.MetadataPairs = metadataPairs
	}
}

func DialOptions(dopts []grpc.DialOption) LcClientOption {
	return func(args *LcClient) {
		args.DialOptions = dopts
	}
}

func RetryOptions(opts ...grpc_retry.CallOption) LcClientOption {
	return func(args *LcClient) {
		args.RetryOptions = opts
	}
}

func ClientConn(clientConn *grpc.ClientConn) LcClientOption {
	return func(args *LcClient) {
		args.ClientConn = clientConn
	}
}

func ServiceClient(serviceClient schema.LcServiceClient) LcClientOption {
	return func(args *LcClient) {
		args.ServiceClient = serviceClient
	}
}

func StateService(rootservice state.StateService) LcClientOption {
	return func(args *LcClient) {
		args.StateService = rootservice
	}
}

func TimestampService(timestampService immuclient.TimestampService) LcClientOption {
	return func(args *LcClient) {
		args.TimestampService = timestampService
	}
}

func ServerSigningPubKey(serverSigningPubKey *ecdsa.PublicKey) LcClientOption {
	return func(args *LcClient) {
		args.serverSigningPubKey = serverSigningPubKey
	}
}

func Logger(logger LcLogger) LcClientOption {
	return func(args *LcClient) {
		args.logger = logger
	}
}
