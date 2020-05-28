package stark

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// checkDir is a function to check that a directory exists
func checkDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("no directory specified")
	}
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %v", dir)
		}
		return fmt.Errorf("can't access adirectory (check permissions): %v", dir)
	}
	return nil
}

// checkFile is a function to check that a file can be read
func checkFile(file string) error {
	fi, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %v", file)
		}
		return fmt.Errorf("can't access file (check permissions): %v", file)
	}
	if fi.Size() == 0 {
		return fmt.Errorf("file appears to be empty: %v", file)
	}
	return nil
}

// checkTimeStamp will return true if the new protobuf timestamp is more recent than the old one.
func checkTimeStamp(old, new *timestamppb.Timestamp) bool {
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
