package stark

import (
	"context"
	"fmt"
	"path/filepath"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core/bootstrap"
	libp2p "github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo"
	icore "github.com/ipfs/interface-go-ipfs-core"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi" // This package is needed so that all the preloaded plugins are loaded automatically
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/libp2p/go-libp2p-core/peer"
)

var (

	// defaultBootstrap nodes used by IPFS.
	defaultBootstrapNodes = []string{

		// IPFS bootstrapper nodes
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",

		/*
			// IPFS cluster pinning nodes
			"/ip4/138.201.67.219/tcp/4001/p2p/QmUd6zHcbkbcs7SMxwLs48qZVX3vpcM8errYS7xEczwRMA",
			"/ip4/138.201.67.219/udp/4001/quic/p2p/QmUd6zHcbkbcs7SMxwLs48qZVX3vpcM8errYS7xEczwRMA",
			"/ip4/138.201.67.220/tcp/4001/p2p/QmNSYxZAiJHeLdkBg38roksAR9So7Y5eojks1yjEcUtZ7i",
			"/ip4/138.201.67.220/udp/4001/quic/p2p/QmNSYxZAiJHeLdkBg38roksAR9So7Y5eojks1yjEcUtZ7i",
			"/ip4/138.201.68.74/tcp/4001/p2p/QmdnXwLrC8p1ueiq2Qya8joNvk3TVVDAut7PrikmZwubtR",
			"/ip4/138.201.68.74/udp/4001/quic/p2p/QmdnXwLrC8p1ueiq2Qya8joNvk3TVVDAut7PrikmZwubtR",
			"/ip4/94.130.135.167/tcp/4001/p2p/QmUEMvxS2e7iDrereVYc5SWPauXPyNwxcy9BXZrC1QTcHE",
			"/ip4/94.130.135.167/udp/4001/quic/p2p/QmUEMvxS2e7iDrereVYc5SWPauXPyNwxcy9BXZrC1QTcHE",
		*/
	}
)

// repoPath is set by init.
var repoPath string

// init function is used to load the plugins.
func init() {
	var err error

	// find the default repo
	repoPath, err = config.PathRoot()
	if err != nil {
		panic(err)
	}

	// setup plugins
	_, err = setupPlugins(repoPath)
	if err != nil {
		panic(err)
	}
}

// client is a wrapper that groups and controls access to the IPFS
// for the starkdb.
type client struct {
	ipfs icore.CoreAPI // the exported API
	repo repo.Repo
	node *core.IpfsNode
}

// endSession closes down the client.
func (client *client) endSession() error {
	if err := client.repo.Close(); err != nil {
		return err
	}
	if err := client.node.Close(); err != nil {
		return err
	}
	client.node.IsOnline = false
	return nil
}

// printListeners will print the peers which this node is listening on.
func (client *client) printListeners() {
	addrs := client.node.PeerHost.Addrs()
	for _, addr := range addrs {
		fmt.Printf("IPFS client node listening on %v/p2p/%v\n", addr, client.node.Identity)
	}
}

// newIPFSclient will initialise the IPFS client.
//
// The IPFS code used in starkdb is based on the IPFS as a
// library example (go-ipfs v0.5.0), see:
// https://github.com/ipfs/go-ipfs/blob/2dc1f691f1bb98cadd7e7867cb924ce69f126751/docs/examples/go-ipfs-as-a-library/main.go
//
func newIPFSclient(ctx context.Context) (*client, error) {

	// setup the empty client
	client := &client{}

	// set up the Peer addresses
	ipfsBootstrapAddrs := make([]ma.Multiaddr, len(defaultBootstrapNodes))
	for i, v := range defaultBootstrapNodes {
		a, err := ma.NewMultiaddr(v)
		if err != nil {
			return nil, err
		}
		ipfsBootstrapAddrs[i] = a
	}

	// open the default repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}
	client.repo = repo

	// defaultNodeOpts are the options used for the IPFS node.
	defaultNodeOpts := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption,
		Repo:    repo,
		ExtraOpts: map[string]bool{
			"pubsub": true,
		},
	}

	// construct the node
	node, err := core.NewNode(ctx, defaultNodeOpts)
	if err != nil {
		return nil, err
	}
	client.node = node

	// bootstrap the node
	addrInfos, err := peer.AddrInfosFromP2pAddrs(ipfsBootstrapAddrs...)
	if err != nil {
		return nil, err
	}
	node.Bootstrap(bootstrap.BootstrapConfigWithPeers(addrInfos))

	// attach the Core API to the constructed node
	api, err := coreapi.NewCoreAPI(node)
	if err != nil {
		return nil, err
	}
	client.ipfs = api
	return client, nil

}

// setupPlugins will create a new IPFS PluginLoader and inject them
// into the session.
func setupPlugins(externalPluginsPath string) (*loader.PluginLoader, error) {

	// load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return nil, fmt.Errorf("error loading plugins: %s", err)
	}

	// load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return nil, fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return nil, fmt.Errorf("error initializing plugins: %s", err)
	}
	return plugins, nil
}
