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
	"log"

	sdk "github.com/vchain-us/ledger-compliance-go/grpcclient"
)

func main() {
	client := sdk.NewLcClient(sdk.ApiKey("jxnqdxcicasvajpfxldqdanquhabiqcjrwdb"), sdk.Host("localhost"), sdk.Port(3324))
	err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}
	index, err := client.VerifiedSet(context.Background(), []byte(`key`), []byte(`val`))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("index:\t\t%v\n\n", index.Id)
	item, err := client.VerifiedGet(context.Background(), []byte(`key`))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("My item:\nkey:\t\t%s\nval:\t\t%s\nindex:\t\t%d\n", item.Key, item.Value, item.Tx)
}
