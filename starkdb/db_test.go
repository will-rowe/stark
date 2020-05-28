package stark

import (
	"os"
	"testing"
)

var (
	testFile    = "../README.md"
	testDir     = "../schema"
	testFileKey = "test file"
	testProject = "test project"
	testKey     = "test entry"

	// fields for a test record
	testAlias       = "test record"
	testDescription = "this is a test record"

	// tmp locations for database and result writes
	tmpDir     = "tmp"
	tmpDB      = tmpDir + "/DB"
	resultFile = tmpDir + "/downloadedFile.md"
	resultDir  = tmpDir + "/downloadedDir"
)

// CleanUp will cleanup the tmp directory created by the tests
func CleanUp() error {
	return os.RemoveAll(tmpDir)
}

// TestNewDB will check database initialisation.
func TestNewDB(t *testing.T) {

	// init the starkDB with an ephemeral IPFS node
	starkdb, teardown, err := OpenDB(testProject, SetLocalStorageDir(tmpDB), SetEphemeral(true), SetPin(true))
	if err != nil {
		t.Fatal(err)
	}

	// check the setup options propagated
	if starkdb.keystorePath != tmpDB {
		t.Fatal("starkdb's keystore path does not match the provided one")
	}
	if !starkdb.ephemeral {
		t.Fatal("starkdb is not labelled as ephemeral")
	}
	if !starkdb.pinning {
		t.Fatal("IPFS node was told to pin but is not set for pinning")
	}

	// check the helper methods
	if !starkdb.IsOnline() {
		t.Fatal("IPFS node is offline")
	}
	nodeID, err := starkdb.GetNodeIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if len(nodeID) == 0 {
		t.Fatal("no IPFS node ID is registered")
	}

	// test the db teardown
	if err := teardown(); err != nil {
		t.Fatal(err)
	}
}

// TestSetGet will check a record can be added and retrieved from the IPFS.
func TestSetGet(t *testing.T) {

	// init the starkDB with a default IPFS node
	db, teardown, err := OpenDB(testProject, SetLocalStorageDir(tmpDB), SetPin(true))
	if err != nil {
		t.Fatal(err)
	}

	// teardown
	defer func() {
		if err := teardown(); err != nil {
			t.Fatal(err)
		}

		// remove the database keystore from disk
		if err := CleanUp(); err != nil {
			t.Fatal("tests could not remove tmp directory")
		}
	}()

	// create a record
	testRecord, err := NewRecord(SetAlias(testAlias))
	if err != nil {
		t.Fatal(err)
	}

	// add record to stark
	if err := db.Set(testKey, testRecord); err != nil {
		t.Fatal(err)
	}

	// check it's in the local keyvalue store
	if _, ok := db.keystoreGet(testKey); !ok {
		t.Fatal("Set method did not add a CID to the local keystore")
	}

	// get record back from starkdb
	retrievedSample, err := db.Get(testKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(retrievedSample)

	// try adding duplicate record
	if err := db.Set(testKey, testRecord); err == nil {
		t.Fatal("duplicate sample was added")
	}
}

// ExampleOpenDB documents the usage of OpenDB.
func ExampleOpenDB() {

	// init the starkDB with functional options
	db, dbCloser, err := OpenDB("my new project", SetLocalStorageDir(tmpDB), SetPin(false))
	if err != nil {
		panic(err)
	}

	// defer the database closer
	defer dbCloser()

	// create a record
	record, err := NewRecord(SetAlias("my first sample"))
	if err != nil {
		panic(err)
	}

	// add record to starkDB
	err = db.Set("lookupKey", record)
	if err != nil {
		panic(err)
	}
}
