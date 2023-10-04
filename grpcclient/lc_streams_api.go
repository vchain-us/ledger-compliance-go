package grpcclient

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"

	"github.com/codenotary/immudb/embedded/store"
	"github.com/codenotary/immudb/pkg/api/schema"
	"github.com/codenotary/immudb/pkg/database"
	"github.com/codenotary/immudb/pkg/stream"
)

// StreamSet ...
func (c *LcClient) StreamSet(ctx context.Context, kvs []*stream.KeyValue) (*schema.TxHeader, error) {
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
func (c *LcClient) StreamGet(ctx context.Context, k *schema.KeyRequest) (*schema.Entry, error) {
	gs, err := c.ServiceClient.StreamGet(ctx, k)
	if err != nil {
		return nil, err
	}

	kvr := c.StreamServiceFactory.NewKvStreamReceiver(c.StreamServiceFactory.NewMsgReceiver(gs))

	key, vr, err := kvr.Next()
	if err != nil {
		return nil, err
	}

	value, err := stream.ReadValue(vr, c.StreamChunkSize)
	if err != nil {
		if err == io.EOF {
			return nil, errors.New("expected data on stream is missing")
		}
		return nil, err
	}

	return &schema.Entry{
		Key:   key,
		Value: value,
	}, nil
}

// StreamVerifiedSet ...
func (c *LcClient) StreamVerifiedSet(ctx context.Context, kvs []*stream.KeyValue) (*schema.TxHeader, error) {
	if len(kvs) == 0 {
		return nil, errors.New("no key-values specified")
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
	stateTxID, err := stream.NumberToBytes(state.TxId)
	if err != nil {
		return nil, err
	}

	//--> collect the keys and values as they need to be used for verifications
	stdKVs := make([]*schema.KeyValue, 0, len(kvs))
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

		stdKVs = append(stdKVs, &schema.KeyValue{Key: key, Value: value})
	}
	//<--

	s, err := c.ServiceClient.StreamVerifiableSet(ctx)
	if err != nil {
		return nil, err
	}

	ss := c.StreamServiceFactory.NewMsgSender(s)
	kvss := c.StreamServiceFactory.NewKvStreamSender(ss)

	err = ss.Send(bytes.NewBuffer(stateTxID), len(stateTxID), nil)
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
	if err != nil {
		return nil, err
	}

	if verifiableTx.Tx.Header.Nentries != int32(len(kvs)) || len(verifiableTx.Tx.Entries) != len(kvs) {
		return nil, store.ErrCorruptedData
	}

	tx := schema.TxFromProto(verifiableTx.Tx)

	entrySpecDigest, err := store.EntrySpecDigestFor(tx.Header().Version)
	if err != nil {
		return nil, err
	}

	var verifies bool

	for i, kv := range stdKVs {
		inclusionProof, err := tx.Proof(database.EncodeKey(kv.Key))
		if err != nil {
			return nil, err
		}

		md := tx.Entries()[i].Metadata()
		e := database.EncodeEntrySpec(kv.Key, md, kv.Value)

		verifies = store.VerifyInclusion(inclusionProof, entrySpecDigest(e), tx.Header().Eh)
		if !verifies {
			return nil, store.ErrCorruptedData
		}
	}

	if tx.Header().Eh != schema.DigestFromProto(verifiableTx.DualProof.TargetTxHeader.EH) {
		return nil, store.ErrCorruptedData
	}

	var sourceID, targetID uint64
	var sourceAlh, targetAlh [sha256.Size]byte

	sourceID = state.TxId
	sourceAlh = schema.DigestFromProto(state.TxHash)
	targetID = tx.Header().ID
	targetAlh = tx.Header().Alh()

	if state.TxId > 0 {
		verifies = store.VerifyDualProof(
			schema.DualProofFromProto(verifiableTx.DualProof),
			sourceID,
			targetID,
			sourceAlh,
			targetAlh,
		)

		if !verifies {
			return nil, store.ErrCorruptedData
		}
	}

	newState := &schema.ImmutableState{
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

// StreamVerifiedGet ...
func (c *LcClient) StreamVerifiedGet(ctx context.Context, req *schema.VerifiableGetRequest) (*schema.Entry, error) {
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
	if err != nil {
		return nil, err
	}

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

	entrySpecDigest, err := store.EntrySpecDigestFor(int(vEntry.VerifiableTx.Tx.Header.Version))
	if err != nil {
		return nil, err
	}

	inclusionProof := schema.InclusionProofFromProto(vEntry.InclusionProof)
	dualProof := schema.DualProofFromProto(vEntry.VerifiableTx.DualProof)

	var eh [sha256.Size]byte

	var sourceID, targetID uint64
	var sourceAlh, targetAlh [sha256.Size]byte

	var vTx uint64
	var e *store.EntrySpec

	if vEntry.Entry.ReferencedBy == nil {
		vTx = vEntry.Entry.Tx
		e = database.EncodeEntrySpec(req.KeyRequest.Key, schema.KVMetadataFromProto(vEntry.Entry.Metadata), vEntry.Entry.Value)
	} else {
		ref := vEntry.Entry.ReferencedBy
		vTx = ref.Tx
		e = database.EncodeReference(ref.Key, schema.KVMetadataFromProto(ref.Metadata), vEntry.Entry.Key, ref.AtTx)
	}

	if state.TxId <= vTx {
		eh = schema.DigestFromProto(vEntry.VerifiableTx.DualProof.TargetTxHeader.EH)
		sourceID = state.TxId
		sourceAlh = schema.DigestFromProto(state.TxHash)
		targetID = vTx
		targetAlh = dualProof.TargetTxHeader.Alh()
	} else {
		eh = schema.DigestFromProto(vEntry.VerifiableTx.DualProof.SourceTxHeader.EH)
		sourceID = vTx
		sourceAlh = dualProof.SourceTxHeader.Alh()
		targetID = state.TxId
		targetAlh = schema.DigestFromProto(state.TxHash)
	}

	verifies := store.VerifyInclusion(inclusionProof, entrySpecDigest(e), eh)
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

	newState := &schema.ImmutableState{
		Db:        c.ApiKey,
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
func (c *LcClient) StreamScan(ctx context.Context, req *schema.ScanRequest) (*schema.Entries, error) {
	gs, err := c.ServiceClient.StreamScan(ctx, req)
	if err != nil {
		return nil, err
	}
	kvr := c.StreamServiceFactory.NewKvStreamReceiver(c.StreamServiceFactory.NewMsgReceiver(gs))
	var entries []*schema.Entry
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
				return nil, errors.New("expected data on stream is missing")
			}
			return nil, err
		}

		entry := &schema.Entry{
			Key:   key,
			Value: value,
		}

		entries = append(entries, entry)
	}
	return &schema.Entries{Entries: entries}, nil
}

// StreamZScan ...
func (c *LcClient) StreamZScan(ctx context.Context, req *schema.ZScanRequest) (*schema.ZEntries, error) {
	gs, err := c.ServiceClient.StreamZScan(ctx, req)
	if err != nil {
		return nil, err
	}
	zr := c.StreamServiceFactory.NewZStreamReceiver(c.StreamServiceFactory.NewMsgReceiver(gs))
	var entries []*schema.ZEntry
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
	return &schema.ZEntries{Entries: entries}, nil
}

// StreamHistory ...
func (c *LcClient) StreamHistory(ctx context.Context, req *schema.HistoryRequest) (*schema.Entries, error) {
	gs, err := c.ServiceClient.StreamHistory(ctx, req)
	if err != nil {
		return nil, err
	}
	kvr := c.StreamServiceFactory.NewKvStreamReceiver(c.StreamServiceFactory.NewMsgReceiver(gs))
	var entries []*schema.Entry
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
				return nil, errors.New("expected data on stream is missing")
			}
			return nil, err
		}

		entry := &schema.Entry{
			Key:   key,
			Value: value,
		}
		entries = append(entries, entry)
	}
	return &schema.Entries{Entries: entries}, nil
}

// StreamExecAll ...
func (c *LcClient) StreamExecAll(ctx context.Context, req *stream.ExecAllRequest) (*schema.TxHeader, error) {
	s, err := c.ServiceClient.StreamExecAll(ctx)
	if err != nil {
		return nil, err
	}

	eas := c.StreamServiceFactory.NewExecAllStreamSender(c.StreamServiceFactory.NewMsgSender(s))

	err = eas.Send(req)
	if err != nil {
		return nil, err
	}

	return s.CloseAndRecv()
}
