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
	client := sdk.NewLcClient(sdk.ApiKey("vipicevtthnqiveufonnkhnoshkoyrbvpeyj"), sdk.Host("localhost"), sdk.Port(3324))
	err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}
	_, err = client.SafeSet(context.Background(), []byte(`aaa`), []byte(`val1`))
	if err != nil {
		log.Fatal(err)
	}
	_, err = client.SafeSet(context.Background(), []byte(`aab`), []byte(`val2`))
	if err != nil {
		log.Fatal(err)
	}
	_, err = client.SafeSet(context.Background(), []byte(`abb`), []byte(`val3`))
	if err != nil {
		log.Fatal(err)
	}
	_, err = client.SafeSet(context.Background(), []byte(`abb`), []byte(`val4`))
	if err != nil {
		log.Fatal(err)
	}

	list, err := client.Scan(context.Background(), &immuschema.ScanOptions{Prefix: []byte(`aa`)})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v\n", list)
	list, err = client.History(context.Background(), &immuschema.HistoryOptions{Key: []byte(`abb`)})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v\n", list)
}
