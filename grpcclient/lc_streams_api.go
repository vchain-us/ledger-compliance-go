package grpcclient

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"time"

	"github.com/codenotary/immudb/embedded/store"
	immuschema "github.com/codenotary/immudb/pkg/api/schema"
	"github.com/codenotary/immudb/pkg/database"
	"github.com/codenotary/immudb/pkg/stream"
)

// StreamSet ...
func (c *LcClient) StreamSet(ctx context.Context, kvs []*stream.KeyValue) (*immuschema.TxMetadata, error) {
	s, err := c.ServiceClient.StreamSet(ctx)
	if err != nil {
		return nil, err
	}

	kvss := c.StreamServiceFactory.NewKvStreamSender(c.StreamServiceFactory.NewMsgSender(s))

	for _, kv := range kvs {
		err = kvss.Send(kv)
		if err != nil {
			return nil, err
		}
	}

	return s.CloseAndRecv()
}

// StreamGet ...
func (c *LcClient) StreamGet(ctx context.Context, k *immuschema.KeyRequest) (*immuschema.Entry, error) {
	gs, err := c.ServiceClient.StreamGet(ctx, k)

	kvr := c.StreamServiceFactory.NewKvStreamReceiver(c.StreamServiceFactory.NewMsgReceiver(gs))

	key, vr, err := kvr.Next()
	if err != nil {
		return nil, err
	}

	value, err := stream.ReadValue(vr, c.StreamChunkSize)
	if err != nil {
		if err == io.EOF {
			return nil, stream.ErrMissingExpectedData
		}
		return nil, err
	}

	return &immuschema.Entry{
		Key:   key,
		Value: value,
	}, nil
}

