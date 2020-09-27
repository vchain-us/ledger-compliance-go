package grpcclient

import (
	"context"
	"errors"
	"google.golang.org/grpc"
)

func (c *LcClient) ConnectionCheckerInterceptor() func(context.Context, string, interface{}, interface{}, *grpc.ClientConn, grpc.UnaryInvoker, ...grpc.CallOption) error {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if c.ClientConn == nil || c.ServiceClient == nil {
			return errors.New("not connected")
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
