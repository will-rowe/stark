// Package stark is an IPFS-backed database for recording and distributing sequencing data.
package stark

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/dgraph-io/badger"
	"github.com/ipfs/go-ipfs/core"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/pkg/errors"
)

// Defaults
const (

	// DefaultLocalDbLocation is used if the user does not provide one.
	DefaultLocalDbLocation = "/tmp/starkDB/"

	// DefaultStarkEnvVariable is the env variable starkdb looks for when told to use encryption.
	DefaultStarkEnvVariable = "STARK_DB_ENCRYPTION_KEY"
)

// Errors
var (

	// ErrNoProject indicates no project name was given.
	ErrNoProject = fmt.Errorf("project name is required for a starkDB")

	// ErrDbOption is issued for incorrect database initialisation options.
	ErrDbOption = fmt.Errorf("starkDB option could not be set")

	// ErrNewDb is issued when NewDb fails.
	ErrNewDb = fmt.Errorf("could not initialise a starkDB")

	// ErrNoEnvSet is issued when no env variable is found.
	ErrNoEnvSet = fmt.Errorf("no private key found in %s", DefaultStarkEnvVariable)

	// ErrEncyptKey is issued when the provided encyption key doesn't meet requirements.
	ErrEncryptKey = fmt.Errorf("cannot load private key")

	// ErrExistingRecord indicates a record with matching UUID is already in the IPFS and has a more recent update timestamp.
	ErrExistingRecord = fmt.Errorf("cannot replace a record in starkDB with an older version")

	// ErrKeyNotFound is issued during a Get request when the key is not present in the local keystore.
	ErrKeyNotFound = fmt.Errorf("key not found in the database")

	// ErrNodeFormat is issued when a CID points to a node with an unsupported format
	ErrNodeFormat = fmt.Errorf("database entry points to a non-CBOR node")

	// ErrNodeOffline
	ErrNodeOffline = fmt.Errorf("IPFS node is offline")

	// ErrNoPeerID
	ErrNoPeerID = fmt.Errorf("no PeerID listed for the current IPFS node")
)

// SetLocalStorageDir is an option setter for the OpenDB constructor that sets the path to the local keystore.
func SetLocalStorageDir(path string) func(*DB) error {
	return func(DB *DB) error {
		return DB.setLocalStorage(path)
	}
}

// SetEncryption is an option setter for the OpenDB constructor that tells starkdb to make encrypted writes to IPFS using the private key in STARK_DB_ENCRYPTION_KEY env variable.
func SetEncryption(val bool) func(*DB) error {
	return func(DB *DB) error {
		return DB.setEncryption(val)
	}
}

// SetEphemeral is an option setter for the OpenDB constructor that sets the underlying IPFS node to be emphermeral.
func SetEphemeral(val bool) func(*DB) error {
	return func(DB *DB) error {
		return DB.setEphemeral(val)
	}
}

// SetPin is an option setter for the OpenDB constructor that sets the underlying IPFS node to pin entries.
func SetPin(val bool) func(*DB) error {
	return func(DB *DB) error {
		return DB.setPinning(val)
	}
}

// SetAnnounce is an option setter for the OpenDB constructor that sets the database to announce new records via PubSub.
func SetAnnounce(val bool) func(*DB) error {
	return func(DB *DB) error {
		return DB.setAnnounce(val)
	}
}

// DB is the starkDB database.
type DB struct {

	// user-defined settings
	project      string // the project which the database instance is managing
	keystorePath string // local keystore location
	ephemeral    bool   // if true, the IPFS node will be emphemeral
	pinning      bool   // if true, IPFS IO will be done with pinning
	announce     bool   // if true, new records added to the IPFS will be broadcast on the pubsub topic for this project
	allowNetwork bool   // controls the IPFS node's network connection // TODO: not yet implemented (thinking of local dbs)
	privateKey   []byte // private key for encrypted DB instances // TODO: not yet implemented (thinking of encrypted dbs)

	// local storage
	keystore *badger.DB // local keystore to relate record UUIDs to IPFS CIDs

	// IPFS
	ipfsCoreAPI icore.CoreAPI  // the IPFS interface
	ipfsNode    *core.IpfsNode // a pointer to the underlying bound IPFS node

	// PubSub
	pubsubSub      icore.PubSubSubscription // the pubsub subscription
	pubsubMessages chan icore.PubSubMessage // used to receive pubsub messages
	pubsubStop     chan struct{}            // used to signal the pubsub goroutine to end
	pubsubStopped  chan struct{}            // used to signal the pubsub goroutine has ended

	// helpers
	ctx        context.Context    // context for IPFS calls etc.
	ctxCancel  context.CancelFunc // cancel function for the context
	sync.Mutex                    // protects access to the bound IPFS node
}

