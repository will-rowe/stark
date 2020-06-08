package stark

import (
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/badger"
)

// dbMetadata is used to dump starkDB metadata to
// JSON.
type dbMetadata struct {
	Project      string      `json:"project"`
	Host         string      `json:"host_node"`
	KeystorePath string      `json:"keystore"`
	Pinning      bool        `json:"pinning"`
	Announcing   bool        `json:"announcing"`
	MaxEntries   int         `json:"max_entries"`
	CurrEntries  int         `json:"current_entries"`
	Pairs        [][2]string `json:"contents"`
}

// MarshalJSON is used to satisify the JSON Marshaler
// interface for the Db but restricts data to that
// specified by the dbMetadata struct.
func (Db *Db) MarshalJSON() ([]byte, error) {
	nodeID, err := Db.GetNodeIdentity()
	if err != nil {
		return nil, err
	}
	pairs := make([][2]string, Db.currentNumEntries)
	counter := 0
	for entry := range Db.RangeCIDs() {
		if entry.Error != nil {
			return nil, err
		}
		pairs[counter] = [2]string{entry.Key, entry.CID}
		counter++
	}
	return json.Marshal(dbMetadata{
		Db.project,
		nodeID,
		Db.keystorePath,
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

// RangeCIDs is used to iterate over all the starkDB keys
// and their corresponding Record CIDs.
//
// This method will return a channel of KeyCIDpair from the
// StarkDB. The caller does not need to close the returned
// channel.
func (Db *Db) RangeCIDs() chan KeyCIDpair {

	// setup the channel to send key values
	returnChan := make(chan KeyCIDpair)

	// iterate over the badger key value store
	go func() {
		err := Db.keystore.View(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.PrefetchValues = false
			it := txn.NewIterator(opts)
			defer it.Close()
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				key := item.Key()
				err := item.Value(func(cid []byte) error {
					returnChan <- KeyCIDpair{string(key), string(cid), nil}
					return nil
				})
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			returnChan <- KeyCIDpair{"", "", err}
		}
		close(returnChan)
	}()
	return returnChan
}

// Listen will start a subscription to the IPFS PubSub network
// for messages matching the current database's project. It
// tries pulling Records from the IPFS via the announced
// CIDs, then returns them via a channel to the caller.
//
// It returns the Record channel, an Error channel which reports
// errors during message processing and Record retrieval, as well
// as any error during the PubSub setup.
func (Db *Db) Listen(terminator chan struct{}) (chan *Record, chan error, error) {
	if !Db.IsOnline() {
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
				collectedRecord, err := Db.GetRecordFromCID(cid)
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
	if !Db.IsOnline() {
		return ErrNodeOffline
	}
	if len(Db.project) == 0 {
		return ErrNoProject
	}
	return Db.ipfsClient.SendMessage(Db.ctx, Db.project, message)
}

// IsOnline returns true if the starkDB is in online mode
// and the IPFS daemon is reachable.
// TODO: this needs some more work.
func (Db *Db) IsOnline() bool {
	return Db.ipfsClient.Online() && Db.allowNetwork
}

// GetNodeIdentity returns the PeerID of the underlying IPFS
// node for the starkDB instance.
func (Db *Db) GetNodeIdentity() (string, error) {
	Db.lock.Lock()
	defer Db.lock.Unlock()
	if !Db.IsOnline() {
		return "", ErrNodeOffline
	}
	id := Db.ipfsClient.PrintNodeID()
	if len(id) == 0 {
		return "", ErrNoPeerID
	}
	return id, nil
}

// refreshCount will clear the record counter and then
// recount the number of record keys that are in the
// starkDB.
func (Db *Db) refreshCount() error {
	Db.lock.Lock()
	defer Db.lock.Unlock()
	Db.currentNumEntries = 0
	err := Db.keystore.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			Db.currentNumEntries++
		}
		return nil
	})
	if err != nil {
		return err
	}
	if Db.currentNumEntries > Db.maxEntries {
		return ErrMaxEntriesExceeded
	}
	return nil
}
