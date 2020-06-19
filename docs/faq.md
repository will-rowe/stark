# FAQ

## can anyone access my data?

Yes. Anything put into a stark database will be available on the public IPFS network. However, you could configure your own private network if you wanted to.

stark does have an option to encrypt record fields. The record itself will still be accessible on the public network but it will require a passphrase to decrypt the fields.

Note: Encryption is a WIP. Currently only a Record's UUID will be encrypted as a proof of functionality. Encrypted Records are decrypted on retrieval, but this will fail if the database instance requesting them doesn't have the correct password.

## is my data persistent?

Yes. stark pins the records it adds by default, which means that the node you are using to run the database should delete it during garbage collection (see [here](https://docs.ipfs.io/concepts/persistence/)). If no other nodes request the records you add to a database on your node, the records will only exist on your node. This runs the risk that they could be deleted (e.g. if you wipe your nodes storage).

To be safe, and to speed up sharing, you can use the `withPinata` option to use the [pinata API](https://pinata.cloud/) and pin your data to their nodes (**requires a Pinata account**).

You can deactivate pinning (`withNoPinning`), which means records added to a stark database can be collected by the IPFS garbage collector, although this sort of defeats the point.

## can I use the IFPS tool to interact with records and projects?

Yes. Here are some examples:

* to list records in a project

```
ipfs ls <project cid>
```

* to get a record from a project:

```
ipfs dag get <project cid>/<record alias>
```

* to get a field from a record:

```
ipfs dag get <project cid>/<record alias>/<field>
```
