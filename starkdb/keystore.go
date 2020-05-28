package stark

import (
	"fmt"

	"github.com/dgraph-io/badger"
)

// GetExplorerLink will return an IPFS explorer link for a CID in the local keystore given the provided lookup key.
func (DB *DB) GetExplorerLink(key string) (string, error) {
	cid, ok := DB.keystoreGet(key)
	if !ok {
		return "", fmt.Errorf("could not retrieve CID from local keystore")
	}
	return fmt.Sprintf("IPLD Explorer link: https://explore.ipld.io/#/explore/%s \n", cid), nil
}

// keystoreSet will add a key value pair to the local keystore.
func (DB *DB) keystoreSet(key, value string) error {
	txn := DB.keystore.NewTransaction(true)
	defer txn.Discard()
	err := txn.Set([]byte(key), []byte(value))
	if err != nil {
		return err
	}
	return txn.Commit()
}

// keystoreGet will get a key value pair from the local keystore.
func (DB *DB) keystoreGet(key string) (string, bool) {
	txn := DB.keystore.NewTransaction(false)
	defer txn.Discard()
	item, err := txn.Get([]byte(key))
	if err != nil {
		if err != badger.ErrKeyNotFound {
			panic(err)
		}
		return "", false
	}
	var returnedValue []byte
	err = item.Value(func(val []byte) error {
		returnedValue = val
		return nil
	})
	if err != nil {
		panic(err)
	}
	return string(returnedValue), true
}
