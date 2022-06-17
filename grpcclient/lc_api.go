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
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/codenotary/immudb/embedded/store"
	immuschema "github.com/codenotary/immudb/pkg/api/schema"
	"github.com/codenotary/immudb/pkg/database"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/vchain-us/ledger-compliance-go/schema"
	"google.golang.org/grpc"
)

// Deprecated: use VCNSetArtifacts instead
func (c *LcClient) Set(ctx context.Context, key []byte, value []byte) (*immuschema.TxHeader, error) {
	return c.ServiceClient.Set(ctx, &immuschema.SetRequest{KVs: []*immuschema.KeyValue{{Key: key, Value: value}}})
}

// Deprecated: use VCNGetArtifacts instead
func (c *LcClient) Get(ctx context.Context, key []byte) (*immuschema.Entry, error) {
	return c.ServiceClient.Get(ctx, &immuschema.KeyRequest{Key: key})
}

// Deprecated: use VCNGetArtifacts instead
func (c *LcClient) GetAt(ctx context.Context, key []byte, tx uint64) (*immuschema.Entry, error) {
	return c.ServiceClient.Get(ctx, &immuschema.KeyRequest{Key: key,
		AtTx: tx,
	})
}

// Deprecated: use VCNSetArtifacts instead
func (c *LcClient) ExecAll(ctx context.Context, in *immuschema.ExecAllRequest) (*immuschema.TxHeader, error) {
	result, err := c.ServiceClient.ExecAll(ctx, in)
	return result, err
}

// Deprecated: use VCNGetArtifacts instead
func (c *LcClient) GetAll(ctx context.Context, in *immuschema.KeyListRequest) (*immuschema.Entries, error) {
	return c.ServiceClient.GetAll(ctx, in)
}

// Deprecated: use VCNSetArtifacts instead
func (c *LcClient) SetAll(ctx context.Context, req *immuschema.SetRequest) (*immuschema.TxHeader, error) {
	return c.ServiceClient.Set(ctx, req)
}

// Deprecated: use VCNSetArtifacts instead
func (c *LcClient) SetMulti(ctx context.Context, req *schema.SetMultiRequest) (*schema.SetMultiResponse, error) {
	return c.ServiceClient.SetMulti(ctx, req)
}

// VCNSetArtifacts ...
func (c *LcClient) VCNSetArtifacts(ctx context.Context, req *schema.VCNArtifactsRequest) (*schema.VCNArtifactsResponse, error) {
	return c.ServiceClient.VCNSetArtifacts(ctx, req)
}

// VCNSearchArtifacts ...
func (c *LcClient) VCNSearchArtifacts(ctx context.Context, req *schema.VCNSearchRequest) (*schema.EntryList, error) {
	return c.ServiceClient.VCNSearchArtifacts(ctx, req)
}

// VCNGetArtifacts ...
func (c *LcClient) VCNGetArtifacts(ctx context.Context, req *schema.VCNArtifactsGetRequest) (*schema.EntryList, error) {
	if len(req.Hashes) > 1 && req.Verify {
		return nil, errors.New("verify can only be used with one hash")
	}
	if len(req.Hashes) != 1 || !req.Verify {
		resp, err := c.ServiceClient.VCNGetArtifacts(ctx, req)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}

	err := c.StateService.CacheLock()
	if err != nil {
		return nil, err
	}
	defer c.StateService.CacheUnlock()

	state, err := c.StateService.GetState(ctx, c.ApiKey)
	if err != nil {
		return nil, err
	}

	req.ProveSinceTx = state.TxId
	resp, err := c.ServiceClient.VCNGetArtifacts(ctx, req)
	if err != nil {
		return nil, err
	}

	hashDecoded, err := hex.DecodeString(req.Hashes[0])
	if err != nil {
		return nil, err
	}
	// check if the artifact hash is contained in the original immudb full key. We can enforce this check using a predefined key composition format
	if !bytes.Contains(resp.Entries[0].Proof.ImmudbFullKey, hashDecoded) {
		return nil, store.ErrCorruptedData
	}

	ventry := &immuschema.VerifiableEntry{
		Entry: &immuschema.Entry{
			Tx:       resp.Entries[0].Tx,
			Key:      resp.Entries[0].Proof.ImmudbFullKey,
			Value:    resp.Entries[0].Value,
			Metadata: resp.Entries[0].Proof.Metadata,
		},
		VerifiableTx:   resp.Entries[0].Proof.VerifiableTx,
		InclusionProof: resp.Entries[0].Proof.InclusionProof,
	}

	kReq := &immuschema.KeyRequest{
		Key: resp.Entries[0].Proof.ImmudbFullKey,
	}

	newState, err := verifyGet(state, ventry, kReq, c.ApiKeyHash)
	if err != nil {
		return nil, err
	}

	err = c.StateService.SetState(c.ApiKey, newState)
	if err != nil {
		return nil, err
	}

	if c.serverSigningPubKey != nil {
		ok, err := newState.CheckSignature(c.serverSigningPubKey)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, store.ErrCorruptedData
		}
	}
	return resp, nil
}

