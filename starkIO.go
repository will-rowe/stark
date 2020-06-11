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
func (starkdb *Db) Set(key string, record *Record) error {
	starkdb.Lock()
	defer starkdb.Unlock()

	// max entry check
	if starkdb.currentNumEntries == starkdb.maxEntries {
		return ErrMaxEntriesExceeded
	}

	// check the local keystore to see if this key has been used before
	if existingCID, exists := starkdb.cidLookup[key]; exists {

		// retrieve the record for this key
		existingRecord, err := starkdb.getRecordFromCID(existingCID)
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
	if len(starkdb.cipherKey) != 0 && !record.GetEncrypted() {
		if err := record.Encrypt(starkdb.cipherKey); err != nil {
			return err
		}
	}

	// marshal Record data to JSON
	jsonData, err := json.Marshal(record)
	if err != nil {
		return err
	}

	// create DAG node in IPFS for Record data
	cid, err := starkdb.ipfsClient.DagPut(starkdb.ctx, jsonData, starkdb.pinning)
	if err != nil {
		return err
	}

	// link the record CID to the project directory and take a snapshot
	snapshotUpdate, err := starkdb.ipfsClient.AddLink(starkdb.ctx, starkdb.snapshotCID, cid, key)
	if err != nil {
		return errors.Wrap(err, ErrSnapshotUpdate.Error())
	}
	starkdb.snapshotCID = snapshotUpdate

	// if announcing, do it now
	if starkdb.announcing {

		// TODO: send proto data instead of CID
		if err := starkdb.publishAnnouncement([]byte(cid)); err != nil {
			return err
		}
	}

	// add the returned CID to the local keystore
	starkdb.cidLookup[key] = cid
	starkdb.currentNumEntries++
	return nil
}

// Get will retrieve a Record from the starkDB using the provided key.
func (starkdb *Db) Get(key string) (*Record, error) {
	starkdb.Lock()
	defer starkdb.Unlock()

	// check the local keystore for the provided key
	cid, ok := starkdb.cidLookup[key]
	if !ok {
		return nil, ErrNotFound(key)
	}

	// use the helper method to retrieve the Record
	return starkdb.getRecordFromCID(cid)
}

// Delete will delete an entry from starkdb. This involves
// removing the key and Record CID from the local store,
// as well as unpinning the Record from the IPFS.
//
// Note: I'm not sure how this behaves if the Record
// wasn't pinned in the IPFS in the first place.
func (starkdb *Db) Delete(key string) error {
	starkdb.Lock()
	defer starkdb.Unlock()

	// check the local keystore for the provided key
	cid, ok := starkdb.cidLookup[key]
	if !ok {
		return ErrNotFound(key)
	}

	// unlink the record CID from the project directory and update a snapshot
	snapshotUpdate, err := starkdb.ipfsClient.RmLink(starkdb.ctx, starkdb.snapshotCID, key)
	if err != nil {
		return errors.Wrap(err, ErrSnapshotUpdate.Error())
	}
	starkdb.snapshotCID = snapshotUpdate

	// unpin the file
	if err := starkdb.ipfsClient.Unpin(starkdb.ctx, cid); err != nil {
		return err
	}

	// TODO: if using a pinning service - will need to request an unpin there

	// remove from the keystore
	delete(starkdb.cidLookup, key)
	starkdb.currentNumEntries--
	return nil
}

// getRecordFromCID is a helper method that collects a Record from
// the IPFS using its CID string.
func (starkdb *Db) getRecordFromCID(cid string) (*Record, error) {
	if len(cid) == 0 {
		return nil, ErrNoCID
	}

	// retrieve the record data from the IPFS
	retrievedNode, err := starkdb.ipfsClient.DagGet(starkdb.ctx, cid)
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
		if err := record.Decrypt(starkdb.cipherKey); err != nil {
			return nil, errors.Wrap(err, ErrEncrypted.Error())
		}
	}

	// add the pulled CID to this record
	record.PreviousCID = cid
	return record, nil
}
