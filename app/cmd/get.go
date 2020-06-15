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
	"github.com/will-rowe/stark/app/config"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

var (
	recEncoding *string
	hReadable   *bool
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a record from an open database",
	Long:  `Get a record from an open database.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runGet(args[0])
	},
}

func init() {
	recEncoding = getCmd.Flags().StringP("encoding", "E", "json", "Encoding to use when printing the retrieved Record (json or proto)")
	hReadable = getCmd.Flags().BoolP("humanReadable", "H", false, "If true, output will be human readable")
	rootCmd.AddCommand(getCmd)
}

func runGet(key string) {

	// check the options
	if (*recEncoding != "json") && (*recEncoding != "proto") {
		log.Fatalf("unsupported encoding requested: %s", *recEncoding)
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
	c := stark.NewStarkDbClient(conn)

	// make a Get request
	record, err := c.Get(ctx, &stark.Key{Id: key})
	config.CheckResponseErr(err)

	// print the returned Record
	var data []byte
	if *recEncoding == "proto" {
		data, err = proto.Marshal(record)
	} else {
		data, err = json.MarshalIndent(record, "", "\t")
	}
	if err != nil {
		log.Fatal(err)
	}
	if *hReadable {
		fmt.Println(string(data))
	} else {
		fmt.Println(data)
	}
}
