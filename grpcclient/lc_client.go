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
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/codenotary/immudb/pkg/client/state"
	"os"
	"sync"
	"time"

	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/vchain-us/ledger-compliance-go/schema"
	"google.golang.org/grpc/keepalive"

	"github.com/codenotary/immudb/pkg/client/cache"
	"github.com/codenotary/immudb/pkg/client/timestamp"
	"github.com/codenotary/immudb/pkg/logger"
	"google.golang.org/grpc"

	immuschema "github.com/codenotary/immudb/pkg/api/schema"
	immuclient "github.com/codenotary/immudb/pkg/client"
)

// LcClientIf ...
type LcClientIf interface {
	Set(ctx context.Context, key []byte, value []byte) (*immuschema.TxMetadata, error)
	VerifiedSet(ctx context.Context, key []byte, value []byte) (*immuschema.TxMetadata, error)

	Get(ctx context.Context, key []byte) (*immuschema.Entry, error)
	GetAt(ctx context.Context, key []byte, tx uint64) (*immuschema.Entry, error)
	VerifiedGet(ctx context.Context, key []byte) (*immuschema.Entry, error)
	VerifiedGetSince(ctx context.Context, key []byte, tx uint64) (*immuschema.Entry, error)
	VerifiedGetAt(ctx context.Context, key []byte, tx uint64) (*immuschema.Entry, error)

	GetAll(ctx context.Context, in *immuschema.KeyListRequest) (*immuschema.Entries, error)

	ExecAll(ctx context.Context, in *immuschema.ExecAllRequest) (*immuschema.TxMetadata, error)

	Scan(ctx context.Context, req *immuschema.ScanRequest) (*immuschema.Entries, error)
	ZScan(ctx context.Context, req *immuschema.ZScanRequest) (*immuschema.ZEntries, error)

	History(ctx context.Context, req *immuschema.HistoryRequest) (*immuschema.Entries, error)

	ZScanExt(ctx context.Context, options *immuschema.ZScanRequest) (*schema.ZItemExtList, error)
	HistoryExt(ctx context.Context, options *immuschema.HistoryRequest) (sl *schema.ItemExtList, err error)

	Health(ctx context.Context) (*immuschema.HealthResponse, error)

	VerifiedGetExt(ctx context.Context, key []byte) (*schema.VerifiableItemExt, error)
	VerifiedGetExtSince(ctx context.Context, key []byte, tx uint64) (*schema.VerifiableItemExt, error)
	VerifiedGetExtAt(ctx context.Context, key []byte, tx uint64) (itemExt *schema.VerifiableItemExt, err error)

	Connect() (err error)
}

type LcClient struct {
	Dir                 string
	Host                string
	Port                int
	ApiKey              string
	DialOptions         []grpc.DialOption
	Logger              logger.Logger
	ClientConn          *grpc.ClientConn
	ServiceClient       schema.LcServiceClient
	StateService        state.StateService
	TimestampService    immuclient.TimestampService
	serverSigningPubKey *ecdsa.PublicKey
	sync.RWMutex
}

func NewLcClient(setters ...LcClientOption) *LcClient {
	dt, _ := timestamp.NewDefaultTimestamp()

	// Default Options
	cli := &LcClient{
		Dir:              "",
		Host:             "localhost",
		Port:             3324,
		ApiKey:           "",
		Logger:           logger.NewSimpleLogger("immuclient", os.Stderr),
		TimestampService: immuclient.NewTimestampService(dt),
	}

	cli.DialOptions = []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                20 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	for _, setter := range setters {
		setter(cli)
	}

	var uic []grpc.UnaryClientInterceptor

	if cli.serverSigningPubKey != nil {
		uic = append(uic, cli.SignatureVerifierInterceptor)
	}

	uic = append(uic, cli.ConnectionCheckerInterceptor(), cli.ApiKeySetterInterceptor())

	cli.DialOptions = append(cli.DialOptions, grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(uic...)))

	return cli
}

func (c *LcClient) Connect() (err error) {
	if c.ApiKey == "" {
		return errors.New("api key not provided")
	}
	c.ClientConn, err = grpc.Dial(fmt.Sprintf("%s:%d", c.Host, c.Port), c.DialOptions...)
	if err != nil {
		return err
	}

	c.ServiceClient = schema.NewLcServiceClient(c.ClientConn)

	uuidPrv := NewLcUUIDProvider(c.ServiceClient)
	stateProvider := NewLcStateProvider(c.ServiceClient)

	c.StateService, err = state.NewStateService(cache.NewFileCache(c.Dir), c.Logger, stateProvider, uuidPrv)
	if err != nil {
		return err
	}

	return nil
}

func (c *LcClient) Disconnect() (err error) {
	c.ServiceClient = nil
	return c.ClientConn.Close()
}
