package impi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type StdLocalAndThirdPartyTestSuite struct {
	VerifierTestSuite
}

func (s *StdLocalAndThirdPartyTestSuite) SetupSuite() {
	s.options.Scheme = ImportGroupVerificationSchemeStdLocalThirdParty
	s.options.LocalPrefix = "github.com/pavius/impi"
}

func (s *StdLocalAndThirdPartyTestSuite) TestValidAllGroups() {

	verificationTests := []struct {
		name                    string
		contents                string
		expectedErrorStrings    []string
		nonExpectedErrorStrings []string
	}{
		{
			name: "Std (valid)",
			contents: `package fixtures
import (
    "fmt"
    "os"
    "path"
)
`,
		},
		{
			name: "Local (valid)",
			contents: `package fixtures
import (
    "github.com/pavius/impi/a"
    // some comment
    "github.com/pavius/impi/b"
    "github.com/pavius/impi/c"
)
`,
		},
		{
			name: "Third party (valid)",
			contents: `package fixtures
import (
    "github.com/another/3rdparty"
    "github.com/some/thirdparty"
)
`,
		},
		{
			name: "Std -> Local (valid)",
			contents: `package fixtures
import (
    "fmt"
    "os"
    "path"

    "github.com/pavius/impi/a"
    // some comment
    "github.com/pavius/impi/b"
    "github.com/pavius/impi/c"
)
`,
		},
		{
			name: "Std -> Third party (valid)",
			contents: `package fixtures
import (
    "fmt"
    "os"
    "path"

    // another comment
    "github.com/another/3rdparty"
    "github.com/some/thirdparty"
)
`,
		},
		{
			name: "Local -> Third party (valid)",
			contents: `package fixtures
import (

    "github.com/pavius/impi/a"
    // some comment
    "github.com/pavius/impi/b"
    "github.com/pavius/impi/c"

    // another comment
    "github.com/another/3rdparty"
    "github.com/some/thirdparty"
)
`,
		},
		{
			name: "Std -> Local -> Third party (valid)",
			contents: `package fixtures
import (
    "fmt"
    "os"
    "path"

    "github.com/pavius/impi/a"
    // some comment
    "github.com/pavius/impi/b"
    "github.com/pavius/impi/c"

    // another comment
    "github.com/another/3rdparty"
    "github.com/some/thirdparty"
)
`,
		},
		{
			name: "Std -> Local -> Third party (valid)",
			contents: `package fixtures
import (
    "fmt"
    "os"
    "path"

    "github.com/pavius/impi/a"
    // some comment
    "github.com/pavius/impi/b"
    "github.com/pavius/impi/c"

    // another comment
    "github.com/another/3rdparty"
    "github.com/some/thirdparty"
)
`,
		},
		{
			name: "Std -> Third party -> Local (invalid)",
			contents: `package fixtures
import (
    "fmt"
    "os"
    "path"

    // another comment
    "github.com/another/3rdparty"
    "github.com/some/thirdparty"

    "github.com/pavius/impi/a"
    "github.com/pavius/impi/b"
    // some comment
    "github.com/pavius/impi/c"
)
`,
			expectedErrorStrings: []string{
				"Import groups are not in the proper order",
			},
		},
		{
			name: "Too many groups",
			contents: `package fixtures
import (
    "fmt"
    "os"

    "github.com/pavius/impi/a"
    "github.com/pavius/impi/c"

    "github.com/pavius/impi/a"
    "github.com/pavius/impi/c"

    // another comment
    "github.com/another/3rdparty"
    "github.com/some/thirdparty"
)
`, expectedErrorStrings: []string{"Expected no more than 3 groups, got 4"},
		},
		{
			name: "Improper sorting",
			contents: `package fixtures
import (
    "os"
    "fmt"
    "path"

    "github.com/pavius/impi/b"
    "github.com/pavius/impi/a"
    // some comment
    "github.com/pavius/impi/c"

    // another comment
    "github.com/another/3rdparty"
    "github.com/some/thirdparty"
)
`,
			expectedErrorStrings: []string{
				"Import group 0 is not sorted",
				"Import group 1 is not sorted",
			},
			nonExpectedErrorStrings: []string{
				"Import group 2 is not sorted",
			},
		},
		{
			name: "3rd party in local",
			contents: `package fixtures
import (
    "fmt"
    "os"
    "path"

    "github.com/another/3rdparty"
    "github.com/pavius/impi/a"
    "github.com/pavius/impi/b"
    // some comment
    "github.com/pavius/impi/c"

    // another comment
    "github.com/another/3rdparty"
    "github.com/some/thirdparty"
)
`,
			expectedErrorStrings: []string{
				"Imports of different types are not allowed in the same group",
			},
		},
		{
			name: "Unsorted and mixed",
			contents: `package fixtures
import (
    "fmt"
    "os"
    "path"

    "github.com/another/3rdparty"
    "github.com/pavius/impi/a"
    "github.com/pavius/impi/b"
    // some comment
    "github.com/pavius/impi/c"

    // another comment
    "github.com/another/3rdparty"
    "github.com/some/thirdparty"
	"context"
)
`,
			expectedErrorStrings: []string{
				"Imports of different types are not allowed in the same group",
			},
		},
	}

	for _, verificationTest := range verificationTests {
		err := s.verify(verificationTest.contents)

		if verificationTest.expectedErrorStrings == nil {
			s.Require().NoError(err)
			continue
		}

		for _, expectedErrorStrings := range verificationTest.expectedErrorStrings {
			s.Require().Error(err)
			s.Require().Contains(err.Error(), expectedErrorStrings)
		}

		for _, nonExpectedErrorStrings := range verificationTest.nonExpectedErrorStrings {
			if err != nil {
				s.Require().NotContains(err.Error(), nonExpectedErrorStrings)
			}
		}
	}
}

//
// Base for other modes
//

type VerifierTestSuite struct {
	suite.Suite
	verifier *verifier
	options  VerifyOptions
}

func (s *VerifierTestSuite) SetupTest() {
	var err error

	s.verifier, err = newVerifier()
	s.Require().NoError(err)
}

func (s *VerifierTestSuite) verify(contents string) error {
	return s.verifier.verify(strings.NewReader(contents), &s.options)
}

func TestVerifierSuite(t *testing.T) {
	suite.Run(t, new(StdLocalAndThirdPartyTestSuite))
}
