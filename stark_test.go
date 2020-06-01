package stark

// ExampleOpenDB documents the usage of OpenDB.
func ExampleOpenDB() {

	// init the starkDB with functional options
	starkdb, dbCloser, err := OpenDB(SetProject("my project"), SetLocalStorageDir("/tmp/starkdb"), WithPinning())
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
