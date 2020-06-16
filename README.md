<div align="center">
  <img src="docs/stark-logo-with-text.png?raw=true?" alt="stark-logo" width="250">
  <h3>Sequence Transmission And Record Keeping</h3>
  <hr>
  <a href="https://travis-ci.org/will-rowe/stark"><img src="https://travis-ci.org/will-rowe/stark.svg?branch=master" alt="travis"></a>
  <a href="https://godoc.org/github.com/will-rowe/stark"><img src="https://godoc.org/github.com/will-rowe/stark?status.svg" alt="GoDoc"></a>
  <a href="https://goreportcard.com/report/github.com/will-rowe/stark"><img src="https://goreportcard.com/badge/github.com/will-rowe/stark" alt="goreportcard"></a>
  <a href="https://codecov.io/gh/will-rowe/stark"><img src="https://codecov.io/gh/will-rowe/stark/branch/master/graph/badge.svg" alt="codecov"></a>
  <a href='https://stark-docs.readthedocs.io/en/latest/?badge=latest'><img src='https://readthedocs.org/projects/stark-docs/badge/?version=latest' alt='Documentation Status'></a>
</div>

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
- a database is aliased by a `project name` which groups the `records`
- `projects` and `records` are DAG nodes in the [IPFS](https://ipfs.io/)
- DAG `links` are created between `records` and the `projects` that use them
- `records` and `projects` are pointed to by `content identifiers (CIDs)`
- the `CIDs` change when the content they point to is altered, so databases track them locally using `keys`
- databases are re-opened and shared using the `project` `CID` (termed a `snapshot`)

### Records

- `records` are a data structure used to represent a Nanopore sequencing run (but can be hijacked and extended to be more generic or to represent Samples and Libraries)
- `records` are defined in [protobuf](https://developers.google.com/protocol-buffers) format (which is compiled with Go bindings using [this makefile](./schema/Makefile))
- currently, `records` are serialised to JSON for IPFS transactions

## Installation

### Requirements

Both the Go package and the Command Line Utility require `go-ipfs`. See download and install instructions [here](https://docs.ipfs.io/guides/guides/install/).

### As a Go package

```sh
go get -d github.com/will-rowe/stark
```

### As an app

TODO

## Documentation

View the [Go Documentation](https://pkg.go.dev/github.com/will-rowe/stark) site for package documentation or visit [readthedocs](https://stark-docs.readthedocs.io/en/latest/) pages for more information on the app.
