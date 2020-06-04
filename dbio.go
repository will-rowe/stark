package stark

import (
	"bytes"
	"encoding/json"
	"fmt"
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
// Set adds a comment to the Record's history before adding it to the
// IPFS.
func (Db *Db) Set(key string, record *Record) error {
	Db.lock.Lock()
	defer Db.lock.Unlock()

	// max entry check
	if Db.currentNumEntries == Db.maxEntries {
		return ErrMaxEntriesExceeded
	}

	// check the local keystore to see if this key has been used before
	if existingCID, exists := Db.keystoreGet(key); exists {

		// retrieve the record for this key
		existingRecord, err := Db.GetRecordFromCID(existingCID)
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
	cid, err := Db.dagPut(jsonData)
	if err != nil {
		return err
	}

	// if announcing, do it now
	if Db.announcing {
		if err := Db.publishAnnouncement([]byte(cid)); err != nil {
			return err
		}
	}

	// add the returned CID to the local keystore
	if err := Db.keystoreSet(key, cid); err != nil {
		return err
	}
	Db.currentNumEntries++
	return nil
}

// Get will retrieve a Record from the starkDB using the provided key.
func (Db *Db) Get(key string) (*Record, error) {
	Db.lock.Lock()
	defer Db.lock.Unlock()

	// check the local keystore for the provided key
	cid, exists := Db.keystoreGet(key)
	if !exists {
		return nil, fmt.Errorf("%v: %v", ErrKeyNotFound, key)
	}

	// use the helper method to retrieve the Record
	return Db.GetRecordFromCID(cid)
}

// Delete will delete an entry from starkDB. This involves
// removing the key and Record CID from the local store,
// as well as unpinning the Record from the IPFS.
//
// Note: I'm not sure how this behaves if the Record
// wasn't pinned in the IPFS in the first place.
func (Db *Db) Delete(key string) error {
	Db.lock.Lock()
	defer Db.lock.Unlock()

	// check the local keystore for the provided key
	cid, exists := Db.keystoreGet(key)
	if !exists {
		return fmt.Errorf("%v: %v", ErrKeyNotFound, key)
	}

	// unpin
	if err := Db.ipfsClient.ipfs.Pin().Rm(Db.ctx, path.New(cid)); err != nil {
		return err
	}

	// remove from the keystore
	if err := Db.keystoreDelete(key); err != nil {
		return err
	}
	Db.currentNumEntries--
	return nil
}

// GetRecordFromCID is a helper method that collects a Record from
// the IPFS using its CID string.
func (Db *Db) GetRecordFromCID(cid string) (*Record, error) {
	if len(cid) == 0 {
		return nil, ErrNoCID
	}

	// retrieve the record data from the IPFS
	retrievedNode, err := Db.dagGet(cid)
	if err != nil {
		return nil, err
	}

	// double check it's CBOR
	cborNode, isCborNode := retrievedNode.(*cbor.Node)
	if !isCborNode {
		return nil, fmt.Errorf("%v: %v", ErrNodeFormat, cid)
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

// GetExplorerLink will return an IPFS explorer link for
// a CID in the starkDB given the provided lookup key.
func (Db *Db) GetExplorerLink(key string) (string, error) {
	cid, ok := Db.keystoreGet(key)
	if !ok {
		return "", fmt.Errorf("could not retrieve CID from local keystore")
	}
	return fmt.Sprintf("IPLD Explorer link: https://explore.ipld.io/#/explore/%s \n", cid), nil
}

// Snapshot copies the current database to the IPFS and
// returns the CID needed for retrieval/sharing.
func (Db *Db) Snapshot() (string, error) {
	Db.lock.Lock()
	defer Db.lock.Unlock()
	if err := Db.keystore.Sync(); err != nil {
		return "", err
	}
	if err := checkDir(Db.keystorePath); err != nil {
		return "", err
	}
	cid, err := Db.addFile(Db.keystorePath)
	if err != nil {
		return "", err
	}
	return cid, nil
}

// dagPut will append to an IPFS dag.
func (Db *Db) dagPut(data []byte) (string, error) {

	// get the node adder
	var adder ipld.NodeAdder = Db.ipfsClient.ipfs.Dag()
	if Db.pinning {
		adder = Db.ipfsClient.ipfs.Dag().Pinning()
	}
	b := ipld.NewBatch(Db.ctx, adder)

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
		err := b.Add(Db.ctx, nd)
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
func (Db *Db) dagGet(queryCID string) (interface{}, error) {

	// get the IPFS path
	rp, err := Db.ipfsClient.ipfs.ResolvePath(Db.ctx, path.New(queryCID))
	if err != nil {
		return nil, err
	}

	// get holders ready
	var obj ipld.Node

	// detect what we're dealing with and check we're good before collecting the data
	if rp.Cid().Type() == cid.DagCBOR {
		obj, err = Db.ipfsClient.ipfs.Dag().Get(Db.ctx, rp.Cid())
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
func (Db *Db) keystoreSet(key, value string) error {
	txn := Db.keystore.NewTransaction(true)
	defer txn.Discard()
	err := txn.Set([]byte(key), []byte(value))
	if err != nil {
		return err
	}
	return txn.Commit()
}

// keystoreGet will get a key value pair from the local keystore.
func (Db *Db) keystoreGet(key string) (string, bool) {
	txn := Db.keystore.NewTransaction(false)
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

// keystoreDelete will remove a key value pair from the local keystore.
func (Db *Db) keystoreDelete(key string) error {
	return Db.keystore.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// addFile will add a file (or directory) to the IPFS and return
// the CID.
func (Db *Db) addFile(filePath string) (string, error) {

	// convert the file to an IPFS File Node
	ipfsFile, err := getUnixfsNode(filePath)
	if err != nil {
		return "", fmt.Errorf("could not convert file to IPFS format file: %s", err)
	}

	// access the UnixfsAPI interface for the go-ipfs node and add file to IPFS
	cid, err := Db.ipfsClient.ipfs.Unixfs().Add(Db.ctx, ipfsFile, options.Unixfs.Pin(Db.pinning))
	if err != nil {
		return "", fmt.Errorf("could not add file to IPFS: %s", err)
	}
	return cid.String(), nil
}

// getFile will get a file (or directory) from the IPFS and write it
// to the supplied outputPath.
func (Db *Db) getFile(cidStr, outputPath string) error {

	// convert the CID to an IPFS Path
	cid := icorepath.New(cidStr)
	rootNode, err := Db.ipfsClient.ipfs.Unixfs().Get(Db.ctx, cid)
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
