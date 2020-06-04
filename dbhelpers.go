package stark

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dgraph-io/badger"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// dbMetadata is used to dump starkDB metadata to
// JSON.
type dbMetadata struct {
	Project      string `json:"project"`
	Host         string `json:"host_node"`
	KeystorePath string `json:"keystore"`
	Pinning      bool   `json:"pinning"`
	Announcing   bool   `json:"announcing"`
	MaxEntries   int    `json:"max_entries"`
	CurrEntries  int    `json:"current_entries"`
}

// MarshalJSON is used to satisify the JSON Marshaler
// interface for the Db but restricts data to that
// specified by the dbMetadata struct.
func (Db *Db) MarshalJSON() ([]byte, error) {
	nodeID, err := Db.GetNodeIdentity()
	if err != nil {
		return nil, err
	}
	return json.Marshal(dbMetadata{
		Db.project,
		nodeID,
		Db.keystorePath,
		Db.pinning,
		Db.announcing,
		Db.maxEntries,
		Db.currentNumEntries,
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

// IsOnline returns true if the starkDB is in online mode
// and the IPFS daemon is reachable.
func (Db *Db) IsOnline() bool {
	return Db.ipfsClient.node.IsOnline && Db.allowNetwork
}

// GetNodeIdentity returns the PeerID of the underlying IPFS
// node for the starkDB.
func (Db *Db) GetNodeIdentity() (string, error) {
	Db.lock.Lock()
	defer Db.lock.Unlock()
	if !Db.IsOnline() {
		return "", ErrNodeOffline
	}
	if len(Db.ipfsClient.node.Identity) == 0 {
		return "", ErrNoPeerID
	}
	return Db.ipfsClient.node.Identity.Pretty(), nil
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

// checkDir is a function to check that a directory exists
func checkDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("no directory specified")
	}
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %v", dir)
		}
		return fmt.Errorf("can't access adirectory (check permissions): %v", dir)
	}
	return nil
}

// checkFile is a function to check that a file can be read
func checkFile(file string) error {
	fi, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %v", file)
		}
		return fmt.Errorf("can't access file (check permissions): %v", file)
	}
	if fi.Size() == 0 {
		return fmt.Errorf("file appears to be empty: %v", file)
	}
	return nil
}

// checkTimeStamp will return true if the new protobuf timestamp is more recent than the old one.
func checkTimeStamp(old, new *timestamppb.Timestamp) bool {
	if old.GetSeconds() > new.GetSeconds() {
		return false
	}
	if old.GetSeconds() == new.GetSeconds() {
		if old.GetNanos() >= new.GetNanos() {
			return false
		}
	}
	return true
}
