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
	"crypto/sha256"
	"github.com/codenotary/immudb/embedded/store"
	immuschema "github.com/codenotary/immudb/pkg/api/schema"
	"github.com/codenotary/immudb/pkg/database"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/vchain-us/ledger-compliance-go/schema"
	"google.golang.org/grpc"
	"time"
)

// Set ...
func (c *LcClient) Set(ctx context.Context, key []byte, value []byte) (*immuschema.TxMetadata, error) {
	return c.ServiceClient.Set(ctx, &immuschema.SetRequest{KVs: []*immuschema.KeyValue{{Key: key, Value: value}}})
}

// Get ...
func (c *LcClient) Get(ctx context.Context, key []byte) (*immuschema.Entry, error) {
	return c.ServiceClient.Get(ctx, &immuschema.KeyRequest{Key: key})
}

// GetAt ...
func (c *LcClient) GetAt(ctx context.Context, key []byte, tx uint64) (*immuschema.Entry, error) {
	return c.ServiceClient.Get(ctx, &immuschema.KeyRequest{Key: key,
		AtTx: tx,
	})
}

// ExecAll ...
func (c *LcClient) ExecAll(ctx context.Context, in *immuschema.ExecAllRequest) (*immuschema.TxMetadata, error) {
	result, err := c.ServiceClient.ExecAll(ctx, in)
	return result, err
}

// GetAll ...
func (c *LcClient) GetAll(ctx context.Context, in *immuschema.KeyListRequest) (*immuschema.Entries, error) {
	return c.ServiceClient.GetAll(ctx, in)
}

