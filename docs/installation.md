# Installation

### Requirements

Both the package and the app require IPFS (specifically, the Go implementation: `go-ipfs`). See download and install instructions [here](https://docs.ipfs.io/guides/guides/install/).

Once you have IPFS installed, make sure that you have a repository initialised (run `ipfs init` on the command line).

### Installing STARK

#### option 1: use Go

Both the app and the package can be installed at the same time using the Go toolchain.

- To install a tagged release:

```sh
export GO111MODULE=on
release=0.0.1
go get -v github.com/will-rowe/stark/...@${release}
```

- To install the latest master:

```sh
export GO111MODULE=on
go get -v github.com/will-rowe/stark/...@master
```

#### option 2: use a release

If you just want the app, download a release for your platform from the GitHub [releases page](https://github.com/will-rowe/stark/releases).

#### option 3: use Conda

tbc
