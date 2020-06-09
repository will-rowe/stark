package pinata

import (
	"encoding/json"
	"testing"
	"time"
)

// TestMetadata will test struct and methods
// for the Pinata metadata.
func TestMetadata(t *testing.T) {
	name := "testName"
	testMeta := NewMetadata(name)
	if testMeta.Name != name {
		t.Fatal("did not set metadata name")
	}

	// check the allowed types
	if err := testMeta.Add("int", 1); err != nil {
		t.Fatal(err)
	}
	if err := testMeta.Add("string", "some string"); err != nil {
		t.Fatal(err)
	}
	if err := testMeta.Add("time", time.Now()); err != nil {
		t.Fatal(err)
	}

	// make sure a bad type can't get in
	myMap := make(map[int]string)
	if err := testMeta.Add("bad", myMap); err == nil {
		t.Fatal("added an unsupported type to metadata")
	}

	// check the limit is enforced
	for i := 0; i < 7; i++ {
		if err := testMeta.Add(string(i), i); err != nil {
			t.Fatal(err)
		}
	}
	if err := testMeta.Add("too many", 666); err != ErrMetaLimit {
		t.Fatal("metadata exceeded keyvalue pair limit")
	}

	// check it marshals to JSON
	b, err := json.Marshal(testMeta)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))
}
