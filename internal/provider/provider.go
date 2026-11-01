// Package provider defines the contracts that every problem source implements.
//
// Adapters describe what exists (courses, stages, tester availability and
// license posture). They never render, prompt or execute testers: downloading
// belongs to internal/cache, consent policy to internal/licensing, and process
// execution to internal/runner.
package provider

// LicenseStatus describes the licensing of a provider's tester or content.
type LicenseStatus string

const (
	// LicenseMIT means the work is published under the MIT License.
	LicenseMIT LicenseStatus = "MIT"
	// LicenseNone means the work ships no license file, so all rights are
	// reserved by the author. Absence of a license is not permission to use.
	LicenseNone LicenseStatus = "No License"
)

// Stage is one ordered step of a course. Number is 1-based and is the order
// shown to the user; ID is the upstream slug.
type Stage struct {
	ID          string
	Number      int
	Name        string
	Difficulty  string
	Description string // Markdown
	Extension   string // extension slug, empty for base stages
}

// TesterPlan describes whether a provider can run a tester for this course on
// the current platform, without downloading anything.
type TesterPlan struct {
	// Available reports whether a pinned build exists for Platform.
	Available bool
	// Platform is the requested os/arch key, e.g. "linux_amd64".
	Platform string
	// AssetName is the published release asset filename, for display only.
	AssetName string
	// Release is the pinned tester release tag.
	Release string
	// License is the tester's licensing posture.
	License LicenseStatus
}

// Course is a provider-neutral view of a guided project or problem set.
type Course struct {
	ID           string
	ProviderID   string
	ProviderName string

	Name    string
	Summary string

	Languages []string
	Stages    []Stage
	Tester    TesterPlan

	// Note explains why a course is not fully ready, if it is not.
	Note string
}

// Ready reports whether a course has a stage list loaded.
func (c Course) Ready() bool { return len(c.Stages) > 0 }

// Provider is a source of courses. Implementations must be cheap and
// data-only: no network, download or execution.
type Provider interface {
	// ID is a stable identifier used for selection state, e.g. "codecrafters".
	ID() string
	// Name is the display name.
	Name() string
	// Note is a short description of the provider and its licensing posture.
	Note() string
	// Planned reports whether the provider is listed but not usable yet.
	Planned() bool
	// Courses returns the available courses.
	Courses() []Course
}

// DefaultLanguages is used when a provider or definition does not specify a
// language list, so language selection is never empty.
var DefaultLanguages = []string{
	"go", "rust", "python", "javascript", "typescript", "c", "cpp",
	"java", "kotlin", "zig",
}

// Placeholder is a provider that is listed but not implemented yet.
type Placeholder struct {
	id, name, note string
}

// NewPlaceholder returns a planned, course-less provider.
func NewPlaceholder(id, name, note string) Placeholder {
	return Placeholder{id: id, name: name, note: note}
}

func (p Placeholder) ID() string   { return p.id }
func (p Placeholder) Name() string { return p.name }
func (p Placeholder) Note() string { return p.note }

func (p Placeholder) Planned() bool     { return true }
func (p Placeholder) Courses() []Course { return nil }
