/*
Copyright © 2020 Will Rowe <w.p.m.rowe@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// collectCmd represents the collect command
var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collect records for a database via PubSub",
	Long:  `Collect database records for a project via PubSub.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("collect called")
	},
}

func init() {
	rootCmd.AddCommand(collectCmd)
}

func runCollect() {

	/*

		// subscribe to the requested project
		log.Info("subscribing the node...")
		if err := node.Subscribe(ctx, viper.GetString("project")); err != nil {
			log.Fatal(err)
		}
		defer func() {
			if err := node.Unsubscribe(); err != nil {
				log.Fatal(err)
			}
		}()
		log.Infof("\tlistening for: %v", viper.GetString("project"))

		// setup the pubsub listener
		msgChan := make(chan *ipfs.Message)
		errChan := make(chan error, 1)
		sigChan := make(chan struct{})
		go node.Listen(msgChan, errChan, sigChan)

		// catch the os interupt for graceful close down
		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {

			// wait for interupt
			<-c
			log.Info("interrupt received - shutting down")

			// quit the PubSub listener
			close(sigChan)
			os.Exit(0)
		}()

		// process incoming messages
		for {
			select {

			// collect any messages
			case msg := <-msgChan:

				// check the message over
				log.Infof("\tmessage received from: %v", msg.From.Pretty())
				log.Infof("\tcontent: %v", msg.Data)

				// handle it
				//var doc map[string]interface{}
				//json.Unmarshal([]byte(s), &doc)
				//context, hasContext := doc["@context"]

			// collect any errors from the PubSub
			case err := <-errChan:
				log.Warn(err)
			}

		}


	*/

}
