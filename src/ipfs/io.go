package ipfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core/coredag"
	cbor "github.com/ipfs/go-ipld-cbor"
	ipld "github.com/ipfs/go-ipld-format"
	merkle "github.com/ipfs/go-merkledag"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
)

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

// NewDagNode will create a new UNIXFS formatted DAG node in the IPFS.
func (client *Client) NewDagNode(ctx context.Context) (string, error) {
	path, err := client.ipfs.Object().New(ctx, options.Object.Type("unixfs-dir"))
	if err != nil {
		return "", err
	}
	return path.Cid().String(), nil
}

// AddLink will add a link under the specified path. Child path can point to a
// subdirectory within the patent, it will be created if not present.
// It will return the new base CID and any error.
func (client *Client) AddLink(ctx context.Context, baseCID, childCID, linkLabel string) (string, error) {
	path, err := client.ipfs.Object().AddLink(ctx, path.New(baseCID), linkLabel, path.New(childCID), options.Object.Create(true))
	if err != nil {
		return "", err
	}
	return path.Cid().String(), nil
}

// RmLink will remove a link under the specified path.
// It will return the new base CID and any error.
func (client *Client) RmLink(ctx context.Context, baseCID, linkLabel string) (string, error) {
	path, err := client.ipfs.Object().RmLink(ctx, path.New(baseCID), linkLabel)
	if err != nil {
		return "", err
	}
	return path.Cid().String(), nil
}

// GetNodeLinks returns the links from a node.
func (client *Client) GetNodeLinks(ctx context.Context, nodeCID string) ([]*ipld.Link, error) {
	linkList, err := client.ipfs.Object().Links(ctx, path.New(nodeCID))
	if err != nil {
		return nil, err
	}
	if len(linkList) == 0 {
		return nil, ErrNoLinks
	}
	return linkList, nil
}

// GetNodeData will output a reader for the raw bytes
// contained in an IPFS DAG node.
func (client *Client) GetNodeData(ctx context.Context, nodeCID string) (io.Reader, error) {
	dataReader, err := client.ipfs.Object().Data(ctx, path.New(nodeCID))
	if err != nil {
		return nil, err
	}
	return dataReader, nil
}

// DagPut will append to an IPFS dag.
func (client *Client) DagPut(ctx context.Context, data []byte, pinning bool) (string, error) {

	// convert the data to an IPFS file
	file := files.NewReaderFile(bytes.NewReader(data))
	if file == nil {
		return "", fmt.Errorf("failed to convert input data for IPFS")
	}

	// create an IPLD format node(s)
	nds, err := coredag.ParseInputs(DefaultIenc, DefaultFormatParser, file, DefaultMhType, -1)
	if err != nil {
		return "", err
	}
	if len(nds) == 0 {
		return "", fmt.Errorf("no node returned from ParseInputs")
	}

	// get the node adder
	var adder ipld.NodeAdder = client.ipfs.Dag()
	if pinning {
		adder = client.ipfs.Dag().Pinning()
	}

	// buffer the nodes
	buf := ipld.NewBatch(ctx, adder)
	for _, nd := range nds {
		if err := buf.Add(ctx, nd); err != nil {
			return "", err
		}
	}

	// commit the batched nodes
	if err := buf.Commit(); err != nil {
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

	// get the root node
	var obj ipld.Node
	obj, err = client.ipfs.Dag().Get(ctx, rp.Cid())
	if err != nil {
		return nil, err
	}

	// detect what we're dealing with
	switch rp.Cid().Type() {
	case cid.DagProtobuf:
		_, isProtoNode := obj.(*merkle.ProtoNode)
		if !isProtoNode {
			return nil, fmt.Errorf("node can't be cast to protobuf for: %v", rp.Cid().String())
		}
	case cid.DagCBOR:
		_, isCborNode := obj.(*cbor.Node)
		if !isCborNode {
			return nil, fmt.Errorf("node can't be cast to cbor for: %v", rp.Cid().String())
		}
	default:
		return nil, fmt.Errorf("unsupported IPLD CID format (%v = %v)", queryCID, rp.Cid().Type())
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
