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
	"bufio"
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	starkdb "github.com/will-rowe/stark"
	"github.com/will-rowe/stark/src/helpers"
	"github.com/will-rowe/stark/stark/config"
	"google.golang.org/grpc"
)

var (
	inputFile *string
	useProto  *bool
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <key>",
	Short: "Add a record to an open database",
	Long: `Adds a Record to an open database using the
	provided lookup key.
	
	This command will read from an input file if provided,
	otherwise it will check STDIN. If no Record is found,
	it will use an interactive prompt to populate a new 
	Record before adding it to the open database.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runAdd(args[0])
	},
}

func init() {
	inputFile = addCmd.Flags().StringP("inputFile", "f", "", "File containing Record")
	useProto = addCmd.Flags().Bool("useProto", false, "Input Record is in Protobuf format, not JSON")
	rootCmd.AddCommand(addCmd)
}

// runAdd is the main block for the add subcommand
func runAdd(key string) {

	// check for input data
	var data []byte
	if len(*inputFile) != 0 {
		fh, err := os.Open(*inputFile)
		if err != nil {
			log.Fatal(err)
		}
		defer fh.Close()
		data, err = ioutil.ReadAll(fh)
		if err != nil {
			log.Fatal(err)
		}
	} else if helpers.StdinAvailable() {
		log.Info("collecting Record from STDIN...")
		var err error
		data, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
	}

	// create the record
	record := &starkdb.Record{}
	if data != nil {
		if *useProto {
			if err := proto.Unmarshal(data, record); err != nil {
				log.Fatal(err)
			}
		} else {
			if err := json.Unmarshal(data, record); err != nil {
				log.Fatal(err)
			}
		}
	} else {
		log.Info("no Record provided, collecting data interactively...")
		log.Info("enter Record description:")
		descp, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		descp = strings.Replace(descp, "\n", "", -1)
		log.Info("enter Record alias:")
		alias, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		key = strings.Replace(alias, "\n", "", -1)
		record, err = starkdb.NewRecord(starkdb.SetAlias(alias), starkdb.SetDescription(descp))
		if err != nil {
			log.Fatal(err)
		}
		log.Info("new Record created, proceeding to add")
	}

	// get context
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// connect to the server
	conn, err := grpc.DialContext(ctx, viper.GetString("Address"), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("could not connect to a stark database: %v", err)
	}
	defer conn.Close()
	client := starkdb.NewStarkDbClient(conn)

	// make a Set request
	response, err := client.Set(ctx, &starkdb.KeyRecordPair{Key: key, Record: record})
	config.CheckResponseErr(err)

	// print the key and cid
	log.Info("added Record to database:")
	log.Infof("\t%v -> %v\n", response.GetRecord().GetAlias(), response.GetRecord().GetPreviousCID())
}
