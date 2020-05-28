package stark

import (
	"testing"
)

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
