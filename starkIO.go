package stark

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/gogo/protobuf/jsonpb"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/pkg/errors"
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
	if existingCID, exists := Db.cidLookup[key]; exists {

		// retrieve the record for this key
		existingRecord, err := Db.getRecordFromCID(existingCID)
		if err != nil {
			return err
		}

		// check UUIDs
		if existingRecord.GetUuid() != record.GetUuid() {
			return ErrAttemptedOverwrite
		}

		// if the existingCID in the local keystore does not match the previousCID of the incoming Record it is an attempted overwrite
		if existingCID != record.GetPreviousCID() {
			return ErrRecordHistory
		}

		// otherwise this is an attempted update, check that the incoming Record is more recent
		if !starkhelpers.CheckTimeStamp(existingRecord.GetLastUpdatedTimestamp(), record.GetLastUpdatedTimestamp()) {
			return ErrAttemptedUpdate
		}
		record.AddComment("Set: updating record.")

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

	// link the record CID to the project directory and take a snapshot
	snapshotUpdate, err := Db.ipfsClient.AddLink(Db.ctx, Db.snapshotCID, cid, key)
	if err != nil {
		return errors.Wrap(err, ErrSnapshotUpdate.Error())
	}
	Db.snapshotCID = snapshotUpdate

	// if announcing, do it now
	if Db.announcing {

		// TODO: send proto data instead of CID
		if err := Db.publishAnnouncement([]byte(cid)); err != nil {
			return err
		}
	}

	// add the returned CID to the local keystore
	Db.cidLookup[key] = cid
	Db.currentNumEntries++
	return nil
}

// Get will retrieve a Record from the starkDB using the provided key.
func (Db *Db) Get(key string) (*Record, error) {
	Db.lock.Lock()
	defer Db.lock.Unlock()

	// check the local keystore for the provided key
	cid, ok := Db.cidLookup[key]
	if !ok {
		return nil, ErrNotFound(key)
	}

	// use the helper method to retrieve the Record
	return Db.getRecordFromCID(cid)
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
	cid, ok := Db.cidLookup[key]
	if !ok {
		return ErrNotFound(key)
	}

	// unlink the record CID from the project directory and update a snapshot
	snapshotUpdate, err := Db.ipfsClient.RmLink(Db.ctx, Db.snapshotCID, key)
	if err != nil {
		return errors.Wrap(err, ErrSnapshotUpdate.Error())
	}
	Db.snapshotCID = snapshotUpdate

	// unpin the file
	if err := Db.ipfsClient.Unpin(Db.ctx, cid); err != nil {
		return err
	}

	// TODO: if using a pinning service - will need to request an unpin there

	// remove from the keystore
	delete(Db.cidLookup, key)
	Db.currentNumEntries--
	return nil
}

// getRecordFromCID is a helper method that collects a Record from
// the IPFS using its CID string.
func (Db *Db) getRecordFromCID(cid string) (*Record, error) {
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
			return nil, errors.Wrap(err, ErrEncrypted.Error())
		}
	}

	// add the pulled CID to this record
	record.PreviousCID = cid
	return record, nil
}
