package impi

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type StdThirdPartyLocalSchemeTestSuite struct {
	VerifierTestSuite
}

func (s *StdThirdPartyLocalSchemeTestSuite) SetupSuite() {
	s.options.Scheme = ImportGroupVerificationSchemeStdThirdPartyLocal
	s.options.LocalPrefix = "github.com/pavius/impi"
}

func (s *StdThirdPartyLocalSchemeTestSuite) TestValidAllGroups() {

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
			name: "Third party -> Local (valid)",
			contents: `package fixtures
import (
    // another comment
    "github.com/another/3rdparty"
    "github.com/some/thirdparty"

    "github.com/pavius/impi/a"
    // some comment
    "github.com/pavius/impi/b"
    "github.com/pavius/impi/c"
)
`,
		},
		{
			name: "Std -> Third party -> Local (valid)",
			contents: `package fixtures
import (
    "fmt"
    "os"
    "path"

    // another comment
    "github.com/another/3rdparty"
    "github.com/some/thirdparty"

    "github.com/pavius/impi/a"
    // some comment
    "github.com/pavius/impi/b"
    "github.com/pavius/impi/c"
)
`,
		},
		{
			name: "Std -> Local -> Third party (invalid)",
			contents: `package fixtures
import (
    "fmt"
    "os"
    "path"

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
				`Import groups are not in the proper order: ["Std" "Local" "Third party"]`,
			},
		},
		{
			name: "Too many groups",
			contents: `package fixtures
import (
    "fmt"
    "os"

    // another comment
    "github.com/another/3rdparty"
    "github.com/some/thirdparty"

    "github.com/pavius/impi/a"
    "github.com/pavius/impi/c"

    "github.com/pavius/impi/a"
    "github.com/pavius/impi/c"
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

    // another comment
    "github.com/another/3rdparty"
    "github.com/some/thirdparty"

    "github.com/pavius/impi/b"
    "github.com/pavius/impi/a"
    // some comment
    "github.com/pavius/impi/c"
)
`,
			expectedErrorStrings: []string{
				"Import group 0 is not sorted",
				"Import group 2 is not sorted",
			},
			nonExpectedErrorStrings: []string{
				"Import group 1 is not sorted",
			},
		},
		{
			name: "3rd party in local",
			contents: `package fixtures
import (
    "fmt"
    "os"
    "path"

    // another comment
    "github.com/another/3rdparty"
    "github.com/some/thirdparty"

    "github.com/another/3rdparty"
    "github.com/pavius/impi/a"
    "github.com/pavius/impi/b"
    // some comment
    "github.com/pavius/impi/c"

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

    // another comment
    "github.com/another/3rdparty"
    "github.com/some/thirdparty"
	"context"

    "github.com/another/3rdparty"
    "github.com/pavius/impi/a"
    "github.com/pavius/impi/b"
    // some comment
    "github.com/pavius/impi/c"
)
`,
			expectedErrorStrings: []string{
				"Imports of different types are not allowed in the same group",
			},
		},
	}

	s.verifyTestCases(verificationTestCases)
}

func TestStdThirdPartyLocalSchemeTestSuite(t *testing.T) {
	suite.Run(t, new(StdThirdPartyLocalSchemeTestSuite))
}
