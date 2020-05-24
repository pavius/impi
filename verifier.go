package impi

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// This regex is to appear in generated code.
var generatedRegex = regexp.MustCompile("// Code generated .* DO NOT EDIT\\.")

type verifier struct {
	verifyOptions *VerifyOptions
}

type importInfoGroup struct {
	importInfos []importInfo
}

type importType int

const (
	importTypeUnknown = importType(iota)
	importTypeStd
	importTypeLocal
	importTypeThirdParty
	importTypeLocalOrThirdParty
)

var importTypeName = []string{
	"Unknown",
	"Std",
	"Local",
	"Third party",
	"Local or third party",
}

type verificationScheme interface {
	// getMaxNumGroups returns max number of groups the scheme allows
	getMaxNumGroups() int

	// getMixedGroupsAllowed returns whether a group can contain imports of different types
	getMixedGroupsAllowed() bool

	// getAllowedImportOrders returns which group orders are allowed
	getAllowedImportOrders() [][]importType
}

type importDeclaration struct {
	lineNumStart int
	lineNumEnd   int
	importInfos  []importInfo
}

type importInfo struct {
	lineNumStart   int
	lineNumEnd     int
	lineNumImport  int
	path           string
	classifiedType importType
}

func newVerifier() (*verifier, error) {
	return &verifier{}, nil
}

func (v *verifier) verify(sourceFileReader io.ReadSeeker, verifyOptions *VerifyOptions) error {
	v.verifyOptions = verifyOptions

	if verifyOptions.IgnoreGenerated {
		// The line specifying that the code was generated can be found anywhere
		// within a file. In practice, it is the first line.
		fileContents, err := ioutil.ReadAll(sourceFileReader)
		if err != nil {
			return err
		}

		if generatedRegex.Match(fileContents) {
			return nil
		}

		if _, err := sourceFileReader.Seek(0, 0); err != nil {
			return err
		}
	}

	// get lines on which imports start and end
	importDecls, err := v.parseImports(sourceFileReader)
	if err != nil {
		return err
	}

	// special case: we permit a separate declaration for `import "C"` as this is typically
	// preceded by comment preamble
	importDecls = filterImportC(importDecls)

	// if there's nothing, do nothing
	if len(importDecls) == 0 {
		return nil
	}

	// we do not permit multiple declarations (other than the special case mentioned above)
	if len(importDecls) > 1 {
		return fmt.Errorf("Multiple import declarations not permitted, %d observed", len(importDecls))
	}

	// group the import lines we got based on newlines separating the groups
	importInfoGroups := v.groupImports(importDecls)

	// get scheme by type
	verificationScheme, err := v.getVerificationScheme()
	if err != nil {
		return err
	}

	// verify that we don't have too many groups
	if verificationScheme.getMaxNumGroups() < len(importInfoGroups) {
		return fmt.Errorf("Expected no more than 3 groups, got %d", len(importInfoGroups))
	}

	// if the scheme disallowed mixed groups, check that there are no mixed groups
	if !verificationScheme.getMixedGroupsAllowed() {
		if err := v.verifyNonMixedGroups(importInfoGroups); err != nil {
			return err
		}

		// verify group order
		if err := v.verifyGroupOrder(importInfoGroups, verificationScheme.getAllowedImportOrders()); err != nil {
			return err
		}
	}

	// verify that all groups are sorted amongst themselves
	if err := v.verifyImportInfoGroupsOrder(importInfoGroups); err != nil {
		return err
	}

	return nil
}

func (v *verifier) groupImports(importDecls []importDeclaration) []importInfoGroup {
	var groups []importInfoGroup

	for _, importDecl := range importDecls {
		var (
			lastLine int
			group    importInfoGroup
		)
		for _, info := range importDecl.importInfos {
			if lastLine > 0 && info.lineNumStart != lastLine+1 {
				// line number has jumped ahead by more than 1, so start a new group
				groups = append(groups, group)
				group = importInfoGroup{}
			}

			group.importInfos = append(group.importInfos, info)
			lastLine = info.lineNumEnd
		}
		if len(group.importInfos) > 0 {
			// ensure the final group is appended
			groups = append(groups, group)
		}
	}

	return groups
}

// filter out single `import "C"` from groups since it needs to be on it's own line
func filterImportC(importDecls []importDeclaration) []importDeclaration {
	var filteredDecls []importDeclaration

	for _, importDecl := range importDecls {
		var (
			cImport   bool
			stdImport bool
		)
		for _, decl := range importDecl.importInfos {
			if decl.path == "C" {
				cImport = true
			} else {
				stdImport = true
			}
		}
		if cImport && !stdImport {
			// this is `import "C"` only, so can be skipped over
			continue
		}
		filteredDecls = append(filteredDecls, importDecl)
	}

	return filteredDecls
}

