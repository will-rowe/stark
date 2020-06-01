package stark

import (
	"context"
	"os"
	"strings"
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

// cleanUp will cleanup the tmp directory created by the tests
func cleanUp() error {
	return os.RemoveAll(tmpDir)
}

// TestIPFSclient will run an IPFS client and check some methods.
func TestIPFSclient(t *testing.T) {

	// get some context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// start the client
	client, err := newIPFSclient(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// print the listeners
	client.printListeners()

	// test the closer
	if err := client.endSession(); err != nil {
		t.Fatal(err)
	}
}

// TestNewDB will check database initialisation and set/get operation.
func TestNewDB(t *testing.T) {

	// init the starkDB
	starkdb, teardown, err := OpenDB(SetProject(testProject), SetLocalStorageDir(tmpDB), WithPinning())
	if err != nil {
		t.Fatal(err)
	}

	// check the setup options propagated
	if starkdb.project != strings.ReplaceAll(testProject, " ", "_") {
		t.Fatal("starkdb's project does not match the provided one")
	}
	if starkdb.keystorePath != tmpDB {
		t.Fatal("starkdb's keystore path does not match the provided one")
	}
	if !starkdb.pinning {
		t.Fatal("IPFS node was told to pin but is not set for pinning")
	}

	// create a record
	testRecord, err := NewRecord(SetAlias(testAlias))
	if err != nil {
		t.Fatal(err)
	}

	// add record to stark
	if err := starkdb.Set(testKey, testRecord); err != nil {
		t.Fatal(err)
	}

	// check it's in the local keyvalue store
	if _, ok := starkdb.keystoreGet(testKey); !ok {
		t.Fatal("Set method did not add a CID to the local keystore")
	}

	// get record back from starkdb
	retrievedSample, err := starkdb.Get(testKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(retrievedSample)

	// try adding duplicate record
	if err := starkdb.Set(testKey, testRecord); err == nil {
		t.Fatal("duplicate sample was added")
	}

	// test the db teardown
	if err := teardown(); err != nil {
		t.Fatal(err)
	}
}

// TestReopenDB will check database re-opening.
func TestReopenDB(t *testing.T) {

	// test you can reopen the starkDB
	starkdb, teardown, err := OpenDB(SetProject(testProject), SetLocalStorageDir(tmpDB), WithPinning())
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	// get record back from starkdb
	retrievedSample, err := starkdb.Get(testKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(retrievedSample)

	// clean up the test
	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}

}

/*
// TestFileIO will check a file can be added and retrieved from the IPFS
func TestFileIO(t *testing.T) {

	// init the starkdb with a default IPFS node
	db, teardown, err := OpenDB(testProject, SetLocalStorageDir(tmpDB))
	if err != nil {
		t.Fatal(err)
	}

	// teardown
	defer func() {
		if err := teardown(); err != nil {
			t.Fatal(err)
		}
	}()

	// add a file to IPFS
	if err := db.addFile(testFile, testFileKey); err != nil {
		t.Fatal(err)
	}

	// check you can't add with the same key again
	if err := db.addFile(testFile, testFileKey); err == nil {
		t.Fatal("used a duplicate key")
	}

	// check the local database has the CID
	if link, err := db.GetExplorerLink(testFileKey); err != nil {
		t.Fatal(err)
	} else {
		t.Log(link)
	}

	// get the file back
	if err := db.getFile(testFileKey, resultFile); err != nil {
		t.Fatal(err)
	}

	// check that it exists on the local filesystem
	if err := checkFile(resultFile); err != nil {
		t.Fatal(err)
	}
}

*/
