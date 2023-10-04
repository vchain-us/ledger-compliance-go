/*
Copyright 2019-2023 vChain, Inc.

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

package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"log"

	"github.com/codenotary/immudb/pkg/api/schema"
	"github.com/codenotary/immudb/pkg/stream"
	sdk "github.com/vchain-us/ledger-compliance-go/grpcclient"
)

func main() {
	client := sdk.NewLcClient(
		sdk.ApiKey("APIKey1.JPOzsgjjXOQxNqTbTpvtoFXfmoHPzNuzqRvX"),
		sdk.Host("localhost"),
		sdk.Port(3324))
	err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}

	// StreamSet and StreamGet
	k1 := []byte("key1")
	v1 := randomValue(31 << 20) // 31 MB
	if err := streamSetAndGet(client, k1, v1); err != nil {
		log.Fatal(err)
	}

	// StreamVerifiedSet and StreamVerifiedGet
	k2 := []byte("key2")
	v2 := randomValue(6 << 20) // 6 MB
	k3 := []byte("key3")
	v3 := randomValue(7 << 20) // 7 MB
	if err := streamVerifiedSetAndGet(client, [][]byte{k2, k3}, [][]byte{v2, v3}); err != nil {
		log.Fatal(err)
	}

	// StreamExecAll
	k4 := k3
	v4 := randomValue(8 << 20) // 8 MB
	set := []byte("set1")
	score1 := 11.
	score4 := 33.
	txMeta, err := client.StreamExecAll(context.Background(), &stream.ExecAllRequest{
		Operations: []*stream.Op{
			{Operation: &stream.Op_KeyValue{
				KeyValue: &stream.KeyValue{
					Key:   &stream.ValueSize{Content: bufio.NewReader(bytes.NewBuffer(k4)), Size: len(k4)},
					Value: &stream.ValueSize{Content: bufio.NewReader(bytes.NewBuffer(v4)), Size: len(v4)},
				},
			}},
			{Operation: &stream.Op_ZAdd{
				ZAdd: &schema.ZAddRequest{Set: set, Score: score1, Key: k1},
			}},
			{Operation: &stream.Op_ZAdd{
				ZAdd: &schema.ZAddRequest{Set: set, Score: score4, Key: k4},
			}},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("StreamExecAll OK - TxID: %d\n", txMeta.GetId())

	// StreamScan
	entries, err := client.StreamScan(context.Background(), &schema.ScanRequest{Prefix: []byte("key")})
	if err != nil {
		log.Fatal(err)
	}
	if len(entries.GetEntries()) != 3 {
		log.Fatalf("StreamScan returned no error, but was expecting 3 entries, got %d", len(entries.GetEntries()))
	}
	fmt.Printf("StreamScan OK - Nb entries: %d\n", len(entries.GetEntries()))
	for _, e := range entries.GetEntries() {
		switch {
		case bytes.Equal(e.Key, k1), bytes.Equal(e.Key, k2), bytes.Equal(e.Key, k3):
			fmt.Printf("  - key %s found, value len: %d\n", e.Key, len(e.Value))
		default:
			fmt.Printf("  WARNING: unexpected key %s found\n", e.Key)
		}
	}

	// StreamZScan
	zentries, err := client.StreamZScan(context.Background(), &schema.ZScanRequest{Set: set})
	if err != nil {
		log.Fatal(err)
	}
	if len(zentries.GetEntries()) == 0 {
		log.Fatalf("StreamZScan returned 0 entries")
	}
	fmt.Printf("StreamZScan OK - Nb Z entries: %d\n", len(zentries.GetEntries()))
	for _, ze := range zentries.GetEntries() {
		switch {
		case bytes.Equal(ze.Key, k1), bytes.Equal(ze.Key, k3):
			fmt.Printf("  - key %s found in set %s with score %.0f\n", ze.Key, ze.Set, ze.Score)
		default:
			fmt.Printf("  WARNING: unexpected key %s found in set %s\n", ze.Key, ze.Set)
		}
	}

	// StreamHistory
	entries, err = client.StreamHistory(context.Background(), &schema.HistoryRequest{Key: k4})
	if err != nil {
		log.Fatal(err)
	}
	if len(entries.GetEntries()) == 0 {
		log.Fatalf("StreamHistory returned 0 entries")
	}
	fmt.Printf("StreamHistory OK - Nb historical entries: %d\n", len(entries.GetEntries()))
	for _, e := range entries.GetEntries() {
		fmt.Printf("  - key %s, value len: %d, tx: %d\n", e.Key, len(e.Value), e.Tx)
	}
}

func randomValue(MB int) []byte {
	v := make([]byte, MB)
	_, err := rand.Read(v)
	if err != nil {
		log.Fatal(err)
	}
	return v
}

func streamSetAndGet(client *sdk.LcClient, k, v []byte) error {
	ctx := context.Background()
	txMeta, err := client.StreamSet(ctx, []*stream.KeyValue{
		{
			Key:   &stream.ValueSize{Content: bufio.NewReader(bytes.NewBuffer(k)), Size: len(k)},
			Value: &stream.ValueSize{Content: bufio.NewReader(bytes.NewBuffer(v)), Size: len(v)},
		},
	})
	if err != nil {
		return err
	}
	fmt.Printf("StreamSet OK - TxID: %d\n", txMeta.GetId())

	entry, err := client.StreamGet(ctx, &schema.KeyRequest{Key: k})
	if err != nil {
		return err
	}
	fmt.Printf(
		"StreamGet OK - Entry key: %s, entry value length: %d (same as the set value: %t)\n",
		entry.Key, len(entry.Value), len(entry.Value) == len(v))
	return nil
}

func streamVerifiedSetAndGet(client *sdk.LcClient, ks, vs [][]byte) error {
	ctx := context.Background()
	sKVs := make([]*stream.KeyValue, 0, len(ks))
	for i, k := range ks {
		sKVs = append(sKVs, &stream.KeyValue{
			Key:   &stream.ValueSize{Content: bufio.NewReader(bytes.NewBuffer(k)), Size: len(k)},
			Value: &stream.ValueSize{Content: bufio.NewReader(bytes.NewBuffer(vs[i])), Size: len(vs[i])},
		})
	}
	txHeader, err := client.StreamVerifiedSet(ctx, sKVs)
	if err != nil {
		return err
	}
	fmt.Printf("StreamVerifiedSet OK - TxID: %d\n", txHeader.GetId())

	for i, k := range ks {
		entry, err := client.StreamVerifiedGet(ctx, &schema.VerifiableGetRequest{
			KeyRequest: &schema.KeyRequest{Key: k},
		})
		if err != nil {
			return err
		}
		fmt.Printf(
			"StreamVerifiedGet OK - Entry key: %s, entry value length: %d (same as the set value: %t)\n",
			entry.Key, len(entry.Value), len(entry.Value) == len(vs[i]))
	}

	return nil
}
