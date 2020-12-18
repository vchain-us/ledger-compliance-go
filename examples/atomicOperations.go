// +build ignore

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
	client := sdk.NewLcClient(sdk.ApiKey("iygjyyaoudcpelitsbdvciughnzxdujmbhxy"), sdk.Host("localhost"), sdk.Port(3324))
	err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}

	aOps := &immuschema.Ops{
		Operations: []*immuschema.Op{
			{
				Operation: &immuschema.Op_KVs{
					KVs: &immuschema.KeyValue{
						Key:   []byte(`key1`),
						Value: []byte(`val1`),
					},
				},
			},
			{
				Operation: &immuschema.Op_KVs{
					KVs: &immuschema.KeyValue{
						Key:   []byte(`key2`),
						Value: []byte(`val2`),
					},
				},
			},
			{
				Operation: &immuschema.Op_ZOpts{
					ZOpts: &immuschema.ZAddOptions{
						Set: []byte(`mySet`),
						Key: []byte(`key1`),
						Score: &immuschema.Score{
							Score: 5.6,
						},
					},
				},
			},
			{
				Operation: &immuschema.Op_ZOpts{
					ZOpts: &immuschema.ZAddOptions{
						Set: []byte(`mySet`),
						Key: []byte(`key2`),
						Score: &immuschema.Score{
							Score: 15.9,
						},
					},
				},
			},
		},
	}
	_, err = client.ExecAllOps(context.Background(), aOps)
	if err != nil {
		log.Fatal(err)
	}
	list, err := client.ZScan(context.Background(), &immuschema.ZScanOptions{
		Set: []byte(`mySet`),
	})

	fmt.Printf("%v\n", list)
}
