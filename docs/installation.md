# Installation

### Requirements

Both the package and the app require `go-ipfs`. See download and install instructions [here](https://docs.ipfs.io/guides/guides/install/).

### Install

#### option 1: use Go

Both the app and the package can be installed at the same time using the Go toolchain.

- To install a tagged release:

```sh
export GO111MODULE=on
release=0.0.1
go get -v github.com/will-rowe/stark/...@$(release)
```

- To install the latest master:

```sh
export GO111MODULE=on
go get -v github.com/will-rowe/stark/...@master
```

#### option 2: use a release

Download a release for your platform from the GitHub [releases page](https://github.com/will-rowe/stark/releases).

#### option 3: use Conda

tbc
