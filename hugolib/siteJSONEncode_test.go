package hugolib

import (
	"encoding/json"
	"testing"
)

// Issue #1123
// Testing prevention of cyclic refs in JSON encoding
// May be smart to run with: -timeout 4000ms
func TestEncodePage(t *testing.T) {

	// borrowed from menu_test.go
	s := createTestSite(MENU_PAGE_SOURCES)
	testSiteSetup(s, t)

	_, err := json.Marshal(s)
	check(t, err)

	_, err = json.Marshal(s.Pages[0])
	check(t, err)
}

func check(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Failed %s", err)
	}
}
