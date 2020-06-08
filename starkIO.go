package stark

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/jsonpb"
	cbor "github.com/ipfs/go-ipld-cbor"
	starkhelpers "github.com/will-rowe/stark/src/helpers"
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
		if !starkhelpers.CheckTimeStamp(existingRecord.GetLastUpdatedTimestamp(), record.GetLastUpdatedTimestamp()) {

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

	// if encrypting requested and Record isn't already, do it now
	if len(Db.cipherKey) != 0 && !record.GetEncrypted() {
		if err := record.Encrypt(Db.cipherKey); err != nil {
			return err
		}
	}

	// marshal Record data to JSON
	jsonData, err := json.Marshal(record)
	if err != nil {
		return err
	}

	// create DAG node in IPFS for Record data
	cid, err := Db.ipfsClient.DagPut(Db.ctx, jsonData, Db.pinning)
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
	if err := Db.ipfsClient.Unpin(Db.ctx, cid); err != nil {
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
	retrievedNode, err := Db.ipfsClient.DagGet(Db.ctx, cid)
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

	// if it's an encrypted Record, see if we can decrypt
	if record.GetEncrypted() {
		if err := record.Decrypt(Db.cipherKey); err != nil {
			return nil, err
		}
	}

	// add the pulled CID to this record
	record.PreviousCID = cid
	return record, nil
}

// GetCID will return an IPFS CID in the starkDB
// for the provided lookup key.
func (Db *Db) GetCID(key string) (string, error) {
	cid, ok := Db.keystoreGet(key)
	if !ok {
		return "", fmt.Errorf("could not retrieve CID from local keystore")
	}
	return cid, nil
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
	if err := starkhelpers.CheckDir(Db.keystorePath); err != nil {
		return "", err
	}
	cid, err := Db.ipfsClient.AddFile(Db.ctx, Db.keystorePath, Db.pinning)
	if err != nil {
		return "", err
	}
	return cid, nil
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
