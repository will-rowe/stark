# Using STARK as a package

View the [Go Documentation](https://pkg.go.dev/github.com/will-rowe/stark) site for the complete **stark** package documentation.

This page will document some examples.

## Usage example

```go
// This basic program will create a new database, add a record to it and then retrieve a copy of that record.
package main

import (
	"fmt"

	"github.com/will-rowe/stark"
)

func main() {

	// init a starkDB
	db, dbCloser, err := stark.OpenDB(stark.SetProject("my project"))
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


}
```
