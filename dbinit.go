package stark

import (
	"context"
	"encoding/hex"
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

// SetEncryption is an option setter for the OpenDB constructor
// that tells starkDB to make encrypted writes to IPFS using the
// private key in STARK_DB_ENCRYPTION_KEY env variable.
func SetEncryption(val bool) DbOption {
	return func(Db *Db) error {
		return Db.setEncryption(val)
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

// WithPinning is an option setter that specifies the IPFS
// node pin entries.
//
// Note: If not provided to the constructor, the node will
// not pin entries.
func WithPinning() DbOption {
	return func(Db *Db) error {
		return Db.setPinning(true)
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
		pinning:      false,
		announcing:   false,
		maxEntries:   DefaultMaxEntries,
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

// Listen will start a subscription and emit Records as they
// are announced on the PubSub network and match the
// database's topic.
func (Db *Db) Listen(terminator chan struct{}) (chan *Record, chan error) {

	// cidTracker skips over duplicate CIDs
	cidTracker := make(map[string]struct{})

	// channels used to send Records and errors back to the caller
	recChan := make(chan *Record, DefaultBufferSize)
	errChan := make(chan error)

	// subscribe the database
	if err := Db.subscribe(); err != nil {
		errChan <- err
	}

	// process the incoming messages
	go func() {
		for {
			select {
			case msg := <-Db.pubsubMessages:

				// TODO: check sender peerID
				//msg.From()

				// get the CID
				cid := string(msg.Data())
				if _, ok := cidTracker[cid]; ok {
					continue
				}
				cidTracker[cid] = struct{}{}

				// collect the Record from IPFS
				collectedRecord, err := Db.GetRecordFromCID(cid)
				if err != nil {
					errChan <- err
				} else {

					// add a comment to say this Record was from PubSub
					collectedRecord.AddComment(fmt.Sprintf("collected from %s via pubsub.", msg.From()))

					// send the record on to the caller
					recChan <- collectedRecord
				}

			case err := <-Db.pubsubErrors:
				errChan <- err

			case <-terminator:
				if err := Db.unsubscribe(); err != nil {
					errChan <- err
				}
				close(recChan)
				close(errChan)
				return
			}
		}
	}()
	return recChan, errChan
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
	if len(addresses) < MinBootstrappers {
		return ErrBootstrappers
	}
	Db.bootstrappers = addresses
	return nil
}

// setPinning sets the underlying IPFS node to
// pin entries.
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
	Db.privateKey = key
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
