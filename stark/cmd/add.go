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
	"fmt"
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
	inputFile   *string
	useStdin    *bool
	useProto    *bool
	printRecord *bool
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <key>",
	Short: "Add a record to an open database",
	Long: `Add a record to an open database using the
	record alias as the database key.
	
	This command will read from an input file if provided,
	or STDIN if the flag is set. Otherwise, it will use
	an interactive prompt to populate a Record before
	adding it to the open database.`,
	Run: func(cmd *cobra.Command, args []string) {
		runAdd()
	},
}

func init() {
	inputFile = addCmd.Flags().StringP("inputFile", "f", "", "File containing Record")
	useStdin = addCmd.Flags().Bool("useStdin", false, "Read record from STDIN")
	useProto = addCmd.Flags().Bool("useProto", false, "Input Record is in Protobuf format, not JSON")
	printRecord = addCmd.Flags().BoolP("printRecord", "P", false, "Print the Record once it has been added to the database")
	rootCmd.AddCommand(addCmd)
}

// runAdd is the main block for the add subcommand
func runAdd() {

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
	} else if *useStdin {
		err := helpers.CheckSTDIN()
		if err != nil {
			log.Fatal(err)
		}
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
		fmt.Printf("no Record provided, collecting data interactively\nenter record description:\n")
		descp, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		descp = strings.Replace(descp, "\n", "", -1)
		fmt.Println("enter record alias (key):")
		key, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		key = strings.Replace(key, "\n", "", -1)
		record, err = starkdb.NewRecord(starkdb.SetAlias(key), starkdb.SetDescription(descp))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("record created, proceeding to add")
	}

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

	// make a Set request
	rec, err := client.Set(ctx, record)
	config.CheckResponseErr(err)
	if *printRecord {
		fmt.Println(rec)
	}
}
