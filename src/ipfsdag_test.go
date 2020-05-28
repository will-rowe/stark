// Package stark description.
package stark

import (
	"encoding/json"
	"testing"
)

// TestDagPutGet will check serialised data can be added and retrieved from the IPLD
func TestDagPutGet(t *testing.T) {
	defer CleanUp()

	// init the starkDB with a default IPFS node
	starkdb, teardown, err := OpenDB(testProject, SetLocalStorageDir(tmpDB), SetPin(true))
	if err != nil {
		t.Fatal(err)
	}

	// teardown
	defer func() {
		if err := teardown(); err != nil {
			t.Fatal(err)
		}
	}()

	// create a record
	testRecord, err := NewRecord(SetAlias(testAlias))
	if err != nil {
		t.Fatal(err)
	}

	// marshal to JSON
	jsonData, err := json.Marshal(testRecord)
	if err != nil {
		t.Fatal(err)
	}

	// dag put
	cid, err := starkdb.dagPut(jsonData)
	if err != nil {
		t.Fatal(err)
	}

	// dag get
	retrievedData, err := starkdb.dagGet(cid + "/uuid")
	if err != nil {
		t.Fatal(err)
	}
	if retrievedData != testRecord.GetUuid() {
		t.Fatal("failed to retrieve correct field value for uuid")
	}
}
