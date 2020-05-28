// This basic program will create a new database, add a record to it and then retrieve a copy of that record.
//
// You can compile and run this example with:
//   go run main.go
//
package main

import (
	"fmt"

	sdb "github.com/will-rowe/stark/starkdb"
)

func main() {

	// init a starkDB
	db, dbCloser, err := sdb.OpenDB("my project", sdb.SetLocalStorageDir("/tmp/starkdb"), sdb.SetEphemeral(false))
	if err != nil {
		panic(err)
	}

	// defer the database closer
	defer dbCloser()

	// create a record
	record, err := sdb.NewRecord(sdb.SetAlias("my first sample"))
	if err != nil {
		panic(err)
	}

	// add record to starkDB
	err = db.Set("lookupKey", record)
	if err != nil {
		panic(err)
	}

	// retrieve record from the starkDB
	retrievedSample, err := db.Get("lookupKey")
	if err != nil {
		panic(err)
	}
	fmt.Println(retrievedSample.GetAlias())

	// you can also view the record in the IPFS
	link, err := db.GetExplorerLink("lookupKey")
	if err != nil {
		panic(err)
	}
	fmt.Printf("view record on the IPFS: %v\n", link)
}