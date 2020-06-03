package stark

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dgraph-io/badger"
	"google.golang.org/protobuf/types/known/timestamppb"
)

/*
// RangeObject is returned from the Range method
// for each entry in the starkDB. It contains
// the Key, Record CID and any error.
type RangeObject struct {
	Key   string
	CID   string
	Error error
}

// Range will return a channel of RangeObjects from the
// StarkDB. Each Range object will contain a starkDB
// key, associated Record CID and any error encountered
// during lookup.
func (Db *Db) Range() chan *RangeObject {
	dataChan := make(chan *RangeObject)
	go func() {
		stream := Db.keystore.NewStream()

		stream.Send = func(kvlist *pb.KVList) (err error) {
			for _, kv := range kvlist.Kv {
				k, v := kv.GetKey(), kv.GetValue()
				if _, err = fmt.Fprintf(out, "+%d,%d:%s->", len(k), len(v), k); err != nil {
					return err
				}
				if _, err := out.Write(v); err != nil {
					return err
				}
				if err = out.WriteByte('\n'); err != nil {
					return err
				}
			}
			return nil
		}


		stream.Send = func(list *pb.KVList) error {
			return proto.MarshalText(w, list) // Write to w.
		  }
	}()
	return dataChan
}
*/

// dbMetadata is used to dump database metadata to
// JSON.
type dbMetadata struct {
	Project      string `json:"project"`
	KeystorePath string `json:"keystore"`
	Pinning      bool   `json:"pinning"`
	Announcing   bool   `json:"announcing"`
	NumKeys      int    `json:"number_of_keys"`
}

// MarshalJSON is used to satisify the JSON Marshaler
// interface for the Db but restricts data to that
// specified by the dbMetadata struct.
func (Db *Db) MarshalJSON() ([]byte, error) {
	return json.Marshal(dbMetadata{
		Db.project,
		Db.keystorePath,
		Db.pinning,
		Db.announcing,
		Db.numKeys,
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
	Db.numKeys = 0
	err := Db.keystore.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			Db.numKeys++
			//item := it.Item()
			//k := item.Key()
		}
		return nil
	})
	return err
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
