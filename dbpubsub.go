package stark

import (
	"context"
	"io"

	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
)

// publishAnnouncement will send a PubSub message on the topic
// of the database project.
func (Db *Db) publishAnnouncement(message []byte) error {
	if !Db.IsOnline() {
		return ErrNodeOffline
	}
	if len(Db.project) == 0 {
		return ErrNoProject
	}
	return Db.ipfsClient.ipfs.PubSub().Publish(Db.ctx, Db.project, message)
}

// subscribe will start the database listening to PubSub for messages
// on its registered project.
//
// Note: see https://blog.ipfs.io/25-pubsub/ for good intro on PubSub
func (Db *Db) subscribe() error {
	if !Db.IsOnline() {
		return ErrNodeOffline
	}

	// use the DHT to find other peers
	discover := true

	// setup the subscription
	sub, err := Db.ipfsClient.ipfs.PubSub().Subscribe(Db.ctx, Db.project, options.PubSub.Discover(discover))
	if err != nil {
		return err
	}
	Db.pubsubSub = sub

	// create the channels
	Db.pubsubMessages = make(chan icore.PubSubMessage)
	Db.pubsubErrors = make(chan error)
	Db.pubsubStop = make(chan struct{})
	Db.pubsubStopped = make(chan struct{})

	// start listening for pubsub messages
	go func() {

		// close the stoppedchan when this func exits
		defer close(Db.pubsubStopped)

		// collect messages and wait for signals
		for {
			select {

			default:

				// collect the message and check it out
				// before sending it on the collection chan
				message, err := Db.pubsubSub.Next(Db.ctx)
				if err == io.EOF || err == context.Canceled {
					continue
				} else if err != nil {
					Db.pubsubErrors <- err
				} else {
					Db.pubsubMessages <- message
				}
			case <-Db.pubsubStop:
				return
			}
		}
	}()
	return nil
}

// unsubscribe will stop the database from listening on
// PubSub.
func (Db *Db) unsubscribe() error {

	// signal the listener to stop
	close(Db.pubsubStop)

	// wait until it's stopped
	<-Db.pubsubStopped

	// close down the remaining chans
	close(Db.pubsubMessages)
	close(Db.pubsubErrors)

	// unset the subscription
	if err := Db.pubsubSub.Close(); err != nil {
		return err
	}
	Db.pubsubSub = nil
	return nil
}
