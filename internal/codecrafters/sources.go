package codecrafters

// LicenseStatus describes the licensing of a CodeCrafters repository.
type LicenseStatus string

const (
	LicenseMIT  LicenseStatus = "MIT"
	LicenseNone LicenseStatus = "No License"
)

type Source struct {
	CourseName string `yaml:"course_name"`
	// RepoURL is the course specification repository.
	RepoURL     string        `yaml:"repo_url"`
	RepoLicense LicenseStatus `yaml:"repo_license"`
	// TesterURL is the tester suite repository.
	TesterURL     string        `yaml:"tester_url"`
	TesterLicense LicenseStatus `yaml:"tester_license"`
}

// Sources lists the CodeCrafters course specification and tester suite
// repositories. License status was verified against each repository's
// license metadata.
var Sources = []Source{
	{
		CourseName:    "Build your own Shell",
		RepoURL:       "https://github.com/codecrafters-io/build-your-own-shell",
		RepoLicense:   LicenseMIT,
		TesterURL:     "https://github.com/codecrafters-io/shell-tester",
		TesterLicense: LicenseNone,
	},
	{
		CourseName:    "Build your own Redis",
		RepoURL:       "https://github.com/codecrafters-io/build-your-own-redis",
		RepoLicense:   LicenseMIT,
		TesterURL:     "https://github.com/codecrafters-io/redis-tester",
		TesterLicense: LicenseNone,
	},
	{
		CourseName:    "Build your own Grep",
		RepoURL:       "https://github.com/codecrafters-io/build-your-own-grep",
		RepoLicense:   LicenseMIT,
		TesterURL:     "https://github.com/codecrafters-io/grep-tester",
		TesterLicense: LicenseNone,
	},
	{
		CourseName:    "Build your own Claude Code",
		RepoURL:       "https://github.com/codecrafters-io/build-your-own-claude-code",
		RepoLicense:   LicenseMIT,
		TesterURL:     "https://github.com/codecrafters-io/claude-code-tester",
		TesterLicense: LicenseNone,
	},
	{
		CourseName:    "Build your own Interpreter",
		RepoURL:       "https://github.com/codecrafters-io/build-your-own-interpreter",
		RepoLicense:   LicenseMIT,
		TesterURL:     "https://github.com/codecrafters-io/interpreter-tester",
		TesterLicense: LicenseNone,
	},
	{
		CourseName:    "Build your own BitTorrent",
		RepoURL:       "https://github.com/codecrafters-io/build-your-own-bittorrent",
		RepoLicense:   LicenseMIT,
		TesterURL:     "https://github.com/codecrafters-io/bittorrent-tester",
		TesterLicense: LicenseNone,
	},
	{
		CourseName:  "Build your own Kafka",
		RepoURL:     "https://github.com/codecrafters-io/build-your-own-kafka",
		RepoLicense: LicenseMIT,
		TesterURL:   "https://github.com/codecrafters-io/kafka-tester",
		// Unlike the other testers, kafka-tester ships an MIT LICENSE file.
		TesterLicense: LicenseMIT,
	},
	{
		CourseName:    "Build your own Git",
		RepoURL:       "https://github.com/codecrafters-io/build-your-own-git",
		RepoLicense:   LicenseMIT,
		TesterURL:     "https://github.com/codecrafters-io/git-tester",
		TesterLicense: LicenseNone,
	},
	{
		CourseName:    "Build your own SQLite",
		RepoURL:       "https://github.com/codecrafters-io/build-your-own-sqlite",
		RepoLicense:   LicenseMIT,
		TesterURL:     "https://github.com/codecrafters-io/sqlite-tester",
		TesterLicense: LicenseNone,
	},
	{
		CourseName:    "Build your own DNS server",
		RepoURL:       "https://github.com/codecrafters-io/build-your-own-dns-server",
		RepoLicense:   LicenseMIT,
		TesterURL:     "https://github.com/codecrafters-io/dns-server-tester",
		TesterLicense: LicenseNone,
	},
	{
		CourseName:    "Build your own HTTP server",
		RepoURL:       "https://github.com/codecrafters-io/build-your-own-http-server",
		RepoLicense:   LicenseMIT,
		TesterURL:     "https://github.com/codecrafters-io/http-server-tester",
		TesterLicense: LicenseNone,
	},
	{
		CourseName:    "Build your own Docker",
		RepoURL:       "https://github.com/codecrafters-io/build-your-own-docker",
		RepoLicense:   LicenseMIT,
		TesterURL:     "https://github.com/codecrafters-io/docker-tester",
		TesterLicense: LicenseNone,
	},
}
