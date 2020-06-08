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
	projectPath *string
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init <project name>",
	Short: "Initialise a stark database",
	Long: `This subcommand will initialise a stark database.
	
	A database will be setup for the provided project name.
	The database project and local storage directory will be
	added to the stark app's config file.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runInit(args[0])
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	projectPath = initCmd.Flags().StringP("projectPath", "p", "/tmp", "Directory to store local project data")
}

func runInit(projectName string) {
	config.StartLog("init")
	log.Info("initialising database...")
	log.Infof("\tproject name: %v", projectName)
	log.Infof("\tlocal data: %v/%v", *projectPath, projectName)

	// check we don't have a database for this project yet
	projs := viper.GetStringMapString("Databases")
	if _, ok := projs[projectName]; ok {
		log.Warn("project with this name already exists")
		os.Exit(0)
	}

	// otherwise load the config in so we can update
	conf, err := config.DumpConfig2Mem()
	if err != nil {
		log.Fatal(err)
	}
	conf.Databases[projectName] = *projectPath

	// check the DB can be opened at the location
	log.Info("checking new database...")
	_, dbCloser, err := starkdb.OpenDB(starkdb.SetProject(projectName), starkdb.SetLocalStorageDir(*projectPath))
	if err != nil {
		log.Fatal(err)
	}
	if err := dbCloser(); err != nil {
		log.Fatal(err)
	}

	// write config back
	if err := conf.WriteConfig(); err != nil {
		log.Fatal(err)
	}
	log.Info("done.")
}
