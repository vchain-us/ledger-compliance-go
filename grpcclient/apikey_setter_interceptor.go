package grpcclient

import (
	"context"
	"google.golang.org/grpc"
)

func (c *LcClient) ApiKeySetterInterceptor() func(context.Context, string, interface{}, interface{}, *grpc.ClientConn, grpc.UnaryInvoker, ...grpc.CallOption) error {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		opts = append(opts, grpc.PerRPCCredentials(ApiKeyAuth{
			ApiKey: c.ApiKey,
		}))
		return invoker(ctx, method, req, reply, cc, opts...)
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
