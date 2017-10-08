package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/pavius/impi"
)

type consoleErrorReporter struct{}

func (cer *consoleErrorReporter) Report(err impi.VerificationError) {
	fmt.Printf("%s: %s\n", err.FilePath, err.Error())
}

var localPrefix = flag.String("local", "", "prefix of the local repository")

func run() error {
	numCPUs := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPUs)

	// parse flags
	flag.Parse()

	// TODO: can parallelize across root paths
	for argIndex := 0; argIndex < flag.NArg(); argIndex++ {
		rootPath := flag.Arg(argIndex)

		impiInstance, err := impi.NewImpi(numCPUs)
		if err != nil {
			return fmt.Errorf("Failed to create impi: %s", err.Error())
		}

		err = impiInstance.Verify(rootPath, &impi.VerifyOptions{
			SkipTests:   false,
			LocalPrefix: *localPrefix,
		}, &consoleErrorReporter{})

		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}
