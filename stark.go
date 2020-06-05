/*Package stark is an IPFS-backed database for distributed
Sequence Recording And Record Keeping.

It is both a library and a Command Line Utility for running
and interacting with **stark databases**.

Features include:

- snapshot and sync entire databases over the IPFS
- use PubSub messaging to share and collect data records as they are created
- track record history and rollback revisions (rollback feature WIP)
- attach and sync files to records (WIP)
- encrypt record fields*/
package stark // import "github.com/will-rowe/stark"

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/dgraph-io/badger" // Db is the starkDB database.
	ma "github.com/multiformats/go-multiaddr"
)

const (

	// DefaultBufferSize is the maximum number of records stored in channels.
	DefaultBufferSize = 42

	// DefaultCipherKeyLength is the required number of bytes for a cipher key.
	DefaultCipherKeyLength = 32

	// DefaultFormat is the format of the input data for IPFS.
	DefaultFormat = "cbor"

	// DefaultIenc is the input encoding for the data will be added to the IPFS DAG.
	DefaultIenc = "json"

	// DefaultLocalDbLocation is used if the user does not provide one.
	DefaultLocalDbLocation = "/tmp/starkDB/"

	// DefaultMaxEntries is the maximum number of keys a starkDB can hold.
	DefaultMaxEntries = 10000

	// DefaultMhType is the multihash to use for DAG put operations.
	DefaultMhType = uint64(math.MaxUint64) // use default hash (sha256 for cbor, sha1 for git..)

	// DefaultMinBootstrappers is the minimum number of reachable bootstrappers required.
	DefaultMinBootstrappers = 3

	// DefaultProject is the default project name used if none provided to the OpenDB function.
	DefaultProject = "starkDB-default-project"

	// DefaultStarkEnvVariable is the env variable starkDB looks for when told to use encryption.
	DefaultStarkEnvVariable = "STARK_DB_PASSWORD"
)