// OpenDB opens a new instance of starkDB.
// It returns the initialised database, a teardown function and any error encountered.
func OpenDB(project string, options ...func(*DB) error) (*DB, func() error, error) {
	if len(project) == 0 {
		return nil, nil, ErrNoProject
	}

	// sanitize the project name
	project = strings.ReplaceAll(project, " ", "_")

	// create the uninitialised DB
	starkDB := &DB{
		project: project,

		// defaults
		pinning:  true,
		announce: false,

		// add in the currently unsettable options
		allowNetwork: true,
	}

	// add the provided options
	for _, option := range options {
		err := option(starkDB)
		if err != nil {
			return nil, nil, errors.Wrap(err, ErrDbOption.Error())
		}
	}

	// setup the local keystore
	if len(starkDB.keystorePath) == 0 {
		starkDB.keystorePath = DefaultLocalDbLocation
	}
	dirPath := fmt.Sprintf("%s/%s", starkDB.keystorePath, starkDB.project)
	badgerOpts := badger.DefaultOptions(dirPath).WithLogger(nil)
	ldb, err := badger.Open(badgerOpts)
	if err != nil {
		return nil, nil, errors.Wrap(err, ErrNewDb.Error())
	}
	starkDB.keystore = ldb

	// get some context
	starkDB.ctx, starkDB.ctxCancel = context.WithCancel(context.Background())

	// setup the IPFS Core API
	starkDB.ipfsCoreAPI, starkDB.ipfsNode, err = setupIPFS(starkDB.ctx, starkDB.ephemeral)
	if err != nil {
		starkDB.teardown()
		return nil, nil, errors.Wrap(err, ErrNewDb.Error())
	}

	// return the teardown so we can ensure it happens
	return starkDB, starkDB.teardown, nil
}

/////////////////////////
// Exported methods:

// IsOnline returns true if the starkDB is in online mode and the IPFS daemon is reachable.
func (DB *DB) IsOnline() bool {
	DB.Lock()
	allowNetwork := DB.allowNetwork
	DB.Unlock()
	return DB.ipfsNode.IsOnline && DB.ipfsNode.IsDaemon && allowNetwork
}

// GetNodeIdentity returns the PeerID of the underlying IPFS node for the starkDB.
func (DB *DB) GetNodeIdentity() (string, error) {
	if !DB.IsOnline() {
		return "", ErrNodeOffline
	}
	if len(DB.ipfsNode.Identity) == 0 {
		return "", ErrNoPeerID
	}
	return DB.ipfsNode.Identity.Pretty(), nil
}

/////////////////////////
// Unexported methods:

// setLocalStorage will check if a directory exists, try and make it if not, then set the field on IPFSnode
func (DB *DB) setLocalStorage(path string) error {
	if len(path) == 0 {
		return fmt.Errorf("no path provided for local database")
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("can't access adirectory (check permissions): %v", path)
		}
	}
	DB.keystorePath = path
	return nil
}

// setEphemeral sets the underlying IPFS node to be emphermeral.
func (DB *DB) setEphemeral(ephemeral bool) error {
	DB.ephemeral = ephemeral
	return nil
}

// setPinning sets the underlying IPFS node to pin entries.
func (DB *DB) setPinning(pin bool) error {
	DB.pinning = pin
	return nil
}

// setAnnounce sets the database to announce new records via PubSub.
func (DB *DB) setAnnounce(announce bool) error {
	DB.announce = announce
	return nil
}

// setEncryption tells starkdb to make encrypted writes.
func (DB *DB) setEncryption(val bool) error {

	// check for the env variable
	encryptKey, exists := os.LookupEnv(DefaultStarkEnvVariable)
	if !exists {
		return ErrNoEnvSet
	}

	// get the key
	key, err := hex.DecodeString(encryptKey)
	if err != nil {
		return errors.Wrap(err, ErrEncryptKey.Error())
	}

	// check key is correct length
	if len(key) != 32 {
		return errors.Wrap(fmt.Errorf("encrypt key must be 32 bytes"), ErrEncryptKey.Error())
	}

	// set the key
	DB.privateKey = key
	return nil
}

// teardown will close down all the open guff nicely
func (DB *DB) teardown() error {

	// TODO: work on this some more once the API has taken shape

	// end the context
	DB.ctxCancel()

	// close the local db and return any error
	return DB.keystore.Close()
}
