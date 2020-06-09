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
	"encoding/json"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	starkdb "github.com/will-rowe/stark"
	"github.com/will-rowe/stark/app/config"
)

var (
	outFile *string
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get <project name> <key>",
	Short: "Get a record from a database",
	Long:  `Get a record from a database.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		runGet(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	outFile = getCmd.Flags().StringP("outFile", "o", "record.json", "Outfile name for retrieved record")
}

// runGet is the main block for the add subcommand
func runGet(projectName, key string) {
	config.StartLog("get")

	// init the database
	log.Info("fetching database...")
	projs := viper.GetStringMapString("Databases")
	projectPath, ok := projs[projectName]
	if !ok {
		log.Fatalf("no project found for: %v", projectName)
		os.Exit(1)
	}
	log.Info("\tproject name: ", projectName)

	// setup the db opts
	dbOpts := []starkdb.DbOption{
		starkdb.SetProject(projectName),
		starkdb.SetLocalStorageDir(projectPath),
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

	// defer close of the db
	defer func() {
		if err := dbCloser(); err != nil {
			log.Fatal(err)
		}
	}()

	// get the Record
	log.Info("getting the record...")
	record, err := db.Get(key)
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(record, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	if err := ioutil.WriteFile(*outFile, data, 0644); err != nil {
		log.Fatal(err)
	}
	log.Infof("\trecord key: %s", key)
	log.Infof("\twritten to file: %s", *outFile)
	log.Info("done.")
}
