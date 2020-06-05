package stark

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
)

// SetProject is an option setter for the OpenDB
// constructor that sets the project for the
// database.
func SetProject(project string) DbOption {
	return func(Db *Db) error {
		return Db.setProject(project)
	}
}

// SetLocalStorageDir is an option setter for the OpenDB
// constructor that sets the path to the local keystore.
// It will create the director(y/ies) if not found.
// If not provided, a default path will be used in /tmp
//
// Note: stark will create a project directory in this
// location. This means the same local storage directory
// can be provided to multiple starkDBs.
func SetLocalStorageDir(path string) DbOption {
	return func(Db *Db) error {
		return Db.setLocalStorage(path)
	}
}

// SetBootstrappers is an option setter for the OpenDB
// constructor that sets list of bootstrapper nodes to
// use for IPFS peer discovery.
//
// Note: a default list of bootstrappers will be used
// if this option setter is omitted.
func SetBootstrappers(bootstrapperList []string) DbOption {
	return func(Db *Db) error {
		return Db.setBootstrappers(bootstrapperList)
	}
}

// SetKeyLimit is an option setter for the OpenDB constructor
// that tells starkDB instance the maximum number of keys it
// can hold.
func SetKeyLimit(val int) DbOption {
	return func(Db *Db) error {
		return Db.setKeyLimit(val)
	}
}

// WithNoPinning is an option setter that specifies the IPFS
// node should NOT pin entries.
//
// Note: If not provided to the constructor, the node will
// pin entries by default.
func WithNoPinning() DbOption {
	return func(Db *Db) error {
		return Db.setPinning(false)
	}
}

// WithAnnouncing is an option setter for the OpenDB constructor
// that sets the database to announcing new records via PubSub
// as they are added to the database.
//
// When Set is called and WithAnnouncing is set, the CID of the
// set Record is broadcast on IPFS with the database project
// as the topic.
func WithAnnouncing() DbOption {
	return func(Db *Db) error {
		return Db.setAnnouncing(true)
	}
}

// WithSnapshot is an option setter that opens a database
// and then pulls in an existing database via a snapshot.
//
// Note: If opening an existing database, this will be
// erased in place of the snapshotted database.
func WithSnapshot(snapshotCID string) DbOption {
	return func(Db *Db) error {
		return Db.setSnapshotCID(snapshotCID)
	}
}

// WithEncryption is an option setter for the OpenDB constructor
// that tells starkDB to make encrypted writes to IPFS using the
// password in STARK_DB_PASSWORD env variable.
//
// Note: If existing Records were encrypted, Get operations will
// fail unless this option is set.
func WithEncryption() DbOption {
	return func(Db *Db) error {
		return Db.setEncryption(true)
	}
}

// OpenDB opens a new instance of a starkDB.
//
// If there is an existing database in the specified local
// storage location, which has the specified project name,
// the DB will open that.
//
// It returns the initialised database, a teardown function
// and any error encountered.
func OpenDB(options ...DbOption) (*Db, func() error, error) {

	// context for the lifetime of the DB
	ctx, cancel := context.WithCancel(context.Background())

	// create the uninitialised DB
	starkDB := &Db{
		ctx:       ctx,
		ctxCancel: cancel,

		// defaults
		project:      DefaultProject,
		keystorePath: DefaultLocalDbLocation,
		snapshotCID:  "",
		pinning:      true,
		announcing:   false,
		maxEntries:   DefaultMaxEntries,
		cipherKey:    nil,
		allowNetwork: true, // currently un-implemented
	}

	// add the provided options
	for _, option := range options {
		err := option(starkDB)
		if err != nil {
			return nil, nil, errors.Wrap(err, ErrDbOption.Error())
		}
	}

	// add in the default bootstrappers if none were provided
	if starkDB.bootstrappers == nil {
		addresses, err := setupBootstrappers(DefaultBootstrappers)
		if err != nil {
			return nil, nil, err
		}
		starkDB.bootstrappers = addresses
	}

	// now update the keystorePath variable to point to the requested project
	starkDB.keystorePath = fmt.Sprintf("%s/%s", starkDB.keystorePath, starkDB.project)

	// init the IPFS client
	client, err := newIPFSclient(starkDB.ctx, starkDB.bootstrappers)
	if err != nil {
		return nil, nil, err
	}
	starkDB.ipfsClient = client

	// if there was a snapshot CID provided, remove anything already in the
	// keystore path that could conflict and then retrieve the database
	// snapshot
	if len(starkDB.snapshotCID) != 0 {
		if err := os.RemoveAll(starkDB.keystorePath); err != nil {
			return nil, nil, err
		}
		if err := starkDB.getFile(starkDB.snapshotCID, starkDB.keystorePath); err != nil {
			return nil, nil, err
		}
	}

	// open up badger and attach to the starkDB
	badgerOpts := badger.DefaultOptions(starkDB.keystorePath).WithLogger(nil)
	ldb, err := badger.Open(badgerOpts)
	if err != nil {
		return nil, nil, errors.Wrap(err, ErrNewDb.Error())
	}
	starkDB.keystore = ldb

	// update the database counts
	if err := starkDB.refreshCount(); err != nil {
		return nil, nil, err
	}

	// return the teardown so we can ensure it happens
	return starkDB, starkDB.teardown, nil
}