func (v *verifier) parseImports(sourceFileReader io.ReadSeeker) ([]importDeclaration, error){
	sourceFileSet := token.NewFileSet()

	sourceNode, err := parser.ParseFile(sourceFileSet, "", sourceFileReader, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var importDecls []importDeclaration

	// Read each import declaration
	for _, decl := range sourceNode.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}

		importDecl := importDeclaration{
			lineNumStart: sourceFileSet.Position(genDecl.Pos()).Line,
			lineNumEnd:   sourceFileSet.Position(genDecl.End()).Line,
		}

		for _, spec := range genDecl.Specs {
			importSpec := spec.(*ast.ImportSpec)
			importLine := sourceFileSet.Position(importSpec.Pos()).Line
			importEndLine := sourceFileSet.Position(importSpec.End()).Line
			lineStart := importLine
			if importSpec.Doc != nil && len(importSpec.Doc.List) > 0 {
				// if there are comments we'll use the line of the first comment
				lineStart = sourceFileSet.Position(importSpec.Doc.List[0].Pos()).Line
			}
			importPath := strings.Trim(importSpec.Path.Value, `"`) // remove outer quotes
			importDecl.importInfos = append(importDecl.importInfos, importInfo{
				lineNumStart:   lineStart,
				lineNumEnd:     importEndLine,
				lineNumImport:  importLine,
				path:           importPath,
				classifiedType: v.classifyImportType(importPath),
			})
		}

		importDecls = append(importDecls, importDecl)
	}

	return importDecls, nil
}

func (v *verifier) verifyImportInfoGroupsOrder(importInfoGroups []importInfoGroup) error {
	var errorString string

	for importInfoGroupIndex, importInfoGroup := range importInfoGroups {
		var importPaths []string

		// create slice of strings so we can compare
		for _, importInfo := range importInfoGroup.importInfos {
			importPaths = append(importPaths, importInfo.path)
		}

		// check that group is sorted
		if !sort.StringsAreSorted(importPaths) {

			// created a sorted copy for logging
			sortedImportGroup := make([]string, len(importPaths))
			copy(sortedImportGroup, importPaths)
			sort.Sort(sort.StringSlice(sortedImportGroup))

			errorString += fmt.Sprintf("\n- Import group %d is not sorted\n-- Got:\n%s\n\n-- Expected:\n%s\n",
				importInfoGroupIndex,
				strings.Join(importPaths, "\n"),
				strings.Join(sortedImportGroup, "\n"))
		}
	}

	if len(errorString) != 0 {
		return errors.New(errorString)
	}

	return nil
}

func (v *verifier) classifyImportType(path string) importType {
	// if the value doesn't contain dot, it's a standard import
	if !strings.Contains(path, ".") {
		return importTypeStd
	}

	// if there's no prefix specified, it's either standard or local
	if len(v.verifyOptions.LocalPrefix) == 0 {
		return importTypeLocalOrThirdParty
	}

	if strings.HasPrefix(path, v.verifyOptions.LocalPrefix) {
		return importTypeLocal
	}

	return importTypeThirdParty
}

func (v *verifier) getVerificationScheme() (verificationScheme, error) {
	switch v.verifyOptions.Scheme {
	case ImportGroupVerificationSchemeStdLocalThirdParty:
		return newStdLocalThirdPartyScheme(), nil
	case ImportGroupVerificationSchemeStdThirdPartyLocal:
		return newStdThirdPartyLocalScheme(), nil
	default:
		return nil, errors.New("Unsupported verification scheme")
	}
}

func (v *verifier) verifyNonMixedGroups(importInfoGroups []importInfoGroup) error {
	for importInfoGroupIndex, importInfoGroup := range importInfoGroups {
		importGroupImportType := importInfoGroup.importInfos[0].classifiedType

		for _, importInfo := range importInfoGroup.importInfos {
			if importInfo.classifiedType != importGroupImportType {
				return fmt.Errorf("Imports of different types are not allowed in the same group (%d): %s != %s",
					importInfoGroupIndex,
					importInfoGroup.importInfos[0].path,
					importInfo.path)
			}
		}
	}

	return nil
}

func (v *verifier) verifyGroupOrder(importInfoGroups []importInfoGroup, allowedImportOrders [][]importType) error {
	var existingImportOrder []importType

	// use the first import type as indicative of the following. TODO: to support ImportGroupVerificationSchemeStdNonStd
	// this will need to do a full pass
	for _, importInfoGroup := range importInfoGroups {
		existingImportOrder = append(existingImportOrder, importInfoGroup.importInfos[0].classifiedType)
	}

	for _, allowedImportOrder := range allowedImportOrders {
		if reflect.DeepEqual(allowedImportOrder, existingImportOrder) {
			return nil
		}
	}

	// convert to string for a clearer error
	existingImportOrderString := []string{}
	for _, importType := range existingImportOrder {
		existingImportOrderString = append(existingImportOrderString, importTypeName[importType])
	}

	return fmt.Errorf("Import groups are not in the proper order: %q", existingImportOrderString)
}
