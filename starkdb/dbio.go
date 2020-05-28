package stark

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/gogo/protobuf/jsonpb"
	files "github.com/ipfs/go-ipfs-files"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
)

/////////////////////////
// Exported methods:

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
			record.AddComment("updating record.")
		} else {
			record.AddComment("overwriting record.")
		}
	}
	record.AddComment("adding record to IPFS.")

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
	record.AddComment("record retrieved from IPFS.")
	return record, nil
}

/////////////////////////
// Unexported methods:

// addFile will add a file or directory to the IPFS and record the CID lookup locally using the provided key.
func (DB *DB) addFile(path, key string) error {

	// check the local key isn't already in use
	_, ok := DB.keystoreGet(key)
	if ok {
		return fmt.Errorf("key already in the local keystore")
	}

	// convert the file to an IPFS File Node
	ipfsFile, err := getUnixfsNode(path)
	if err != nil {
		return fmt.Errorf("could not convert file to IPFS format file: %s", err)
	}

	// access the UnixfsAPI interface for the go-ipfs node and add file to IPFS
	cid, err := DB.ipfsCoreAPI.Unixfs().Add(DB.ctx, ipfsFile, options.Unixfs.Pin(DB.pinning))
	if err != nil {
		return fmt.Errorf("could not add file to IPFS: %s", err)
	}

	// record the CID locally
	return DB.keystoreSet(key, cid.String())
}

// getFile will get a file or directory from the IPFS using the provided lookup key.
func (DB *DB) getFile(key, outputPath string) error {

	// use the lookup key to get the stored CID
	cid, ok := DB.keystoreGet(key)
	if !ok {
		return fmt.Errorf("could not retrieve CID from local keystore")
	}

	// convert the string CID to an IPFS Path
	ipfspath := path.New(cid)

	// access the UnixfsAPI interface for the go-ipfs node and get data
	rootNodeFile, err := DB.ipfsCoreAPI.Unixfs().Get(DB.ctx, ipfspath)
	if err != nil {
		return fmt.Errorf("could not get CID from IPFS: %s", err)
	}

	// convert to a file on the local filesystem
	return files.WriteTo(rootNodeFile, outputPath)
}
