package stark

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

var (
	testFile    = "README.md"
	testDir     = "schema"
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
	if starkdb.announce {
		t.Fatal("starkdb is announcing but was not told to")
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

// TestReopenDB will check database re-opening and getting.
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

	// try the explorer link
	link, err := starkdb.GetExplorerLink(testKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(link)
}

// TestFileIO will check a file can be added and retrieved from the IPFS.
func TestFileIO(t *testing.T) {
	starkdb, teardown, err := OpenDB(SetProject(testProject), SetLocalStorageDir(tmpDB), WithPinning())
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	// add a file to IPFS
	cid, err := starkdb.addFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	// get the file back
	reader, err := starkdb.getFile(cid)
	if err != nil {
		t.Fatal(err)
	}

	// keep it as a byte slice for ease
	retrievedFile, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	// compare the retrieved file to the original
	origFile, err := ioutil.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(retrievedFile) != string(origFile) {
		t.Fatalf("retrieved file does not match original: %v vs %v", string(retrievedFile), string(origFile))
	}
}

// TestPubSub will check registering, announcing and listening.
func TestPubSub(t *testing.T) {
	starkdb, teardown, err := OpenDB(SetProject(testProject), SetLocalStorageDir(tmpDB), WithPinning(), WithAnnounce())
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()
	if !starkdb.announce {
		t.Fatal("db has no announce flag set")
	}

	// use a go routine to setup a Listener
	terminator := make(chan struct{})
	testErrs := make(chan error)
	receivedRecord := false
	go func() {
		recs, errs := starkdb.Listen(terminator)
		select {
		case rec := <-recs:
			t.Log("received record via PubSub: ", rec)
			receivedRecord = true
			break
		case err := <-errs:
			testErrs <- err
			break
		}
		close(terminator)
	}()

	// create a record
	testRecord, err := NewRecord(SetAlias(testAlias))
	if err != nil {
		t.Fatal(err)
	}

	// add record to starkdb and announce it
	if err := starkdb.Set(testKey, testRecord); err != nil {
		t.Fatal(err)
	}

	// wait to receive the record over PubSub
	select {
	default:
		<-terminator
	case err := <-testErrs:
		t.Fatal(err)
	}
	if !receivedRecord {
		t.Fatal("did not receive record via PubSub")
	}
}

// TestCleanup will cleanup the test tmp files.
func TestCleanup(t *testing.T) {
	if err := os.RemoveAll(tmpDir); err != nil {
		t.Fatal(err)
	}
}
