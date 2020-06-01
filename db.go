package stark

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"

	"github.com/dgraph-io/badger"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/pkg/errors"
)

const (

	// DefaultProject is the default project name used if none provided to the OpenDB function.
	DefaultProject = "starkdb-default-project"

	// DefaultLocalDbLocation is used if the user does not provide one.
	DefaultLocalDbLocation = "/tmp/starkDB/"

	// DefaultStarkEnvVariable is the env variable starkdb looks for when told to use encryption.
	DefaultStarkEnvVariable = "STARK_DB_ENCRYPTION_KEY"

	// Ienc is the format in which the data will be added to the IPFS DAG.
	Ienc = "json"

	// Format is the format of the input data.
	Format = "cbor"

	// MhType is the hash to use for DAG put operations.
	MhType = uint64(math.MaxUint64) // use default hash (sha256 for cbor, sha1 for git..)

)

var (

	// ErrNoProject indicates no project name was given.
	ErrNoProject = fmt.Errorf("project name is required for a starkDB")

	// ErrDbOption is issued for incorrect database initialisation options.
	ErrDbOption = fmt.Errorf("starkDB option could not be set")

	// ErrNewDb is issued when NewDb fails.
	ErrNewDb = fmt.Errorf("could not initialise a starkDB")

	// ErrNoEnvSet is issued when no env variable is found.
	ErrNoEnvSet = fmt.Errorf("no private key found in %s", DefaultStarkEnvVariable)

	// ErrEncryptKey is issued when the provided encyption key doesn't meet requirements.
	ErrEncryptKey = fmt.Errorf("cannot load private key")

	// ErrExistingRecord indicates a record with matching UUID is already in the IPFS and has a more recent update timestamp.
	ErrExistingRecord = fmt.Errorf("cannot replace a record in starkDB with an older version")

	// ErrKeyNotFound is issued during a Get request when the key is not present in the local keystore.
	ErrKeyNotFound = fmt.Errorf("key not found in the database")

	// ErrNodeFormat is issued when a CID points to a node with an unsupported format.
	ErrNodeFormat = fmt.Errorf("database entry points to a non-CBOR node")

	// ErrNodeOffline indicates the node is offline.
	ErrNodeOffline = fmt.Errorf("IPFS node is offline")

	// ErrNodeOnline indicates the node is online.
	ErrNodeOnline = fmt.Errorf("IPFS node is online")

	// ErrNoPeerID indicates the IPFS node has no peer ID.
	ErrNoPeerID = fmt.Errorf("no PeerID listed for the current IPFS node")

	// ErrLinkExists indicates a record is already linked to the provided UUID.
	ErrLinkExists = fmt.Errorf("record already linked to the provided UUID")
)

// DbOption is a wrapper struct used to pass functional
// options to the starkDB constructor.
type DbOption func(DB *DB) error

// SetProject is an option setter for the OpenDB
// constructor that sets the project for the
// database.
func SetProject(project string) DbOption {
	return func(DB *DB) error {
		return DB.setProject(project)
	}
}

// SetLocalStorageDir is an option setter for the OpenDB
// constructor that sets the path to the local keystore.
func SetLocalStorageDir(path string) DbOption {
	return func(DB *DB) error {
		return DB.setLocalStorage(path)
	}
}

// SetEncryption is an option setter for the OpenDB constructor
// that tells starkdb to make encrypted writes to IPFS using the
// private key in STARK_DB_ENCRYPTION_KEY env variable.
func SetEncryption(val bool) DbOption {
	return func(DB *DB) error {
		return DB.setEncryption(val)
	}
}

// WithPinning is an option setter that specifies the IPFS
// node pin entries.
//
// Note: If not provided to the constructor, the node will
// not pin entries.
func WithPinning() DbOption {
	return func(DB *DB) error {
		return DB.setPinning(true)
	}
}

// WithAnnounce is an option setter for the OpenDB constructor
// that sets the database to announce new records via PubSub.
func WithAnnounce() DbOption {
	return func(DB *DB) error {
		return DB.setAnnounce(true)
	}
}

