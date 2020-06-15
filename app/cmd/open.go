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
	"net"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/will-rowe/stark"
	starkdb "github.com/will-rowe/stark"
	"github.com/will-rowe/stark/app/config"
	"google.golang.org/grpc"
)

var (
	listen *bool
)

// openCmd represents the open command
var openCmd = &cobra.Command{
	Use:   "open",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runOpen(args[0])
	},
}

func init() {
	listen = openCmd.Flags().BoolP("listen", "l", false, "If true, database will listen for records being added on the network and make a copy in the current database")
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

	// setup the db opts
	dbOpts := []starkdb.DbOption{
		starkdb.SetProject(projectName),
	}
	if len(projectSnapshot) != 0 {
		dbOpts = append(dbOpts, starkdb.SetSnapshotCID(projectSnapshot))
	}
	if announce {
		log.Info("\tusing announce")
		dbOpts = append(dbOpts, starkdb.WithAnnouncing())
	}
	if encrypt {
		log.Info("\tusing encryption")
		dbOpts = append(dbOpts, starkdb.WithEncryption())
	}

	// open the db
	db, dbCloser, err := starkdb.OpenDB(dbOpts...)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("\tcurrent number of entries in database: %d", db.GetNumEntries())

	// set up the gRPC server
	errChan := make(chan error)
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

	//
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			errChan <- err
		}
	}()
	log.Info("ready...")

	// setup the interupt
	interupt := make(chan os.Signal)
	signal.Notify(interupt, os.Interrupt, syscall.SIGTERM)

	// block until interupt or server error
	select {
	case err := <-errChan:
		log.Warnf("server error: %v\n", err)
	case <-interupt:
	}
	log.Info("interupt received")
}
