# Using stark as a package

View the [Go Documentation](https://pkg.go.dev/github.com/will-rowe/stark) site for the complete **stark** package documentation.

## Examples

TODO

For now, have a look at the app code to get some ideas.

## Notes

- each instance of a database is linked to a project, re-opening a database with the same project name will edit that database
- the `OpenDB` and `NewRecord` consructor functions use functional options to set struct values - this is in an effort to keep the API stable (see [here](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis))
- if a record is retrieved from the database and updated, you need to then re-add it to the database. In other words, a **stark database** only records the most recent version of a record commited to the IPFS
- records have a history, which can be used to rollback changes to other version of the record that entered the IPFS
- even though schema is in protobuf, most of the time it's marshaling to JSON to pass stuff around
- Record methods are not threadsafe - the database passes around copies of Records so this isn't much of an issue atm. The idea is that users of the library will end up turning Record data into something more usable and won't operate on them after initial Set/Gets