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
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	starkdb "github.com/will-rowe/stark"
)

// dumpCmd represents the dump command
var dumpCmd = &cobra.Command{
	Use:   "dump <project name>",
	Short: "Dump a database to STDOUT",
	Long: `Dump will produce a JSON formatted
	metadata string for the specified database, then 
	print it to STDOUT.
	
	The JSON will also contain all keys and linked
	IPFS CIDs contained within the database. Full
	records will not be returned.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runDump(args[0])
	},
}

func init() {
	rootCmd.AddCommand(dumpCmd)
}

func runDump(projectName string) {

	// get the database local storage path
	projs := viper.GetStringMapString("Databases")
	projectPath, ok := projs[projectName]
	if !ok {
		log.Fatalf("no project found for: %v", projectName)
		os.Exit(1)
	}

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

	// dump the db metadata
	json, err := db.DumpMetadata()
	if err != nil {
		log.Fatal(err)
	}

	// print the json
	fmt.Println(json)
}
