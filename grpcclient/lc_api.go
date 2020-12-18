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
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/vchain-us/ledger-compliance-go/schema"
	"google.golang.org/grpc"
)

// Set ...
func (c *LcClient) Set(ctx context.Context, key []byte, value []byte) (*immuschema.TxMetadata, error) {
	return c.ServiceClient.Set(ctx, &immuschema.SetRequest{KVs: []*immuschema.KeyValue{{Key: key, Value: value}}})
}

// Get ...
func (c *LcClient) Get(ctx context.Context, key []byte) (*immuschema.Item, error) {
	return c.ServiceClient.Get(ctx, &immuschema.KeyRequest{Key: key})
}

// SetBatch ...
func (c *LcClient) ExecAll(ctx context.Context, in *immuschema.ExecAllRequest) (*immuschema.TxMetadata, error) {
	result, err := c.ServiceClient.ExecAllOps(ctx, in)
	return result, err
}

// GetBatch ...
func (c *LcClient) GetAll(ctx context.Context, in *immuschema.KeyListRequest) (*immuschema.ItemList, error) {
	return c.ServiceClient.GetAll(ctx, in)
}

// SafeSet ...
func (c *LcClient) VerifiedSet(ctx context.Context, key []byte, value []byte) (*immuschema.TxMetadata, error) {
	c.Lock()
	defer c.Unlock()
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

	inclusionProof, err := tx.Proof(key)
	if err != nil {
		return nil, err
	}

	verifies := store.VerifyInclusion(inclusionProof, &store.KV{Key: key, Value: value}, tx.Eh())
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

	// TODO: FIX state signing
	if newState.Signature != nil {
		ok, err := newState.CheckSignature()
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

// SafeGet ...
func (c *LcClient) VerifiedGet(ctx context.Context, key []byte) (*immuschema.Item, error) {
	c.Lock()
	defer c.Unlock()

	state, err := c.StateService.GetState(ctx, c.ApiKey)
	if err != nil {
		return nil, err
	}

	req := &immuschema.VerifiableGetRequest{
		KeyRequest:   &immuschema.KeyRequest{Key: key, SinceTx: state.TxId},
		ProveSinceTx: state.TxId,
	}

	vItem, err := c.ServiceClient.VerifiableGet(ctx, req)
	if err != nil {
		return nil, err
	}

	inclusionProof := immuschema.InclusionProofFrom(vItem.InclusionProof)
	dualProof := immuschema.DualProofFrom(vItem.VerifiableTx.DualProof)

	var eh [sha256.Size]byte

	var sourceID, targetID uint64
	var sourceAlh, targetAlh [sha256.Size]byte

	if state.TxId <= vItem.Item.Tx {
		eh = immuschema.DigestFrom(vItem.VerifiableTx.DualProof.TargetTxMetadata.EH)

		sourceID = state.TxId
		sourceAlh = immuschema.DigestFrom(state.TxHash)
		targetID = vItem.Item.Tx
		targetAlh = dualProof.TargetTxMetadata.Alh()
	} else {
		eh = immuschema.DigestFrom(vItem.VerifiableTx.DualProof.SourceTxMetadata.EH)

		sourceID = vItem.Item.Tx
		sourceAlh = dualProof.SourceTxMetadata.Alh()
		targetID = state.TxId
		targetAlh = immuschema.DigestFrom(state.TxHash)
	}

	verifies := store.VerifyInclusion(
		inclusionProof,
		&store.KV{Key: key, Value: vItem.Item.Value},
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
		Signature: vItem.VerifiableTx.Signature,
	}

	// TODO: FIX state signing
	if newState.Signature != nil {
		ok, err := newState.CheckSignature()
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

	return vItem.Item, nil
}

// Scan ...
func (c *LcClient) Scan(ctx context.Context, req *immuschema.ScanRequest) (*immuschema.ItemList, error) {
	return c.ServiceClient.Scan(ctx, req)
}

// ZScan ...
func (c *LcClient) ZScan(ctx context.Context, req *immuschema.ZScanRequest) (*immuschema.ZItemList, error) {
	return c.ServiceClient.ZScan(ctx, req)
}

// History ...
func (c *LcClient) History(ctx context.Context, req *immuschema.HistoryRequest) (*immuschema.ItemList, error) {
	return c.ServiceClient.History(ctx, req)
}

// ZAdd ...
func (c *LcClient) ZAddAt(ctx context.Context, options *immuschema.ZAddRequest) (*immuschema.TxMetadata, error) {
	return c.ServiceClient.ZAdd(ctx, options)
}

// SafeZAdd ...
func (c *LcClient) VerifiedZAddAt(ctx context.Context, req *immuschema.VerifiableZAddRequest) (*immuschema.TxMetadata, error) {
	c.Lock()
	defer c.Unlock()
	state, err := c.StateService.GetState(ctx, c.ApiKey)
	if err != nil {
		return nil, err
	}

	req.ProveSinceTx = state.TxId
	var metadata runtime.ServerMetadata

	result, err := c.ServiceClient.VerifiableZAdd(
		ctx,
		req,
		grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD),
	)
	if err != nil {
		return nil, err
	}

	/*
		verified, err := c.verifyAndSetRoot(result, root, ctx)
		if err != nil {
			return nil, err
		}
	*/

	return result.Tx.Metadata, nil
}

func (c *LcClient) ZScanExt(ctx context.Context, options *immuschema.ZScanRequest) (*schema.ZItemExtList, error) {
	return c.ServiceClient.ZScanExt(ctx, options)
}

func (c *LcClient) HistoryExt(ctx context.Context, options *immuschema.HistoryRequest) (*schema.ItemExtList, error) {
	return c.ServiceClient.HistoryExt(ctx, options)
}

func (c *LcClient) VerifiedGetExt(ctx context.Context, key []byte) (itemExt *schema.ItemExt, err error) {
	c.Lock()
	defer c.Unlock()
	/*
		root, err := c.StateService.GetRoot(ctx, c.ApiKey)
		if err != nil {
			return nil, err
		}

		sgOpts := &immuschema.SafeGetOptions{
			Key: key,
			RootIndex: &immuschema.Index{
				Index: root.GetIndex(),
			},
		}

		safeItemExt, err := c.ServiceClient.SafeGetExt(ctx, sgOpts)
		if err != nil {
			return nil, err
		}

		h, err := safeItemExt.Item.Hash()
		if err != nil {
			return nil, err
		}
		verified := safeItemExt.Item.Proof.Verify(h, *root)
		if verified {
			// saving a fresh root
			tocache := immuschema.NewRoot()
			tocache.SetIndex(safeItemExt.Item.Proof.At)
			tocache.SetRoot(safeItemExt.Item.Proof.Root)
			err = c.StateService.SetRoot(tocache, c.ApiKey)
			if err != nil {
				return nil, err
			}
		}

		sitem, err := safeItemExt.Item.ToSafeSItem()
		if err != nil {
			return nil, err
		}

		return &schema.VerifiedItemExt{
			Item: &immuclient.VerifiedItem{
				Key:      sitem.Item.GetKey(),
				Value:    sitem.Item.Value.Payload,
				Index:    sitem.Item.GetIndex(),
				Time:     sitem.Item.Value.Timestamp,
				Verified: verified,
			},
			Timestamp: safeItemExt.Timestamp,
		}, nil*/
	return
}
