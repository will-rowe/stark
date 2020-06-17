# About

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
