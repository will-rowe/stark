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
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
)

const (

	// DefaultProject is the default project name used if none provided to the OpenDB function.
	DefaultProject = "starkdb-default-project"

	// DefaultLocalDbLocation is used if the user does not provide one.
	DefaultLocalDbLocation = "/tmp/starkDB/"

	// DefaultStarkEnvVariable is the env variable starkdb looks for when told to use encryption.
	DefaultStarkEnvVariable = "STARK_DB_ENCRYPTION_KEY"

	// DefaultBufferSize is the maximum number of records stored in channels.
	DefaultBufferSize = 42

	// Ienc is the format in which the data will be added to the IPFS DAG.
	Ienc = "json"

	// Format is the format of the input data.
	Format = "cbor"

	// MhType is the hash to use for DAG put operations.
	MhType = uint64(math.MaxUint64) // use default hash (sha256 for cbor, sha1 for git..)

	// MinBootstrappers is the minimum number of reachable bootstrappers required.
	MinBootstrappers = 3
)

var (

	// ErrBootstrappers is issued when not enough bootstrappers are accessible.
	ErrBootstrappers = fmt.Errorf("not enough bootstrappers found (minimum required: %d)", MinBootstrappers)

	// ErrDbOption is issued for incorrect database initialisation options.
	ErrDbOption = fmt.Errorf("starkDB option could not be set")

	// ErrEncryptKey is issued when the provided encyption key doesn't meet requirements.
	ErrEncryptKey = fmt.Errorf("cannot load private key")

	// ErrExistingRecord indicates a record with matching UUID is already in the IPFS and has a more recent update timestamp.
	ErrExistingRecord = fmt.Errorf("cannot replace a record in starkDB with an older version")

	// ErrKeyNotFound is issued during a Get request when the key is not present in the local keystore.
	ErrKeyNotFound = fmt.Errorf("key not found in the database")

	// ErrLinkExists indicates a record is already linked to the provided UUID.
	ErrLinkExists = fmt.Errorf("record already linked to the provided UUID")

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
	ErrNoEnvSet = fmt.Errorf("no private key found in %s", DefaultStarkEnvVariable)

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

// DbOption is a wrapper struct used to pass functional
// options to the starkDB constructor.
type DbOption func(Db *Db) error

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
// if this option setter is ommitted.
func SetBootstrappers(bootstrapperList []string) DbOption {
	return func(Db *Db) error {
		return Db.setBootstrappers(bootstrapperList)
	}
}

// SetEncryption is an option setter for the OpenDB constructor
// that tells starkdb to make encrypted writes to IPFS using the
// private key in STARK_DB_ENCRYPTION_KEY env variable.
func SetEncryption(val bool) DbOption {
	return func(Db *Db) error {
		return Db.setEncryption(val)
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
// Note: If opening an existing datbase, this will be
// erased in place of the snapshotted database.
func WithSnapshot(snapshotCID string) DbOption {
	return func(Db *Db) error {
		return Db.setSnapshotCID(snapshotCID)
	}
}

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

	// not yet implemented:
	allowNetwork bool   // controls the IPFS node's network connection // TODO: not yet implemented (thinking of local dbs)
	privateKey   []byte // private key for encrypted DB instances // TODO: not yet implemented (thinking of encrypted dbs)

	// local storage
	keystore *badger.DB // local keystore to relate record UUIDs to IPFS CIDs
	numKeys  int        // the number of keys in the keystore (checked on db open and then incremented/decremented during Set/Delete ops)

	// IPFS
	ipfsClient *client // wraps the IPFS core API

	// PubSub
	pubsubSub      icore.PubSubSubscription // the pubsub subscription
	pubsubMessages chan icore.PubSubMessage // used to receive pubsub messages
	pubsubErrors   chan error               // used to receive pubsub errors
	pubsubStop     chan struct{}            // used to signal the pubsub goroutine to end
	pubsubStopped  chan struct{}            // used to signal the pubsub goroutine has ended
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

// setEncryption tells starkdb to make encrypted
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

// setSnapshotCID will record the CID of
// a snapshotted database.
func (Db *Db) setSnapshotCID(snapshotCID string) error {
	if len(snapshotCID) == 0 {
		return ErrNoCID
	}
	Db.snapshotCID = snapshotCID
	return nil
}
