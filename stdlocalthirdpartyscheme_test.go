package impi

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type StdLocalAndThirdPartySchemeTestSuite struct {
	VerifierTestSuite
}

func (s *StdLocalAndThirdPartySchemeTestSuite) SetupSuite() {
	s.options.Scheme = ImportGroupVerificationSchemeStdLocalThirdParty
	s.options.LocalPrefix = "github.com/pavius/impi"
}

func (s *StdLocalAndThirdPartySchemeTestSuite) TestValidAllGroups() {

	verificationTestCases := []verificationTestCase{
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
				`Import groups are not in the proper order: ["Std" "Third party" "Local"]`,
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
		{
			name: `import "C"`,
			contents: `package impi

import (
	"fmt"
	"os"

	"github.com/pavius/impi"

	"github.com/pkg/errors"
)

/*
#include <stdlib.h>
*/
import "C"
`,
		},
	}

	s.verifyTestCases(verificationTestCases)
}

func TestStdLocalAndThirdPartySchemeTestSuite(t *testing.T) {
	suite.Run(t, new(StdLocalAndThirdPartySchemeTestSuite))
}
