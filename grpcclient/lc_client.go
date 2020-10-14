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
	"errors"
	"fmt"
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
	"github.com/codenotary/immudb/pkg/client/rootservice"
)

// ImmuClient ...
type LcClientIf interface {
	Set(ctx context.Context, key []byte, value []byte) (*immuschema.Index, error)
	Get(ctx context.Context, key []byte) (*immuschema.StructuredItem, error)
	SafeSet(ctx context.Context, key []byte, value []byte) (*immuclient.VerifiedIndex, error)
	SafeGet(ctx context.Context, key []byte) (*immuclient.VerifiedItem, error)
	SetBatch(ctx context.Context, in *immuschema.KVList) (*immuschema.Index, error)
	GetBatch(ctx context.Context, in *immuschema.KeyList) (*immuschema.StructuredItemList, error)
	Scan(ctx context.Context, prefix []byte) (*immuschema.StructuredItemList, error)
	History(ctx context.Context, key []byte) (sl *immuschema.StructuredItemList, err error)
	ZAdd(ctx context.Context, set []byte, score float64, key []byte) (*immuschema.Index, error)
	SafeZAdd(ctx context.Context, set []byte, score float64, key []byte) (*immuclient.VerifiedIndex, error)
	ZScan(ctx context.Context, set []byte) (*immuschema.StructuredItemList, error)

	Connect() (err error)
}

type LcClient struct {
	Dir              string
	Host             string
	Port             int
	ApiKey           string
	DialOptions      []grpc.DialOption
	Logger           logger.Logger
	ClientConn       *grpc.ClientConn
	ServiceClient    schema.LcServiceClient
	RootService      rootservice.RootService
	TimestampService immuclient.TimestampService
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
	cli.DialOptions = append(cli.DialOptions, grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
		cli.ConnectionCheckerInterceptor(),
		cli.ApiKeySetterInterceptor())))

	return cli
}

func (c *LcClient) Connect() (err error) {
	if c.ApiKey == "" {
		return errors.New("api key not provided")
	}
	c.ClientConn, err = grpc.Dial(fmt.Sprintf("%s:%d", c.Host, c.Port), c.DialOptions...)
	if err != nil {
		c.Logger.Errorf("fail to dial: %v", err)
		return err
	}

	c.ServiceClient = schema.NewLcServiceClient(c.ClientConn)

	uuidPrv := NewLcUUIDProvider(c.ServiceClient)
	rootPrv := NewLcRootProvider(c.ServiceClient)
	c.RootService, err = rootservice.NewRootService(cache.NewFileCache(c.Dir), c.Logger, rootPrv, uuidPrv)
	if err != nil {
		c.Logger.Errorf("fail to instantiate root service: %v", err)
		return err
	}

	return nil
}

func (c *LcClient) Disconnect() (err error) {
	c.ServiceClient = nil
	return c.ClientConn.Close()
}

func (c *LcClient) NewSKV(key []byte, value []byte) *immuschema.StructuredKeyValue {
	return &immuschema.StructuredKeyValue{
		Key: key,
		Value: &immuschema.Content{
			Timestamp: uint64(c.TimestampService.GetTime().Unix()),
			Payload:   value,
		},
	}
}

func (c *LcClient) verifyAndSetRoot(result *immuschema.Proof, root *immuschema.Root) (bool, error) {
	verified := result.Verify(result.Leaf, *root)
	var err error
	if verified {
		toCache := immuschema.NewRoot()
		toCache.SetIndex(result.Index)
		toCache.SetRoot(result.Root)
		err = c.RootService.SetRoot(toCache, c.ApiKey)
	}
	return verified, err
}

func (c *LcClient) NewSKVList(list *immuschema.KVList) *immuschema.SKVList {
	slist := &immuschema.SKVList{}
	for _, kv := range list.KVs {
		slist.SKVs = append(slist.SKVs, &immuschema.StructuredKeyValue{
			Key: kv.Key,
			Value: &immuschema.Content{
				Timestamp: uint64(c.TimestampService.GetTime().Unix()),
				Payload:   kv.Value,
			},
		})
	}
	return slist
}
