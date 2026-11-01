package licensing

import (
	"strings"
	"testing"

	"github.com/aadhii-yz/quest/internal/provider"
)

func TestRequiresConsent(t *testing.T) {
	cases := []struct {
		license provider.LicenseStatus
		want    bool
	}{
		{provider.LicenseNone, true},
		{provider.LicenseMIT, false},
		{"", false},
	}
	for _, tc := range cases {
		plan := provider.TesterPlan{License: tc.license}
		if got := RequiresConsent(plan); got != tc.want {
			t.Errorf("RequiresConsent(%q) = %v, want %v", tc.license, got, tc.want)
		}
	}
}

func TestNoticeNamesMissingLicense(t *testing.T) {
	notice := Notice(provider.TesterPlan{License: provider.LicenseNone})
	if !strings.Contains(notice, "no license") || !strings.Contains(notice, "consent") {
		t.Fatalf("notice = %q", notice)
	}
	if got := Notice(provider.TesterPlan{License: provider.LicenseMIT}); got != "" {
		t.Fatalf("MIT tester should need no notice, got %q", got)
	}
}