func (c *LcClient) VCNGetClientSignature(ctx context.Context, in *schema.VCNGetClientSignatureRequest, opts ...grpc.CallOption) (*schema.VCNGetClientSignatureResponse, error) {
	return c.ServiceClient.VCNGetClientSignature(ctx, in, opts...)
}

// Deprecated: use VCNSetArtifacts instead
func (c *LcClient) VerifiedSet(ctx context.Context, key []byte, value []byte) (*immuschema.TxHeader, error) {
	err := c.StateService.CacheLock()
	if err != nil {
		return nil, err
	}
	defer c.StateService.CacheUnlock()

	start := time.Now()
	defer c.Logger.Debugf("VerifiedSet finished in %s", time.Since(start))

	state, err := c.StateService.GetState(ctx, c.ApiKey)
	if err != nil {
		return nil, err
	}

	req := &immuschema.VerifiableSetRequest{
		SetRequest:   &immuschema.SetRequest{KVs: []*immuschema.KeyValue{{Key: key, Value: value}}},
		ProveSinceTx: state.TxId,
	}

	var metadata runtime.ServerMetadata

	verifiableTx, err := c.ServiceClient.VerifiableSet(
		ctx,
		req,
		grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD),
	)
	if err != nil {
		return nil, err
	}

	if verifiableTx.Tx.Header.Nentries != 1 || len(verifiableTx.Tx.Entries) != 1 {
		return nil, store.ErrCorruptedData
	}

	tx := immuschema.TxFromProto(verifiableTx.Tx)

	entrySpecDigest, err := store.EntrySpecDigestFor(tx.Header().Version)
	if err != nil {
		return nil, err
	}

	inclusionProof, err := tx.Proof(database.EncodeKey(key))
	if err != nil {
		return nil, err
	}

	md := tx.Entries()[0].Metadata()

	if md != nil && md.Deleted() {
		return nil, store.ErrCorruptedData
	}

	e := database.EncodeEntrySpec(key, md, value)

	verifies := store.VerifyInclusion(inclusionProof, entrySpecDigest(e), tx.Header().Eh)
	if !verifies {
		return nil, store.ErrCorruptedData
	}

	if tx.Header().Eh != immuschema.DigestFromProto(verifiableTx.DualProof.TargetTxHeader.EH) {
		return nil, store.ErrCorruptedData
	}

	var sourceID, targetID uint64
	var sourceAlh, targetAlh [sha256.Size]byte

	sourceID = state.TxId
	sourceAlh = immuschema.DigestFromProto(state.TxHash)
	targetID = tx.Header().ID
	targetAlh = tx.Header().Alh()

	if state.TxId > 0 {
		verifies = store.VerifyDualProof(
			immuschema.DualProofFromProto(verifiableTx.DualProof),
			sourceID,
			targetID,
			sourceAlh,
			targetAlh,
		)

		if !verifies {
			return nil, store.ErrCorruptedData
		}
	}

	newState := &immuschema.ImmutableState{
		Db:        c.ApiKey,
		TxId:      targetID,
		TxHash:    targetAlh[:],
		Signature: verifiableTx.Signature,
	}

	if c.serverSigningPubKey != nil {
		ok, err := newState.CheckSignature(c.serverSigningPubKey)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, store.ErrCorruptedData
		}
	}

	err = c.StateService.SetState(c.ApiKey, newState)
	if err != nil {
		return nil, err
	}

	return verifiableTx.Tx.Header, nil
}

// Deprecated: use VCNGetArtifacts instead
func (c *LcClient) VerifiedGet(ctx context.Context, key []byte) (*immuschema.Entry, error) {
	return c.verifiedGet(ctx, &immuschema.KeyRequest{
		Key: key,
	})
}

