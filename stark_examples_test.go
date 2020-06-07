package stark

import (
	"fmt"
	"log"
)

// ExampleOpenDB documents the usage of OpenDB.
func ExampleOpenDB() {

	// init the starkDB with functional options
	starkdb, dbCloser, err := OpenDB(SetProject("my project"), SetLocalStorageDir("/tmp/starkdb"), WithAnnouncing())
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
	err = starkdb.Set("db key", record)
	if err != nil {
		panic(err)
	}
}

// ExampleRangeCIDs documents the usage of RangeCIDs.
func ExampleRangeCIDs() {

	// init the starkDB with functional options
	starkdb, dbCloser, err := OpenDB(SetProject("my project"), SetLocalStorageDir("/tmp/starkdb"), WithNoPinning())
	if err != nil {
		panic(err)
	}

	// defer the database closer
	defer dbCloser()

	// range over the database entries (as KeyCIDpairs)
	for entry := range starkdb.RangeCIDs() {
		if entry.Error != nil {
			panic(err)
		}
		fmt.Printf("key: %s, value: %s\n", entry.Key, entry.CID)
	}
}

// ExampleListen documents the usage of Listen.
func ExampleListen() {

	// init the starkDB with functional options
	starkdb, dbCloser, err := OpenDB(SetProject("my project"), SetLocalStorageDir("/tmp/starkdb"))
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
			log.Printf("received Record from another DB: %v", rec.GetAlias())

		case err := <-errs:

			// error handling
			log.Printf("received error whilst processing PubSub message: %w", err)
			break
		}
		close(terminator)
	}()

	// add additional processing here

}
