// Package starkexample documents examples for the stark package.
package starkexample

import (
	stark "github.com/will-rowe/stark/v1"
)

// ExampleOpenDB documents the usage of OpenDB.
func ExampleOpenDB() {

	// init the starkDB with functional options
	starkdb, dbCloser, err := stark.OpenDB(stark.SetProject("my project"), stark.SetLocalStorageDir("/tmp/starkdb"), stark.WithPinning())
	if err != nil {
		panic(err)
	}

	// defer the database closer
	defer dbCloser()

	// create a record
	record, err := stark.NewRecord(stark.SetAlias("my first sample"))
	if err != nil {
		panic(err)
	}

	// add record to starkDB
	err = starkdb.Set("db key", record)
	if err != nil {
		panic(err)
	}
}
