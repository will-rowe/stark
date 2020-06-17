//Package ipfs wraps the ipfs core api with some useful client methods for PubSub and IO.
package ipfs

import (
	"context"
	"fmt"
	"math"
	"net"
	"path/filepath"
	"strings"
	"sync"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core" // This package is needed so that all the preloaded plugins are loaded automatically
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/will-rowe/stark/src/helpers"
)

const (
	// DefaultBufferSize is used in the buffered PubSub channels.
	DefaultBufferSize = 42

	// DefaultIenc is the input encoding for the data will be added to the IPFS DAG.
	DefaultIenc = "json"

	// DefaultFormatParser is the format of the input data for IPFS.
	DefaultFormatParser = "cbor"

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

	// ErrNoLinks is issued when no links are found in an IPFS DAG node.
	ErrNoLinks = fmt.Errorf("no links found in IPFS DAG node")

	// ErrOffline indicates node is offline.
	ErrOffline = fmt.Errorf("the IPFS node is offline")
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

	// check if repo is already locked
	locked, err := fsrepo.LockedByOtherProcess(defaultIpfsRepo)
	if err != nil {
		panic(err)
	}
	if locked {
		//fmt.Println("IPFS already runnning - skipping IPFS plugin initialisation")
		return
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

// PrintNodeID will print the node's identity.
func (client *Client) PrintNodeID() string {
	return client.node.Identity.Pretty()
}

// GetPublicIPv4Addr uses the host addresses to return the
// public ipv4 address of the host machine, if available.
func (client *Client) GetPublicIPv4Addr() (string, error) {
	for _, addr := range client.node.PeerHost.Addrs() {
		parts := strings.Split(addr.String(), "/")
		if len(parts) < 3 {
			continue
		}
		if parts[1] != "ip4" {
			continue
		}
		parsed := net.ParseIP(parts[2])
		if parsed != nil && helpers.IsPublicIPv4(parsed) {
			return addr.String(), nil
		}
	}
	return "", fmt.Errorf("no public IPv4 address was found for IPFS node")
}

// Online will return true if the node is online.
func (client *Client) Online() bool {
	return client.node.IsOnline
}

// Connect will connect the client to peers.
// It is adapted from https://github.com/ipfs/go-ipfs/tree/master/docs/examples/go-ipfs-as-a-library
func (client *Client) Connect(ctx context.Context, peers []string, logger chan interface{}) {
	var wg sync.WaitGroup
	peerInfos := make(map[peer.ID]*peerstore.PeerInfo, len(peers))
	for _, addrStr := range peers {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			if logger != nil {
				logger <- err
			}
			return
		}
		pii, err := peerstore.InfoFromP2pAddr(addr)
		if err != nil {
			if logger != nil {
				logger <- err
			}
			return
		}
		pi, ok := peerInfos[pii.ID]
		if !ok {
			pi = &peerstore.PeerInfo{ID: pii.ID}
			peerInfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, pii.Addrs...)
	}
	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peerstore.PeerInfo) {
			defer wg.Done()
			err := client.ipfs.Swarm().Connect(ctx, *peerInfo)
			if err != nil && logger != nil {
				logger <- fmt.Errorf("IPFS bootstrapper - %w", err)
			}
		}(peerInfo)
	}
	wg.Wait()
	return
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// NewIPFSclient will initialise the IPFS client.
//
// The IPFS code used in starkDB is based on the IPFS as a
// library example (go-ipfs v0.5.0)
//
// Note: if no repoPath provided, this function will attempt to
// use the IPFS default repo path.
func NewIPFSclient(ctx context.Context) (*Client, error) {

	// open the repo
	repo, err := fsrepo.Open(defaultIpfsRepo)
	if err != nil {
		return nil, err
	}

	// defaultNodeOpts are the options used for the IPFS node.
	defaultNodeOpts := &core.BuildCfg{
		Online:    true,
		Permanent: true,
		Repo:      repo,
		Routing:   libp2p.DHTOption,
		ExtraOpts: map[string]bool{
			"pubsub":   true,
			"psrouter": true,
			"dht":      true,
			"p2p":      true,
		},
	}

	// construct the node
	node, err := core.NewNode(ctx, defaultNodeOpts)
	if err != nil {
		return nil, err
	}

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
