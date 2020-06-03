package stark

import (
	"encoding/json"
	"fmt"
	"os"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// dbMetadata is used to dump database metadata to
// JSON.
type dbMetadata struct {
	Project      string `json:"project"`
	KeystorePath string `json:"keystore"`
	Pinning      bool   `json:"pinning"`
	Announcing   bool   `json:"announcing"`
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
