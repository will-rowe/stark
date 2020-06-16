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
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/will-rowe/stark"
	"github.com/will-rowe/stark/stark/config"
	"google.golang.org/grpc"
)

// dumpCmd represents the dump command
var dumpCmd = &cobra.Command{
	Use:   "dump <project name>",
	Short: "Dump an open database to STDOUT",
	Long: `Dump will produce a JSON formatted
	database dump for the currently open 
	database, then print it to STDOUT.
	
	The JSON will also contain all keys and linked
	IPFS CIDs contained within the database. Full
	records will not be returned.`,
	Run: func(cmd *cobra.Command, args []string) {
		runDump()
	},
}

func init() {
	rootCmd.AddCommand(dumpCmd)
}

func runDump() {

	// get context
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// connect to the server
	conn, err := grpc.DialContext(ctx, viper.GetString("Address"), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("could not connect to a database: %v", err)
	}
	defer conn.Close()
	c := stark.NewStarkDbClient(conn)

	// make a Dump request
	dump, err := c.Dump(ctx, &stark.Key{})
	config.CheckResponseErr(err)

	// print the json
	data, err := json.MarshalIndent(dump, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
