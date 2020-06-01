package stark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core/coredag"
	cbor "github.com/ipfs/go-ipld-cbor"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
)

// Set will add a Record to the starkDB, linking it with the provided key.
func (DB *DB) Set(key string, record *Record) error {

	// check the local keystore to see if this key has been used before
	if existingCID, exists := DB.keystoreGet(key); exists {

		// retrieve the record for this key
		existingRecord, err := DB.Get(key)
		if err != nil {
			return err
		}

		// make sure new record has a more recent timestamp than the existing record
		if !checkTimeStamp(existingRecord.GetLastUpdatedTimestamp(), record.GetLastUpdatedTimestamp()) {

			// can't replace an existing record in the DB with one that is not as recent
			return ErrExistingRecord
		}

		// if the existingCID in the local keystore matches the previousCID of the incoming Record, assume this is an update, otherwise it is an overwrite
		if existingCID == record.GetPreviousCID() {
			record.AddComment("Set: updating record.")
		} else {
			record.AddComment("Set: overwriting record.")
		}
	}
	record.AddComment("Set: adding record to IPFS.")

	// marshal Record data to JSON
	jsonData, err := json.Marshal(record)
	if err != nil {
		return err
	}

	// create DAG node in IPFS for Record data
	cid, err := DB.dagPut(jsonData)
	if err != nil {
		return err
	}

	// add the returned CID to the local keystore
	return DB.keystoreSet(key, cid)
}

// Get will retrieve a Record from the starkDB using the provided key.
func (DB *DB) Get(key string) (*Record, error) {

	// check the local keystore for the provided key
	cid, exists := DB.keystoreGet(key)
	if !exists {
		return nil, fmt.Errorf("%v: %v", ErrKeyNotFound, key)
	}

	// retrieve the record data from the IPFS
	retrievedNode, err := DB.dagGet(cid)
	if err != nil {
		return nil, err
	}

	// double check it's CBOR (dagGet has already done this though)
	cborNode, isCborNode := retrievedNode.(*cbor.Node)
	if !isCborNode {
		return nil, fmt.Errorf("%v: %v", ErrNodeFormat, key)
	}

	// get JSON data from node
	data, err := cborNode.MarshalJSON()
	if err != nil {
		return nil, err
	}

	// unmarshal it to a Record
	record := &Record{}
	um := &jsonpb.Unmarshaler{}
	if err := um.Unmarshal(bytes.NewReader(data), record); err != nil {
		return nil, err
	}

	// add the pulled CID to this record
	record.PreviousCID = cid
	return record, nil
}

// GetExplorerLink will return an IPFS explorer link for a CID in the starkdb given the provided lookup key.
func (DB *DB) GetExplorerLink(key string) (string, error) {
	cid, ok := DB.keystoreGet(key)
	if !ok {
		return "", fmt.Errorf("could not retrieve CID from local keystore")
	}
	return fmt.Sprintf("IPLD Explorer link: https://explore.ipld.io/#/explore/%s \n", cid), nil
}

// dagPut will append to an IPFS dag.
func (DB *DB) dagPut(data []byte) (string, error) {

	// get the node adder
	var adder ipld.NodeAdder = DB.ipfsClient.ipfs.Dag()
	if DB.pinning {
		adder = DB.ipfsClient.ipfs.Dag().Pinning()
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

// dagGet will fetch a DAG node from the IPFS using the
// provided CID.
func (DB *DB) dagGet(queryCID string) (interface{}, error) {

	// get the IPFS path
	rp, err := DB.ipfsClient.ipfs.ResolvePath(DB.ctx, path.New(queryCID))
	if err != nil {
		return nil, err
	}

	// get holders ready
	var obj ipld.Node

	// detect what we're dealing with and check we're good before collecting the data
	if rp.Cid().Type() == cid.DagCBOR {
		obj, err = DB.ipfsClient.ipfs.Dag().Get(DB.ctx, rp.Cid())
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

// keystoreSet will add a key value pair to the local keystore.
func (DB *DB) keystoreSet(key, value string) error {
	txn := DB.keystore.NewTransaction(true)
	defer txn.Discard()
	err := txn.Set([]byte(key), []byte(value))
	if err != nil {
		return err
	}
	return txn.Commit()
}

// keystoreGet will get a key value pair from the local keystore.
func (DB *DB) keystoreGet(key string) (string, bool) {
	txn := DB.keystore.NewTransaction(false)
	defer txn.Discard()
	item, err := txn.Get([]byte(key))
	if err != nil {
		if err != badger.ErrKeyNotFound {
			panic(err)
		}
		return "", false
	}
	var returnedValue []byte
	err = item.Value(func(val []byte) error {
		returnedValue = val
		return nil
	})
	if err != nil {
		panic(err)
	}
	return string(returnedValue), true
}

// addFile will add a file (or directory) to the IPFS and return
// the CID.
func (DB *DB) addFile(filePath string) (string, error) {

	// convert the file to an IPFS File Node
	ipfsFile, err := getUnixfsNode(filePath)
	if err != nil {
		return "", fmt.Errorf("could not convert file to IPFS format file: %s", err)
	}

	// access the UnixfsAPI interface for the go-ipfs node and add file to IPFS
	cid, err := DB.ipfsClient.ipfs.Unixfs().Add(DB.ctx, ipfsFile, options.Unixfs.Pin(DB.pinning))
	if err != nil {
		return "", fmt.Errorf("could not add file to IPFS: %s", err)
	}
	return cid.String(), nil
}

// getFile will get a file (or directory) from the IPFS and return
// a reader.
func (DB *DB) getFile(cidStr string) (io.ReadCloser, error) {

	// convert the CID to an IPFS Path
	cid := icorepath.New(cidStr)
	rootNode, err := DB.ipfsClient.ipfs.Unixfs().Get(DB.ctx, cid)
	if err != nil {
		return nil, err
	}

	// return a reader starting from the root node
	file := files.ToFile(rootNode)
	return file, nil
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
