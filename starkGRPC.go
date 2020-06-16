package stark

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"
	starkhelpers "github.com/will-rowe/stark/src/helpers"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// Get will retrieve a copy of a Record
// from the starkDB using the provided
// lookup key.
func (starkdb *Db) Get(ctx context.Context, key *Key) (*Record, error) {
	starkdb.Lock()
	defer starkdb.Unlock()

	// check the local keystore for the provided key
	cid, ok := starkdb.cidLookup[key.GetId()]
	if !ok {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("no Record in database for key: %v", key.GetId()))
	}

	// use the helper method to retrieve the Record
	record, err := starkdb.getRecordFromCID(cid)
	if err != nil {
		return nil, err
	}

	starkdb.send2log(fmt.Sprintf("record retrieved: %v->%v", key.GetId(), cid))
	return record, nil
}

// Set will add a copy of a Record to the
// starkDB, using the Record Alias as a
// lookup key.
//
// This method will add comments to the
// Record's history before adding it to the
// IPFS.
//
// It will return a pointer to a copy of the
// Record that was added to the starkDB,
// which contains the updated Record history
// and the CID for the Record in the IPFS.
func (starkdb *Db) Set(ctx context.Context, record *Record) (*Record, error) {
	starkdb.Lock()
	defer starkdb.Unlock()

	// use the record alias as the lookup key
	key := record.GetAlias()
	if len(key) == 0 {
		return nil, ErrNoKey
	}

	// check the local keystore to see if this key has been used before
	if existingCID, exists := starkdb.cidLookup[key]; exists {

		// retrieve the record for this key
		existingRecord, err := starkdb.getRecordFromCID(existingCID)
		if err != nil {
			return nil, err
		}

		// check UUIDs
		if existingRecord.GetUuid() != record.GetUuid() {
			return nil, ErrAttemptedOverwrite
		}

		// if the existingCID in the local keystore does not match the previousCID of the incoming Record it is an attempted overwrite
		if existingCID != record.GetPreviousCID() {
			return nil, ErrRecordHistory
		}

		// otherwise this is an attempted update, check that the incoming Record is more recent
		if !starkhelpers.CheckTimeStamp(existingRecord.GetLastUpdatedTimestamp(), record.GetLastUpdatedTimestamp()) {
			return nil, ErrAttemptedUpdate
		}
		record.AddComment("Set: updating record.")

	}
	record.AddComment("Set: adding record to IPFS.")

	// if encrypting requested and Record isn't already, do it now
	if len(starkdb.cipherKey) != 0 && !record.GetEncrypted() {
		if err := record.Encrypt(starkdb.cipherKey); err != nil {
			return nil, err
		}
	}

	// marshal Record data to JSON
	jsonData, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}

	// create DAG node in IPFS for Record data
	cid, err := starkdb.ipfsClient.DagPut(starkdb.ctx, jsonData, starkdb.pinning)
	if err != nil {
		return nil, err
	}

	// link the record CID to the project directory and take a snapshot
	snapshotUpdate, err := starkdb.ipfsClient.AddLink(starkdb.ctx, starkdb.snapshotCID, cid, key)
	if err != nil {
		return nil, errors.Wrap(err, ErrSnapshotUpdate.Error())
	}
	starkdb.snapshotCID = snapshotUpdate

	// if announcing, do it now
	if starkdb.announcing {

		// TODO: send proto data instead of CID
		if err := starkdb.publishAnnouncement([]byte(cid)); err != nil {
			return nil, err
		}
	}

	// add the returned CID to the local keystore
	starkdb.cidLookup[key] = cid
	starkdb.currentNumEntries++
	starkdb.sessionEntries++

	// job done
	starkdb.send2log(fmt.Sprintf("record added: %v->%v", key, cid))

	// use pinata if session interval reached
	if starkdb.pinataInterval > 0 {
		if starkdb.sessionEntries%starkdb.pinataInterval == 0 {
			starkdb.send2log("pinning interval reached, uploading database to Pinata")
			var k, s string
			var ok1, ok2 bool
			if k, ok1 = os.LookupEnv(DefaultPinataAPIkey); !ok1 {
				return nil, ErrPinataKey
			}
			if s, ok2 = os.LookupEnv(DefaultPinataSecretKey); !ok2 {
				return nil, ErrPinataSecret
			}
			go func() {
				pinataResp, err := starkdb.PinataPublish(k, s)
				if err != nil {
					starkdb.send2log(fmt.Sprintf("pinata error: %v", err))
					return
				}
				starkdb.send2log(fmt.Sprintf("pinata API response: %v", pinataResp.Status))
			}()
		}
	}

	// add the CID to the record and return
	record.PreviousCID = cid
	return record, nil
}

// Dump returns the metadata from a starkDB instance.
//
// Note: input key is currently unused.
func (starkdb *Db) Dump(ctx context.Context, key *Key) (*DbMeta, error) {
	starkdb.Lock()
	defer starkdb.Unlock()

	nodeAdd, err := starkdb.GetNodeAddr()
	if err != nil {
		return nil, err
	}
	return &DbMeta{
		Project:     starkdb.project,
		Snapshot:    starkdb.snapshotCID,
		NodeAddress: nodeAdd,
		Pinning:     starkdb.pinning,
		Announcing:  starkdb.announcing,
		CurrEntries: int32(starkdb.currentNumEntries),
		Pairs:       starkdb.cidLookup,
	}, nil
}
