package impi

type stdLocalThirdParty struct{}

// NewstdLocalThirdParty returns a new stdLocalThirdParty
func newStdLocalThirdParty() *stdLocalThirdParty {
	return &stdLocalThirdParty{}
}

// GetMaxNumGroups returns max number of groups the scheme allows
func (sltp *stdLocalThirdParty) GetMaxNumGroups() int {
	return 3
}

// GetMixedGroupsAllowed returns whether a group can contain imports of different types
func (sltp *stdLocalThirdParty) GetMixedGroupsAllowed() bool {
	return false
}

// GetAllowedGroupOrders returns which group orders are allowed
func (sltp *stdLocalThirdParty) GetAllowedImportOrders() [][]importType {
	return [][]importType{
		{importTypeStd},
		{importTypeLocal},
		{importTypeThirdParty},
		{importTypeStd, importTypeLocal},
		{importTypeStd, importTypeThirdParty},
		{importTypeLocal, importTypeThirdParty},
		{importTypeStd, importTypeLocal, importTypeThirdParty},
	}
}
