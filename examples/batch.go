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
	"context"
	"fmt"
	immuschema "github.com/codenotary/immudb/pkg/api/schema"
	sdk "github.com/vchain-us/ledger-compliance-go/grpcclient"
	"log"
)

func main() {
	client := sdk.NewLcClient(sdk.ApiKey("jxnqdxcicasvajpfxldqdanquhabiqcjrwdb"), sdk.Host("localhost"), sdk.Port(3324))
	err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}

	skv := &immuschema.KVList{
		KVs: []*immuschema.KeyValue{
			{
				Key:   []byte("key1"),
				Value: []byte("val1"),
			},
			{
				Key:   []byte("key2"),
				Value: []byte("val3"),
			},
			{
				Key:   []byte("key3"),
				Value: []byte("val3"),
			},
		},
	}
	_, err = client.SetBatch(context.Background(), skv)
	if err != nil {
		log.Fatal(err)
	}

	keys := &immuschema.KeyList{
		Keys: []*immuschema.Key{
			{
				Key: []byte("key1"),
			},
			{
				Key: []byte("key2"),
			},
			{
				Key: []byte("key3"),
			},
		},
	}
	list, err := client.GetBatch(context.Background(), keys)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%v\n", list)
}
