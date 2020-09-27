package grpcclient

import (
	immuclient "github.com/codenotary/immudb/pkg/client"
	"github.com/codenotary/immudb/pkg/client/rootservice"
	"github.com/codenotary/immudb/pkg/logger"
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

func DialOptions(dopts []grpc.DialOption) LcClientOption {
	return func(args *LcClient) {
		args.DialOptions = dopts
	}
}

func Logger(logger logger.Logger) LcClientOption {
	return func(args *LcClient) {
		args.Logger = logger
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

func RootService(rootservice rootservice.RootService) LcClientOption {
	return func(args *LcClient) {
		args.RootService = rootservice
	}
}

func TimestampService(timestampService immuclient.TimestampService) LcClientOption {
	return func(args *LcClient) {
		args.TimestampService = timestampService
	}
}
