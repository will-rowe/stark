// Package config is used to set up and manage the stark config file.
package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"

	"github.com/will-rowe/stark/src/helpers"
)

var (

	// DefaultConfigName is the default config file name.
	DefaultConfigName = ".stark"

	// DefaultConfigLoc is the default location for the config file.
	DefaultConfigLoc = getHome()

	// DefaultType of file for the config file.
	DefaultType = "json"

	// DefaultConfigPath for the config file.
	DefaultConfigPath = fmt.Sprintf("%s/%s.%s", DefaultConfigLoc, DefaultConfigName, DefaultType)

	// DefaultLicense for any file being created.
	DefaultLicense = "MIT"

	// ErrInvalidPath is used when the config file path is bad or doesn't exist.
	ErrInvalidPath = fmt.Errorf("invalid config filepath")
)

// StarkConfig is a struct to hold the config
// data.
type StarkConfig struct {
	ConfigPath string            `json:"configPath"`
	FileType   string            `json:"fileType"`
	License    string            `json:"license"`
	Databases  map[string]string `json:"databases"`
}

// NewConfig returns an initialised empty StarkConfig.
func NewConfig() *StarkConfig {
	return &StarkConfig{
		Databases: make(map[string]string),
	}
}

// WriteConfig will write a StarkConfig to disk.
func (x *StarkConfig) WriteConfig() error {
	if len(x.ConfigPath) == 0 {
		return ErrInvalidPath
	}
	fh, err := os.Create(x.ConfigPath)
	defer fh.Close()
	d, err := json.MarshalIndent(x, "", "\t")
	if err != nil {
		return err
	}
	_, err = fh.Write(d)
	return err
}

// GenerateDefault will generate the default
// config on disk. If no filePath provided,
// it will use the DefaultConfigPath.
//
// Note: regardless of filepath, this function
// will generate the DefaultFileType.
func GenerateDefault(filePath string) error {
	if len(filePath) == 0 {
		filePath = DefaultConfigPath
	}

	// set up the default config data
	defaultConfig := &StarkConfig{
		ConfigPath: filePath,
		FileType:   DefaultType,
		License:    DefaultLicense,
		Databases:  make(map[string]string),
	}
	return defaultConfig.WriteConfig()
}

// ResetConfig will remove any existing config and
// replace it with the default one.
//
// NOTE: the caller must reload
// the config into viper
func ResetConfig(configPath string) error {

	// remove the existing config if it exists
	if helpers.CheckFileExists(configPath) {
		if err := os.Remove(configPath); err != nil {
			return err
		}
	}

	// now generate the default and write it to disk
	return GenerateDefault("")
}

// DumpConfig2Mem will unmarshall the config from
// Viper to a struct in memory.
func DumpConfig2Mem() (*StarkConfig, error) {
	c := NewConfig()
	err := viper.Unmarshal(c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// DumpConfig2JSON will unmarshall the config
// from Viper to a JSON string.
func DumpConfig2JSON() (string, error) {
	c, err := DumpConfig2Mem()
	if err != nil {
		return "", err
	}
	d, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return "", err
	}
	return string(d), nil
}

// getHome is used to find the DefaultConfigLoc.
func getHome() string {
	homeDir, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	return homeDir
}
