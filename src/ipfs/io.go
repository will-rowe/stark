package ipfs

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core/coredag"
	cbor "github.com/ipfs/go-ipld-cbor"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
)

// DagPut will append to an IPFS dag.
func (client *Client) DagPut(ctx context.Context, data []byte, pinning bool) (string, error) {

	// get the node adder
	var adder ipld.NodeAdder = client.ipfs.Dag()
	if pinning {
		adder = client.ipfs.Dag().Pinning()
	}
	b := ipld.NewBatch(ctx, adder)

	//
	file := files.NewBytesFile(data)
	if file == nil {
		return "", fmt.Errorf("failed to convert input data for IPFS")
	}

	//
	nds, err := coredag.ParseInputs(DefaultIenc, DefaultFormat, file, DefaultMhType, -1)
	if err != nil {
		return "", err
	}
	if len(nds) == 0 {
		return "", fmt.Errorf("no node returned from ParseInputs")
	}

	//
	for _, nd := range nds {
		err := b.Add(ctx, nd)
		if err != nil {
			return "", err
		}
	}

	// commit the batched nodes
	if err := b.Commit(); err != nil {
		return "", err
	}
	return nds[0].Cid().String(), nil
}

// DagGet will fetch a DAG node from the IPFS using the
// provided CID.
func (client *Client) DagGet(ctx context.Context, queryCID string) (interface{}, error) {

	// get the IPFS path
	rp, err := client.ipfs.ResolvePath(ctx, path.New(queryCID))
	if err != nil {
		return nil, err
	}

	// get holders ready
	var obj ipld.Node

	// detect what we're dealing with and check we're good before collecting the data
	if rp.Cid().Type() == cid.DagCBOR {
		obj, err = client.ipfs.Dag().Get(ctx, rp.Cid())
		if err != nil {
			return nil, err
		}
		_, isCborNode := obj.(*cbor.Node)
		if !isCborNode {
			return nil, fmt.Errorf("unsupported IPLD CID format (cid: %v)", queryCID)
		}

		// TODO: we could handle other types of node - using a type switch or some such
		/*
			} else if rp.Cid().Type() == cid.Raw {
				obj, err = ipfsClient.coreAPI.Dag().Get(ipfsClient.ctx, rp.Cid())
				if err != nil {
					return nil, err
				}
				//	data = obj.RawData()
		*/
	} else {
		return nil, fmt.Errorf("unsupported IPLD CID format (cid: %v)", queryCID)
	}

	// get the data ready for return
	var out interface{} = obj

	// grab the specified field if one was given
	if len(rp.Remainder()) > 0 {
		rem := strings.Split(rp.Remainder(), "/")
		final, _, err := obj.Resolve(rem)
		if err != nil {
			return nil, err
		}
		out = final
	}

	return out, nil
}

// Unpin will unpin a CID from the IPFS.
func (client *Client) Unpin(ctx context.Context, cidStr string) error {
	return client.ipfs.Pin().Rm(ctx, path.New(cidStr))
}

// AddFile will add a file (or directory) to the IPFS and return
// the CID.
func (client *Client) AddFile(ctx context.Context, filePath string, pinning bool) (string, error) {

	// convert the file to an IPFS File Node
	ipfsFile, err := getUnixfsNode(filePath)
	if err != nil {
		return "", fmt.Errorf("could not convert file to IPFS format file: %s", err)
	}

	// access the UnixfsAPI interface for the go-ipfs node and add file to IPFS
	cid, err := client.ipfs.Unixfs().Add(ctx, ipfsFile, options.Unixfs.Pin(pinning))
	if err != nil {
		return "", fmt.Errorf("could not add file to IPFS: %s", err)
	}
	return cid.String(), nil
}

// GetFile will get a file (or directory) from the IPFS using the
// supplied CID and then write it to the supplied outputPath.
func (client *Client) GetFile(ctx context.Context, cidStr, outputPath string) error {

	// convert the CID to an IPFS Path
	cid := icorepath.New(cidStr)
	rootNode, err := client.ipfs.Unixfs().Get(ctx, cid)
	if err != nil {
		return err
	}

	// write to local filesystem
	err = files.WriteTo(rootNode, outputPath)
	if err != nil {
		return (fmt.Errorf("could not write out the fetched CID: %s", err))
	}
	return nil
}

// getUnixfsNode takes a file/directory path and returns
// an IPFS File Node representing the file, directory or
// special file.
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
