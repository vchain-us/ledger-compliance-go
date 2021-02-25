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
	"log"

	sdk "github.com/vchain-us/ledger-compliance-go/grpcclient"
)

func main() {
	client := sdk.NewLcClient(
		sdk.ApiKey("rfqjwunwrceixjvxcpdcxhyztdughmtibepw"),
		sdk.Host("localhost"),
		sdk.Port(3324))
	err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}
	fileKey := []byte("fileKey")

	inputFilePath := "./some-input-file.txt"
	if _, err = client.SetFile(context.Background(), fileKey, inputFilePath); err != nil {
		log.Fatal(err)
	}
	log.Printf(
		"SetFile SUCCESSful:\n   content from local file \"%s\" has been written to LC with key \"%s\"",
		inputFilePath, fileKey)

	outputFilePath := "./some-output-file.txt"
	if _, err = client.GetFile(context.Background(), fileKey, outputFilePath); err != nil {
		log.Fatal(err)
	}
	log.Printf(
		"GetFile SUCCESSful:\n   file content from LC key \"%s\" written to local file \"%s\"",
		fileKey, outputFilePath)
}