// Deprecated: use VCNGetArtifacts instead
func (c *LcClient) VerifiedGetAt(ctx context.Context, key []byte, tx uint64) (*immuschema.Entry, error) {
	return c.verifiedGet(ctx, &immuschema.KeyRequest{
		Key:  key,
		AtTx: tx,
	})
}

// Deprecated: use VCNGetArtifacts instead
func (c *LcClient) VerifiedGetSince(ctx context.Context, key []byte, tx uint64) (*immuschema.Entry, error) {
	return c.verifiedGet(ctx, &immuschema.KeyRequest{
		Key:     key,
		SinceTx: tx,
	})
}

// Deprecated: use VCNGetArtifacts instead
func (c *LcClient) VerifiedGetExt(ctx context.Context, key []byte) (itemExt *schema.VerifiableItemExt, err error) {
	return c.verifiedGetExt(ctx, &immuschema.KeyRequest{
		Key: key,
	})
}

// Deprecated: use VCNGetArtifacts instead
func (c *LcClient) VerifiedGetExtSince(ctx context.Context, key []byte, tx uint64) (itemExt *schema.VerifiableItemExt, err error) {
	return c.verifiedGetExt(ctx, &immuschema.KeyRequest{
		Key:     key,
		SinceTx: tx,
	})
}

// Deprecated: use VCNGetArtifacts instead
func (c *LcClient) VerifiedGetExtAt(ctx context.Context, key []byte, tx uint64) (itemExt *schema.VerifiableItemExt, err error) {
	return c.verifiedGetExt(ctx, &immuschema.KeyRequest{
		Key:  key,
		AtTx: tx,
	})
}

// Deprecated: use VCNGetArtifacts instead
func (c *LcClient) VerifiedGetExtAtMulti(ctx context.Context, keys [][]byte, txs []uint64) (itemsExt []*schema.VerifiableItemExt, errs []string, err error) {
	if len(keys) != len(txs) {
		err = errors.New("keys and txs must have the same length")
		return
	}

	reqs := make([]*immuschema.KeyRequest, 0, len(keys))
	for i, key := range keys {
		reqs = append(reqs, &immuschema.KeyRequest{Key: key, AtTx: txs[i]})
	}

	itemsExt, errs, err = c.verifiedGetExtMulti(ctx, reqs)
	return
}

// Deprecated: use VCNSearchArtifacts instead
func (c *LcClient) Scan(ctx context.Context, req *immuschema.ScanRequest) (*immuschema.Entries, error) {
	return c.ServiceClient.Scan(ctx, req)
}

// Deprecated: use VCNSearchArtifacts instead
func (c *LcClient) ZScan(ctx context.Context, req *immuschema.ZScanRequest) (*immuschema.ZEntries, error) {
	return c.ServiceClient.ZScan(ctx, req)
}

// Deprecated: use VCNSearchArtifacts instead
func (c *LcClient) History(ctx context.Context, req *immuschema.HistoryRequest) (*immuschema.Entries, error) {
	return c.ServiceClient.History(ctx, req)
}

// Deprecated: use VCNSearchArtifacts instead
func (c *LcClient) ZAddAt(ctx context.Context, options *immuschema.ZAddRequest) (*immuschema.TxHeader, error) {
	return c.ServiceClient.ZAdd(ctx, options)
}

// Deprecated: use VCNSearchArtifacts instead
func (c *LcClient) ZScanExt(ctx context.Context, options *immuschema.ZScanRequest) (*schema.ZItemExtList, error) {
	return c.ServiceClient.ZScanExt(ctx, options)
}

// Deprecated: use VCNSearchArtifacts instead
func (c *LcClient) HistoryExt(ctx context.Context, options *immuschema.HistoryRequest) (*schema.ItemExtList, error) {
	return c.ServiceClient.HistoryExt(ctx, options)
}

func (c *LcClient) Health(ctx context.Context) (*immuschema.HealthResponse, error) {
	return c.ServiceClient.Health(ctx, &empty.Empty{})
}

func (c *LcClient) CurrentState(ctx context.Context) (*immuschema.ImmutableState, error) {
	return c.ServiceClient.CurrentState(ctx, &empty.Empty{})
}

func (c *LcClient) Feats(ctx context.Context) (*schema.Features, error) {
	return c.ServiceClient.Feats(ctx, &empty.Empty{})
}

// SetFile ...
func (c *LcClient) SetFile(ctx context.Context, key []byte, filePath string) (*immuschema.TxHeader, error) {
	bs, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return c.ServiceClient.Set(ctx, &immuschema.SetRequest{KVs: []*immuschema.KeyValue{{Key: key, Value: bs}}})
}

