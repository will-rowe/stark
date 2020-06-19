# Using stark as an app

## Overview

The app is used to manage stark databases. Here are a few key points:

* a database must be open in order to run `add`/`get`/`dump`
* one database can be open at a time
* an open database is interfaced by a [gRPC](https://grpc.io/docs/what-is-grpc/introduction/) server
* `records` are passed to and from the database via protobuf messages
* `records` are added and retrieved using a `key`, which is the `record's` `alias` field.
* to keep track of projects locally, `stark` has a config file which stores the most recent database `snapshot` (default location: `~/.stark.json`)

## Subcommands

- `stark open <project>` - Open a database for a `project`.
- `stark add` - Add a `record` to an open database.
- `stark get <key>` - Get a `record` from an open database.
- `stark dump` - Dump the current metadata from an open database.

***

### Open

To open a database, just use the `open` subcommand:

```sh
stark open my-project
```

This will check the stark config file and see if a database has been opened for this `project` before:

- if the `project` is found it will recover the most recent `snapshot CID` for this `project` and then collect all the `record` links in a key-value store
- if the `project` is not found, `stark` will open a new database and add it to the config for next time
- it's easiest to open the database in one terminal window and then run `add` and `get` in another

#### Flags

`--withListen`

- tells the database to listen for `records` being added to other database instances for the same `project`
- for instance, if I had a database open for **metagenomics-project-101** and a collaborator also had a database open with this `project` name, my database instance could pull in all `records` that my collaborator was adding to their database (provided they were using the `--withAnnounce` flag)
- this works best if `--withPeers` is used to connect the two databases directly

`--withAnnounce`

- this is used to announce `records` as they are added to the database
- announced `records` can be picked up by databases that are listening (via the `--withListen` flag)

`--withEncrypt`

- encrypts `record` fields when adding a `record` to the database
- this flag must also be used to get encrypted `records`
- if you try to `get` an encrypted `record` without this flag, the `get` will fail
- to provide the encryption password, use the `STARK_DB_PASSWORD` environment variable

`--withPinata <int>`

`--withPeers <string>`

***

### Add

To add a `record` to an open database:

```sh
cat record.json | stark add
```

- the `record` must follow the schema or the `add` will fail
- the `record` alias is used as the database `key`, which is needed for `record` retrieval
- if no STDIN or file is provided, the `add` subcommand will collect the `record` interactively using a user prompt (this is a WIP)

#### Flags

`--useProto`

- tells the database that the `record` being added is in protobuf format, not JSON

`--inputFile <string>`

- use this flag to provide the `record` via file instead of STDIN or interactively

***

### Get

To get a `record` from an open database:

```sh
stark get <key>
```

#### Flags

`--humanReadable`

- use this flag to print the `record` as human readable text

`--useProto`

- tells the database return the `record` in protobuf format, not JSON

***

### Dump

To dump the current metadata from an open database:

```sh
stark dump
```