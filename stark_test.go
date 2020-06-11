package stark

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	starkipfs "github.com/will-rowe/stark/src/ipfs"
)

var (

	// IPFS tests:
	tstTopic = "stark test topic"
	tstMsg   = []byte("stark test message")
	tstFile  = "./README.md"
	tmpFile  = "./README.copy.md"

	// database tests:
	testSnapshot    = ""
	testProject     = "test_project"
	testAltProject  = "snapshotted_project"
	testKey         = "test entry"
	testAlias       = "test record"
	testDescription = "this is a test record"
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

	// init the starkDB
	starkdb, teardown, err := OpenDB(SetProject(testProject), SetBootstrappers(starkipfs.DefaultBootstrappers[:numBootstrappers]))
	if err != nil {
		t.Fatal(err)
	}

	// check the setup options propagated
	if starkdb.project != strings.ReplaceAll(testProject, " ", "_") {
		t.Fatal("starkDB's project does not match the provided one")
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
	if starkdb.GetNumEntries() != 0 {
		t.Fatal("new starkDB is not empty")
	}

	// create a record
	testRecord, err := NewRecord(SetAlias(testAlias))
	if err != nil {
		t.Fatal(err)
	}

	// add record to starkDB
	if err := starkdb.Set(testKey, testRecord); err != nil {
		t.Fatal(err)
	}

	// get record back from starkDB
	retrievedSample, err := starkdb.Get(testKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(retrievedSample)
	if starkdb.GetNumEntries() != 1 {
		t.Fatal("starkDB has incorrect entry count: ", starkdb.GetNumEntries())
	}

	// try adding duplicate record
	if err := starkdb.Set(testKey, testRecord); err == nil {
		t.Fatal("duplicate sample was added")
	}

	// test JSON dump of metadata
	jsonDump, err := starkdb.DumpMetadata()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(jsonDump)

	// test snapshot
	testSnapshot = starkdb.GetSnapshot()
	if testSnapshot == "" {
		t.Fatal("no snapshot produced by starkdb")
	}
	t.Log("snapshot: ", testSnapshot)

	// test the db teardown
	if err := teardown(); err != nil {
		t.Fatal(err)
	}
	if starkdb.cidLookup != nil {
		t.Fatal("starkdb teardown did not clear struct")
	}
}

// TestReopenDB will check database re-opening, ranging and deleting.
func TestReopenDB(t *testing.T) {

	// test you can reopen the starkDB
	starkdb, teardown, err := OpenDB(SetProject(testProject), SetSnapshotCID(testSnapshot), WithNoPinning())
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()
	if starkdb.pinning {
		t.Fatal("IPFS node was told not to pin but is set to do so")
	}
	if starkdb.GetSnapshot() != testSnapshot {
		t.Fatal("starkDB did not build from existing snapshot")
	}

	// range over the starkDB and check we have an entry
	cids := starkdb.GetCIDs()
	if len(cids) != 1 {
		t.Fatal("starkDB did not collect record from existing snapshot")
	}
	for key := range cids {
		if key != testKey {
			t.Fatalf("encountered unexpected key in starkDB after GetCIDs(): %v", key)
		}
	}

	// get record back from starkDB
	retrievedSample, err := starkdb.Get(testKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(retrievedSample)

	// try deleting
	if err := starkdb.Delete(testKey); err != nil {
		t.Fatal(err)
	}
	if starkdb.currentNumEntries != 0 {
		t.Fatal("db is not empty after delete operation on sole entry")
	}

	// take another snapshot now db is empty
	testSnapshot = starkdb.GetSnapshot()
	if testSnapshot != "" {
		t.Fatal("snapshot produced by empty starkdb")
	}
}

// TestMessages will check registering, announcing and listening.
func TestMessages(t *testing.T) {
	starkdb, teardown, err := OpenDB(SetProject(testProject), WithNoPinning(), WithAnnouncing())
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()
	if !starkdb.announcing {
		t.Fatal("starkdb told to announce but flag is unset")
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

	// add record to starkDB and announce it
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

	if starkdb.currentNumEntries != 1 {
		t.Fatal("starkdb does not contain newly added entry")
	}

	testSnapshot = starkdb.GetSnapshot()
	if testSnapshot == "" {
		t.Fatal("no snapshot produced by starkdb")
	}
	t.Log("snapshot: ", testSnapshot)
}

// TestEncyption will test the Record encryption.
func TestEncyption(t *testing.T) {

	// set a dummy encyption key
	if err := os.Setenv("STARK_DB_PASSWORD", "dummy password"); err != nil {
		t.Fatal(err)
	}

	// open the db with encryption
	starkdb, teardown, err := OpenDB(SetProject(testProject), WithEncryption())
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
	testSnapshot = starkdb.GetSnapshot()
	if testSnapshot == "" {
		t.Fatal("no snapshot produced by starkdb")
	}
	t.Log("snapshot: ", testSnapshot)

	// close the DB and open unencrypted version
	if err := teardown(); err != nil {
		t.Fatal(err)
	}

	// check encrypted records can't be Getted by db with no encyption key
	starkdb, teardown, err = OpenDB(SetProject(testProject), SetSnapshotCID(testSnapshot))
	if err != nil {
		t.Fatal(err)
	}
	retrievedRec, err := starkdb.Get(testKey)
	if err == nil {
		t.Fatal("retrieved and decypted an encrypted record via a db with no cipher key")
	}

	// close the DB and open encrypted version
	if err := teardown(); err != nil {
		t.Fatal(err)
	}
	starkdb, teardown, err = OpenDB(SetProject(testProject), SetSnapshotCID(testSnapshot), WithEncryption())
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

/*

// Examples:

// ExampleOpenDB documents the usage of OpenDB.
func ExampleOpenDB() {

	// init the starkDB with functional options
	starkdb, dbCloser, err := OpenDB(SetProject("my project"), SetSnapshotCID("/tmp/starkdb"), WithAnnouncing())
	if err != nil {
		panic(err)
	}

	// defer the database closer
	defer dbCloser()

	// create a record
	record, err := NewRecord(SetAlias("my first record"))
	if err != nil {
		panic(err)
	}

	// add record to starkDB
	err = starkdb.Set("lookupKey", record)
	if err != nil {
		panic(err)
	}

	// retrieve record from the starkDB
	retrievedRecord, err := starkdb.Get("lookupKey")
	if err != nil {
		panic(err)
	}
	fmt.Println(retrievedRecord.GetAlias())

	// Output: my first record
}

// ExampleRangeCIDs documents the usage of RangeCIDs.
func ExampleRangeCIDs() {

	// init the starkDB with functional options
	starkdb, dbCloser, err := OpenDB(SetProject("rangeCID example"), SetSnapshotCID("/tmp/starkdb"), WithNoPinning())
	if err != nil {
		panic(err)
	}

	// defer the database closer
	defer dbCloser()

	// create a record
	record, err := NewRecord(SetAlias("my first record"))
	if err != nil {
		panic(err)
	}

	// add record to starkDB
	err = starkdb.Set("lookupKey", record)
	if err != nil {
		panic(err)
	}

	// range over the database entries (as KeyCIDpairs)
	for entry := range starkdb.RangeCIDs() {
		if entry.Error != nil {
			panic(err)
		}
		//fmt.Printf("key: %s, value: %s\n", entry.Key, entry.CID)
		fmt.Printf("key=%v", entry.Key)

		// Output: key=lookupKey
	}
}

// ExampleListen documents the usage of Listen.
func ExampleListen() {

	// init the starkDB with functional options
	starkdb, dbCloser, err := OpenDB(SetProject("listen example"), SetSnapshotCID("/tmp/starkdb"))
	if err != nil {
		panic(err)
	}

	// defer the database closer
	defer dbCloser()

	// create a terminator for the listener
	terminator := make(chan struct{})

	// start the listener
	recs, errs, err := starkdb.Listen(terminator)
	if err != nil {
		panic(err)
	}

	// process the listener channels in a Go routine
	go func() {
		select {
		case rec := <-recs:
			// record handling
			fmt.Printf("received Record from another DB: %v", rec.GetAlias())

		case err := <-errs:

			// error handling
			fmt.Printf("received error whilst processing PubSub message: %v", err)
			break
		}
		close(terminator)
	}()

	// add additional processing here

}

*/
