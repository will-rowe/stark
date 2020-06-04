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
// for the starkDB.
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
// The IPFS code used in starkDB is based on the IPFS as a
// library example (go-ipfs v0.5.0), see:
// https://github.com/ipfs/go-ipfs/blob/2dc1f691f1bb98cadd7e7867cb924ce69f126751/docs/examples/go-ipfs-as-a-library/main.go
//
func newIPFSclient(ctx context.Context, bootstrappers []ma.Multiaddr) (*client, error) {

	// setup the empty client
	client := &client{}

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
	addrInfos, err := peer.AddrInfosFromP2pAddrs(bootstrappers...)
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
