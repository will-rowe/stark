<div align="center">
  <h1>STARK</h1>
  <h3>Sequence Transmission And Record Keeping</h3>
  <hr>
  <a href="https://travis-ci.org/will-rowe/stark"><img src="https://travis-ci.org/will-rowe/stark.svg?branch=master" alt="travis"></a>
  <a href="https://godoc.org/github.com/will-rowe/stark"><img src="https://godoc.org/github.com/will-rowe/stark?status.svg" alt="GoDoc"></a>
  <a href="https://goreportcard.com/report/github.com/will-rowe/stark"><img src="https://goreportcard.com/badge/github.com/will-rowe/stark" alt="goreportcard"></a>
  <a href="https://codecov.io/gh/will-rowe/stark"><img src="https://codecov.io/gh/will-rowe/stark/branch/master/graph/badge.svg" alt="codecov"></a>
</div>

> WIP: The database works and the API is relatively stable. The CLI is in development.

## Overview

**stark** is an IPFS-backed database for recording and distributing sequencing data. It is both a library and a Command Line Utility for running and interacting with **stark databases**. Features include:

- snapshot, sync and share entire databases over the IPFS
- use PubSub messaging to share and collect data records as they are created
- track record history and rollback revisions (rollback feature WIP)
- attach and sync files to records (WIP)
- encrypt record fields
- submit databases to [pinata](https://pinata.cloud/) pinning service for easy backup and distribution

### The database

- **stark databases** track, update and share sequence `records`
- the database groups `records` by `projects`, databases can contain `records` from other `projects`
- `projects` and `records` are DAG nodes in the [IPFS](https://ipfs.io/)
- DAG `links` are created between `records` and the `projects` that use them
- `records` and `projects` are pointed to by `content identifiers (CIDs)`
- the `CIDs` change when the content they point to is altered, so databases track them locally using `keys`
- databases are re-opened and shared using the `project` `CID` (termed a `snapshot`)

### Records

- `records` are a data structure used to represent a Nanopore sequencing run (but can be hijacked and extended to be more generic or to represent Samples and Libraries)
- `records` are defined in [protobuf](https://developers.google.com/protocol-buffers) format (which is compiled with Go bindings using [this makefile](./schema/Makefile))
- currently, `records` are serialised to JSON for IPFS transactions

### Requirements

Both the Go package and the Command Line Utility require `go-ipfs`. See download and install instructions [here](https://docs.ipfs.io/guides/guides/install/).

## The Go Package

View the [Go Documentation](https://pkg.go.dev/github.com/will-rowe/stark) site for the **stark** detailed docs.

### Install

```sh
go get -d github.com/will-rowe/stark
```

### Usage example

```Go
// This basic program will create a new database, add a record to it and then retrieve a copy of that record.
package main

import (
	"fmt"

	"github.com/will-rowe/stark"
)

func main() {

	// init a starkDB
	db, dbCloser, err := stark.OpenDB(stark.SetProject("my project"))
	if err != nil {
		panic(err)
	}

	// defer the database closer
	defer dbCloser()

	// create a record
	record, err := stark.NewRecord(stark.SetAlias("my first sample"))
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

## The Command Line Utility

> this is a work in progress....

## Notes

- each instance of a database is linked to a project, re-opening a database with the same project name will edit that database
- the `OpenDB` and `NewRecord` consructor functions use functional options to set struct values - this is in an effort to keep the API stable (see [here](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis))
- if a record is retrieved from the database and updated, you need to then re-add it to the database. In other words, a **stark database** only records the most recent version of a record commited to the IPFS
- records have a history, which can be used to rollback changes to other version of the record that entered the IPFS
- even though schema is in protobuf, most of the time it's marshaling to JSON to pass stuff around
- Record methods are not threadsafe - the database passes around copies of Records so this isn't much of an issue atm. The idea is that users of the library will end up turning Record data into something more usable and won't operate on them after initial Set/Gets
- Encryption is a WIP, currently only a Record's UUID will be encrypted as a proof of functionality. Encrypted Records are decrypted on retrieval, but this will fail if the database instance requesting them doesn't have the correct password.
