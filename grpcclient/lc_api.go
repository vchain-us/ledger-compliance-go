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
	"errors"
	"github.com/codenotary/immudb/pkg/store"
	"time"

	immuschema "github.com/codenotary/immudb/pkg/api/schema"
	immuclient "github.com/codenotary/immudb/pkg/client"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

// Set ...
func (c *LcClient) Set(ctx context.Context, key []byte, value []byte) (*immuschema.Index, error) {
	skv := c.NewSKV(key, value)
	kv, err := skv.ToKV()
	if err != nil {
		return nil, err
	}
	return c.ServiceClient.Set(ctx, kv)
}

// Get ...
func (c *LcClient) Get(ctx context.Context, key []byte) (si *immuschema.StructuredItem, err error) {
	var item *immuschema.Item
	if item, err = c.ServiceClient.Get(ctx, &immuschema.Key{Key: key}); err != nil {
		return nil, err
	}
	return item.ToSItem()
}

// SafeSet ...
func (c *LcClient) SafeSet(ctx context.Context, key []byte, value []byte) (*immuclient.VerifiedIndex, error) {
	start := time.Now()
	c.Lock()
	defer c.Unlock()

	root, err := c.RootService.GetRoot(ctx, c.ApiKey)
	if err != nil {
		return nil, err
	}

	skv := c.NewSKV(key, value)
	kv, err := skv.ToKV()
	if err != nil {
		return nil, err
	}
	opts := &immuschema.SafeSetOptions{
		Kv: kv,
		RootIndex: &immuschema.Index{
			Index: root.GetIndex(),
		},
	}

	var metadata runtime.ServerMetadata

	result, err := c.ServiceClient.SafeSet(
		ctx,
		opts,
		grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD),
	)
	if err != nil {
		return nil, err
	}

	// This guard ensures that result.Leaf is equal to the item's hash computed from
	// request values. From now on, result.Leaf can be trusted.
	sitem := immuschema.StructuredItem{
		Key: key,
		Value: &immuschema.Content{
			Timestamp: skv.Value.Timestamp,
			Payload:   value,
		},
		Index: result.Index,
	}
	item, err := sitem.ToItem()
	if err != nil {
		return nil, err
	}
	h := item.Hash()

	if !bytes.Equal(h, result.Leaf) {
		return nil, errors.New("proof does not match the given item")
	}

	verified, err := c.verifyAndSetRoot(result, root)
	if err != nil {
		return nil, err
	}

	c.Logger.Debugf("safeset finished in %s", time.Since(start))

	return &immuclient.VerifiedIndex{
			Index:    result.Index,
			Verified: verified,
		},
		nil
}

// SafeGet ...
func (c *LcClient) SafeGet(ctx context.Context, key []byte) (vi *immuclient.VerifiedItem, err error) {
	c.Lock()
	defer c.Unlock()

	root, err := c.RootService.GetRoot(ctx, c.ApiKey)
	if err != nil {
		return nil, err
	}

	sgOpts := &immuschema.SafeGetOptions{
		Key: key,
		RootIndex: &immuschema.Index{
			Index: root.GetIndex(),
		},
	}

	safeItem, err := c.ServiceClient.SafeGet(ctx, sgOpts)
	if err != nil {
		return nil, err
	}

	h, err := safeItem.Hash()
	if err != nil {
		return nil, err
	}
	verified := safeItem.Proof.Verify(h, *root)
	if verified {
		// saving a fresh root
		tocache := immuschema.NewRoot()
		tocache.SetIndex(safeItem.Proof.At)
		tocache.SetRoot(safeItem.Proof.Root)
		err = c.RootService.SetRoot(tocache, c.ApiKey)
		if err != nil {
			return nil, err
		}
	}

	sitem, err := safeItem.ToSafeSItem()
	if err != nil {
		return nil, err
	}
	return &immuclient.VerifiedItem{
			Key:      sitem.Item.GetKey(),
			Value:    sitem.Item.Value.Payload,
			Index:    sitem.Item.GetIndex(),
			Time:     sitem.Item.Value.Timestamp,
			Verified: verified,
		},
		nil
}

// Scan ...
func (c *LcClient) Scan(ctx context.Context, prefix []byte) (*immuschema.StructuredItemList, error) {
	list, err := c.ServiceClient.Scan(ctx, &immuschema.ScanOptions{Prefix: prefix})
	if err != nil {
		return nil, err
	}
	return list.ToSItemList()
}

// ZScan ...
func (c *LcClient) ZScan(ctx context.Context, set []byte) (*immuschema.StructuredItemList, error) {
	list, err := c.ServiceClient.ZScan(ctx, &immuschema.ZScanOptions{Set: set})
	if err != nil {
		return nil, err
	}
	return list.ToSItemList()
}

// History ...
func (c *LcClient) History(ctx context.Context, key []byte) (sl *immuschema.StructuredItemList, err error) {
	list, err := c.ServiceClient.History(ctx, &immuschema.Key{
		Key: key,
	})
	if err != nil {
		return nil, err
	}
	sl, err = list.ToSItemList()
	if err != nil {
		return nil, err
	}
	return sl, err
}

// ZAdd ...
func (c *LcClient) ZAdd(ctx context.Context, set []byte, score float64, key []byte) (*immuschema.Index, error) {
	result, err := c.ServiceClient.ZAdd(ctx, &immuschema.ZAddOptions{
		Set:   set,
		Score: score,
		Key:   key,
	})
	return result, err
}

// SafeZAdd ...
func (c *LcClient) SafeZAdd(ctx context.Context, set []byte, score float64, key []byte) (*immuclient.VerifiedIndex, error) {
	c.Lock()
	defer c.Unlock()
	root, err := c.RootService.GetRoot(ctx, c.ApiKey)
	if err != nil {
		return nil, err
	}

	opts := &immuschema.SafeZAddOptions{
		Zopts: &immuschema.ZAddOptions{
			Set:   set,
			Score: score,
			Key:   key,
		},
		RootIndex: &immuschema.Index{
			Index: root.GetIndex(),
		},
	}

	var metadata runtime.ServerMetadata
	result, err := c.ServiceClient.SafeZAdd(
		ctx,
		opts,
		grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD),
	)
	if err != nil {
		return nil, err
	}

	key2, err := store.SetKey(key, set, score)
	if err != nil {
		return nil, err
	}

	// This guard ensures that result.Leaf is equal to the item's hash computed
	// from request values. From now on, result.Leaf can be trusted.
	item := immuschema.Item{
		Key:   key2,
		Value: key,
		Index: result.Index,
	}
	if !bytes.Equal(item.Hash(), result.Leaf) {
		return nil, errors.New("proof does not match the given item")
	}

	verified, err := c.verifyAndSetRoot(result, root)
	if err != nil {
		return nil, err
	}

	return &immuclient.VerifiedIndex{
			Index:    result.Index,
			Verified: verified,
		},
		nil
}
