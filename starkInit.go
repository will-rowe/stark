package stark

import (
	"context"
	"os"
	"strings"

	"github.com/pkg/errors"

	starkcrypto "github.com/will-rowe/stark/src/crypto"
	starkipfs "github.com/will-rowe/stark/src/ipfs"
	starkpinata "github.com/will-rowe/stark/src/pinata"
)

// SetProject is an option setter for the OpenDB
// constructor that sets the project for the
// database instance.
// The project name is used to broadcast
// messages when Records are added to the
// database instance.
func SetProject(project string) DbOption {
	return func(starkdb *Db) error {
		return starkdb.setProject(project)
	}
}

// SetSnapshotCID is an option setter for the OpenDB
// constructor that sets the base CID to use for the
// database instance.
// If none provided it will open an empty database,
// otherwise it will check the provided CID and
// populate the starkDB from the existing records
// contained in the snapshot.
func SetSnapshotCID(path string) DbOption {
	return func(starkdb *Db) error {
		return starkdb.setSnapshotCID(path)
	}
}

// WithPeers is an option setter for the OpenDB
// constructor that adds a list of nodes to the
// default IPFS bootstrappers.
func WithPeers(nodeList []string) DbOption {
	return func(starkdb *Db) error {
		return starkdb.setNodes(nodeList)
	}
}

// WithNoPinning is an option setter that specifies the IPFS
// node should NOT pin entries.
//
// Note: If not provided to the constructor, the node will
// pin entries by default.
func WithNoPinning() DbOption {
	return func(starkdb *Db) error {
		return starkdb.setPinning(false)
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
	return func(starkdb *Db) error {
		return starkdb.setAnnouncing(true)
	}
}

// WithEncryption is an option setter for the OpenDB constructor
// that tells starkDB to make encrypted writes to IPFS using the
// password in STARK_DB_PASSWORD env variable.
//
// Note: If existing Records were encrypted, Get operations will
// fail unless this option is set.
func WithEncryption() DbOption {
	return func(starkdb *Db) error {
		return starkdb.setEncryption(true)
	}
}

// WithPinata is an option setter for the OpenDB constructor
// that tells starkDB to pin it's contents with pinata every
// time the interval is passed during set operations. A value
// of < 1 tells starkDB NOT to use pinata (default).
//
// Note: This option requires the PINATA_API_KEY and the
// PINATA_SECRET_KEY environment variables to be set.
func WithPinata(interval int) DbOption {
	return func(starkdb *Db) error {
		return starkdb.setPinataPinInterval(interval)
	}
}

// WithLogging is an option setter for the OpenDB constructor
// that provides starkDB with a logging channel to send
// internal state messages during the lifetime of the
// starkDB instance back to the caller.
//
// Note: The caller should close this channel when done.
func WithLogging(loggingChan chan interface{}) DbOption {
	return func(starkdb *Db) error {
		return starkdb.setLogger(loggingChan)
	}
}

// OpenDB opens a new instance of a starkdb.
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
	starkdb := &Db{
		ctx:         ctx,
		ctxCancel:   cancel,
		cidLookup:   make(map[string]string),
		loggingChan: nil,

		// defaults
		project:        DefaultProject,
		snapshotCID:    "",
		pinning:        true,
		announcing:     false,
		cipherKey:      nil,
		peers:          starkipfs.DefaultBootstrappers,
		pinataInterval: 0,
		sessionEntries: 0,
	}

	// add the provided options
	for _, option := range options {
		err := option(starkdb)
		if err != nil {
			return nil, nil, errors.Wrap(err, ErrDbOption.Error())
		}
	}

	// validate options
	if !starkdb.pinning && (starkdb.pinataInterval > 0) {
		return nil, nil, ErrPinataOpt
	}
	if len(starkdb.peers) < DefaultMinBootstrappers {
		return nil, nil, ErrBootstrappers
	}
	//	if starkdb.announcing && (len(starkdb.peers) <= len(starkipfs.DefaultBootstrappers)) {
	//		return nil, nil, ErrNoPeers
	//	}

	// init the IPFS client
	client, err := starkipfs.NewIPFSclient(starkdb.ctx)
	if err != nil {
		return nil, nil, err
	}
	starkdb.ipfsClient = client

	// bootstrap the IPFS node
	go starkdb.ipfsClient.Connect(starkdb.ctx, starkdb.peers, starkdb.loggingChan)

	// if no base CID was provided, initialise a snapshot
	if len(starkdb.snapshotCID) == 0 {
		cid, err := starkdb.ipfsClient.NewDagNode(starkdb.ctx)
		if err != nil {
			return nil, nil, errors.Wrap(err, ErrSnapshotUpdate.Error())
		}
		starkdb.snapshotCID = cid
	} else {

		// populate the lookup map with the existing snapshot
		links, err := starkdb.ipfsClient.GetNodeLinks(starkdb.ctx, starkdb.snapshotCID)
		if err != nil {
			return nil, nil, errors.Wrap(err, ErrInvalidSnapshot.Error())
		}
		for _, link := range links {
			starkdb.cidLookup[link.Name] = link.Cid.String()
		}
	}

	// set the stats
	starkdb.currentNumEntries = len(starkdb.cidLookup)

	// return the teardown so we can ensure it happens
	return starkdb, starkdb.teardown, nil
}

