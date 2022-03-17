/*
Copyright 2019-2020 vChain, Inc.

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

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func (c *LcClient) ApiKeySetterInterceptor() func(context.Context, string, interface{}, interface{}, *grpc.ClientConn, grpc.UnaryInvoker, ...grpc.CallOption) error {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, opts = c.prepareGRPcReq(ctx, opts...)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// ApiKeySetterInterceptorStream ...
func (c *LcClient) ApiKeySetterInterceptorStream() func(context.Context, *grpc.StreamDesc, *grpc.ClientConn, string, grpc.Streamer, ...grpc.CallOption) (grpc.ClientStream, error) {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx, opts = c.prepareGRPcReq(ctx, opts...)
		return streamer(ctx, desc, cc, method, opts...)
	}
}

type ApiKeyAuth struct {
	ApiKey string
}

func (t ApiKeyAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"lc-api-key": t.ApiKey,
	}, nil
}

func (ApiKeyAuth) RequireTransportSecurity() bool {
	return false
}

func (c *LcClient) prepareGRPcReq(ctx context.Context, opts ...grpc.CallOption) (context.Context, []grpc.CallOption) {
	var akOpt grpc.CallOption = nil
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		authHeader, ok := md["lc-api-key"]
		if ok && len(authHeader) > 0 && authHeader[0] != "" {
			akOpt = grpc.PerRPCCredentials(ApiKeyAuth{
				ApiKey: authHeader[0],
			})
		}
	}
	if akOpt == nil && c.ApiKey != "" {
		akOpt = grpc.PerRPCCredentials(ApiKeyAuth{
			ApiKey: c.ApiKey,
		})
	}
	if len(c.MetadataPairs) > 0 {
		ctx = metadata.AppendToOutgoingContext(ctx, c.MetadataPairs...)
	}
	if akOpt != nil {
		opts = append(opts, akOpt)
	}
	return ctx, opts
}
