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
	Host        string      `json:"host_node"`
	HostAdd     string      `json:"host_address"`
	Pinning     bool        `json:"pinning"`
	Announcing  bool        `json:"announcing"`
	MaxEntries  int         `json:"max_entries"`
	CurrEntries int         `json:"current_entries"`
	Pairs       [][2]string `json:"contents"`
}

// MarshalJSON is used to satisify the JSON Marshaler
// interface for the Db but restricts data to that
// specified by the dbMetadata struct.
func (Db *Db) MarshalJSON() ([]byte, error) {
	nodeID, err := Db.GetNodeIdentity()
	if err != nil {
		return nil, err
	}
	nodeAdd, err := Db.GetNodeAddr()
	if err != nil {
		return nil, err
	}
	pairs := make([][2]string, Db.currentNumEntries)
	counter := 0
	for key, value := range Db.cidLookup {
		pairs[counter] = [2]string{key, value}
		counter++
	}
	return json.Marshal(dbMetadata{
		Db.project,
		Db.snapshotCID,
		nodeID,
		nodeAdd,
		Db.pinning,
		Db.announcing,
		Db.maxEntries,
		Db.currentNumEntries,
		pairs,
	})
}

// DumpMetadata returns a JSON string of starkDB metadata.
func (Db *Db) DumpMetadata() (string, error) {
	b, err := json.MarshalIndent(Db, "", "    ")
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
func (Db *Db) GetSnapshot() string {
	Db.lock.Lock()
	defer Db.lock.Unlock()
	if Db.currentNumEntries == 0 {
		return ""
	}
	return Db.snapshotCID
}

// GetCIDs will return a map of keys to CIDs for
// all Records currently held in the database.
func (Db *Db) GetCIDs() map[string]string {
	Db.lock.Lock()
	defer Db.lock.Unlock()
	return Db.cidLookup
}

// GetNodeIdentity returns the PeerID of the underlying IPFS
// node for the starkDB instance.
func (Db *Db) GetNodeIdentity() (string, error) {
	Db.lock.Lock()
	defer Db.lock.Unlock()
	if !Db.isOnline() {
		return "", ErrNodeOffline
	}
	id := Db.ipfsClient.PrintNodeID()
	if len(id) == 0 {
		return "", ErrNoPeerID
	}
	return id, nil
}

// GetNodeAddr returns the public address of the
// underlying IPFS node for the starkDB
// instance.
func (Db *Db) GetNodeAddr() (string, error) {
	Db.lock.Lock()
	defer Db.lock.Unlock()
	if !Db.isOnline() {
		return "", ErrNodeOffline
	}
	add, err := Db.ipfsClient.GetPublicIPv4Addr()
	if err != nil {
		return "", err
	}
	id := Db.ipfsClient.PrintNodeID()
	if len(id) == 0 {
		return "", ErrNoPeerID
	}
	return fmt.Sprintf("/ip4/%s/tcp/4001/p2p/%s", add, id), nil
}

/*
// GetExplorerLink will return an IPFS explorer link for
// a CID in the starkDB given the provided lookup key.
func (Db *Db) GetExplorerLink(key string) (string, error) {
	cid, ok := Db.keystoreGet(key)
	if !ok {
		return "", fmt.Errorf("could not retrieve CID from local keystore")
	}
	return fmt.Sprintf("IPLD Explorer link: https://explore.ipld.io/#/explore/%s \n", cid), nil
}
*/

// Listen will start a subscription to the IPFS PubSub network
// for messages matching the current database's project. It
// tries pulling Records from the IPFS via the announced
// CIDs, then returns them via a channel to the caller.
//
// It returns the Record channel, an Error channel which reports
// errors during message processing and Record retrieval, as well
// as any error during the PubSub setup.
func (Db *Db) Listen(terminator chan struct{}) (chan *Record, chan error, error) {
	if !Db.isOnline() {
		return nil, nil, ErrNodeOffline
	}

	// subscribe the node to the starkDB project
	if err := Db.ipfsClient.Subscribe(Db.ctx, Db.project); err != nil {
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
			case msg := <-Db.ipfsClient.GetPSMchan():

				// TODO: check sender peerID
				//msg.From()

				// get the CID
				cid := string(msg.Data())
				if _, ok := cidTracker[cid]; ok {
					continue
				}
				cidTracker[cid] = struct{}{}

				// collect the Record from IPFS
				collectedRecord, err := Db.getRecordFromCID(cid)
				if err != nil {
					errChan <- err
				} else {

					// add a comment to say this Record was from PubSub
					collectedRecord.AddComment(fmt.Sprintf("collected from %s via pubsub.", msg.From()))

					// send the record on to the caller
					recChan <- collectedRecord
				}

			case err := <-Db.ipfsClient.GetPSEchan():
				errChan <- err

			case <-terminator:
				if err := Db.ipfsClient.Unsubscribe(); err != nil {
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
func (Db *Db) publishAnnouncement(message []byte) error {
	if !Db.isOnline() {
		return ErrNodeOffline
	}
	if len(Db.project) == 0 {
		return ErrNoProject
	}
	return Db.ipfsClient.SendMessage(Db.ctx, Db.project, message)
}

// isOnline returns true if the starkDB is in online mode
// and the IPFS daemon is reachable.
// TODO: this needs some more work.
func (Db *Db) isOnline() bool {
	return Db.ipfsClient.Online() && Db.allowNetwork
}
