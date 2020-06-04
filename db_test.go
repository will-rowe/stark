package stark

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

var (
	testFile       = "README.md"
	testDir        = "schema"
	testFileKey    = "test file"
	testProject    = "test_project"
	testAltProject = "snapshotted_project"
	testKey        = "test entry"

	// fields for a test record
	testAlias       = "test record"
	testDescription = "this is a test record"

	// tmp locations for database and result writes
	tmpDir  = "tmp"
	tmpFile = tmpDir + "/downloadedFile.md"
)

// TestIPFSclient will run an IPFS client and check some methods.
func TestIPFSclient(t *testing.T) {

	// get some context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// get some bootstrappers
	bootstrappers, err := setupBootstrappers(DefaultBootstrappers)
	if err != nil {
		t.Fatal(err)
	}

	// start the client
	client, err := newIPFSclient(ctx, bootstrappers)
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
	numBootstrappers := 3
	numEntries := 1

	// init the starkDB
	starkdb, teardown, err := OpenDB(SetProject(testProject), SetLocalStorageDir(tmpDir), SetBootstrappers(DefaultBootstrappers[:numBootstrappers]), SetKeyLimit(numEntries))
	if err != nil {
		t.Fatal(err)
	}

	// check the setup options propagated
	if starkdb.project != strings.ReplaceAll(testProject, " ", "_") {
		t.Fatal("starkDB's project does not match the provided one")
	}
	if starkdb.keystorePath != fmt.Sprintf("%s/%s", tmpDir, testProject) {
		t.Fatalf("starkDB's keystore path does not look right: %v", starkdb.keystorePath)
	}
	if len(starkdb.bootstrappers) != numBootstrappers {
		t.Fatalf("starkDB does not have expected number of bootstrappers listed: %d", len(starkdb.bootstrappers))
	}
	if !starkdb.pinning {
		t.Fatal("IPFS node was told to pin but is not set for pinning")
	}
	if starkdb.announcing {
		t.Fatal("starkDB is announcing but was not told to")
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

	// get record back from starkDB
	retrievedSample, err := starkdb.Get(testKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(retrievedSample)

	// try adding duplicate record
	if err := starkdb.Set(testKey, testRecord); err == nil {
		t.Fatal("duplicate sample was added")
	}

	// try adding a non-duplicate record
	if err := starkdb.Set("another key", testRecord); err != ErrMaxEntriesExceeded {
		t.Fatal("samples in starkDB exceed set limit")
	}

	// test JSON dump of metadata
	jsonDump, err := starkdb.DumpMetadata()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(jsonDump)

	// test the db teardown
	if err := teardown(); err != nil {
		t.Fatal(err)
	}
}

// TestReopenDB will check database re-opening, ranging and deleting.
func TestReopenDB(t *testing.T) {

	// test you can reopen the starkDB
	starkdb, teardown, err := OpenDB(SetProject(testProject), SetLocalStorageDir(tmpDir), WithNoPinning())
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	// range over the starkDB and check we have an entry
	found := false
	for entry := range starkdb.RangeCIDs() {
		if entry.Error != nil {
			t.Fatal(err)
		}
		if entry.Key != testKey {
			t.Fatalf("encountered unexpected key in starkDB: %v", entry.Key)
		} else {
			found = true
		}
	}
	if !found {
		t.Fatal("RangeCIDs failed to return a starkDB entry")
	}

	// get record back from starkDB
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
	t.Logf("explorer link: %v", link)

	// try deleting
	if err := starkdb.Delete(testKey); err != nil {
		t.Fatal(err)
	}
	if starkdb.currentNumEntries != 0 {
		t.Fatal("db is not empty after delete operation on sole entry")
	}
}

// TestFileIO will check a file can be added and retrieved from the IPFS.
func TestFileIO(t *testing.T) {
	starkdb, teardown, err := OpenDB(SetProject(testProject), SetLocalStorageDir(tmpDir), WithNoPinning())
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
	if err := starkdb.getFile(cid, tmpFile); err != nil {
		t.Fatal(err)
	}

	// keep it as a byte slice for ease
	retrievedFile, err := ioutil.ReadFile(tmpFile)
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
	starkdb, teardown, err := OpenDB(SetProject(testProject), SetLocalStorageDir(tmpDir), WithNoPinning(), WithAnnouncing())
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()
	if !starkdb.announcing {
		t.Fatal("db has no announcing flag set")
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

	// add record to starkDB and announcing it
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

// TestSnapshot will test the snapshot method and clone function.
func TestSnapshot(t *testing.T) {

	// open the test database
	starkdb, teardown, err := OpenDB(SetProject(testProject), SetLocalStorageDir(tmpDir), WithNoPinning(), WithAnnouncing())
	if err != nil {
		t.Fatal(err)
	}

	// snapshot it
	snapshotCID, err := starkdb.Snapshot()
	if err != nil {
		teardown()
		t.Fatal(err)
	}

	// close the database and remove it from local filesystem
	if err := teardown(); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(tmpDir); err != nil {
		t.Fatal(err)
	}

	// open a fresh database with a snapshot
	starkdb, teardown, err = OpenDB(SetProject(testAltProject), SetLocalStorageDir(tmpDir), WithSnapshot(snapshotCID))
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	// check for the record from earlier has been recovered
	retrievedSample, err := starkdb.Get(testKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(retrievedSample)

}

// TestCleanup will cleanup the test tmp files.
func TestCleanup(t *testing.T) {
	if err := os.RemoveAll(tmpDir); err != nil {
		t.Fatal(err)
	}
}
