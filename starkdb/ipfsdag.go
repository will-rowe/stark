package stark

import (
	"fmt"
	"math"
	"strings"

	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core/coredag"
	cbor "github.com/ipfs/go-ipld-cbor"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/interface-go-ipfs-core/path"
)

const (

	// Ienc is the format in which the data will be added to the IPFS DAG.
	Ienc = "json"

	// Format is the format of the input data.
	Format = "cbor"

	// MhType is the hash to use for DAG put operations.
	MhType = uint64(math.MaxUint64) // use default hash (sha256 for cbor, sha1 for git..)
)

// dagPut will add serialised data to an IPFS DAG node(s), returning the CID and any error.
func (DB *DB) dagPut(data []byte) (string, error) {

	//
	var adder ipld.NodeAdder = DB.ipfsCoreAPI.Dag()
	if DB.pinning {
		adder = DB.ipfsCoreAPI.Dag().Pinning()
	}
	b := ipld.NewBatch(DB.ctx, adder)

	//
	file := files.NewBytesFile(data)
	if file == nil {
		return "", fmt.Errorf("failed to convert input data for IPFS")
	}

	//
	nds, err := coredag.ParseInputs(Ienc, Format, file, MhType, -1)
	if err != nil {
		return "", err
	}
	if len(nds) == 0 {
		return "", fmt.Errorf("no node returned from ParseInputs")
	}

	//
	for _, nd := range nds {
		err := b.Add(DB.ctx, nd)
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

// dagGet fetches a DAG node from the IPFS using the provided CID.
func (DB *DB) dagGet(queryCID string) (interface{}, error) {

	// get the IPFS path
	rp, err := DB.ipfsCoreAPI.ResolvePath(DB.ctx, path.New(queryCID))
	if err != nil {
		return nil, err
	}

	// get holders ready
	var obj ipld.Node

	// detect what we're dealing with and check we're good before collecting the data
	if rp.Cid().Type() == cid.DagCBOR {
		obj, err = DB.ipfsCoreAPI.Dag().Get(DB.ctx, rp.Cid())
		if err != nil {
			return nil, err
		}
		_, isCborNode := obj.(*cbor.Node)
		if !isCborNode {
			return nil, fmt.Errorf("unsupported IPLD CID format (cid: %v)", queryCID)
		}
		/*
			} else if rp.Cid().Type() == cid.Raw {
				obj, err = DB.ipfsCoreAPI.Dag().Get(DB.ctx, rp.Cid())
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

/*
// Get will get serialised data from an IPFS DAG using a CID and returns a reader.
func (DB *DB) Get(key string) (io.ReadCloser, error) {

	// use the lookup key to get the stored CID
	cid, ok := DB.keystoreGet(key)
	if !ok {
		return nil, fmt.Errorf("could not retrieve CID from local database")
	}

	// convert the string CID to an IPFS Path
	ipfspath, err := DB.ipfsCoreAPI.ResolvePath(DB.ctx, path.New(cid))
	if err != nil {
		return nil, err
	}

	nd, err := DB.ipfsCoreAPI.Dag().Get(DB.ctx, ipfspath.Cid())
	if err != nil {
		return nil, err
	}

	dr, err := unixfsio.NewDagReader(DB.ctx, nd, DB.ipfsCoreAPI.Dag())
	if err != nil {
		return nil, fmt.Errorf("cat: failed to construct DAG reader: %s", err)
	}
	return dr, nil

}
*/
