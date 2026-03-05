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
	"context"
	"fmt"
	"log"

	immuschema "github.com/codenotary/immudb/pkg/api/schema"
	sdk "github.com/vchain-us/ledger-compliance-go/grpcclient"
)

func main() {
	client := sdk.NewLcClient(sdk.ApiKey("msvvnhdhfseqbveblqxruhasfrvkqlutctcj"), sdk.Host("localhost"), sdk.Port(3324))
	err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}
	_, err = client.VerifiedSet(context.Background(), []byte(`key1`), []byte(`val1`))
	if err != nil {
		log.Fatal(err)
	}
	_, err = client.VerifiedSet(context.Background(), []byte(`key2`), []byte(`val2`))
	if err != nil {
		log.Fatal(err)
	}
	_, err = client.VerifiedSet(context.Background(), []byte(`key3`), []byte(`val3`))
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.ZAddAt(context.Background(), &immuschema.ZAddRequest{
		Set:   []byte(`mySortedSet`),
		Score: 5,
		Key:   []byte(`key1`),
	})
	if err != nil {
		log.Fatal(err)
	}
	_, err = client.ZAddAt(context.Background(), &immuschema.ZAddRequest{
		Set:   []byte(`mySortedSet`),
		Score: 99,
		Key:   []byte(`key3`),
	})
	if err != nil {
		log.Fatal(err)
	}
	_, err = client.ZAddAt(context.Background(), &immuschema.ZAddRequest{
		Set:   []byte(`mySortedSet`),
		Score: 1,
		Key:   []byte(`key2`),
	})

	if err != nil {
		log.Fatal(err)
	}
	list, err := client.ZScan(context.Background(), &immuschema.ZScanRequest{
		Set: []byte(`mySortedSet`),
	})

	fmt.Printf("%v\n", list)

}
