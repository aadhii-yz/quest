// Package licensing owns the policy for running third-party testers.
//
// Downloading is not permission: a tester that ships no license may only be
// run after explicit, specific user consent. Keeping this policy out of the UI
// makes it impossible to render a run button that quietly bypasses it.
package licensing

import "github.com/aadhii-yz/quest/internal/provider"

// RequiresConsent reports whether the tester needs explicit user consent
// before it may be downloaded or executed.
func RequiresConsent(plan provider.TesterPlan) bool {
	return plan.License == provider.LicenseNone
}

// Notice returns the consent text for a tester that requires it. It names the
// missing license, because consent must be specific.
func Notice(plan provider.TesterPlan) string {
	if !RequiresConsent(plan) {
		return ""
	}
	return "this tester ships no license; running it needs explicit consent"
}
