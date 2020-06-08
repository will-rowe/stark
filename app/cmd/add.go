// Package cmd is the command line utility for managing a stark database.
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
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	starkdb "github.com/will-rowe/stark"
	"github.com/will-rowe/stark/app/config"
)

var (
	announce          *bool
	recordAlias       *string
	recordDescription *string
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <project name> <key>",
	Short: "Add a record to a database",
	Long: `Add a record to a database.
	
	This command only offers basic Record fields at the moment.
	It may also be subject to change depending on best to add
	Records as the Record structure evolves.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		runAdd(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	announce = addCmd.Flags().Bool("announce", false, "Announce the Record to the PubSub network when it is added to the Project database")
	recordAlias = addCmd.Flags().String("alias", "", "Alias to give the new record")
	recordDescription = addCmd.Flags().String("description", "", "Description to give new record")
}

// runAdd is the main block for the add subcommand
func runAdd(projectName, key string) {
	config.StartLog("add")

	// init the database
	log.Info("fetching database...")
	projs := viper.GetStringMapString("Databases")
	projectPath, ok := projs[projectName]
	if !ok {
		log.Fatalf("no project found for: %v", projectName)
		os.Exit(1)
	}
	log.Info("\tproject name: ", projectName)

	// open the db
	db, dbCloser, err := starkdb.OpenDB(starkdb.SetProject(projectName), starkdb.SetLocalStorageDir(projectPath))
	if err != nil {
		log.Fatal(err)
	}

	// defer close of the db
	defer func() {
		if err := dbCloser(); err != nil {
			log.Fatal(err)
		}
	}()

	// create the Record
	log.Info("creating the record...")
	record, err := starkdb.NewRecord(starkdb.SetAlias(*recordAlias), starkdb.SetDescription(*recordDescription))
	if err != nil {
		log.Fatal(err)
	}

	// add it to the db
	log.Info("adding the record...")
	if err := db.Set(key, record); err != nil {
		log.Fatal(err)
	}
	cid, err := db.GetCID(key)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("\tkey: %v", key)
	log.Infof("\tcid: %v", cid)
	log.Info("done.")
}
