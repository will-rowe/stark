<div align="center">
  <h1>STARK</h1>
  <h3>a Command Line Utility and IPFS-backed database for recording and distributing sequencing data</h3>
  <hr>
  <a href="https://travis-ci.org/will-rowe/stark"><img src="https://travis-ci.org/will-rowe/stark.svg?branch=master" alt="travis"></a>
  <a href="https://godoc.org/github.com/will-rowe/stark/starkdb"><img src="https://godoc.org/github.com/will-rowe/stark?status.svg" alt="GoDoc"></a>
  <a href="https://goreportcard.com/report/github.com/will-rowe/stark"><img src="https://goreportcard.com/badge/github.com/will-rowe/stark" alt="goreportcard"></a>
  <a href="https://codecov.io/gh/will-rowe/stark"><img src="https://codecov.io/gh/will-rowe/stark/branch/master/graph/badge.svg" alt="codecov"></a>
</div>

---

> WIP: The database works and the API is relatively stable. The CLI is in development.

> proposed backronym: STARK - Sequence Transmission And Record Keeping

---

## Overview

**stark** is a Command Line Utility for running and interacting with a **starkDB**.

**starkdb** is a Go package for an IPFS-backed database for recording and distributing sequencing data.

### The database

- **starkDB** tracks, updates and distributes `keys` and `records`
- `records` are kept as DAG nodes in the [IPFS](https://ipfs.io/), which are pointed to by `content identifiers (CIDs)`
- `CIDs` are kept in a persistent local key-value store ([badgerdb](https://github.com/dgraph-io/badger)) and link to the user-defined `keys`

### Records

- `records` are a data structure used to represent a Nanopore sequencing run (but can be hijacked to represent Samples and Libraries too)
- `records` are defined in [protobuf](https://developers.google.com/protocol-buffers) format (which is compiled with Go bindings using [this makefile](./schema/Makefile))

### Requirements

Both the Go package and the Command Line Utility require the `go-ipfs` command line utility. See download and install instructions [here](https://docs.ipfs.io/guides/guides/install/).

## The Go Package

### Install

```sh
go get -d github.com/will-rowe/stark
```

### Usage example

```Go
// This basic program will create a new database, add a record to it and then retrieve a copy of that record.
//
// You can compile and run this example from within the repo:
//   go run examples/database/main.go
//
package main

import (
	"fmt"

	sdb "github.com/will-rowe/stark/starkdb"
)

func main() {

	// init a starkDB
	db, dbCloser, err := sdb.OpenDB("my project", sdb.SetLocalStorageDir("/tmp/starkdb"), sdb.SetEphemeral(false))
	if err != nil {
		panic(err)
	}

	// defer the database closer
	defer dbCloser()

	// create a record
	record, err := sdb.NewRecord(sdb.SetAlias("my first sample"))
	if err != nil {
		panic(err)
	}

	// add record to starkDB
	err = db.Set("lookupKey", record)
	if err != nil {
		panic(err)
	}

	// retrieve record from the starkDB
	retrievedSample, err := db.Get("lookupKey")
	if err != nil {
		panic(err)
	}
	fmt.Println(retrievedSample.GetAlias())

	// you can also view the record in the IPFS
	link, err := db.GetExplorerLink("lookupKey")
	if err != nil {
		panic(err)
	}
	fmt.Printf("view record on the IPFS: %v\n", link)
}
```

### Documentation

View the [Go Documentation]() site for the **starkDB** package documentation. The basic API is:

```
- NewRecord
  - creates a new record to represent a sequencing data object
- OpenDB
  - open a starkdb
- Set
  - add a record to the open starkdb
- Get
  - get a record from the open starkdb
- Snapshot
  - save the current starkdb to the IPFS
- Pull
  - retrieve a starkdb from the IPFS
- Listen
  - links your starkdb with that of another user on the network
  - listens out for records being added to the same project
  - adds detected records to the current starkdb instance
```

## The Command Line Utility

> this is a work in progress....

## Notes

- the `OpenDB` and `NewRecord` consructor functions use functional options to set struct values - this is in an effort to keep the API stable (see [here](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis))
- if a record is retrieved from the database and updated, you need to then re-add it to the database. In other words, **starkDB** only records the most recent version of a record commited to the IPFS
- records have a history, which can be used to rollback changes to other version of the record that entered the IPFS
- even though schema is in protobuf, most of the time it's marshaling to JSON to pass stuff around

## TODO

- the command line tool
- use PubSub to sync/communicate database changes to peers
- work on concurrent access
- work on optional encryption of record fields
- add a rollback feature that exploits the Record's CID history
- library and sample linking are a wip, I think these links should be a list of UUID->CID lookups (so should check validity)
