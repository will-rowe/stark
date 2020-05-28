// Package stark description.
package stark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
)

// TestPubSub
func TestPubSub(t *testing.T) {
	defer CleanUp()

	// init the starkDB with an ephemeral IPFS node
	starkdb, teardown, err := OpenDB(testProject, SetLocalStorageDir(tmpDB))
	if err != nil {
		t.Fatal(err)
	}

	// teardown
	defer func() {

		// clean up the starkdb
		if err := teardown(); err != nil {
			t.Fatal(err)
		}
	}()

	// get the nodeID
	nodeID, err := starkdb.GetNodeIdentity()
	if err != nil {
		t.Fatal(err)
	}

	// subsribe the node to pubsub
	if err := starkdb.Subscribe(); err != nil {
		t.Fatal(err)
	}

	// create a record
	testRecord, err := NewRecord(SetAlias(testAlias))
	if err != nil {
		t.Fatal(err)
	}

	// collect all test errors via chan
	testErrs := make(chan error)

	// process any messages in a Go routine so we can send some messages in the same test
	go func() {
		for msg := range starkdb.pubsubMessages {

			// marshal the msg to the expected struct
			receivedSample := &Record{}
			um := &jsonpb.Unmarshaler{}
			if err := um.Unmarshal(bytes.NewReader(msg.Data()), receivedSample); err != nil {
				testErrs <- err
			}
			t.Log(receivedSample.String())

			// check the received message matches the sent one
			if receivedSample.GetUuid() != testRecord.GetUuid() {
				testErrs <- fmt.Errorf("received message does not match the sent one")
			}

			// check the message source matches the sender address used
			if nodeID != msg.From().Pretty() {
				testErrs <- fmt.Errorf("source address does not match sender address (%v vs %v)", nodeID, msg.From().Pretty())
			}
			t.Log(msg.From().Pretty())
		}
		close(testErrs)
	}()

	// send a marshaled struct over the PubSub network
	testMsg, err := json.Marshal(testRecord)
	if err != nil {
		t.Fatal(err)
	}
	if err := starkdb.Publish(testMsg); err != nil {
		t.Fatal(err)
	}

	// unsubscribe the node
	if err := starkdb.Unsubscribe(); err != nil {
		t.Fatal(err)
	}

	// check errors
	for err := range testErrs {
		t.Fatal(err)
	}
}