// teardown will close down all the open guff
// nicely.
func (Db *Db) teardown() error {
	Db.lock.Lock()
	Db.lock.Unlock()

	// close the local keystore
	if err := Db.keystore.Close(); err != nil {
		return err
	}

	// cancel the db context
	Db.ctxCancel()

	// close IPFS
	if err := Db.ipfsClient.endSession(); err != nil {
		return err
	}

	// check the node is offline
	if Db.IsOnline() {
		return ErrNodeOnline
	}
	return nil
}

// setProject will set the database project.
func (Db *Db) setProject(project string) error {

	// sanitize the project name
	project = strings.ReplaceAll(project, " ", "_")
	if len(project) == 0 {
		return ErrNoProject
	}

	// set it
	Db.project = project
	return nil
}

// setLocalStorage will check if the director(y/ies) exist(s),
// creates them if not, then sets the storage location for
// the local key-value store.
func (Db *Db) setLocalStorage(path string) error {
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
	Db.keystorePath = path
	return nil
}

// setBootstrappers will set the bootstrapper nodes
// to use for IPFS peer discovery.
func (Db *Db) setBootstrappers(nodeList []string) error {
	if len(nodeList) == 0 {
		return fmt.Errorf("no bootstrapper nodes provided")
	}
	addresses, err := setupBootstrappers(nodeList)
	if err != nil {
		return err
	}
	if len(addresses) < DefaultMinBootstrappers {
		return ErrBootstrappers
	}
	Db.bootstrappers = addresses
	return nil
}

// setPinning sets the underlying IPFS node's
// pinning flag.
func (Db *Db) setPinning(pin bool) error {
	Db.pinning = pin
	return nil
}

// setAnnouncing sets the database to announcing
// new records via PubSub.
func (Db *Db) setAnnouncing(announcing bool) error {
	Db.announcing = announcing
	return nil
}

// setEncryption tells starkDB to make encrypted
// writes.
func (Db *Db) setEncryption(val bool) error {
	if val == false {
		Db.cipherKey = nil
		return nil
	}

	// check for the env variable
	password, exists := os.LookupEnv(DefaultStarkEnvVariable)
	if !exists {
		return ErrNoEnvSet
	}

	// convert password to cipher key
	cipherKey, err := password2cipherkey(password)
	if err != nil {
		return err
	}

	// set the key
	Db.cipherKey = cipherKey
	return nil
}

// setKeyLimit tells the starkDB maximum number of
// keys to allow.
func (Db *Db) setKeyLimit(val int) error {
	Db.maxEntries = val
	return nil
}

// setSnapshotCID will record the CID of
// a snapshotted database.
func (Db *Db) setSnapshotCID(snapshotCID string) error {
	if len(snapshotCID) == 0 {
		return ErrNoCID
	}
	Db.snapshotCID = snapshotCID
	return nil
}
