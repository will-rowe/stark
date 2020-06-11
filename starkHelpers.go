package stark

import (
	"encoding/json"
	"fmt"
)

// dbMetadata is used to dump starkDB metadata to
// JSON.
type dbMetadata struct {
	Project     string      `json:"project"`
	Snapshot    string      `json:"snapshot"`
	HostAdd     string      `json:"host_address"`
	Pinning     bool        `json:"pinning"`
	Announcing  bool        `json:"announcing"`
	CurrEntries int         `json:"current_entries"`
	Pairs       [][2]string `json:"contents"`
}

// MarshalJSON is used to satisify the JSON Marshaler
// interface for the Db but restricts data to that
// specified by the dbMetadata struct.
func (starkdb *Db) MarshalJSON() ([]byte, error) {
	nodeAdd, err := starkdb.GetNodeAddr()
	if err != nil {
		return nil, err
	}
	pairs := make([][2]string, starkdb.currentNumEntries)
	counter := 0
	for key, value := range starkdb.cidLookup {
		pairs[counter] = [2]string{key, value}
		counter++
	}
	return json.Marshal(dbMetadata{
		starkdb.project,
		starkdb.snapshotCID,
		nodeAdd,
		starkdb.pinning,
		starkdb.announcing,
		starkdb.currentNumEntries,
		pairs,
	})
}

// DumpMetadata returns a JSON string of starkDB metadata.
func (starkdb *Db) DumpMetadata() (string, error) {
	b, err := json.MarshalIndent(starkdb, "", "    ")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s\n", string(b)), nil
}

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

// GetNodeIdentity returns the PeerID of the underlying IPFS
// node for the starkDB instance.
func (starkdb *Db) GetNodeIdentity() (string, error) {
	starkdb.Lock()
	defer starkdb.Unlock()
	if !starkdb.isOnline() {
		return "", ErrNodeOffline
	}
	id := starkdb.ipfsClient.PrintNodeID()
	if len(id) == 0 {
		return "", ErrNoPeerID
	}
	return id, nil
}

// GetNodeAddr returns the public address of the
// underlying IPFS node for the starkDB
// instance.
func (starkdb *Db) GetNodeAddr() (string, error) {
	nodeID, err := starkdb.GetNodeIdentity()
	if err != nil {
		return "", err
	}
	starkdb.Lock()
	defer starkdb.Unlock()
	add, err := starkdb.ipfsClient.GetPublicIPv4Addr()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/ip4/%s/tcp/4001/p2p/%s", add, nodeID), nil
}

// PinataPublish

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
