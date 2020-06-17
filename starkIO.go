package stark

import (
	"bytes"
	"fmt"

	"github.com/gogo/protobuf/jsonpb"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/pkg/errors"
	starkpinata "github.com/will-rowe/stark/src/pinata"
)

// GetSnapshot returns the current database snapshot
// CID, which can be used to rebuild the current
// database instance.
//
// Note: if the database has no entries, the
// returned snapshot will be a nil string.
func (starkdb *Db) GetSnapshot() string {
	starkdb.Lock()
	defer starkdb.Unlock()
	if starkdb.currentNumEntries == 0 {
		return ""
	}
	return starkdb.snapshotCID
}

// GetNumEntries returns the number of entries
// in the current database instance.
func (starkdb *Db) GetNumEntries() int {
	starkdb.Lock()
	defer starkdb.Unlock()
	return starkdb.currentNumEntries
}

// GetCIDs will return a map of keys to CIDs for
// all Records currently held in the database.
func (starkdb *Db) GetCIDs() map[string]string {
	starkdb.Lock()
	defer starkdb.Unlock()
	return starkdb.cidLookup
}

// GetNodeAddr returns the public address of the
// underlying IPFS node for the starkDB
// instance.
func (starkdb *Db) GetNodeAddr() (string, error) {
	if !starkdb.isOnline() {
		return "", ErrNodeOffline
	}
	nodeID := starkdb.ipfsClient.PrintNodeID()
	add, err := starkdb.ipfsClient.GetPublicIPv4Addr()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/p2p/%s", add, nodeID), nil
}

// PinataPublish will issue an API call to the pinata
// pinByHash endpoint and pin the current database
// instance.
func (starkdb *Db) PinataPublish(apiKey, apiSecret string) (*starkpinata.APIResponse, error) {
	hostAddress, err := starkdb.GetNodeAddr()
	if err != nil {
		return nil, err
	}
	pinataClient, err := starkpinata.NewClient(apiKey, apiSecret, hostAddress)
	if err != nil {
		return nil, err
	}
	meta := starkpinata.NewMetadata(starkdb.project)
	resp, err := pinataClient.PinByHashWithMetadata(starkdb.snapshotCID, meta)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Listen will start a subscription to the IPFS PubSub network
// for messages matching the current database's project. It
// tries pulling Records from the IPFS via the announced
// CIDs, then returns them via a channel to the caller.
//
// It returns the Record channel, an Error channel which reports
// errors during message processing and Record retrieval, as well
// as any error during the PubSub setup.
func (starkdb *Db) Listen(terminator chan struct{}) (chan *Record, chan error, error) {
	if !starkdb.isOnline() {
		return nil, nil, ErrNodeOffline
	}

	// subscribe the node to the starkDB project
	if err := starkdb.ipfsClient.Subscribe(starkdb.ctx, starkdb.project); err != nil {
		return nil, nil, err
	}

	// cidTracker skips listener over duplicate CIDs
	cidTracker := make(map[string]struct{})

	// create channels used to send Records and errors back to the caller
	recChan := make(chan *Record, DefaultBufferSize)
	errChan := make(chan error)

	// process the incoming messages
	go func() {
		for {
			select {
			case msg := <-starkdb.ipfsClient.GetPSMchan():

				// TODO: check sender peerID
				//msg.From()

				// get the CID
				cid := string(msg.Data())
				if _, ok := cidTracker[cid]; ok {
					continue
				}
				cidTracker[cid] = struct{}{}

				// collect the Record from IPFS
				collectedRecord, err := starkdb.getRecordFromCID(cid)
				if err != nil {
					errChan <- err
				} else {

					// add a comment to say this Record was from PubSub
					collectedRecord.AddComment(fmt.Sprintf("collected from %s via pubsub.", msg.From()))

					// send the record on to the caller
					recChan <- collectedRecord
				}

			case err := <-starkdb.ipfsClient.GetPSEchan():
				errChan <- err

			case <-terminator:
				if err := starkdb.ipfsClient.Unsubscribe(); err != nil {
					errChan <- err
				}
				close(recChan)
				close(errChan)
				return
			}
		}
	}()
	return recChan, errChan, nil
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

// publishAnnouncement will send a PubSub message on the topic
// of the database project.
func (starkdb *Db) publishAnnouncement(message []byte) error {
	if !starkdb.isOnline() {
		return ErrNodeOffline
	}
	if len(starkdb.project) == 0 {
		return ErrNoProject
	}
	return starkdb.ipfsClient.SendMessage(starkdb.ctx, starkdb.project, message)
}

// isOnline returns true if the starkDB is in online mode
// and the IPFS daemon is reachable.
// TODO: this needs some more work.
func (starkdb *Db) isOnline() bool {
	return starkdb.ipfsClient.Online()
}

// send2log will send a message to the log if one
// is attached.
func (starkdb *Db) send2log(msg interface{}) {
	if msg == nil || starkdb.loggingChan == nil {
		return
	}
	starkdb.loggingChan <- msg
}
