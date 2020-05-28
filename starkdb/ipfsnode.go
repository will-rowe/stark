package stark

import (
	"context"
	"os"

	"fmt"
	"io/ioutil"
	"path/filepath"

	config "github.com/ipfs/go-ipfs-config"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/corerepo"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader" // This package is needed so that all the preloaded plugins are loaded automatically
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	icore "github.com/ipfs/interface-go-ipfs-core"
)

// setupIPFS will setup the IPFS node and Core API
func setupIPFS(ctx context.Context, ephemeral bool) (icore.CoreAPI, *core.IpfsNode, error) {

	// get the IPFS repo path and setup pluggins
	var repoPath string
	var err error
	if ephemeral {
		if err = setupPlugins(""); err != nil {
			return nil, nil, err
		}
		repoPath, err = createTempRepo(ctx)
		if err != nil {
			return nil, nil, err
		}
	} else {
		repoPath, err = config.PathRoot()
		if err != nil {
			return nil, nil, err
		}
		//if err = setupPlugins(repoPath); err != nil {
		//	return nil, nil, err
		//}
	}

	// open the provided repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, nil, err
	}

	// prepare the IPFS node options
	nodeOptions := &core.BuildCfg{
		Online:    true,
		Permanent: true,
		Routing:   libp2p.DHTOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		// Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo: repo,
		ExtraOpts: map[string]bool{
			"pubsub": true,
		},
	}

	// create an IPFS node
	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, nil, err
	}
	node.IsDaemon = true
	node.IsOnline = true

	// link the IPFS node with the API
	api, err := coreapi.NewCoreAPI(node)
	if err != nil {
		return nil, nil, err
	}

	// start garbage collector
	go corerepo.PeriodicGC(ctx, node)

	return api, node, nil
}

// getUnixfsNode takes a path and returns an IPFS File Node representing the file, directory or special file.
func getUnixfsNode(path string) (files.Node, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	includeHiddenFiles := false
	f, err := files.NewSerialFile(path, includeHiddenFiles, fileInfo)
	if err != nil {
		return nil, err
	}
	return f, nil
}

/*
	// setup the options
	opts := []options.UnixfsAddOption{
		//options.Unixfs.Hash(hashFunCode),
		//options.Unixfs.Inline(inline),
		//options.Unixfs.InlineLimit(inlineLimit),
		//options.Unixfs.Chunker(chunker),
		options.Unixfs.Pin(false),
		//options.Unixfs.HashOnly(hash),
		//options.Unixfs.FsCache(fscache),
		//options.Unixfs.Nocopy(nocopy),
		//options.Unixfs.Progress(progress),
		//options.Unixfs.Silent(silent),
	}
*/

// setupPlugins
func setupPlugins(externalPluginsPath string) error {

	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

// createTempRepo
func createTempRepo(ctx context.Context) (string, error) {
	repoPath, err := ioutil.TempDir("", "ipfs-shell")
	if err != nil {
		return "", fmt.Errorf("failed to get temp dir: %s", err)
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(ioutil.Discard, 2048)
	if err != nil {
		return "", err
	}

	// Create the repo with the config
	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to init ephemeral node: %s", err)
	}

	return repoPath, nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////
/*
// ConnectToPeers is a method to connect an IPFS node to peers
func (wrapper *IPFSnode) ConnectToPeers(ctx context.Context, peers []string) error {
	var wg sync.WaitGroup
	peerInfos := make(map[peer.ID]*peerstore.PeerInfo, len(peers))
	for _, addrStr := range peers {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		pii, err := peerstore.InfoFromP2pAddr(addr)
		if err != nil {
			return err
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
			err := wrapper.Swarm().Connect(ctx, *peerInfo)
			if err != nil {
				log.Printf("failed to connect to %s: %s", peerInfo.ID, err)
			}
		}(peerInfo)
	}
	wg.Wait()
	return nil
}
*/