// teardown will close down all the open guff
// nicely.
func (starkdb *Db) teardown() error {
	starkdb.Lock()

	// cancel the db context
	starkdb.ctxCancel()

	// close IPFS
	if err := starkdb.ipfsClient.EndSession(); err != nil {
		return err
	}
	starkdb.Unlock()
	*starkdb = Db{}
	return nil
}

// setProject will set the database project.
func (starkdb *Db) setProject(project string) error {
	starkdb.Lock()
	defer starkdb.Unlock()

	// sanitize the project name
	project = strings.ReplaceAll(project, " ", "_")
	if len(project) == 0 {
		return ErrNoProject
	}

	// set it
	starkdb.project = project
	return nil
}

// setSnapshotCID will set the snapshot CID.
func (starkdb *Db) setSnapshotCID(cid string) error {
	if len(cid) == 0 {
		return ErrNoCID
	}
	starkdb.snapshotCID = cid
	return nil
}

// setNodes will add nodes to the list of bootstrapper
// nodes to use for IPFS peer discovery.
func (starkdb *Db) setNodes(nodeList []string) error {
	// TODO: add some checking
	starkdb.peers = append(starkdb.peers, nodeList...)
	return nil
}

// setPinning sets the underlying IPFS node's
// pinning flag.
func (starkdb *Db) setPinning(pin bool) error {
	starkdb.pinning = pin
	return nil
}

// setAnnouncing sets the database to announcing
// new records via PubSub.
func (starkdb *Db) setAnnouncing(announcing bool) error {
	starkdb.announcing = announcing
	return nil
}

// setEncryption tells starkDB to make encrypted
// writes.
func (starkdb *Db) setEncryption(val bool) error {
	if val == false {
		starkdb.cipherKey = nil
		return nil
	}

	// check for the env variable
	password, exists := os.LookupEnv(DefaultStarkEnvVariable)
	if !exists {
		return ErrNoEnvSet
	}

	// convert password to cipher key
	cipherKey, err := starkcrypto.Password2cipherkey(password)
	if err != nil {
		return err
	}

	// set the key
	starkdb.cipherKey = cipherKey
	return nil
}

// setPinataPinInterval tells starkDB to pin it's contents
// with pinata every time the interval is reached for
// set operations.
func (starkdb *Db) setPinataPinInterval(interval int) error {

	// less the one then just leave the default interval set
	if interval < 1 {
		return nil
	}

	// check pinata ENV variables are set
	var k, s string
	var ok1, ok2 bool
	if k, ok1 = os.LookupEnv(DefaultPinataAPIkey); !ok1 {
		return ErrPinataKey
	}
	if s, ok2 = os.LookupEnv(DefaultPinataSecretKey); !ok2 {
		return ErrPinataSecret
	}

	// check the API
	_, err := starkpinata.NewClient(k, s, "")
	if err != nil {
		return ErrPinataAPI(err)
	}

	// set the interval
	starkdb.pinataInterval = interval
	return nil
}

// setLogger will attach a logging channel to
// the starkDB instance.
func (starkdb *Db) setLogger(loggingChan chan interface{}) error {
	starkdb.loggingChan = loggingChan
	return nil
}
