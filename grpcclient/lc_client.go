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

	"google.golang.org/grpc/codes"	

	"github.com/codenotary/immudb/pkg/client/cache"
	"github.com/codenotary/immudb/pkg/client/state"
	"github.com/codenotary/immudb/pkg/stream"

	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/vchain-us/ledger-compliance-go/schema"
	"google.golang.org/grpc/keepalive"

	immuschema "github.com/codenotary/immudb/pkg/api/schema"
	immuclient "github.com/codenotary/immudb/pkg/client"
	"github.com/codenotary/immudb/pkg/client/timestamp"
	"github.com/codenotary/immudb/pkg/logger"
	"google.golang.org/grpc"
)

// LcClientIf ...
type LcClientIf interface {
	VCNSetArtifacts(ctx context.Context, req *schema.VCNArtifactsRequest) (*schema.VCNArtifactsResponse, error)
	VCNSearchArtifacts(ctx context.Context, req *schema.VCNSearchRequest) (*schema.EntryList, error)
	VCNGetArtifacts(ctx context.Context, req *schema.VCNArtifactsGetRequest) (*schema.EntryList, error)
	VCNGetClientSignature(ctx context.Context, in *schema.VCNGetClientSignatureRequest, opts ...grpc.CallOption) (*schema.VCNGetClientSignatureResponse, error)
	Health(ctx context.Context) (*immuschema.HealthResponse, error)
	CurrentState(ctx context.Context) (*immuschema.ImmutableState, error)
	Feats(ctx context.Context) (*schema.Features, error)
	SetFile(ctx context.Context, key []byte, filePath string) (*immuschema.TxHeader, error)
	GetFile(ctx context.Context, key []byte, filePath string) (*immuschema.Entry, error)
	Connect() (err error)
	Disconnect() (err error)
	SetServerSigningPubKey(*ecdsa.PublicKey)
	GetApiKey() string
	GetLogger() logger.Logger
	VCNLabelsSet(ctx context.Context, req []*schema.LabelsSetRequest, opts ...grpc.CallOption) (*schema.VCNLabelsSetResponse, error)
	VCNLabelsUpdate(ctx context.Context, hash string, ops []*schema.VCNLabelsUpdateRequest_VCNLabelsOp, opts ...grpc.CallOption) (*schema.VCNLabelsUpdateResponse, error)
	VCNLabelsGet(ctx context.Context, req []*schema.LabelsGetRequest, opts ...grpc.CallOption) (*schema.VCNLabelsGetResponse, error)
	VCNGetAttachment(ctx context.Context, signerID, artifactHash, attachHash string) (*schema.VCNGetAttachmentResponse, error)

	// Deprecated: use LcClient.VCNGetArtifacts instead
	Set(ctx context.Context, key []byte, value []byte) (*immuschema.TxHeader, error)
	// Deprecated: use LcClient.VCNSetArtifacts instead
	VerifiedSet(ctx context.Context, key []byte, value []byte) (*immuschema.TxHeader, error)
	// Deprecated: use LcClient.VCNGetArtifacts instead
	Get(ctx context.Context, key []byte) (*immuschema.Entry, error)
	// Deprecated: use LcClient.VCNGetArtifacts instead
	GetAt(ctx context.Context, key []byte, tx uint64) (*immuschema.Entry, error)
	// Deprecated: use LcClient.VCNGetArtifacts instead
	VerifiedGet(ctx context.Context, key []byte) (*immuschema.Entry, error)
	// Deprecated: use LcClient.VCNGetArtifacts instead
	VerifiedGetSince(ctx context.Context, key []byte, tx uint64) (*immuschema.Entry, error)
	// Deprecated: use LcClient.VCNGetArtifacts instead
	VerifiedGetAt(ctx context.Context, key []byte, tx uint64) (*immuschema.Entry, error)
	// Deprecated: use LcClient.VCNGetArtifacts instead
	GetAll(ctx context.Context, in *immuschema.KeyListRequest) (*immuschema.Entries, error)
	// Deprecated: use LcClient.VCNSetArtifacts instead
	SetAll(ctx context.Context, kvList *immuschema.SetRequest) (*immuschema.TxHeader, error)
	// Deprecated: use LcClient.VCNSetArtifacts instead
	SetMulti(ctx context.Context, req *schema.SetMultiRequest) (*schema.SetMultiResponse, error)
	// Deprecated: use LcClient.VCNSetArtifacts instead
	ExecAll(ctx context.Context, in *immuschema.ExecAllRequest) (*immuschema.TxHeader, error)
	// Deprecated: use LcClient.VCNSearchArtifacts instead
	Scan(ctx context.Context, req *immuschema.ScanRequest) (*immuschema.Entries, error)
	// Deprecated: use LcClient.VCNSearchArtifacts instead
	ZScan(ctx context.Context, req *immuschema.ZScanRequest) (*immuschema.ZEntries, error)
	// Deprecated: use LcClient.VCNSearchArtifacts instead
	History(ctx context.Context, req *immuschema.HistoryRequest) (*immuschema.Entries, error)
	// Deprecated: use LcClient.VCNSearchArtifacts instead
	ZScanExt(ctx context.Context, options *immuschema.ZScanRequest) (*schema.ZItemExtList, error)
	// Deprecated: use LcClient.VCNSearchArtifacts instead
	HistoryExt(ctx context.Context, options *immuschema.HistoryRequest) (sl *schema.ItemExtList, err error)
	// Deprecated: use LcClient.VCNGetArtifacts instead
	VerifiedGetExt(ctx context.Context, key []byte) (*schema.VerifiableItemExt, error)
	// Deprecated: use LcClient.VCNGetArtifacts instead
	VerifiedGetExtSince(ctx context.Context, key []byte, tx uint64) (*schema.VerifiableItemExt, error)
	// Deprecated: use LcClient.VCNGetArtifacts instead
	VerifiedGetExtAt(ctx context.Context, key []byte, tx uint64) (itemExt *schema.VerifiableItemExt, err error)
	// Deprecated: use LcClient.VCNGetArtifacts instead
	VerifiedGetExtAtMulti(ctx context.Context, keys [][]byte, txs []uint64) (itemsExt []*schema.VerifiableItemExt, errs []string, err error)
	// streams
	// Deprecated: use LcClient.VCNSetArtifacts instead
	StreamSet(ctx context.Context, kvs []*stream.KeyValue) (*immuschema.TxHeader, error)
	// Deprecated: use LcClient.VCNGetArtifacts instead
	StreamGet(ctx context.Context, k *immuschema.KeyRequest) (*immuschema.Entry, error)
	// Deprecated: use LcClient.VCNSetArtifacts instead
	StreamVerifiedSet(ctx context.Context, kvs []*stream.KeyValue) (*immuschema.TxHeader, error)
	// Deprecated: use LcClient.VCNGetArtifacts instead
	StreamVerifiedGet(ctx context.Context, req *immuschema.VerifiableGetRequest) (*immuschema.Entry, error)
	// Deprecated: use LcClient.VCNSearchArtifacts instead
	StreamScan(ctx context.Context, req *immuschema.ScanRequest) (*immuschema.Entries, error)
	// Deprecated: use LcClient.VCNSearchArtifacts instead
	StreamZScan(ctx context.Context, req *immuschema.ZScanRequest) (*immuschema.ZEntries, error)
	// Deprecated: use LcClient.VCNSearchArtifacts instead
	StreamHistory(ctx context.Context, req *immuschema.HistoryRequest) (*immuschema.Entries, error)
	// Deprecated: use LcClient.VCNSetArtifacts instead
	StreamExecAll(ctx context.Context, req *stream.ExecAllRequest) (*immuschema.TxHeader, error)
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
	RetryOptions         []grpc_retry.CallOption
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
		// TODO OGG: StreamChunkSize needs to be made configurable
		StreamChunkSize:      immuclient.DefaultOptions().StreamChunkSize,
		StreamServiceFactory: stream.NewStreamServiceFactory(immuclient.DefaultOptions().StreamChunkSize),
		RetryOptions: []grpc_retry.CallOption{
			grpc_retry.WithMax(0),
			grpc_retry.WithBackoff(grpc_retry.BackoffExponentialWithJitter(1500*time.Millisecond, 0.1)),
			grpc_retry.WithCodes(codes.Aborted, codes.Unavailable, codes.Unknown),
		},
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
	uic = append(uic, cli.ConnectionCheckerInterceptor(), cli.ApiKeySetterInterceptor(), grpc_retry.UnaryClientInterceptor(cli.RetryOptions...))

	var sic []grpc.StreamClientInterceptor
	sic = append(sic, cli.ApiKeySetterInterceptorStream(), grpc_retry.StreamClientInterceptor(cli.RetryOptions...))

	cli.DialOptions = append(
		cli.DialOptions,
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(uic...)),
		grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(sic...)))

	return cli
}

func (c *LcClient) Connect() (err error) {
	if c.ApiKey != "" {
		apiKeyPieces := strings.Split(c.ApiKey, ApiKeySeparator)
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

	c.StateService, err = NewLcStateService(cache.NewFileCache(c.Dir), c.Logger, stateProvider, uuidPrv)
	if err != nil {
		return err
	}

	return nil
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

func (c *LcClient) GetApiKey() string {
	return c.ApiKey
}

func (c *LcClient) GetLogger() logger.Logger {
	return c.Logger
}
