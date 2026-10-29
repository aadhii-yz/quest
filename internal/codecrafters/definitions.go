package codecrafters

type Language struct {
	Slug string `yaml:"slug"`
}

type Testimonial struct {
	AuthorName        string `yaml:"author_name"`
	AuthorDescription string `yaml:"author_description"`
	AuthorAvatar      string `yaml:"author_avatar"`
	Link              string `yaml:"link"`
	Text              string `yaml:"text"`
}

type Extension struct {
	Slug                string `yaml:"slug"`
	Name                string `yaml:"name"`
	DescriptionMarkdown string `yaml:"description_markdown"`
}

type Stage struct {
	Slug                 string `yaml:"slug"`
	PrimaryExtensionSlug string `yaml:"primary_extension_slug"`
	Name                 string `yaml:"name"`
	Difficulty           string `yaml:"difficulty"`
	MarketingMD          string `yaml:"marketing_md"`
}

type Marketing struct {
	Difficulty                     string        `yaml:"difficulty"`
	SampleExtensionIdeaTitle       string        `yaml:"sample_extension_idea_title"`
	SampleExtensionIdeaDescription string        `yaml:"sample_extension_idea_description"`
	Testimonials                   []Testimonial `yaml:"testimonials"`
}

type CourseDefinition struct {
	Slug                 string      `yaml:"slug"`
	Name                 string      `yaml:"name"`
	ShortName            string      `yaml:"short_name"`
	ReleaseStatus        string      `yaml:"release_status"`
	DescriptionMD        string      `yaml:"description_md"`
	ShortDescriptionMD   string      `yaml:"short_description_md"`
	CompletionPercentage string      `yaml:"completion_percentage"`
	Languages            []Language  `yaml:"languages"`
	Marketing            Marketing   `yaml:"marketing"`
	Extensions           []Extension `yaml:"extensions"`
	Stages               []Stage     `yaml:"stages"`
}
