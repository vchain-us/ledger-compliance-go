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
	sdk "github.com/vchain-us/ledger-compliance-go/grpcclient"
	"log"
)

func main() {
	client := sdk.NewLcClient(sdk.ApiKey("vipicevtthnqiveufonnkhnoshkoyrbvpeyj"), sdk.Host("localhost"), sdk.Port(3324))
	err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}
	_, err = client.SafeSet(context.Background(), []byte(`key1`), []byte(`val1`))
	if err != nil {
		log.Fatal(err)
	}
	_, err = client.SafeSet(context.Background(), []byte(`key2`), []byte(`val2`))
	if err != nil {
		log.Fatal(err)
	}
	_, err = client.SafeSet(context.Background(), []byte(`key3`), []byte(`val3`))
	if err != nil {
		log.Fatal(err)
	}

	id, err := client.SafeZAdd(context.Background(), &immuschema.ZAddOptions{
		Set: []byte(`mySortedSet`),
		Score: &immuschema.Score{
			Score: 5,
		},
		Key: []byte(`key1`),
	})
	if err != nil {
		log.Fatal(err)
	}
	println(id.Verified)
	id, err = client.SafeZAdd(context.Background(), &immuschema.ZAddOptions{
		Set: []byte(`mySortedSet`),
		Score: &immuschema.Score{
			Score: 99,
		},
		Key: []byte(`key3`),
	})
	if err != nil {
		log.Fatal(err)
	}
	println(id.Verified)
	id, err = client.SafeZAdd(context.Background(), &immuschema.ZAddOptions{
		Set: []byte(`mySortedSet`),
		Score: &immuschema.Score{
			Score: 1,
		},
		Key: []byte(`key2`),
	})

	if err != nil {
		log.Fatal(err)
	}
	println(id.Verified)
	list, err := client.ZScan(context.Background(), &immuschema.ZScanOptions{
		Set: []byte(`mySortedSet`),
	})

	fmt.Printf("%v\n", list)

}