// StreamVerifiedSet ...
func (c *LcClient) StreamVerifiedSet(ctx context.Context, kvs []*stream.KeyValue) (*immuschema.TxMetadata, error) {
	if len(kvs) == 0 {
		return nil, errors.New("no key-values specified")
	}

	err := c.StateService.CacheLock()
	if err != nil {
		return nil, err
	}
	defer c.StateService.CacheUnlock()

	start := time.Now()
	defer c.Logger.Debugf("StreamVerifiedSet finished in %s", time.Since(start))

	state, err := c.StateService.GetState(ctx, c.ApiKey)
	if err != nil {
		return nil, err
	}
	stateTxID, err := stream.NumberToBytes(state.TxId)
	if err != nil {
		return nil, err
	}

	//--> collect the keys and values as they need to be used for verifications
	stdKVs := make([]*immuschema.KeyValue, 0, len(kvs))
	for i, kv := range kvs {
		var keyBuffer bytes.Buffer
		keyTeeReader := io.TeeReader(kv.Key.Content, &keyBuffer)
		key := make([]byte, kv.Key.Size)
		if _, err := keyTeeReader.Read(key); err != nil {
			return nil, err
		}
		// put a new Reader back
		kvs[i].Key.Content = bufio.NewReader(&keyBuffer)

		var valueBuffer bytes.Buffer
		valueTeeReader := io.TeeReader(kv.Value.Content, &valueBuffer)
		value := make([]byte, kv.Value.Size)
		if _, err = valueTeeReader.Read(value); err != nil {
			return nil, err
		}
		// put a new Reader back
		kvs[i].Value.Content = bufio.NewReader(&valueBuffer)

		stdKVs = append(stdKVs, &immuschema.KeyValue{Key: key, Value: value})
	}
	//<--

	s, err := c.ServiceClient.StreamVerifiableSet(ctx)
	if err != nil {
		return nil, err
	}

	ss := c.StreamServiceFactory.NewMsgSender(s)
	kvss := c.StreamServiceFactory.NewKvStreamSender(ss)

	err = ss.Send(bytes.NewBuffer(stateTxID), len(stateTxID))
	if err != nil {
		return nil, err
	}

	for _, kv := range kvs {
		err = kvss.Send(kv)
		if err != nil {
			return nil, err
		}
	}

	verifiableTx, err := s.CloseAndRecv()

	if verifiableTx.Tx.Metadata.Nentries != int32(len(kvs)) {
		return nil, store.ErrCorruptedData
	}

	tx := immuschema.TxFrom(verifiableTx.Tx)

	var verifies bool

	for _, kv := range stdKVs {
		inclusionProof, err := tx.Proof(database.EncodeKey(kv.Key))
		if err != nil {
			return nil, err
		}
		verifies = store.VerifyInclusion(inclusionProof, database.EncodeKV(kv.Key, kv.Value), tx.Eh())
		if !verifies {
			return nil, store.ErrCorruptedData
		}
	}

	if tx.Eh() != immuschema.DigestFrom(verifiableTx.DualProof.TargetTxMetadata.EH) {
		return nil, store.ErrCorruptedData
	}

	var sourceID, targetID uint64
	var sourceAlh, targetAlh [sha256.Size]byte

	sourceID = state.TxId
	sourceAlh = immuschema.DigestFrom(state.TxHash)
	targetID = tx.ID
	targetAlh = tx.Alh

	if state.TxId > 0 {
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
	}

	newState := &immuschema.ImmutableState{
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

	return verifiableTx.Tx.Metadata, nil
}

// StreamVerifiedGet ...
func (c *LcClient) StreamVerifiedGet(ctx context.Context, req *immuschema.VerifiableGetRequest) (*immuschema.Entry, error) {
	err := c.StateService.CacheLock()
	if err != nil {
		return nil, err
	}
	defer c.StateService.CacheUnlock()

	state, err := c.StateService.GetState(ctx, c.ApiKey)
	if err != nil {
		return nil, err
	}

	gs, err := c.ServiceClient.StreamVerifiableGet(ctx, req)

	ver := c.StreamServiceFactory.NewVEntryStreamReceiver(c.StreamServiceFactory.NewMsgReceiver(gs))

	entryWithoutValueProto, verifiableTxProto, inclusionProofProto, vr, err := ver.Next()
	if err != nil {
		return nil, err
	}

	vEntry, err := stream.ParseVerifiableEntry(
		entryWithoutValueProto, verifiableTxProto, inclusionProofProto, vr, c.StreamChunkSize)
	if err != nil {
		return nil, err
	}

	inclusionProof := immuschema.InclusionProofFrom(vEntry.InclusionProof)
	dualProof := immuschema.DualProofFrom(vEntry.VerifiableTx.DualProof)

	var eh [sha256.Size]byte

	var sourceID, targetID uint64
	var sourceAlh, targetAlh [sha256.Size]byte

	var vTx uint64
	var kv *store.KV

	if vEntry.Entry.ReferencedBy == nil {
		vTx = vEntry.Entry.Tx
		kv = database.EncodeKV(req.KeyRequest.Key, vEntry.Entry.Value)
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

	verifies := store.VerifyInclusion(inclusionProof, kv, eh)
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
		TxId:      targetID,
		TxHash:    targetAlh[:],
		Signature: vEntry.VerifiableTx.Signature,
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

	return vEntry.Entry, nil
}

// StreamScan ...
func (c *LcClient) StreamScan(ctx context.Context, req *immuschema.ScanRequest) (*immuschema.Entries, error) {
	gs, err := c.ServiceClient.StreamScan(ctx, req)
	if err != nil {
		return nil, err
	}
	kvr := c.StreamServiceFactory.NewKvStreamReceiver(c.StreamServiceFactory.NewMsgReceiver(gs))
	var entries []*immuschema.Entry
	for {
		key, vr, err := kvr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		value, err := stream.ReadValue(vr, c.StreamChunkSize)
		if err != nil {
			if err == io.EOF {
				return nil, stream.ErrMissingExpectedData
			}
			return nil, err
		}

		entry := &immuschema.Entry{
			Key:   key,
			Value: value,
		}

		entries = append(entries, entry)
	}
	return &immuschema.Entries{Entries: entries}, nil
}

// StreamZScan ...
func (c *LcClient) StreamZScan(ctx context.Context, req *immuschema.ZScanRequest) (*immuschema.ZEntries, error) {
	gs, err := c.ServiceClient.StreamZScan(ctx, req)
	if err != nil {
		return nil, err
	}
	zr := c.StreamServiceFactory.NewZStreamReceiver(c.StreamServiceFactory.NewMsgReceiver(gs))
	var entries []*immuschema.ZEntry
	for {
		set, key, score, atTx, vr, err := zr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		entry, err := stream.ParseZEntry(set, key, score, atTx, vr, c.StreamChunkSize)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return &immuschema.ZEntries{Entries: entries}, nil
}

// StreamHistory ...
func (c *LcClient) StreamHistory(ctx context.Context, req *immuschema.HistoryRequest) (*immuschema.Entries, error) {
	gs, err := c.ServiceClient.StreamHistory(ctx, req)
	if err != nil {
		return nil, err
	}
	kvr := c.StreamServiceFactory.NewKvStreamReceiver(c.StreamServiceFactory.NewMsgReceiver(gs))
	var entries []*immuschema.Entry
	for {
		key, vr, err := kvr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		value, err := stream.ReadValue(vr, c.StreamChunkSize)
		if err != nil {
			if err == io.EOF {
				return nil, stream.ErrMissingExpectedData
			}
			return nil, err
		}

		entry := &immuschema.Entry{
			Key:   key,
			Value: value,
		}
		entries = append(entries, entry)
	}
	return &immuschema.Entries{Entries: entries}, nil
}
