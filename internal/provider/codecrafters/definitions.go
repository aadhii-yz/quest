// Package codecrafters models the files published by CodeCrafters course
// repositories, most importantly course-definition.yml.
//
// The structs below mirror that schema field-for-field and are decoded with
// gopkg.in/yaml.v3, so the yaml tags must not be changed casually.
package codecrafters

// Language is one implementation language a course is offered in.
type Language struct {
	// Slug is the lowercase language identifier, e.g. "go", "rust", "zig".
	Slug string `yaml:"slug"`
}

// Testimonial is a single marketing quote shown on the course landing page.
type Testimonial struct {
	AuthorName        string `yaml:"author_name"`
	AuthorDescription string `yaml:"author_description"`
	// AuthorAvatar is an absolute URL to the author's avatar image.
	AuthorAvatar string `yaml:"author_avatar"`
	// Link is the author's public profile URL.
	Link string `yaml:"link"`
	// Text is the quote body. It may contain newlines, so it is commonly
	// written as a YAML block scalar.
	Text string `yaml:"text"`
}

// Extension is an optional add-on challenge that extends the base course.
type Extension struct {
	// Slug is referenced by Stage.PrimaryExtensionSlug.
	Slug                string `yaml:"slug"`
	Name                string `yaml:"name"`
	DescriptionMarkdown string `yaml:"description_markdown"`
}

// Stage is a single step of a course. Stages are ordered; the slice index in
// CourseDefinition.Stages is the stage sequence.
type Stage struct {
	Slug string `yaml:"slug"`
	// PrimaryExtensionSlug is the slug of the Extension this stage belongs to.
	// It is empty for stages that are part of the base (non-extension) course.
	PrimaryExtensionSlug string `yaml:"primary_extension_slug,omitempty"`
	Name                 string `yaml:"name"`
	// Difficulty is one of "very_easy", "easy", "medium" or "hard".
	Difficulty string `yaml:"difficulty"`
	// MarketingMD is stage prose for the marketing site. Despite the "MD"
	// suffix it is a Markdown string, not a filename.
	MarketingMD string `yaml:"marketing_md"`
}

// Marketing holds course-level landing page content.
type Marketing struct {
	// Difficulty is the course-level rating ("easy", "medium", ...), separate
	// from the per-stage Stage.Difficulty.
	Difficulty                     string        `yaml:"difficulty"`
	SampleExtensionIdeaTitle       string        `yaml:"sample_extension_idea_title"`
	SampleExtensionIdeaDescription string        `yaml:"sample_extension_idea_description"`
	Testimonials                   []Testimonial `yaml:"testimonials"`
}

// CourseDefinition is the decoded course-definition.yml that ships in every
// codecrafters-io/build-your-own-* repository.
//
// Note that the upstream schema also contains fields this model does not
// capture, such as top-level and per-stage "concept_slugs" and per-stage
// "tester_source_code_url". Unknown YAML keys are ignored by the decoder, so
// they can be added later without breaking existing parsing.
type CourseDefinition struct {
	// Slug is the short identifier, e.g. "grep", "redis", "kafka".
	Slug string `yaml:"slug"`
	// Name is the display name, e.g. "Build your own grep".
	Name string `yaml:"name"`
	// ShortName is a compact form used in URLs and badges, e.g. "grep".
	ShortName string `yaml:"short_name"`
	// ReleaseStatus is the lifecycle state, e.g. "live" or "beta".
	ReleaseStatus string `yaml:"release_status"`
	// DescriptionMD is long-form Markdown for the course overview page.
	DescriptionMD string `yaml:"description_md"`
	// ShortDescriptionMD is a one-line Markdown summary; upstream asks authors
	// to keep it under 70 characters.
	ShortDescriptionMD string `yaml:"short_description_md"`
	// CompletionPercentage is the percent of the original tool's behaviour the
	// course covers, as written in YAML (e.g. "30"). It is kept as a string
	// because the source is not guaranteed to be a plain integer.
	CompletionPercentage string `yaml:"completion_percentage"`
	// Languages are the supported implementation languages.
	Languages []Language `yaml:"languages"`
	// Marketing holds the landing page content.
	Marketing Marketing `yaml:"marketing"`
	// Extensions are the optional add-on challenges.
	Extensions []Extension `yaml:"extensions"`
	// Stages are the ordered course steps.
	Stages []Stage `yaml:"stages"`
}
