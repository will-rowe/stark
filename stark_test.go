package stark

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	starkcrypto "github.com/will-rowe/stark/src/crypto"
	starkipfs "github.com/will-rowe/stark/src/ipfs"
)

var (

	// IPFS tests:
	tstTopic = "stark test topic"
	tstMsg   = []byte("stark test message")
	tstFile  = "./README.md"
	tmpFile  = "./README.copy.md"

	// database tests:
	testProject     = "test_project"
	testAltProject  = "snapshotted_project"
	testKey         = "test entry"
	testAlias       = "test record"
	testDescription = "this is a test record"

	// tmp locations for database and result writes
	tmpDir = "tmp"
)

// IPFS tests:

// IPFS package tests are combined in the stark tests file as the
// plugin injection can only be run once per session.
// Running multiple test files via `go test ./...` will result
// in IPFS lock file contention and a panic.

// TestIpfsClient will test out the client.
func TestIpfsNewClient(t *testing.T) {

	// get some context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// test a client with a new repo
	client, err := starkipfs.NewIPFSclient(ctx, starkipfs.DefaultBootstrappers)
	if err != nil {
		t.Fatal(err)
	}

	// close the client
	if err := client.EndSession(); err != nil {
		t.Fatal(err)
	}
}

// TestIpfsFileIO will test file add and get operations to the IPFS.
func TestIpfsFileIO(t *testing.T) {

	// get some context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// test a client with a new repo
	client, err := starkipfs.NewIPFSclient(ctx, starkipfs.DefaultBootstrappers)
	if err != nil {
		t.Fatal(err)
	}
	defer client.EndSession()

	// add a file to IPFS
	cid, err := client.AddFile(ctx, tstFile, true)
	if err != nil {
		t.Fatal(err)
	}

	// get the file back
	if err := client.GetFile(ctx, cid, tmpFile); err != nil {
		t.Fatal(err)
	}

	// unpin from the IPFS
	if err := client.Unpin(ctx, cid); err != nil {
		t.Fatal(err)
	}

	// keep it as a byte slice for ease
	retrievedFile, err := ioutil.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	// compare the retrieved file to the original
	origFile, err := ioutil.ReadFile(tstFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(retrievedFile) != string(origFile) {
		t.Fatalf("retrieved file does not match original: %v vs %v", string(retrievedFile), string(origFile))
	}

	// remove the tmpFile
	if err := os.Remove(tmpFile); err != nil {
		t.Fatal(err)
	}
}

// TestIpfsPubSub will test the Pubsub.
func TestIpfsPubSub(t *testing.T) {

	// get some context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// test a client with a new repo
	client, err := starkipfs.NewIPFSclient(ctx, starkipfs.DefaultBootstrappers)
	if err != nil {
		t.Fatal(err)
	}
	defer client.EndSession()

	// test PubSub Subscribe
	if err := client.Subscribe(ctx, tstTopic); err != nil {
		t.Fatal(err)
	}
	stopMsgs := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopMsgs:
				return
			default:
				time.Sleep(time.Second * 2)
				if err := client.SendMessage(ctx, tstTopic, tstMsg); err != nil {
					client.GetPSEchan() <- err
				}
			}
		}
	}()
	for {
		select {
		case msg := <-client.GetPSMchan():
			if string(msg.Data()) != string(tstMsg) {
				t.Fatalf("message data received over pubsub did not match sent: %s vs %s", string(msg.Data()), string(tstMsg))
			}
			close(stopMsgs)
			break
		case err := <-client.GetPSEchan():
			if err != nil {
				t.Fatal(err)
				break
			}
		}
		break
	}

	// test PubSub Unsubscribe
	if err := client.Unsubscribe(); err != nil {
		t.Fatal(err)
	}
}

// Database tests:

// TestNewDB will check database initialisation and set/get operation.
func TestNewDB(t *testing.T) {
	numBootstrappers := 3
	numEntries := 1

	// init the starkDB
	starkdb, teardown, err := OpenDB(SetProject(testProject), SetLocalStorageDir(tmpDir), SetBootstrappers(starkipfs.DefaultBootstrappers[:numBootstrappers]), SetKeyLimit(numEntries))
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

// TestMessages will check registering, announcing and listening.
func TestMessages(t *testing.T) {
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
	recs, errs, err := starkdb.Listen(terminator)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
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

// TestEncyption will test the Record encryption.
func TestEncyption(t *testing.T) {

	// set a dummy encyption key
	if err := os.Setenv("STARK_DB_PASSWORD", "dummy password"); err != nil {
		t.Fatal(err)
	}

	// open the db with encryption
	starkdb, teardown, err := OpenDB(SetProject(testProject), SetLocalStorageDir(tmpDir), WithEncryption())
	if err != nil {
		t.Fatal(err)
	}
	if starkdb.cipherKey == nil {
		t.Fatal("encyprted db has no private key set")
	}

	// check a Record Set with encryption
	testRecord, err := NewRecord(SetAlias(testAlias))
	if err != nil {
		t.Fatal(err)
	}
	if err := starkdb.Set(testKey, testRecord); err != nil {
		t.Fatal(err)
	}

	// close the DB and open unencrypted version
	if err := teardown(); err != nil {
		t.Fatal(err)
	}

	// check encrypted records can't be Getted by db with no encyption key
	starkdb, teardown, err = OpenDB(SetProject(testProject), SetLocalStorageDir(tmpDir))
	if err != nil {
		t.Fatal(err)
	}
	retrievedRec, err := starkdb.Get(testKey)
	if err != starkcrypto.ErrCipherKeyMissing {
		t.Fatal("retrieved and decypted an encrypted record via a db with no cipher key")
	}

	// close the DB and open encrypted version
	if err := teardown(); err != nil {
		t.Fatal(err)
	}
	starkdb, teardown, err = OpenDB(SetProject(testProject), SetLocalStorageDir(tmpDir), WithEncryption())
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()
	retrievedRec, err = starkdb.Get(testKey)
	if err != nil {
		t.Fatal(err)
	}

	// retrieved record will have been decrypted via Get, check against original encrypted record
	if retrievedRec.GetUuid() == testRecord.GetUuid() {
		t.Fatal("original record was not encrypted")
	}

	// now decrypt original record - should match retrieved
	if err := testRecord.Decrypt(starkdb.cipherKey); err != nil {
		t.Fatal(err)
	}
	if retrievedRec.GetUuid() != testRecord.GetUuid() {
		t.Fatal("original record was not encrypted")
	}
}

// TestCleanup will cleanup the test tmp files.
func TestCleanup(t *testing.T) {
	if err := os.RemoveAll(tmpDir); err != nil {
		t.Fatal(err)
	}
}
