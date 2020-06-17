/*Package stark is an IPFS-backed database for distributed
Sequence Recording And Record Keeping.

It is both a library and a Command Line Utility for running
and interacting with `stark databases`.

Features include:

- snapshot and sync entire databases over the IPFS

- use PubSub messaging to share and collect data records as they are created

- track record history and rollback revisions (rollback feature WIP)

- attach and sync files to records (WIP)

- encrypt record fields

*/
package stark // import "github.com/will-rowe/stark"

import (
	"context"
	"fmt"
	"sync"

	starkipfs "github.com/will-rowe/stark/src/ipfs"
)

const (

	// DefaultBufferSize is the maximum number of records stored in channels.
	DefaultBufferSize = 42

	// DefaultMinBootstrappers is the minimum number of reachable bootstrappers required.
	DefaultMinBootstrappers = 3

	// DefaultProject is the default project name used if none provided to the OpenDB function.
	DefaultProject = "starkDB-default-project"

	// DefaultPinataAPIkey is the env variable for the pinata API.
	DefaultPinataAPIkey = "PINATA_API_KEY"

	// DefaultPinataSecretKey is the env variable for the pinata secret key.
	DefaultPinataSecretKey = "PINATA_SECRET_KEY"

	// DefaultStarkEnvVariable is the env variable starkDB looks for when told to use encryption.
	DefaultStarkEnvVariable = "STARK_DB_PASSWORD"
)

var (

	// ErrAttemptedOverwrite indicates a starkDB key is already in use for a Record with non-matching UUID.
	ErrAttemptedOverwrite = fmt.Errorf("starkDB key is already in use for a Record with non-matching UUID")

	// ErrAttemptedUpdate indicates a Record with matching UUID is already in the IPFS and has a more recent update timestamp.
	ErrAttemptedUpdate = fmt.Errorf("cannot update a Record in starkDB with an older version")

	// ErrBootstrappers is issued when not enough bootstrappers are accessible.
	ErrBootstrappers = fmt.Errorf("not enough bootstrappers found (minimum required: %d)", DefaultMinBootstrappers)

	// ErrCipherPasswordMismatch is issued when a password does not decrypt a Record.
	ErrCipherPasswordMismatch = fmt.Errorf("provided password cannot decrypt Record")

	// ErrDbOption is issued for incorrect database initialisation options.
	ErrDbOption = fmt.Errorf("starkDB option could not be set")

	// ErrEncrypted is issued when an encryption is attempted on an encrypted Record.
	ErrEncrypted = fmt.Errorf("data is encrypted with passphrase")

	// ErrInvalidSnapshot indicates a snapshotted IPFS DAG node can't be accessed.
	ErrInvalidSnapshot = fmt.Errorf("cannot access the database snapshot")

	// ErrLinkExists indicates a Record is already linked to the provided UUID.
	ErrLinkExists = fmt.Errorf("Record already linked to the provided UUID")

	// ErrNoCID indicates no CID was provided.
	ErrNoCID = fmt.Errorf("no CID was provided")

	// ErrNoKey indicates no record key was provided.
	ErrNoKey = fmt.Errorf("no key could be found, make sure Record Alias is set")

	// ErrNotFound indicates a key was not found in the starkDB.
	ErrNotFound = func(key string) error {
		return fmt.Errorf("key not found: %v", key)
	}

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

	// ErrNoPeers indicates no peers were proided but announcing via PubSub was requested.
	ErrNoPeers = fmt.Errorf("no peers given but PubSub announcements requested, chances of messages being received is low")

	// ErrPinataAPI indicates the Pinata API can't be reached.
	ErrPinataAPI = func(err error) error {
		return fmt.Errorf("failed to reach Pinata API: %w", err)
	}

	// ErrPinataOpt is issued for a db option pinning conflict.
	ErrPinataOpt = fmt.Errorf("can't use WithPinata when WithNoPinning")

	// ErrPinataKey is issued when the no env variable for the Pinata API key is set.
	ErrPinataKey = fmt.Errorf("no %s environment variable found", DefaultPinataAPIkey)

	// ErrPinataSecret is issued when the no env variable for the Pinata secret is set.
	ErrPinataSecret = fmt.Errorf("no %s environment variable found", DefaultPinataSecretKey)

	// ErrRecordHistory indicates two Records with the same UUID a gap in their history.
	ErrRecordHistory = fmt.Errorf("both Records share UUID but have a gap in their history")

	// ErrSnapshotUpdate is issued when a link can't be made between the new Record and existing project base node.
	ErrSnapshotUpdate = fmt.Errorf("could not update database snapshot")
)

// Db is the starkDB database.
type Db struct {
	sync.RWMutex // protects access to the bound IPFS node
	ctx          context.Context
	ctxCancel    context.CancelFunc
	ipfsClient   *starkipfs.Client // wraps the IPFS core API, node and PubSub channels
	cidLookup    map[string]string // quick access to Record CIDs using user-supplied keys

	// user-defined settings
	project        string           // the project which the database instance is managing
	peers          []string         // list of addresses to use for IPFS peer discovery
	snapshotCID    string           // the optional snapshot CID provided during database opening
	pinning        bool             // if true, IPFS IO will be done with pinning
	pinataInterval int              // the number of set operations permitted between pinata pinning (-1 = no pinata pinning)
	announcing     bool             // if true, new records added to the IPFS will be broadcast on the pubsub topic for this project
	cipherKey      []byte           // cipher key for encrypted DB instances
	loggingChan    chan interface{} // user provided channel to collect logging info from database internals

	// db stats
	currentNumEntries int // the number of keys in the keystore (checked on db open and then incremented/decremented during Set/Delete ops)
	sessionEntries    int // the number of keys added during the current database instance (not decremented after Delete ops)
}

// DbOption is a wrapper struct used to pass functional
// options to the starkDB constructor.
type DbOption func(starkdb *Db) error

// RecordOption is a wrapper struct used to pass functional
// options to the Record constructor.
type RecordOption func(record *Record) error