// VerifiedSet ...
func (c *LcClient) VerifiedSet(ctx context.Context, key []byte, value []byte) (*immuschema.TxMetadata, error) {
	c.Lock()
	defer c.Unlock()

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

	tx := immuschema.TxFrom(verifiableTx.Tx)

	inclusionProof, err := tx.Proof(database.EncodeKey(key))
	if err != nil {
		return nil, err
	}

	verifies := store.VerifyInclusion(inclusionProof, database.EncodeKV(key, value), tx.Eh())
	if !verifies {
		return nil, store.ErrCorruptedData
	}

	if tx.Eh() != immuschema.DigestFrom(verifiableTx.DualProof.TargetTxMetadata.EH) {
		return nil, store.ErrCorruptedData
	}

	var sourceID, targetID uint64
	var sourceAlh, targetAlh [sha256.Size]byte

	if state.TxId == 0 {
		sourceID = tx.ID
		sourceAlh = tx.Alh
	} else {
		sourceID = state.TxId
		sourceAlh = immuschema.DigestFrom(state.TxHash)
	}

	targetID = tx.ID
	targetAlh = tx.Alh

	verifies = store.VerifyDualProof(
		immuschema.DualProofFrom(verifiableTx.DualProof),
		sourceID,
		targetID,
		sourceAlh,
		targetAlh,
	)
	if !verifies {
		return nil, store.ErrCorruptedData
	}

	newState := &immuschema.ImmutableState{
		TxId:      tx.ID,
		TxHash:    tx.Alh[:],
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

	return verifiableTx.Tx.Metadata, nil
}

// VerifiedGet ...
func (c *LcClient) VerifiedGet(ctx context.Context, key []byte) (*immuschema.Entry, error) {
	return c.verifiedGet(ctx, &immuschema.KeyRequest{
		Key: key,
	})
}

// VerifiedGetAt ...
func (c *LcClient) VerifiedGetAt(ctx context.Context, key []byte, tx uint64) (*immuschema.Entry, error) {
	return c.verifiedGet(ctx, &immuschema.KeyRequest{
		Key:  key,
		AtTx: tx,
	})
}

// VerifiedGetSince ...
func (c *LcClient) VerifiedGetSince(ctx context.Context, key []byte, tx uint64) (*immuschema.Entry, error) {
	return c.verifiedGet(ctx, &immuschema.KeyRequest{
		Key:     key,
		SinceTx: tx,
	})
}

// VerifiedGetExt ...
func (c *LcClient) VerifiedGetExt(ctx context.Context, key []byte) (itemExt *schema.VerifiableItemExt, err error) {
	return c.verifiedGetExt(ctx, &immuschema.KeyRequest{
		Key: key,
	})
}

// VerifiedGetExtSince ...
func (c *LcClient) VerifiedGetExtSince(ctx context.Context, key []byte, tx uint64) (itemExt *schema.VerifiableItemExt, err error) {
	return c.verifiedGetExt(ctx, &immuschema.KeyRequest{
		Key:     key,
		SinceTx: tx,
	})
}

// VerifiedGetExtAt ...
func (c *LcClient) VerifiedGetExtAt(ctx context.Context, key []byte, tx uint64) (itemExt *schema.VerifiableItemExt, err error) {
	return c.verifiedGetExt(ctx, &immuschema.KeyRequest{
		Key:  key,
		AtTx: tx,
	})
}

// Scan ...
func (c *LcClient) Scan(ctx context.Context, req *immuschema.ScanRequest) (*immuschema.Entries, error) {
	return c.ServiceClient.Scan(ctx, req)
}

// ZScan ...
func (c *LcClient) ZScan(ctx context.Context, req *immuschema.ZScanRequest) (*immuschema.ZEntries, error) {
	return c.ServiceClient.ZScan(ctx, req)
}

// History ...
func (c *LcClient) History(ctx context.Context, req *immuschema.HistoryRequest) (*immuschema.Entries, error) {
	return c.ServiceClient.History(ctx, req)
}

// ZAddAt ...
func (c *LcClient) ZAddAt(ctx context.Context, options *immuschema.ZAddRequest) (*immuschema.TxMetadata, error) {
	return c.ServiceClient.ZAdd(ctx, options)
}

// ZScanExt ...
func (c *LcClient) ZScanExt(ctx context.Context, options *immuschema.ZScanRequest) (*schema.ZItemExtList, error) {
	return c.ServiceClient.ZScanExt(ctx, options)
}

// HistoryExt ...
func (c *LcClient) HistoryExt(ctx context.Context, options *immuschema.HistoryRequest) (*schema.ItemExtList, error) {
	return c.ServiceClient.HistoryExt(ctx, options)
}

func (c *LcClient) verifiedGetExt(ctx context.Context, kReq *immuschema.KeyRequest) (itemExt *schema.VerifiableItemExt, err error) {
	c.Lock()
	defer c.Unlock()

	state, err := c.StateService.GetState(ctx, c.ApiKey)
	if err != nil {
		return nil, err
	}

	if kReq.SinceTx == 0 && kReq.AtTx == 0 {
		kReq.SinceTx = state.TxId
	}

	req := &immuschema.VerifiableGetRequest{
		KeyRequest:   kReq,
		ProveSinceTx: state.TxId,
	}

	vEntryExt, err := c.ServiceClient.VerifiableGetExt(ctx, req)
	if err != nil {
		return nil, err
	}

	newState, err := verifyGet(state, vEntryExt.Item, kReq)
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

func (c *LcClient) verifiedGet(ctx context.Context, kReq *immuschema.KeyRequest) (vi *immuschema.Entry, err error) {
	c.Lock()
	defer c.Unlock()

	state, err := c.StateService.GetState(ctx, c.ApiKey)
	if err != nil {
		return nil, err
	}

	if kReq.SinceTx == 0 && kReq.AtTx == 0 {
		kReq.SinceTx = state.TxId
	}

	req := &immuschema.VerifiableGetRequest{
		KeyRequest:   kReq,
		ProveSinceTx: state.TxId,
	}

	vEntry, err := c.ServiceClient.VerifiableGet(ctx, req)
	if err != nil {
		return nil, err
	}

	newState, err := verifyGet(state, vEntry, kReq)
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

func verifyGet(state *immuschema.ImmutableState, vEntry *immuschema.VerifiableEntry, kReq *immuschema.KeyRequest) (*immuschema.ImmutableState, error) {
	inclusionProof := immuschema.InclusionProofFrom(vEntry.InclusionProof)
	dualProof := immuschema.DualProofFrom(vEntry.VerifiableTx.DualProof)

	var eh [sha256.Size]byte

	var sourceID, targetID uint64
	var sourceAlh, targetAlh [sha256.Size]byte

	var vTx uint64
	var kv *store.KV

	if vEntry.Entry.ReferencedBy == nil {
		vTx = vEntry.Entry.Tx
		kv = database.EncodeKV(kReq.Key, vEntry.Entry.Value)
	} else {
		vTx = vEntry.Entry.ReferencedBy.Tx
		kv = database.EncodeReference(vEntry.Entry.ReferencedBy.Key, vEntry.Entry.Key, vEntry.Entry.ReferencedBy.AtTx)
	}

	if state.TxId <= vTx {
		eh = immuschema.DigestFrom(vEntry.VerifiableTx.DualProof.TargetTxMetadata.EH)

		sourceID = state.TxId
		sourceAlh = immuschema.DigestFrom(state.TxHash)
		targetID = vTx
		targetAlh = dualProof.TargetTxMetadata.Alh()
	} else {
		eh = immuschema.DigestFrom(vEntry.VerifiableTx.DualProof.SourceTxMetadata.EH)

		sourceID = vTx
		sourceAlh = dualProof.SourceTxMetadata.Alh()
		targetID = state.TxId
		targetAlh = immuschema.DigestFrom(state.TxHash)
	}

	verifies := store.VerifyInclusion(
		inclusionProof,
		kv,
		eh)
	if !verifies {
		return nil, store.ErrCorruptedData
	}

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

	newState := &immuschema.ImmutableState{
		TxId:      targetID,
		TxHash:    targetAlh[:],
		Signature: vEntry.VerifiableTx.Signature,
	}

	return newState, nil
}
