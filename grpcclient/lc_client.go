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
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/codenotary/immudb/pkg/client/state"
	"github.com/codenotary/immudb/pkg/stream"

	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/vchain-us/ledger-compliance-go/schema"
	"google.golang.org/grpc/keepalive"

	"github.com/codenotary/immudb/pkg/client/timestamp"
	"github.com/codenotary/immudb/pkg/logger"
	"google.golang.org/grpc"

	immuschema "github.com/codenotary/immudb/pkg/api/schema"
	immuclient "github.com/codenotary/immudb/pkg/client"
)

// LcClientIf ...
type LcClientIf interface {
	Set(ctx context.Context, key []byte, value []byte) (*immuschema.TxHeader, error)
	VerifiedSet(ctx context.Context, key []byte, value []byte) (*immuschema.TxHeader, error)

	Get(ctx context.Context, key []byte) (*immuschema.Entry, error)
	GetAt(ctx context.Context, key []byte, tx uint64) (*immuschema.Entry, error)
	VerifiedGet(ctx context.Context, key []byte) (*immuschema.Entry, error)
	VerifiedGetSince(ctx context.Context, key []byte, tx uint64) (*immuschema.Entry, error)
	VerifiedGetAt(ctx context.Context, key []byte, tx uint64) (*immuschema.Entry, error)

	GetAll(ctx context.Context, in *immuschema.KeyListRequest) (*immuschema.Entries, error)
	SetAll(ctx context.Context, kvList *immuschema.SetRequest) (*immuschema.TxHeader, error)
	SetMulti(ctx context.Context, req *schema.SetMultiRequest) (*schema.SetMultiResponse, error)
	VCNSetArtifacts(ctx context.Context, req *schema.VCNArtifactsRequest) (*schema.VCNArtifactsResponse, error)

	ExecAll(ctx context.Context, in *immuschema.ExecAllRequest) (*immuschema.TxHeader, error)

	Scan(ctx context.Context, req *immuschema.ScanRequest) (*immuschema.Entries, error)
	ZScan(ctx context.Context, req *immuschema.ZScanRequest) (*immuschema.ZEntries, error)

	History(ctx context.Context, req *immuschema.HistoryRequest) (*immuschema.Entries, error)

	ZScanExt(ctx context.Context, options *immuschema.ZScanRequest) (*schema.ZItemExtList, error)
	HistoryExt(ctx context.Context, options *immuschema.HistoryRequest) (sl *schema.ItemExtList, err error)
	Feats(ctx context.Context) (*schema.Features, error)

	Health(ctx context.Context) (*immuschema.HealthResponse, error)
	CurrentState(ctx context.Context) (*immuschema.ImmutableState, error)

	VerifiedGetExt(ctx context.Context, key []byte) (*schema.VerifiableItemExt, error)
	VerifiedGetExtSince(ctx context.Context, key []byte, tx uint64) (*schema.VerifiableItemExt, error)
	VerifiedGetExtAt(ctx context.Context, key []byte, tx uint64) (itemExt *schema.VerifiableItemExt, err error)
	VerifiedGetExtAtMulti(ctx context.Context, keys [][]byte, txs []uint64) (itemsExt []*schema.VerifiableItemExt, errs []string, err error)

	SetFile(ctx context.Context, key []byte, filePath string) (*immuschema.TxHeader, error)
	GetFile(ctx context.Context, key []byte, filePath string) (*immuschema.Entry, error)

	Connect() (err error)
	IsConnected() bool

	ConsistencyCheck(ctx context.Context) error

	GetApiKey() string
	SetApiKey(string)

	// streams
	StreamSet(ctx context.Context, kvs []*stream.KeyValue) (*immuschema.TxHeader, error)
	StreamGet(ctx context.Context, k *immuschema.KeyRequest) (*immuschema.Entry, error)
	StreamVerifiedSet(ctx context.Context, kvs []*stream.KeyValue) (*immuschema.TxHeader, error)
	StreamVerifiedGet(ctx context.Context, req *immuschema.VerifiableGetRequest) (*immuschema.Entry, error)
	StreamScan(ctx context.Context, req *immuschema.ScanRequest) (*immuschema.Entries, error)
	StreamZScan(ctx context.Context, req *immuschema.ZScanRequest) (*immuschema.ZEntries, error)
	StreamHistory(ctx context.Context, req *immuschema.HistoryRequest) (*immuschema.Entries, error)
	StreamExecAll(ctx context.Context, req *stream.ExecAllRequest) (*immuschema.TxHeader, error)

	SetServerSigningPubKey(*ecdsa.PublicKey)
}

