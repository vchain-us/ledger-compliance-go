# ledger compliance GO sdk [![License](https://img.shields.io/github/license/codenotary/immudb4j)](LICENSE)

[![Slack](https://img.shields.io/badge/join%20slack-%23immutability-brightgreen.svg)](https://slack.vchain.us/)
[![Discuss at immudb@googlegroups.com](https://img.shields.io/badge/discuss-immudb%40googlegroups.com-blue.svg)](https://groups.google.com/group/immudb)
[![Coverage](https://coveralls.io/repos/github/vchain-us/ledger-compliance-go/badge.svg?branch=master)](https://coveralls.io/github/vchain-us/ledger-compliance-go?branch=master)

### Official [ledger compliance] client for GO 1.15 and above.

[ledger compliance]: https://tobedefined.io/


## Contents

- [Introduction](#introduction)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Quickstart](#quickstart)
- [Step by step guide](#step-by-step-guide)
    * [Creating a Client](#creating-a-client)
    * [Traditional read and write](#traditional-read-and-write)
    * [Verified or Safe read and write](#verified-or-safe-read-and-write)
- [TLS](#tls)
- [Contributing](#contributing)

## Introduction

Ledger compliance GO sdk implements a [grpc] ledger compliance client.
Backed by [immudb](https://github.com/codenotary/immudb) it uses several common part that are implemented in the immudb go sdk, but simplifying it more.
Latest validated ledger state may be keep in the local filesystem when using default [cache](https://github.com/codenotary/immudb/blob/master/pkg/client/cache/file_cache.go) implementation,
Please read [immudb research paper] for details of how immutability is ensured by [immudb].

[grpc]: https://grpc.io/
[immudb research paper]: https://immudb.io/
[immudb]: https://immudb.io/

## Prerequisites

## Installation
```bash
go get github.com/vchain-us/ledger-compliance-go
```

## Quickstart

Example can be found in the  [example folder](/examples)

## Step by step guide

To obtain a valid apikey please register on ledger compliance frontend, create a new ledger and retrieve the apikey after following creation wizard.

### Creating a Client

The following code snippets shows how to create a client.

```go
client := sdk.NewLcClient(sdk.ApiKey("myApiKey"), sdk.Host("localhost"), sdk.Port(3324))
err := client.Connect()
if err!=nil{
    return err
}

test, err := client.SafeGet(context.TODO(), []byte(`key`))
if err!=nil{
    return err
}
```

### Traditional read and write

Ledger compliance provides read and write operations that behave as a traditional
key-value store i.e. no cryptographic verification is done. This operations
may be used when validations can be post-poned:

```go
index, err := client.Set(ctx, []byte(`key`), []byte(`val`))
if err!=nil{
    return err
}
item, err := client.Get(ctx, []byte(`key`))
if err!=nil{
    return err
}
```

### Verified or Safe read and write

Ledger compliance provides built-in cryptographic verification for any entry. The client
implements the mathematical validations while the application uses as a traditional
read or write operation:

```go
index, err := client.SafeSet(ctx, []byte(`key`), []byte(`val`))
if err!=nil{
    return err
}
test, err := client.SafeGet(ctx, []byte(`key`))
if err!=nil{
    return err
}
```

## TLS

To setup a tls connection use a transport credential dial option.
See  [tls example](examples/tls.go)
```go
    dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(tlsCredentials),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                20 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}
	client := sdk.NewLcClient(sdk.ApiKey("jxnqdxcicasvajpfxldqdanquhabiqcjrwdb"), sdk.Host("localhost"), sdk.Port(3324), sdk.DialOptions(dialOptions))
```

## Contributing

We welcome contributions. Feel free to join the team!

To report bugs or get help, use [GitHub's issues].

[GitHub's issues]: https://github.com/vchain-us/ledger-compliance-go/issues
