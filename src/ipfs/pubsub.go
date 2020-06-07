package ipfs

import (
	"context"
	"io"

	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
)

// GetPSMchan gives access to the PubSub messages channel.
func (client *Client) GetPSMchan() chan icore.PubSubMessage {
	return client.pubsubMessages
}

// GetPSEchan gives access to the PubSub errors channel.
func (client *Client) GetPSEchan() chan error {
	return client.pubsubErrors
}

// SendMessage will broadcast a message via PubSub.
func (client *Client) SendMessage(ctx context.Context, topic string, message []byte) error {
	return client.ipfs.PubSub().Publish(ctx, topic, message)
}

// Subscribe will start a Client's PubSub subscription for the provided topic.
//
// Note: see https://blog.ipfs.io/25-pubsub/ for good intro on PubSub
func (client *Client) Subscribe(ctx context.Context, topic string) error {

	// use the DHT to find other peers
	discover := true

	// setup the subscription
	sub, err := client.ipfs.PubSub().Subscribe(ctx, topic, options.PubSub.Discover(discover))
	if err != nil {
		return err
	}
	client.pubsubSub = sub

	// create the channels
	client.pubsubMessages = make(chan icore.PubSubMessage, DefaultBufferSize)
	client.pubsubErrors = make(chan error, DefaultBufferSize)
	client.pubsubStop = make(chan struct{})
	client.pubsubStopped = make(chan struct{})

	// start listening for pubsub messages
	go func() {

		// close the stoppedchan when this func exits
		defer close(client.pubsubStopped)

		// collect messages and wait for signals
		for {
			select {

			default:

				// collect the message and check it out
				// before sending it on the collection chan
				message, err := client.pubsubSub.Next(ctx)
				if err == io.EOF || err == context.Canceled {
					continue
				} else if err != nil {
					client.pubsubErrors <- err
				} else {
					client.pubsubMessages <- message
				}
			case <-client.pubsubStop:
				return
			}
		}
	}()
	return nil
}

// Unsubscribe will stop a Client's active PubSub subscription.
func (client *Client) Unsubscribe() error {

	// signal the listener to stop
	close(client.pubsubStop)

	// wait until it's stopped
	<-client.pubsubStopped

	// close down the remaining chans
	close(client.pubsubMessages)
	close(client.pubsubErrors)

	// unset the subscription
	if err := client.pubsubSub.Close(); err != nil {
		return err
	}
	client.pubsubSub = nil
	return nil
}