type LcClient struct {
	Dir                  string
	Host                 string
	Port                 int
	ApiKey               string
	ApiKeyHash           string
	MetadataPairs        []string
	DialOptions          []grpc.DialOption
	Logger               logger.Logger
	ClientConn           *grpc.ClientConn
	ServiceClient        schema.LcServiceClient
	StateService         state.StateService
	TimestampService     immuclient.TimestampService
	StreamChunkSize      int
	StreamServiceFactory stream.ServiceFactory
	serverSigningPubKey  *ecdsa.PublicKey
	sync.RWMutex
	akm sync.RWMutex
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
		// TODO OGG: StreamChunkSize needs to be made configurable
		StreamChunkSize:      immuclient.DefaultOptions().StreamChunkSize,
		StreamServiceFactory: stream.NewStreamServiceFactory(immuclient.DefaultOptions().StreamChunkSize),
	}

	cli.DialOptions = []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                20 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxCallRecvMsgSize)),
	}

	for _, setter := range setters {
		setter(cli)
	}

	var uic []grpc.UnaryClientInterceptor
	if cli.serverSigningPubKey != nil {
		uic = append(uic, cli.SignatureVerifierInterceptor)
	}
	uic = append(uic, cli.ConnectionCheckerInterceptor(), cli.ApiKeySetterInterceptor())

	var sic []grpc.StreamClientInterceptor
	sic = append(sic, cli.ApiKeySetterInterceptorStream())

	cli.DialOptions = append(
		cli.DialOptions,
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(uic...)),
		grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(sic...)))

	return cli
}

func (c *LcClient) Connect() (err error) {
	if c.GetApiKey() != "" {
		apiKeyPieces := strings.Split(c.GetApiKey(), ApiKeySeparator)
		if len(apiKeyPieces) >= 2 {
			signerID := strings.Join(apiKeyPieces[:len(apiKeyPieces)-1], ApiKeySeparator)
			hashed := sha256.Sum256([]byte(apiKeyPieces[len(apiKeyPieces)-1]))
			c.ApiKeyHash = signerID + ApiKeySeparator + base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hashed[:])
		}
	}

	c.ClientConn, err = grpc.Dial(fmt.Sprintf("%s:%d", c.Host, c.Port), c.DialOptions...)
	if err != nil {
		return err
	}

	c.ServiceClient = schema.NewLcServiceClient(c.ClientConn)

	uuidPrv := NewLcUUIDProvider(c.ServiceClient)
	stateProvider := NewLcStateProvider(c.ServiceClient)

	c.StateService, err = NewLcStateService(NewLcFileCache(c.Dir), c.Logger, stateProvider, uuidPrv)
	if err != nil {
		return err
	}

	return nil
}

func (c *LcClient) IsConnected() bool {
	if c.ClientConn != nil && c.ServiceClient != nil && c.GetApiKey() != "" {
		return true
	}
	return false
}

func (c *LcClient) Disconnect() (err error) {
	c.ServiceClient = nil
	return c.ClientConn.Close()
}

func (c *LcClient) SetServerSigningPubKey(k *ecdsa.PublicKey) {
	c.Lock()
	defer c.Unlock()
	c.serverSigningPubKey = k
}

func (c *LcClient) SetApiKey(apiKey string) {
	c.akm.Lock()
	defer c.akm.Unlock()
	c.ApiKey = apiKey
}

func (c *LcClient) GetApiKey() string {
	c.akm.RLock()
	defer c.akm.RUnlock()
	return c.ApiKey
}
