package cmd

import (
	"fmt"
	"os"
)

var errorChannel = make(chan error)

const MaxNumOfWorkers = 3

func logError(fileError <-chan error) {
	for err := range fileError {
		fmt.Fprintln(os.Stderr, err)
	}
}
