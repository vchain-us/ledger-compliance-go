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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	sdk "github.com/vchain-us/ledger-compliance-go/grpcclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"io/ioutil"
	"log"
	"time"
)

func main() {

	tlsCredentials, err := loadTLSCertificate()

	if err != nil {
		log.Fatal("cannot load TLS credentials: ", err)
	}

	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(tlsCredentials),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                20 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	client := sdk.NewLcClient(sdk.ApiKey("jxnqdxcicasvajpfxldqdanquhabiqcjrwdb"), sdk.Host("localhost"), sdk.Port(3324), sdk.DialOptions(dialOptions))

	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}

	index, err := client.SafeSet(context.Background(), []byte(`key`), []byte(`val`))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("index:\t\t%v\nverified:\t%t\n\n", index.Index, index.Verified)
	item, err := client.SafeGet(context.Background(), []byte(`key`))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("My item:\nkey:\t\t%s\nval:\t\t%s\nindex:\t\t%d\ntimestamp:\t%v\nverified:\t%t\n", item.Key, item.Value, item.Index, item.Time, item.Verified)
}

func loadTLSCertificate() (credentials.TransportCredentials, error) {
	cert, err := ioutil.ReadFile("cert/ca-cert.pem")
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(cert) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}
	config := &tls.Config{
		RootCAs: certPool,
	}
	return credentials.NewTLS(config), nil
}
