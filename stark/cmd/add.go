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
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	starkdb "github.com/will-rowe/stark"
	"github.com/will-rowe/stark/stark/config"
	"google.golang.org/grpc"
)

var (
	printRecord       *bool
	recordDescription *string
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <key>",
	Short: "Add a record to an open database",
	Long: `Add a record to an open database.
	
	This command only offers basic Record fields at the moment.
	It may also be subject to change depending on best to add
	Records as the Record structure evolves.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runAdd(args[0])
	},
}

func init() {
	recordDescription = addCmd.Flags().String("description", "", "Description to give new record (enclose in quotes)")
	printRecord = addCmd.Flags().BoolP("printRecord", "P", false, "Print the record once it has been added to the database")
	rootCmd.AddCommand(addCmd)
}

// runAdd is the main block for the add subcommand
func runAdd(key string) {

	// get context
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// connect to the server
	conn, err := grpc.DialContext(ctx, viper.GetString("Address"), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("could not connect to a database: %v", err)
	}
	defer conn.Close()
	client := starkdb.NewStarkDbClient(conn)

	// create the Record
	record, err := starkdb.NewRecord(starkdb.SetAlias(key), starkdb.SetDescription(*recordDescription))
	if err != nil {
		log.Fatal(err)
	}

	// make a Set request
	rec, err := client.Set(ctx, record)
	config.CheckResponseErr(err)
	if *printRecord {
		fmt.Println(rec)
	}
}
