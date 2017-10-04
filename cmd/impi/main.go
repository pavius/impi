package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/pavius/impi"
)

type consoleErrorReporter struct {}

func (cer *consoleErrorReporter) Report(err impi.VerificationError) {
	fmt.Printf("%s: %s\n", err.FilePath, err.Error())
}

var localPrefix = flag.String("local", "", "prefix of the local repository")

func main() {
	numCPUs := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPUs)

	// parse flags
	flag.Parse()

	// TODO: can parallelize across root paths
	for argIndex := 0; argIndex < flag.NArg(); argIndex++ {
		rootPath := flag.Arg(argIndex)

		impiInstance, err := impi.NewImpi(numCPUs)
		if err != nil {
			fmt.Errorf("Failed to create impi: %s", err.Error())
			os.Exit(1)
		}

		err = impiInstance.Verify(rootPath, &impi.VerifyOptions{
			SkipTests:   false,
			LocalPrefix: *localPrefix,
		}, &consoleErrorReporter{})

		if err != nil {
			os.Exit(1)
		}
	}
}
