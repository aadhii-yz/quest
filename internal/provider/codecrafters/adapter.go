package codecrafters

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/aadhii-yz/quest/internal/provider"
	"gopkg.in/yaml.v3"
)

// Adapter exposes the pinned CodeCrafters sources as a provider.Provider.
// Definitions are read from disk; tester binaries are not downloaded and no
// network calls are made.
type Adapter struct {
	defs    map[string]CourseDefinition
	defsDir string
	osArch  string
}

// New builds an adapter that looks for course-definition.yml files under
// defsDir and reports tester availability for the current platform.
func New(defsDir string) *Adapter {
	return newAdapter(defsDir, runtime.GOOS+"_"+runtime.GOARCH)
}

func newAdapter(defsDir, osArch string) *Adapter {
	return &Adapter{
		defs:    loadDefinitions(defsDir),
		defsDir: defsDir,
		osArch:  osArch,
	}
}

func (a *Adapter) ID() string   { return "codecrafters" }
func (a *Adapter) Name() string { return "CodeCrafters" }

func (a *Adapter) Note() string {
	return "Build real tools from scratch. Tester binaries are downloaded per pinned release, verified by sha256 and run locally."
}

func (a *Adapter) Planned() bool { return false }

func (a *Adapter) Courses() []provider.Course {
	courses := make([]provider.Course, 0, len(Sources))
	for i := range Sources {
		courses = append(courses, a.course(&Sources[i]))
	}
	return courses
}

func (a *Adapter) course(src *Source) provider.Course {
	slug := slugFromRepo(src.RepoURL)
	c := provider.Course{
		ID:           slug,
		ProviderID:   a.ID(),
		ProviderName: a.Name(),
		Name:         strings.TrimPrefix(src.CourseName, "Build your own "),
		Languages:    provider.DefaultLanguages,
		Tester:       testerPlan(src, a.osArch),
	}

	if def, ok := a.defs[slug]; ok {
		c.Summary = plain(def.ShortDescriptionMD)
		c.Languages = nil
		for _, l := range def.Languages {
			c.Languages = append(c.Languages, l.Slug)
		}
		c.Stages = stages(def)
	}
	if len(c.Languages) == 0 {
		c.Languages = provider.DefaultLanguages
	}
	if !c.Ready() {
		c.Note = "no course-definition.yml found under " + a.defsDir
	}
	return c
}

// testerPlan reports what would be executed for this course on osArch. It
// deliberately carries no download URL or hash: cache/runner own those.
func testerPlan(src *Source, osArch string) provider.TesterPlan {
	plan := provider.TesterPlan{
		Platform: osArch,
		Release:  src.TesterRelease,
		License:  src.TesterLicense,
	}
	if asset, ok := src.AssetFor(osArch); ok {
		plan.Available = true
		plan.AssetName = filepath.Base(asset.URL)
	}
	return plan
}

func stages(def CourseDefinition) []provider.Stage {
	out := make([]provider.Stage, 0, len(def.Stages))
	for i, stage := range def.Stages {
		out = append(out, provider.Stage{
			ID:          stage.Slug,
			Number:      i + 1,
			Name:        stage.Name,
			Difficulty:  stage.Difficulty,
			Description: stage.MarketingMD,
			Extension:   stage.PrimaryExtensionSlug,
		})
	}
	return out
}

func slugFromRepo(repoURL string) string {
	slug := filepath.Base(strings.TrimSuffix(repoURL, "/"))
	return strings.TrimPrefix(slug, "build-your-own-")
}

// loadDefinitions reads course-definition.yml files from root itself and one
// or two directory levels below it (temp/<course>/, temp/<provider>/<course>/).
func loadDefinitions(root string) map[string]CourseDefinition {
	out := map[string]CourseDefinition{}
	patterns := []string{
		filepath.Join(root, "course-definition.yml"),
		filepath.Join(root, "*", "course-definition.yml"),
		filepath.Join(root, "*", "*", "course-definition.yml"),
	}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, path := range matches {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var def CourseDefinition
			if err := yaml.Unmarshal(data, &def); err != nil || def.Slug == "" {
				continue
			}
			if _, exists := out[def.Slug]; exists {
				continue
			}
			out[def.Slug] = def
		}
	}
	return out
}

// plain strips the light Markdown used in course definitions and collapses
// whitespace, for one-line summaries.
func plain(s string) string {
	replacer := strings.NewReplacer("**", "", "`", "", "__", "")
	return strings.Join(strings.Fields(replacer.Replace(stripLinks(s))), " ")
}

// stripLinks rewrites [text](url) as text, keeping prose readable where URLs
// would otherwise dominate a one-line summary.
func stripLinks(s string) string {
	if !strings.ContainsRune(s, '[') {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '[' {
			mid := strings.IndexByte(s[i:], ']')
			if mid > 0 {
				close := i + mid
				if close+1 < len(s) && s[close+1] == '(' {
					if end := strings.IndexByte(s[close:], ')'); end > 0 {
						b.WriteString(s[i+1 : close])
						i = close + end + 1
						continue
					}
				}
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