// DB is the starkDB database.
type DB struct {
	lock      sync.Mutex // protects access to the bound IPFS node and badger db
	ctx       context.Context
	ctxCancel context.CancelFunc

	// user-defined settings
	project      string // the project which the database instance is managing
	keystorePath string // local keystore location
	pinning      bool   // if true, IPFS IO will be done with pinning
	announce     bool   // if true, new records added to the IPFS will be broadcast on the pubsub topic for this project
	allowNetwork bool   // controls the IPFS node's network connection // TODO: not yet implemented (thinking of local dbs)
	privateKey   []byte // private key for encrypted DB instances // TODO: not yet implemented (thinking of encrypted dbs)

	// local storage
	keystore *badger.DB // local keystore to relate record UUIDs to IPFS CIDs

	// IPFS
	ipfsClient *client // wraps the IPFS core API

	// PubSub
	pubsubSub      icore.PubSubSubscription // the pubsub subscription
	pubsubMessages chan icore.PubSubMessage // used to receive pubsub messages
	pubsubStop     chan struct{}            // used to signal the pubsub goroutine to end
	pubsubStopped  chan struct{}            // used to signal the pubsub goroutine has ended
}

// OpenDB opens a new instance of starkDB.
//
// If there is an existing database in the specified local
// storage location, which has the specified project name,
// the DB will open that.
//
// It returns the initialised database, a teardown function
// and any error encountered.
func OpenDB(options ...DbOption) (*DB, func() error, error) {

	// context for the lifetime of the DB
	ctx, cancel := context.WithCancel(context.Background())

	// create the uninitialised DB
	starkDB := &DB{
		ctx:       ctx,
		ctxCancel: cancel,
		project:   DefaultProject,

		// defaults
		pinning:  false,
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

	// init the IPFS client
	client, err := newIPFSclient(ctx)
	if err != nil {
		return nil, nil, err
	}
	starkDB.ipfsClient = client

	// setup the PubSub if requested
	if starkDB.announce {
		//todo
	}

	// return the teardown so we can ensure it happens
	return starkDB, starkDB.teardown, nil
}

// IsOnline returns true if the starkDB is in online mode and the IPFS daemon is reachable.
func (DB *DB) IsOnline() bool {
	DB.lock.Lock()
	allowNetwork := DB.allowNetwork
	DB.lock.Unlock()
	return DB.ipfsClient.node.IsOnline && allowNetwork
}

// GetNodeIdentity returns the PeerID of the underlying IPFS node for the starkDB.
func (DB *DB) GetNodeIdentity() (string, error) {
	if !DB.IsOnline() {
		return "", ErrNodeOffline
	}
	DB.lock.Lock()
	defer DB.lock.Unlock()
	if len(DB.ipfsClient.node.Identity) == 0 {
		return "", ErrNoPeerID
	}
	return DB.ipfsClient.node.Identity.Pretty(), nil
}

// setProject will set the database project.
func (DB *DB) setProject(project string) error {

	// sanitize the project name
	project = strings.ReplaceAll(project, " ", "_")
	if len(project) == 0 {
		return ErrNoProject
	}

	// set it
	DB.project = project
	return nil
}

// setLocalStorage will check if a directory exists,
// try and make it if not, then set the field on
// IPFSnode.
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

// setPinning sets the underlying IPFS node to
// pin entries.
func (DB *DB) setPinning(pin bool) error {
	DB.pinning = pin
	return nil
}

// setAnnounce sets the database to announce
// new records via PubSub.
func (DB *DB) setAnnounce(announce bool) error {
	DB.announce = announce
	return nil
}

// setEncryption tells starkdb to make encrypted
// writes.
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

// teardown will close down all the open guff
// nicely.
func (DB *DB) teardown() error {
	DB.lock.Lock()
	DB.lock.Unlock()

	// close the local keystore
	if err := DB.keystore.Close(); err != nil {
		return err
	}

	// cancel the db context
	DB.ctxCancel()

	// close any currently running plugins
	if err := DB.ipfsClient.endSession(); err != nil {
		return err
	}

	// check the node is offline
	if DB.IsOnline() {
		return ErrNodeOnline
	}
	return nil
}
