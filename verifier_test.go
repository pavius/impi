package impi

import (
	"strings"

	"github.com/stretchr/testify/suite"
)

type VerifierTestSuite struct {
	suite.Suite
	verifier *verifier
	options  VerifyOptions
}

type verificationTestCase struct {
	name                    string
	contents                string
	expectedErrorStrings    []string
	nonExpectedErrorStrings []string
}

func (s *VerifierTestSuite) SetupTest() {
	var err error

	s.verifier, err = newVerifier()
	s.Require().NoError(err)
}

func (s *VerifierTestSuite) verify(contents string) error {
	return s.verifier.verify(strings.NewReader(contents), &s.options)
}

func (s *VerifierTestSuite) verifyTestCases(verificationTestCases []verificationTestCase) {
	for _, verificationTestCase := range verificationTestCases {
		err := s.verify(verificationTestCase.contents)

		if verificationTestCase.expectedErrorStrings == nil {
			s.Require().NoError(err, verificationTestCase.name)
			continue
		}

		for _, expectedErrorStrings := range verificationTestCase.expectedErrorStrings {
			s.Require().Error(err, verificationTestCase.name)
			s.Require().Contains(err.Error(), expectedErrorStrings, verificationTestCase.name)
		}

		for _, nonExpectedErrorStrings := range verificationTestCase.nonExpectedErrorStrings {
			if err != nil {
				s.Require().NotContains(err.Error(), nonExpectedErrorStrings, verificationTestCase.name)
			}
		}
	}
}
