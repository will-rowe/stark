// Package stark description.
package stark

import (
	"context"
	"fmt"
	"io"

	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
)

// Note: see https://blog.ipfs.io/25-pubsub/ for good intro on PubSub

// Subscribe will subscribe the database to the pubsub network for it's project.
func (DB *DB) Subscribe() error {

	// use the DHT to find other peers
	// TODO: find out more about this
	discover := true

	// setup the sub
	sub, err := DB.ipfsCoreAPI.PubSub().Subscribe(DB.ctx, DB.project, options.PubSub.Discover(discover))
	if err != nil {
		return err
	}
	DB.pubsubSub = sub

	// create the channels
	DB.pubsubMessages = make(chan icore.PubSubMessage)
	DB.pubsubStop = make(chan struct{})
	DB.pubsubStopped = make(chan struct{})

	// start listening for pubsub messages
	go func() {

		// close the stoppedchan when this func exits
		defer close(DB.pubsubStopped)

		// collect messages and wait for signals
		for {
			select {

			default:

				// collect the message and check it out before sending it on the collection chan
				message, err := DB.pubsubSub.Next(DB.ctx)
				if err == io.EOF || err == context.Canceled {
					continue
				} else if err != nil {
					panic(err)
				} else {
					DB.pubsubMessages <- message
				}

			case <-DB.pubsubStop:
				return
			}
		}
	}()

	return nil
}

// Unsubscribe will unsubscribe the node from a topic.
func (DB *DB) Unsubscribe() error {

	// signal the listener to stop
	close(DB.pubsubStop)

	// wait until it's stopped
	<-DB.pubsubStopped

	// close down the message chan
	close(DB.pubsubMessages)

	// unset the subscription and topic
	if err := DB.pubsubSub.Close(); err != nil {
		return err
	}
	DB.project = ""
	return nil
}

// Publish will publish a message about the registered topic
func (DB *DB) Publish(message []byte) error {
	if len(DB.project) == 0 {
		return fmt.Errorf("the node is has no registered topic")
	}
	return DB.ipfsCoreAPI.PubSub().Publish(DB.ctx, DB.project, message)
}
