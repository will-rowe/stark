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
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/will-rowe/stark"
	starkdb "github.com/will-rowe/stark"
	"github.com/will-rowe/stark/stark/config"
	"google.golang.org/grpc"
)

var (
	announce       *bool
	encrypt        *bool
	listen         *bool
	pinataInterval *int
)

// openCmd represents the open command
var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Open a stark database",
	Long: `Open a stark database. The database must
	have already neen initialised with the init subcommand.
	
	This command will serve a database via a gRPC server.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runOpen(args[0])
	},
}

func init() {
	announce = openCmd.Flags().BoolP("withAnnounce", "a", false, "Announce all records over PubSub as they are added to the open database.")
	encrypt = openCmd.Flags().BoolP("withEncrypt", "e", false, fmt.Sprintf("Encrypt record fields using the password stored in the %v env variable.", starkdb.DefaultStarkEnvVariable))
	listen = openCmd.Flags().BoolP("withListen", "l", false, "Listen for records being announced over PubSub and make a copy in the open database.")
	pinataInterval = openCmd.Flags().IntP("withPinata", "p", 0, fmt.Sprintf("Sets Pinata interval for pinning db contents - requires %v and %v to be set. (<1 == Pinata disabled)", starkdb.DefaultPinataAPIkey, starkdb.DefaultPinataSecretKey))
	rootCmd.AddCommand(openCmd)
}

func runOpen(projectName string) {
	config.StartLog("open")

	// get the database info
	log.Info("fetching database...")
	projs := viper.GetStringMapString("Databases")
	projectSnapshot, ok := projs[projectName]
	if !ok {
		log.Fatalf("no project found for: %v", projectName)
		os.Exit(1)
	}
	log.Info("\tproject name: ", projectName)
	if len(projectSnapshot) != 0 {
		log.Infof("\tsnapshot: %v", projectSnapshot)
	}

	// create a message channel for internal logging
	msgChan := make(chan interface{})
	defer close(msgChan)
	go func() {
		for msg := range msgChan {
			log.Infof("\t%v", msg)
		}
	}()

	// setup the db opts
	dbOpts := []starkdb.DbOption{
		starkdb.SetProject(projectName),
		starkdb.WithLogging(msgChan),
	}
	if len(projectSnapshot) != 0 {
		dbOpts = append(dbOpts, starkdb.SetSnapshotCID(projectSnapshot))
	}
	if *announce {
		log.Info("\tusing announce")
		dbOpts = append(dbOpts, starkdb.WithAnnouncing())
	}
	if *encrypt {
		log.Info("\tusing encryption")
		dbOpts = append(dbOpts, starkdb.WithEncryption())
	}
	if *pinataInterval > 0 {
		log.Infof("\tusing Pinata every %d records", *pinataInterval)
		dbOpts = append(dbOpts, starkdb.WithPinata(*pinataInterval))
	}
	if *listen {
		log.Info("\tusing listen")
	}

	// open the db
	db, dbCloser, err := starkdb.OpenDB(dbOpts...)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("\tcurrent number of entries in database: %d", db.GetNumEntries())

	// set up some control channels
	errChan := make(chan error)
	terminator := make(chan struct{})

	// start the listener if requested
	if *listen {
		log.Info("starting the PubSub listener...")
		recs, errs, err := db.Listen(terminator)
		if err != nil {
			log.Fatal(err)
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			select {
			case rec := <-recs:
				log.Infof("\tPubSub: found record (%v)", rec.GetUuid())
				_, err := db.Set(ctx, rec)
				if err != nil {
					errChan <- err
				}
				log.Infof("\tPubSub: added new record (%v->%v)", rec.GetAlias(), rec.GetPreviousCID())
			case err := <-errs:
				errChan <- err
			}
		}()
	}

	// set up the gRPC server
	address := viper.GetString("Address")
	log.Info("starting gRPC server...")
	log.Infof("\taddress: %s", address)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	stark.RegisterStarkDbServer(grpcServer, db)

	// defer the database close down
	defer func() {
		log.Info("shutting down...")
		log.Infof("\tcurrent number of entries in database: %d", db.GetNumEntries())

		// close the listener
		close(terminator)

		// update the snapshot
		newSnapshot := db.GetSnapshot()
		conf, err := config.DumpConfig2Mem()
		if err != nil {
			log.Fatal(err)
		}
		conf.Databases[projectName] = newSnapshot
		if err := conf.WriteConfig(); err != nil {
			log.Fatal(err)
		}
		if len(newSnapshot) != 0 {
			log.Infof("\tsnapshot: %v", newSnapshot)
		}

		// close the database and IPFS
		if err := dbCloser(); err != nil {
			log.Fatal(err)
		}
		log.Info("\tclosed database")

		// close down the server
		grpcServer.GracefulStop()
		log.Info("\tstopped the gRPC server")
		log.Info("finished.")
	}()

	// start serving
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			errChan <- err
		}
	}()

	// setup the interupt
	interupt := make(chan os.Signal)
	signal.Notify(interupt, os.Interrupt, syscall.SIGTERM)

	// block until interupt or server error
	log.Info("ready...")
	select {
	case err := <-errChan:
		log.Errorf("\t%v", err)
	case <-interupt:
	}
	log.Info("interupt received")
}
