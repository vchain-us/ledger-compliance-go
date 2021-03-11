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

package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"log"

	"github.com/codenotary/immudb/pkg/api/schema"
	"github.com/codenotary/immudb/pkg/stream"
	sdk "github.com/vchain-us/ledger-compliance-go/grpcclient"
)

func main() {
	client := sdk.NewLcClient(
		sdk.ApiKey("huqymoysjtjzvkmligcvbucicjejhknlmrsk"),
		sdk.Host("localhost"),
		sdk.Port(3324))
	err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}

	v := make([]byte, 31<<20) // 31 MB
	_, err = rand.Read(v)
	if err != nil {
		log.Fatal(err)
	}
	k := []byte("key1")

	txMeta, err := client.StreamSet(context.Background(), []*stream.KeyValue{
		{Key: &stream.ValueSize{
			Content: bufio.NewReader(bytes.NewBuffer(k)),
			Size:    len(k),
		},
			Value: &stream.ValueSize{
				Content: bufio.NewReader(bytes.NewBuffer(v)),
				Size:    len(v),
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("StreamSet OK - TxID: %d\n", txMeta.GetId())

	entry, err := client.StreamGet(context.Background(), &schema.KeyRequest{Key: k})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf(
		"StreamGet OK - Entry key: %s, entry value length: %d (same as the set value: %t)",
		entry.Key, len(entry.Value), len(entry.Value) == len(v))
}