// GetFile ...
func (c *LcClient) GetFile(ctx context.Context, key []byte, filePath string) (*immuschema.Entry, error) {
	entry, err := c.ServiceClient.Get(ctx, &immuschema.KeyRequest{Key: key})
	if err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(filePath, entry.Value, os.ModePerm); err != nil {
		return nil, err
	}
	return entry, nil
}

func (c *LcClient) verifiedGetExt(ctx context.Context, kReq *immuschema.KeyRequest) (itemExt *schema.VerifiableItemExt, err error) {
	err = c.StateService.CacheLock()
	if err != nil {
		return nil, err
	}
	defer c.StateService.CacheUnlock()

	state, err := c.StateService.GetState(ctx, c.ApiKey)
	if err != nil {
		return nil, err
	}

	req := &immuschema.VerifiableGetRequest{
		KeyRequest:   kReq,
		ProveSinceTx: state.TxId,
	}

	vEntryExt, err := c.ServiceClient.VerifiableGetExt(ctx, req)
	if err != nil {
		return nil, err
	}

	newState, err := verifyGet(state, vEntryExt.Item, kReq, c.ApiKeyHash)
	if err != nil {
		return nil, err
	}
	err = c.StateService.SetState(c.ApiKey, newState)
	if err != nil {
		return nil, err
	}

	if c.serverSigningPubKey != nil {
		ok, err := newState.CheckSignature(c.serverSigningPubKey)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, store.ErrCorruptedData
		}
	}

	return vEntryExt, nil
}

func (c *LcClient) verifiedGetExtMulti(
	ctx context.Context,
	reqs []*immuschema.KeyRequest,
) (itemExt []*schema.VerifiableItemExt, errs []string, err error) {

	err = c.StateService.CacheLock()
	if err != nil {
		return nil, nil, err
	}
	defer c.StateService.CacheUnlock()

	state, err := c.StateService.GetState(ctx, c.ApiKey)
	if err != nil {
		return nil, nil, err
	}

	req := &schema.VerifiableGetExtMultiRequest{}
	req.Requests = make([]*immuschema.VerifiableGetRequest, 0, len(reqs))
	for _, kr := range reqs {
		req.Requests = append(req.Requests, &immuschema.VerifiableGetRequest{
			KeyRequest:   kr,
			ProveSinceTx: state.TxId,
		})
	}

	resp, err := c.ServiceClient.VerifiableGetExtMulti(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	if len(resp.GetItems()) != len(req.Requests) || len(resp.GetErrors()) != len(req.Requests) {
		return resp.GetItems(), resp.GetErrors(), fmt.Errorf(
			"expected %d entries and %d errors, got %d entries and %d errors",
			len(req.Requests), len(req.Requests), len(resp.GetItems()), len(resp.GetErrors()))
	}

	itemsExt := resp.GetItems()
	errs = resp.GetErrors()

	for i, itemExt := range itemsExt {
		if errs[i] != "" {
			continue
		}
		newState, err := verifyGet(state, itemExt.GetItem(), reqs[i], c.ApiKeyHash)
		if err != nil {
			return nil, nil, err
		}
		err = c.StateService.SetState(c.ApiKey, newState)
		if err != nil {
			return nil, nil, err
		}
		if c.serverSigningPubKey != nil {
			ok, err := newState.CheckSignature(c.serverSigningPubKey)
			if err != nil {
				return nil, nil, err
			}
			if !ok {
				return nil, nil, store.ErrCorruptedData
			}
		}
	}

	return itemsExt, errs, nil
}

func (c *LcClient) verifiedGet(ctx context.Context, kReq *immuschema.KeyRequest) (vi *immuschema.Entry, err error) {
	err = c.StateService.CacheLock()
	if err != nil {
		return nil, err
	}
	defer c.StateService.CacheUnlock()

	state, err := c.StateService.GetState(ctx, c.ApiKey)
	if err != nil {
		return nil, err
	}

	req := &immuschema.VerifiableGetRequest{
		KeyRequest:   kReq,
		ProveSinceTx: state.TxId,
	}

	vEntry, err := c.ServiceClient.VerifiableGet(ctx, req)
	if err != nil {
		return nil, err
	}

	newState, err := verifyGet(state, vEntry, kReq, c.ApiKeyHash)
	if err != nil {
		return nil, err
	}

	err = c.StateService.SetState(c.ApiKey, newState)
	if err != nil {
		return nil, err
	}

	if c.serverSigningPubKey != nil {
		ok, err := newState.CheckSignature(c.serverSigningPubKey)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, store.ErrCorruptedData
		}
	}

	return vEntry.Entry, nil
}

