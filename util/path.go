package util

import (
	"fmt"
	"os"
)

func IsDir(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return fmt.Errorf("Error os.Stat: %v", err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("Is not a dir")
	}
	return nil
}