var (

	// ErrBootstrappers is issued when not enough bootstrappers are accessible.
	ErrBootstrappers = fmt.Errorf("not enough bootstrappers found (minimum required: %d)", DefaultMinBootstrappers)

	// ErrDbOption is issued for incorrect database initialisation options.
	ErrDbOption = fmt.Errorf("starkDB option could not be set")

	// ErrEncrypted is issued when an encryption is attempted on an encrypted Record.
	ErrEncrypted = fmt.Errorf("data is encrypted, needs decrypt")

	// ErrCipherKeyMissing is issued when an encrypt/decrypt needed but we don't have a cipher key.
	ErrCipherKeyMissing = fmt.Errorf("no cipher key provided")

	// ErrCipherKeyLength is issued when a key is not long enough.
	ErrCipherKeyLength = fmt.Errorf("cipher key must be %d bytes", DefaultCipherKeyLength)

	// ErrCipherPassword is issued when a cipher key cannot be generated from the provided password.
	ErrCipherPassword = fmt.Errorf("cannot generate cipher key from provided password")

	// ErrCipherPasswordMismatch is issued when a password does not decrypt a Record.
	ErrCipherPasswordMismatch = fmt.Errorf("provided password cannot decrypt Record")

	// ErrExistingRecord indicates a Record with matching UUID is already in the IPFS and has a more recent update timestamp.
	ErrExistingRecord = fmt.Errorf("cannot replace a Record in starkDB with an older version")

	// ErrKeyNotFound is issued during a Get request when the key is not present in the local keystore.
	ErrKeyNotFound = fmt.Errorf("key not found in the database")

	// ErrLinkExists indicates a Record is already linked to the provided UUID.
	ErrLinkExists = fmt.Errorf("Record already linked to the provided UUID")

	// ErrMaxEntriesExceeded inidicates maximum number of entries has been exceeded by starkDB.
	ErrMaxEntriesExceeded = fmt.Errorf("maximum number of entries exceeded by starkDB")

	// ErrNewDb is issued when NewDb fails.
	ErrNewDb = fmt.Errorf("could not initialise a starkDB")

	// ErrNoCID indicates no CID was provided.
	ErrNoCID = fmt.Errorf("no CID was provided")

	// ErrNodeFormat is issued when a CID points to a node with an unsupported format.
	ErrNodeFormat = fmt.Errorf("database entry points to a non-CBOR node")

	// ErrNodeOffline indicates the node is offline.
	ErrNodeOffline = fmt.Errorf("IPFS node is offline")

	// ErrNodeOnline indicates the node is online.
	ErrNodeOnline = fmt.Errorf("IPFS node is online")

	// ErrNoEnvSet is issued when no env variable is found.
	ErrNoEnvSet = fmt.Errorf("no %s environment variable found", DefaultStarkEnvVariable)

	// ErrNoPeerID indicates the IPFS node has no peer ID.
	ErrNoPeerID = fmt.Errorf("no PeerID listed for the current IPFS node")

	// ErrNoProject indicates no project name was given.
	ErrNoProject = fmt.Errorf("project name is required for a starkDB")

	// ErrNoSub indicates the IPFS node is not registered for PubSub.
	ErrNoSub = fmt.Errorf("IPFS node has no topic registered for PubSub")

	// DefaultBootstrappers are nodes used for IPFS peer discovery.
	DefaultBootstrappers = []string{

		// IPFS bootstrapper nodes
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",

		// IPFS cluster pinning nodes
		"/ip4/138.201.67.219/tcp/4001/p2p/QmUd6zHcbkbcs7SMxwLs48qZVX3vpcM8errYS7xEczwRMA",
		"/ip4/138.201.67.219/udp/4001/quic/p2p/QmUd6zHcbkbcs7SMxwLs48qZVX3vpcM8errYS7xEczwRMA",
		"/ip4/138.201.67.220/tcp/4001/p2p/QmNSYxZAiJHeLdkBg38roksAR9So7Y5eojks1yjEcUtZ7i",
		"/ip4/138.201.67.220/udp/4001/quic/p2p/QmNSYxZAiJHeLdkBg38roksAR9So7Y5eojks1yjEcUtZ7i",
		"/ip4/138.201.68.74/tcp/4001/p2p/QmdnXwLrC8p1ueiq2Qya8joNvk3TVVDAut7PrikmZwubtR",
		"/ip4/138.201.68.74/udp/4001/quic/p2p/QmdnXwLrC8p1ueiq2Qya8joNvk3TVVDAut7PrikmZwubtR",
		"/ip4/94.130.135.167/tcp/4001/p2p/QmUEMvxS2e7iDrereVYc5SWPauXPyNwxcy9BXZrC1QTcHE",
		"/ip4/94.130.135.167/udp/4001/quic/p2p/QmUEMvxS2e7iDrereVYc5SWPauXPyNwxcy9BXZrC1QTcHE",
	}
)

// Db is the starkDB database.
type Db struct {
	lock      sync.Mutex // protects access to the bound IPFS node and badger db
	ctx       context.Context
	ctxCancel context.CancelFunc

	// user-defined settings
	project       string         // the project which the database instance is managing
	keystorePath  string         // local keystore location
	bootstrappers []ma.Multiaddr // list of addresses to use for IPFS peer discovery
	snapshotCID   string         // the optional snapshot CID provided during database opening
	pinning       bool           // if true, IPFS IO will be done with pinning
	announcing    bool           // if true, new records added to the IPFS will be broadcast on the pubsub topic for this project
	maxEntries    int            // the maximum number of keys a starkDB instance can hold
	cipherKey     []byte         // cipher key for encrypted DB instances

	// not yet implemented:
	allowNetwork bool // controls the IPFS node's network connection // TODO: not yet implemented (thinking of local dbs)

	// local storage
	keystore          *badger.DB // local keystore to relate record UUIDs to IPFS CIDs
	currentNumEntries int        // the number of keys in the keystore (checked on db open and then incremented/decremented during Set/Delete ops)

	// IPFS
	ipfsClient *client // wraps the IPFS core API, node and PubSub channels
}

// DbOption is a wrapper struct used to pass functional
// options to the starkDB constructor.
type DbOption func(Db *Db) error

// RecordOption is a wrapper struct used to pass functional
// options to the Record constructor.
type RecordOption func(Record *Record) error

// KeyCIDpair wraps the starkDB Key, corresponding
// Record CID and any access error for each entry
// in the starkDB.
//
// It is used by the RangeCIDs method to return a
// copy of each entry in the starkDB.
type KeyCIDpair struct {
	Key   string
	CID   string
	Error error
}
