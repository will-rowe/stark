<div align="center">
  <img src="https://raw.githubusercontent.com/will-rowe/stark/master/docs/stark-logo-with-text.png" alt="stark-logo" width="250">
  <h3>Sequence Transmission And Record Keeping</h3>
  <hr>
  <a href="https://travis-ci.org/will-rowe/stark"><img src="https://travis-ci.org/will-rowe/stark.svg?branch=master" alt="travis"></a>
  <a href="https://godoc.org/github.com/will-rowe/stark"><img src="https://godoc.org/github.com/will-rowe/stark?status.svg" alt="GoDoc"></a>
  <a href="https://goreportcard.com/report/github.com/will-rowe/stark"><img src="https://goreportcard.com/badge/github.com/will-rowe/stark" alt="goreportcard"></a>
  <a href="https://codecov.io/gh/will-rowe/stark"><img src="https://codecov.io/gh/will-rowe/stark/branch/master/graph/badge.svg" alt="codecov"></a>
  <a href='https://stark-docs.readthedocs.io/en/latest/?badge=latest'><img src='https://readthedocs.org/projects/stark-docs/badge/?version=latest' alt='Documentation Status'></a>
</div>

## Overview

**stark** is an IPFS-backed database for recording and distributing sequencing data. It is both an application and a Go Package for running and interacting with **stark databases**. Features include:

- snapshot, sync and share entire databases over the IPFS
- use PubSub messaging to share and collect data records as they are created
- track record history and rollback revisions (rollback feature WIP)
- attach and sync files to records (WIP)
- encrypt record fields
- submit databases to [pinata](https://pinata.cloud/) pinning service for easy backup and distribution

## Quickstart

### Requirements

Both the app and the Go package require IPFS (specifically, the Go implementation: `go-ipfs`). See download and install instructions [here](https://docs.ipfs.io/guides/guides/install/).

Then make sure you have an IPFS repository initialised on your machine.

### Install

The easiest way to install is using Go (v1.14):

```sh
export GO111MODULE=on
release=0.0.0
go get -v github.com/will-rowe/stark/...@$(release)
```

### Usage

For using the Go package, see the [Go documentation](https://pkg.go.dev/github.com/will-rowe/stark) and the [examples](https://stark-docs.readthedocs.io/en/latest/package/#usage-example).

The following are some basic commands for using the **stark** app:

* Use the `open` subcommand to open a database and serve it via [gRPC](https://grpc.io/docs/what-is-grpc/introduction/):

```
stark open my-project
```

* Use the `add` subcommand to add a [Record](https://stark-docs.readthedocs.io/en/latest/about/#records) to an open database:

```
stark add -f record.json
```

* Or use the `add` subcommand with no arguments to create a Record interactively and then add it: 

```
stark add
```

* Use the `get` subcommand to retrieve a Record from an open database:

```
stark get my-record-key -H > record.json
```

## Documentation

Visit the [stark documentation site](https://stark-docs.readthedocs.io/en/latest/) and the [Go package documentation](https://pkg.go.dev/github.com/will-rowe/stark) for more information.