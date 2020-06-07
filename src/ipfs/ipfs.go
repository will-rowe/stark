//Package ipfs wraps the ipfs core api with some useful client methods for PubSub and IO.
package ipfs

import (
	"context"
	"fmt"
	"math"
	"path/filepath"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core/bootstrap"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ipfs/go-ipfs/core" // This package is needed so that all the preloaded plugins are loaded automatically
	"github.com/ipfs/go-ipfs/repo/fsrepo"
)

const (
	// DefaultBufferSize is used in the buffered PubSub channels.
	DefaultBufferSize = 42

	// DefaultFormat is the format of the input data for IPFS.
	DefaultFormat = "cbor"

	// DefaultIenc is the input encoding for the data will be added to the IPFS DAG.
	DefaultIenc = "json"

	// DefaultMhType is the multihash to use for DAG put operations.
	DefaultMhType = uint64(math.MaxUint64) // use default hash (sha256 for cbor, sha1 for git..)
)

var (

	// defaultIpfsRepo is the IPFS repository to use (leave blank to attempt to find system default).
	defaultIpfsRepo = ""

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

	// ErrRepoAlreadyInitialised is issued when an IPFS repo has already been initialised.
	ErrRepoAlreadyInitialised = fmt.Errorf("specified IPFS repo is already initialised")
)

// init will setup the IPFS repo
func init() {
	var err error
	if len(defaultIpfsRepo) == 0 {
		defaultIpfsRepo, err = config.PathRoot()
		if err != nil {
			panic(err)
		}
	}
	if !fsrepo.IsInitialized(defaultIpfsRepo) {
		panic("default IPFS repo needs initialising (run `ipfs init`)")
	}

	// wait until lock is free
	for {
		locked, err := fsrepo.LockedByOtherProcess(defaultIpfsRepo)
		if err != nil {
			panic(err)
		}
		if !locked {
			break
		}
	}

	// initialise the plugins
	if _, err := initIPFSplugins(defaultIpfsRepo); err != nil {
		panic(err)
	}
}

// Client is a wrapper that groups and controls access to the IPFS
// for the starkDB.
type Client struct {

	// API:
	ipfs icore.CoreAPI
	node *core.IpfsNode
	repo repo.Repo

	// PubSub:
	pubsubSub      icore.PubSubSubscription // the pubsub subscription
	pubsubMessages chan icore.PubSubMessage // used to receive pubsub messages
	pubsubErrors   chan error               // used to receive pubsub errors
	pubsubStop     chan struct{}            // used to signal the pubsub goroutine to end
	pubsubStopped  chan struct{}            // used to signal the pubsub goroutine has ended
}

// EndSession closes down the client.
func (client *Client) EndSession() error {
	if err := client.node.Close(); err != nil {
		return err
	}
	if err := client.repo.Close(); err != nil {
		return err
	}
	client.node.IsOnline = false
	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// TODO: the following methods are untested and subject to change:

// PrintListeners will print the peers which this node is listening on.
func (client *Client) PrintListeners() {
	addrs := client.node.PeerHost.Addrs()
	for _, addr := range addrs {
		fmt.Printf("IPFS client node listening on %v/p2p/%v\n", addr, client.node.Identity)
	}
}

// PrintNodeID will print the node's identity.
func (client *Client) PrintNodeID() string {
	return client.node.Identity.Pretty()
}

// Online will return true if the node is online.
func (client *Client) Online() bool {
	return client.node.IsOnline
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// NewIPFSclient will initialise the IPFS client.
//
// The IPFS code used in starkDB is based on the IPFS as a
// library example (go-ipfs v0.5.0)
//
// Note: if no repoPath provided, this function will attempt to
// use the IPFS default repo path.
func NewIPFSclient(ctx context.Context, bootstrappers []string) (*Client, error) {

	// open the repo
	repo, err := fsrepo.Open(defaultIpfsRepo)
	if err != nil {
		return nil, err
	}

	// defaultNodeOpts are the options used for the IPFS node.
	defaultNodeOpts := &core.BuildCfg{
		Online:    true,
		Permanent: false,
		Repo:      repo,
		Routing:   libp2p.DHTOption,
		ExtraOpts: map[string]bool{
			"pubsub": true,
		},
	}

	// construct the node
	node, err := core.NewNode(ctx, defaultNodeOpts)
	if err != nil {
		return nil, err
	}

	// bootstrap the node
	adds, err := setupBootstrappers(bootstrappers)
	addrInfos, err := peer.AddrInfosFromP2pAddrs(adds...)
	if err != nil {
		return nil, err
	}
	node.Bootstrap(bootstrap.BootstrapConfigWithPeers(addrInfos))

	// get the API
	api, err := coreapi.NewCoreAPI(node)
	if err != nil {
		return nil, err
	}

	// return the client
	return &Client{
		ipfs: api,
		node: node,
		repo: repo,
	}, nil
}

// initIPFSplugins is used to initialized IPFS plugins
// before creating a new IPFS node.
//
// Note: This should only be called once per sesh.
func initIPFSplugins(repoPath string) (*loader.PluginLoader, error) {
	pl, err := loader.NewPluginLoader(filepath.Join(repoPath, "plugins"))
	if err != nil {
		return nil, err
	}
	err = pl.Initialize()
	if err != nil {
		return nil, err
	}
	if err := pl.Inject(); err != nil {
		return nil, err
	}
	return pl, err
}

// setupBootstrappers takes a list of bootstrapper nodes
// and returns their multiaddresses.
func setupBootstrappers(nodes []string) ([]ma.Multiaddr, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no list of bootstrapping nodes provided")
	}
	adds := []ma.Multiaddr{}
	for _, node := range nodes {
		add, err := ma.NewMultiaddr(node)
		if err != nil {
			return nil, err
		}
		adds = append(adds, add)
	}
	return adds, nil
}