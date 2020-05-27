package impi

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/kisielk/gotool"
	"golang.org/x/sync/errgroup"
)

// Impi is a single instance that can perform verification on a path
type Impi struct {
	numWorkers      int
	verifyOptions   *VerifyOptions
	SkipPathRegexes []*regexp.Regexp
}

// ImportGroupVerificationScheme specifies what to check when inspecting import groups
type ImportGroupVerificationScheme int

const (
	// ImportGroupVerificationSchemeSingle allows for a single, sorted group
	ImportGroupVerificationSchemeSingle = ImportGroupVerificationScheme(iota)

	// ImportGroupVerificationSchemeStdNonStd allows for up to two groups in the following order:
	// - standard imports
	// - non-standard imports
	ImportGroupVerificationSchemeStdNonStd

	// ImportGroupVerificationSchemeStdLocalThirdParty allows for up to three groups in the following order:
	// - standard imports
	// - local imports (where local prefix is specified in verification options)
	// - non-standard imports
	ImportGroupVerificationSchemeStdLocalThirdParty

	// ImportGroupVerificationSchemeStdThirdPartyLocal allows for up to three groups in the following order:
	// - standard imports
	// - non-standard imports
	// - local imports (where local prefix is specified in verification options)
	ImportGroupVerificationSchemeStdThirdPartyLocal
)

// VerifyOptions specifies how to perform verification
type VerifyOptions struct {
	SkipTests       bool
	Scheme          ImportGroupVerificationScheme
	LocalPrefix     string
	SkipPaths       []string
	IgnoreGenerated bool
}

// VerificationError holds an error and a file path on which the error occurred
type VerificationError struct {
	error
	FilePath string
}

// ErrorReporter receives error reports as they are detected by the workers
type ErrorReporter interface {
	Report(VerificationError)
}

// NewImpi creates a new impi instance
func NewImpi(numWorkers int) (*Impi, error) {
	newImpi := &Impi{
		numWorkers: numWorkers,
	}

	return newImpi, nil
}

// Verify will iterate over the path and start verifying import correctness within
// all .go files in the path. Path follows go tool semantics (e.g. ./...)
func (i *Impi) Verify(rootPath string, verifyOptions *VerifyOptions, errorReporter ErrorReporter) error {
	// save stuff for current session
	i.verifyOptions = verifyOptions

	// compile skip regex
	for _, skipPath := range verifyOptions.SkipPaths {
		skipPathRegex, err := regexp.Compile(skipPath)
		if err != nil {
			return err
		}

		i.SkipPathRegexes = append(i.SkipPathRegexes, skipPathRegex)
	}

	numErrors := 0
	resultsCh := make(chan VerificationError)
	filePathsCh := make(chan string, i.numWorkers)

	g, ctx := errgroup.WithContext(context.TODO())
	g.Go(func() error {
		for res := range resultsCh {
			errorReporter.Report(res)
			numErrors++
		}
		return nil
	})
	g.Go(func() error {
		defer close(filePathsCh)
		// When the populate paths function finishes up (error or not), filePathsCh will be closed. This will
		// allow the workers goroutine to finish up, as all iterations over this channel will stop.
		return i.populatePathsChan(ctx, rootPath, filePathsCh)
	})
	g.Go(func() error {
		defer close(resultsCh)
		// If all workers fail, the results reading goroutine will safely stop as resultsCh becomes closed. The
		// file path populating goroutine will end up trying to write to filePathsCh whilst nothing is reading
		// from it; deadlock is prevented here because errgroup will cancel the context that is passed down.
		// resultsCh is always going to be read to completion (there is no error cases in the results reading
		// goroutine), so there is no possibility of deadlock when trying to write to this channel.
		return i.createWorkers(filePathsCh, resultsCh)
	})
	if err := g.Wait(); err != nil {
		return err
	}

	if numErrors != 0 {
		return fmt.Errorf("Found %d errors", numErrors)
	}

	return nil
}

func (i *Impi) populatePathsChan(ctx context.Context, rootPath string, filePathsCh chan<- string) error {
	// TODO: this should be done in parallel

	// get all the packages in the root path, following go 1.9 semantics
	packagePaths := gotool.ImportPaths([]string{rootPath})

	if len(packagePaths) == 0 {
		return fmt.Errorf("Could not find packages in %s", packagePaths)
	}

	// iterate over these paths:
	// - for files, just shove to paths
	// - for dirs, find all go sources
	for _, packagePath := range packagePaths {
		if isDir(packagePath) {

			// iterate over files in directory
			fileInfos, err := ioutil.ReadDir(packagePath)
			if err != nil {
				return err
			}

			for _, fileInfo := range fileInfos {
				if fileInfo.IsDir() {
					continue
				}

				if err := i.addFilePathToFilePathsChan(ctx, path.Join(packagePath, fileInfo.Name()), filePathsCh); err != nil {
					return err
				}
			}

		} else {
			// shove path to channel if passes filter
			if err := i.addFilePathToFilePathsChan(ctx, packagePath, filePathsCh); err != nil {
				return err
			}
		}
	}

	return nil
}

func (i *Impi) createWorkers(filePathsCh <-chan string, resultsCh chan<- VerificationError) error {
	var g errgroup.Group
	for idx := 0; idx < i.numWorkers; idx++ {
		g.Go(func() error {
			// create a verifier with which we'll verify modules
			verifier, err := newVerifier()
			if err != nil {
				return err
			}

			for filePath := range filePathsCh {
				f, err := os.Open(filePath)
				if err != nil {
					return err
				}

				// verify the path and report an error if one is found
				if err = verifier.verify(f, i.verifyOptions); err != nil {
					resultsCh <- VerificationError{error: err, FilePath: filePath}
				}
			}
			return nil
		})
	}
	return g.Wait()
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func (i *Impi) addFilePathToFilePathsChan(ctx context.Context, filePath string, filePathsCh chan<- string) error {
	// skip non-go files
	if !strings.HasSuffix(filePath, ".go") {
		return nil
	}

	// skip tests if not desired
	if strings.HasSuffix(filePath, "_test.go") && i.verifyOptions.SkipTests {
		return nil
	}

	// cmd/impi/main.go should check the patters
	for _, skipPathRegex := range i.SkipPathRegexes {
		if skipPathRegex.Match([]byte(filePath)) {
			return nil
		}
	}

	// write to paths chan
	select {
	case <-ctx.Done():
		return ctx.Err()
	case filePathsCh <- filePath:
		return nil
	}
}
