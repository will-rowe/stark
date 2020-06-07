//Package helpers contains some basic helper functions for stark.
package helpers

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// CheckDir is a function to check that a directory
// exists and tries to make it if it doesn't.
func CheckDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("no directory specified")
	}
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, 0755)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("cannot write to %s, incorrect permissions", dir)
		}
		return fmt.Errorf("check dir failed for %v (%w)", dir, err)
	}
	return nil
}

// CheckTimeStamp will return true if the new protobuf
// timestamp is more recent than the old one.
func CheckTimeStamp(old, new *timestamppb.Timestamp) bool {
	if old.GetSeconds() > new.GetSeconds() {
		return false
	}
	if old.GetSeconds() == new.GetSeconds() {
		if old.GetNanos() >= new.GetNanos() {
			return false
		}
	}
	return true
}