func verifyGet(state *immuschema.ImmutableState, vEntry *immuschema.VerifiableEntry, kReq *immuschema.KeyRequest, apiKeyHash string) (*immuschema.ImmutableState, error) {
	entrySpecDigest, err := store.EntrySpecDigestFor(int(vEntry.VerifiableTx.Tx.Header.Version))
	if err != nil {
		return nil, err
	}

	inclusionProof := immuschema.InclusionProofFromProto(vEntry.InclusionProof)
	dualProof := immuschema.DualProofFromProto(vEntry.VerifiableTx.DualProof)

	var eh [sha256.Size]byte

	var sourceID, targetID uint64
	var sourceAlh, targetAlh [sha256.Size]byte

	var vTx uint64
	var e *store.EntrySpec

	if vEntry.Entry.ReferencedBy == nil {
		vTx = vEntry.Entry.Tx
		e = database.EncodeEntrySpec(kReq.Key, immuschema.KVMetadataFromProto(vEntry.Entry.Metadata), vEntry.Entry.Value)
	} else {
		ref := vEntry.Entry.ReferencedBy
		vTx = ref.Tx
		e = database.EncodeReference(ref.Key, immuschema.KVMetadataFromProto(ref.Metadata), vEntry.Entry.Key, ref.AtTx)
	}

	if state.TxId <= vTx {
		eh = immuschema.DigestFromProto(vEntry.VerifiableTx.DualProof.TargetTxHeader.EH)

		sourceID = state.TxId
		sourceAlh = immuschema.DigestFromProto(state.TxHash)
		targetID = vTx
		targetAlh = dualProof.TargetTxHeader.Alh()
	} else {
		eh = immuschema.DigestFromProto(vEntry.VerifiableTx.DualProof.SourceTxHeader.EH)

		sourceID = vTx
		sourceAlh = dualProof.SourceTxHeader.Alh()
		targetID = state.TxId
		targetAlh = immuschema.DigestFromProto(state.TxHash)
	}

	verifies := store.VerifyInclusion(
		inclusionProof,
		entrySpecDigest(e),
		eh)
	if !verifies {
		return nil, store.ErrCorruptedData
	}

	if state.TxId > 0 {
		verifies = store.VerifyDualProof(
			dualProof,
			sourceID,
			targetID,
			sourceAlh,
			targetAlh,
		)
		if !verifies {
			return nil, store.ErrCorruptedData
		}
	}

	newState := &immuschema.ImmutableState{
		Db:        apiKeyHash,
		TxId:      targetID,
		TxHash:    targetAlh[:],
		Signature: vEntry.VerifiableTx.Signature,
	}

	return newState, nil
}

func (c *LcClient) VCNLabelsSet(ctx context.Context, req []*schema.LabelsSetRequest, opts ...grpc.CallOption) (*schema.VCNLabelsSetResponse, error) {
	return c.ServiceClient.VCNLabelsSet(ctx, &schema.VCNLabelsSetRequest{Request: req}, opts...)
}

func (c *LcClient) VCNLabelsUpdate(ctx context.Context, hash string, ops []*schema.VCNLabelsUpdateRequest_VCNLabelsOp, opts ...grpc.CallOption) (*schema.VCNLabelsUpdateResponse, error) {
	return c.ServiceClient.VCNLabelsUpdate(ctx, &schema.VCNLabelsUpdateRequest{
		Hash: hash,
		Ops:  ops},
		opts...)
}

func (c *LcClient) VCNLabelsGet(ctx context.Context, req []*schema.LabelsGetRequest, opts ...grpc.CallOption) (*schema.VCNLabelsGetResponse, error) {
	return c.ServiceClient.VCNLabelsGet(ctx, &schema.VCNLabelsGetRequest{Request: req}, opts...)
}

func (c *LcClient) VCNGetAttachment(ctx context.Context, signerID, artifactHash, attachHash string) (*schema.VCNGetAttachmentResponse, error) {
	return c.ServiceClient.VCNGetAttachment(ctx, &schema.VCNGetAttachmentRequest{
		SignerID:       signerID,
		ArtifactHash:   artifactHash,
		AttachmentHash: attachHash,
	})
}
