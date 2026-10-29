package codecrafters

// LicenseStatus describes the licensing of a CodeCrafters repository.
type LicenseStatus string

const (
	// LicenseMIT means the repository is published under the MIT License.
	LicenseMIT LicenseStatus = "MIT"
	// LicenseNone means the repository ships no license file, so all rights are
	// reserved by the author. Absence of a license is not permission to use.
	LicenseNone LicenseStatus = "No License"
)

// OS/architecture keys used in Source.Assets. They match the values
// produced by `go tool dist list`, so callers can build the key from
// runtime.GOOS and runtime.GOARCH.
const (
	ArchDarwinAMD64 = "darwin_amd64"
	ArchDarwinARM64 = "darwin_arm64"
	ArchLinuxAMD64  = "linux_amd64"
	ArchLinuxARM64  = "linux_arm64"
	ArchLinux386    = "linux_386"
)

// Asset is one prebuilt tester binary published as a GitHub release asset.
type Asset struct {
	// URL is the tar.gz download URL.
	URL string
	// SHA256 is the hex digest of the tar.gz, as published in the release's
	// checksums.txt. Verify it before extracting or executing.
	SHA256 string
}

// Source pins one course's specification repo commit and tester release.
type Source struct {
	CourseName string

	// RepoURL is the course specification repository.
	RepoURL string
	// RepoCommit is a pinned full commit SHA. Course specs are MIT licensed, but
	// pinning keeps generated content reproducible and avoids surprise upstream
	// edits changing stage text out from under a saved session.
	RepoCommit  string
	RepoLicense LicenseStatus

	// TesterURL is the tester suite repository.
	TesterURL string
	// TesterRelease is the pinned release tag. Never resolve "latest" at runtime:
	// it is unreviewed code and the tag is needed to build asset names.
	TesterRelease string
	// TesterChecksumsURL points at the release's checksums.txt.
	TesterChecksumsURL string
	TesterLicense      LicenseStatus

	// Assets maps an OS/architecture key (see the Arch* constants) to the
	// prebuilt tester binary for that platform. Not every course ships every
	// platform; use AssetFor rather than indexing directly.
	Assets map[string]Asset
}

// AssetFor returns the tester asset for the given OS/architecture key.
// ok is false when the course publishes no build for that platform.
func (s Source) AssetFor(osArch string) (Asset, bool) {
	a, ok := s.Assets[osArch]
	return a, ok
}

// Sources lists the CodeCrafters courses. Spec repo commits and tester
// releases are pinned to reviewed versions; asset digests were taken from
// each release's published checksums.txt.
//
// The tester binaries are fetched and executed on the user's machine. Most
// tester repositories carry no license, so treat TesterLicense as a gate: do
// not download or run an unlicensed tester without explicit user consent.
var Sources = []Source{
	{
		CourseName: "Build your own Shell",

		RepoURL:     "https://github.com/codecrafters-io/build-your-own-shell",
		RepoCommit:  "1b6a5b9a16f695429352f12e8289b96b7a0314bb",
		RepoLicense: LicenseMIT,

		TesterURL:          "https://github.com/codecrafters-io/shell-tester",
		TesterRelease:      "v133",
		TesterChecksumsURL: "https://github.com/codecrafters-io/shell-tester/releases/download/v133/shell-tester_133_checksums.txt",
		TesterLicense:      LicenseNone,

		Assets: map[string]Asset{
			ArchDarwinAMD64: {
				URL:    "https://github.com/codecrafters-io/shell-tester/releases/download/v133/v133_darwin_amd64.tar.gz",
				SHA256: "fb3b9e0d359cb7ad876b7c70d4781d944133eb340252a7fa0c548f6b02522d34",
			},
			ArchDarwinARM64: {
				URL:    "https://github.com/codecrafters-io/shell-tester/releases/download/v133/v133_darwin_arm64.tar.gz",
				SHA256: "45b82486cc2692cf842a22ef24d10132b39db3f76a0c5ad2d913456f0613b639",
			},
			ArchLinuxAMD64: {
				URL:    "https://github.com/codecrafters-io/shell-tester/releases/download/v133/v133_linux_amd64.tar.gz",
				SHA256: "34c69047f644db485e3941bfc709736c05596a0862a60f03e4dac47c0db70358",
			},
			ArchLinuxARM64: {
				URL:    "https://github.com/codecrafters-io/shell-tester/releases/download/v133/v133_linux_arm64.tar.gz",
				SHA256: "76df5e9c1c408b1e227153cf704810e5a2af7dd604c1d0b0db6d08b2c7fd8402",
			},
		},
	},
	{
		CourseName: "Build your own Redis",

		RepoURL:     "https://github.com/codecrafters-io/build-your-own-redis",
		RepoCommit:  "09b175e62b02299d190465092d45aeaaa6435855",
		RepoLicense: LicenseMIT,

		TesterURL:          "https://github.com/codecrafters-io/redis-tester",
		TesterRelease:      "v213",
		TesterChecksumsURL: "https://github.com/codecrafters-io/redis-tester/releases/download/v213/redis-tester_213_checksums.txt",
		TesterLicense:      LicenseNone,

		Assets: map[string]Asset{
			ArchDarwinAMD64: {
				URL:    "https://github.com/codecrafters-io/redis-tester/releases/download/v213/v213_darwin_amd64.tar.gz",
				SHA256: "9eea737e102eda19581d10efedd831bec4cf563799a8f5e53c7eb7bb10636d11",
			},
			ArchDarwinARM64: {
				URL:    "https://github.com/codecrafters-io/redis-tester/releases/download/v213/v213_darwin_arm64.tar.gz",
				SHA256: "84b05ff9e4e127c386b5030c277d62f1dc557706968df374a27b154077b840fc",
			},
			ArchLinuxAMD64: {
				URL:    "https://github.com/codecrafters-io/redis-tester/releases/download/v213/v213_linux_amd64.tar.gz",
				SHA256: "9fcdb2eeb15d67f181ddc052caaf64422a280a3cca1417c5f4252431e6b96353",
			},
			ArchLinuxARM64: {
				URL:    "https://github.com/codecrafters-io/redis-tester/releases/download/v213/v213_linux_arm64.tar.gz",
				SHA256: "78c1fd94075b99858865a09acb8ab25cd5a8f9d2581d9325d2034995e7bed539",
			},
		},
	},
	{
		CourseName: "Build your own Grep",

		RepoURL:     "https://github.com/codecrafters-io/build-your-own-grep",
		RepoCommit:  "6d54daaad5632b077289c750f01e0df61b4fde98",
		RepoLicense: LicenseMIT,

		TesterURL:          "https://github.com/codecrafters-io/grep-tester",
		TesterRelease:      "v65",
		TesterChecksumsURL: "https://github.com/codecrafters-io/grep-tester/releases/download/v65/grep-tester_65_checksums.txt",
		TesterLicense:      LicenseNone,

		Assets: map[string]Asset{
			ArchDarwinAMD64: {
				URL:    "https://github.com/codecrafters-io/grep-tester/releases/download/v65/v65_darwin_amd64.tar.gz",
				SHA256: "4dca9622d8f3899e4b10d0fc53dec28cdcaed144d084049d4eda1b35a780da4e",
			},
			ArchLinuxAMD64: {
				URL:    "https://github.com/codecrafters-io/grep-tester/releases/download/v65/v65_linux_amd64.tar.gz",
				SHA256: "2ddcfbdf62d40be9c5d022744c46a33f490775315f14c123961c7f59260373f9",
			},
			ArchLinuxARM64: {
				URL:    "https://github.com/codecrafters-io/grep-tester/releases/download/v65/v65_linux_arm64.tar.gz",
				SHA256: "c61584041e48324145b531f991c4d9cb301adbe08cd19f8e86a4f98452f6bc85",
			},
			ArchLinux386: {
				URL:    "https://github.com/codecrafters-io/grep-tester/releases/download/v65/v65_linux_386.tar.gz",
				SHA256: "3b92e2c7adcf9e909d6d7e9ddac10e7a19564f1142d32fd9f068f07346a45718",
			},
		},
	},
	{
		CourseName: "Build your own Claude Code",

		RepoURL:     "https://github.com/codecrafters-io/build-your-own-claude-code",
		RepoCommit:  "56a12aeef52d2748eaf51a20e44e613f4bbee59a",
		RepoLicense: LicenseMIT,

		TesterURL:          "https://github.com/codecrafters-io/claude-code-tester",
		TesterRelease:      "v14",
		TesterChecksumsURL: "https://github.com/codecrafters-io/claude-code-tester/releases/download/v14/claude-code-tester_14_checksums.txt",
		TesterLicense:      LicenseNone,

		Assets: map[string]Asset{
			ArchDarwinAMD64: {
				URL:    "https://github.com/codecrafters-io/claude-code-tester/releases/download/v14/v14_darwin_amd64.tar.gz",
				SHA256: "ac1d504d214dff00d23603d3dd23e9bf7a6137e53a51769705c0a6ae6f522d27",
			},
			ArchLinuxAMD64: {
				URL:    "https://github.com/codecrafters-io/claude-code-tester/releases/download/v14/v14_linux_amd64.tar.gz",
				SHA256: "93411d0316003b5b2a2e581a86ed073e6322533ada7f0305472ba5b6956b76aa",
			},
			ArchLinuxARM64: {
				URL:    "https://github.com/codecrafters-io/claude-code-tester/releases/download/v14/v14_linux_arm64.tar.gz",
				SHA256: "0ee5f593eb5dce4902a4d1ef83a4752cacb853167870ee9a977d1a7393ad8cff",
			},
			ArchLinux386: {
				URL:    "https://github.com/codecrafters-io/claude-code-tester/releases/download/v14/v14_linux_386.tar.gz",
				SHA256: "bc5f9628a0048a73fc314ee61d9d0bbc5218b101e492faaa86e1135081df79ef",
			},
		},
	},
	{
		CourseName: "Build your own Interpreter",

		RepoURL:     "https://github.com/codecrafters-io/build-your-own-interpreter",
		RepoCommit:  "e05193b218567c80504f5358585fbfe2908ffbca",
		RepoLicense: LicenseMIT,

		TesterURL:          "https://github.com/codecrafters-io/interpreter-tester",
		TesterRelease:      "v64",
		TesterChecksumsURL: "https://github.com/codecrafters-io/interpreter-tester/releases/download/v64/interpreter-tester_64_checksums.txt",
		TesterLicense:      LicenseNone,

		Assets: map[string]Asset{
			ArchDarwinAMD64: {
				URL:    "https://github.com/codecrafters-io/interpreter-tester/releases/download/v64/v64_darwin_amd64.tar.gz",
				SHA256: "4f107867bed615b868f6c4fc7ea421656ab64fad85a5a7ce4e5fbf73415b2c0f",
			},
			ArchLinuxAMD64: {
				URL:    "https://github.com/codecrafters-io/interpreter-tester/releases/download/v64/v64_linux_amd64.tar.gz",
				SHA256: "07a31d558c8c4c55faa58a72c03ed1025e6f0dc0460f7b123aeded032570a201",
			},
			ArchLinuxARM64: {
				URL:    "https://github.com/codecrafters-io/interpreter-tester/releases/download/v64/v64_linux_arm64.tar.gz",
				SHA256: "2c2b26dd998903b4a6ab77495ce88b1b692ef634d67165a2a9dab1ab317f61fd",
			},
			ArchLinux386: {
				URL:    "https://github.com/codecrafters-io/interpreter-tester/releases/download/v64/v64_linux_386.tar.gz",
				SHA256: "58c7b6fa3509f12ffd8e36d6a40a40ae4f334689f0e611fde610c46454ba559b",
			},
		},
	},
	{
		CourseName: "Build your own BitTorrent",

		RepoURL:     "https://github.com/codecrafters-io/build-your-own-bittorrent",
		RepoCommit:  "2c63fcce4328dee6faeb682b21206d2b26b510a5",
		RepoLicense: LicenseMIT,

		TesterURL:          "https://github.com/codecrafters-io/bittorrent-tester",
		TesterRelease:      "v81",
		TesterChecksumsURL: "https://github.com/codecrafters-io/bittorrent-tester/releases/download/v81/bittorrent-tester_81_checksums.txt",
		TesterLicense:      LicenseNone,

		Assets: map[string]Asset{
			ArchDarwinAMD64: {
				URL:    "https://github.com/codecrafters-io/bittorrent-tester/releases/download/v81/v81_darwin_amd64.tar.gz",
				SHA256: "a5df7430eb62615419a8ac158fb4a6b6754d139d1e427218242ced5119a3f4fa",
			},
			ArchLinuxAMD64: {
				URL:    "https://github.com/codecrafters-io/bittorrent-tester/releases/download/v81/v81_linux_amd64.tar.gz",
				SHA256: "a46fb0cc387e4d2daf6b6af8fc82dd93f970633cbfae7bd7905244cff9a0c50b",
			},
			ArchLinuxARM64: {
				URL:    "https://github.com/codecrafters-io/bittorrent-tester/releases/download/v81/v81_linux_arm64.tar.gz",
				SHA256: "89f770933f17734446df4e8ff43704bcaecf03b2cef764d40e756272f20ed5ce",
			},
			ArchLinux386: {
				URL:    "https://github.com/codecrafters-io/bittorrent-tester/releases/download/v81/v81_linux_386.tar.gz",
				SHA256: "2b7ea6808a1854afab9ca419e55df59ad8d9b2d72c7994b606413160a79860ea",
			},
		},
	},
	{
		CourseName: "Build your own Kafka",

		RepoURL:     "https://github.com/codecrafters-io/build-your-own-kafka",
		RepoCommit:  "d759358f0f880994ae6ba86871523310b39fc0c9",
		RepoLicense: LicenseMIT,

		TesterURL:          "https://github.com/codecrafters-io/kafka-tester",
		TesterRelease:      "v44",
		TesterChecksumsURL: "https://github.com/codecrafters-io/kafka-tester/releases/download/v44/kafka-tester_44_checksums.txt",
		// Unlike the other testers, this one ships an MIT LICENSE file.
		TesterLicense: LicenseMIT,

		Assets: map[string]Asset{
			ArchDarwinAMD64: {
				URL:    "https://github.com/codecrafters-io/kafka-tester/releases/download/v44/v44_darwin_amd64.tar.gz",
				SHA256: "b98fef0e597bb1d935a8f747bf7ddb4ccda215504363f27e39e7bd51a1d5ece8",
			},
			ArchDarwinARM64: {
				URL:    "https://github.com/codecrafters-io/kafka-tester/releases/download/v44/v44_darwin_arm64.tar.gz",
				SHA256: "3ee2f7ac642542201eebd974dec9a805aa3bc6a0c4da244340357c80554b2176",
			},
			ArchLinuxAMD64: {
				URL:    "https://github.com/codecrafters-io/kafka-tester/releases/download/v44/v44_linux_amd64.tar.gz",
				SHA256: "abf4799741beb22c0e02170cea72d49f24d2849c6964dce1022340591d76ac29",
			},
			ArchLinuxARM64: {
				URL:    "https://github.com/codecrafters-io/kafka-tester/releases/download/v44/v44_linux_arm64.tar.gz",
				SHA256: "d14caee0e0e28f78dc1980e9e94afa57d42972ef87fec259d7ff04981dd27216",
			},
		},
	},
	{
		CourseName: "Build your own Git",

		RepoURL:     "https://github.com/codecrafters-io/build-your-own-git",
		RepoCommit:  "34f4e91a91f8a435830eca21080b6e5e0b1ec48a",
		RepoLicense: LicenseMIT,

		TesterURL:          "https://github.com/codecrafters-io/git-tester",
		TesterRelease:      "v57",
		TesterChecksumsURL: "https://github.com/codecrafters-io/git-tester/releases/download/v57/git-tester_57_checksums.txt",
		TesterLicense:      LicenseNone,

		Assets: map[string]Asset{
			ArchDarwinAMD64: {
				URL:    "https://github.com/codecrafters-io/git-tester/releases/download/v57/v57_darwin_amd64.tar.gz",
				SHA256: "1d502e7dd88948f60cc5c57739080c51ac0c5b364b5ffd34a51eae7a17d1be0d",
			},
			ArchLinuxAMD64: {
				URL:    "https://github.com/codecrafters-io/git-tester/releases/download/v57/v57_linux_amd64.tar.gz",
				SHA256: "7a1ad75e7ac05c0525c9354c92c50b7d10956173721c6d0ed89c696d7812143d",
			},
			ArchLinuxARM64: {
				URL:    "https://github.com/codecrafters-io/git-tester/releases/download/v57/v57_linux_arm64.tar.gz",
				SHA256: "4641c8079cc3bf822eecb19af6cc1936a9966b01bd175c71ac0be01d9d8e3161",
			},
			ArchLinux386: {
				URL:    "https://github.com/codecrafters-io/git-tester/releases/download/v57/v57_linux_386.tar.gz",
				SHA256: "53d4c068494416a447c6d4135aad55792a4137a28d1fa3673719d4566325bcbd",
			},
		},
	},
	{
		CourseName: "Build your own SQLite",

		RepoURL:     "https://github.com/codecrafters-io/build-your-own-sqlite",
		RepoCommit:  "b58b43a5b0b5321ac39fe05bc7298bd30d924654",
		RepoLicense: LicenseMIT,

		TesterURL:          "https://github.com/codecrafters-io/sqlite-tester",
		TesterRelease:      "v73",
		TesterChecksumsURL: "https://github.com/codecrafters-io/sqlite-tester/releases/download/v73/sqlite-tester_73_checksums.txt",
		TesterLicense:      LicenseNone,

		// These assets are ~365 MB each because the tester embeds sqlite
		// binaries. Surface the download size before starting.
		Assets: map[string]Asset{
			ArchDarwinAMD64: {
				URL:    "https://github.com/codecrafters-io/sqlite-tester/releases/download/v73/v73_darwin_amd64.tar.gz",
				SHA256: "c18d322c459d86ffe205095510ed55a7462733c308f7e2a89c5024410d087f2c",
			},
			ArchLinuxAMD64: {
				URL:    "https://github.com/codecrafters-io/sqlite-tester/releases/download/v73/v73_linux_amd64.tar.gz",
				SHA256: "4c43acf20e0aac8cfcfedfd40b401ea5421f2bd008dd21e294928c5002001ae5",
			},
			ArchLinuxARM64: {
				URL:    "https://github.com/codecrafters-io/sqlite-tester/releases/download/v73/v73_linux_arm64.tar.gz",
				SHA256: "9615d6d42671cc7945d1ca4a5265be84985642effd1488950c5137f50fefaef3",
			},
		},
	},
	{
		CourseName: "Build your own DNS server",

		RepoURL:     "https://github.com/codecrafters-io/build-your-own-dns-server",
		RepoCommit:  "6665dd2fd72ab3ef709c14a0236befac1941ab38",
		RepoLicense: LicenseMIT,

		TesterURL:          "https://github.com/codecrafters-io/dns-server-tester",
		TesterRelease:      "v29",
		TesterChecksumsURL: "https://github.com/codecrafters-io/dns-server-tester/releases/download/v29/dns-server-tester_29_checksums.txt",
		TesterLicense:      LicenseNone,

		Assets: map[string]Asset{
			ArchDarwinAMD64: {
				URL:    "https://github.com/codecrafters-io/dns-server-tester/releases/download/v29/v29_darwin_amd64.tar.gz",
				SHA256: "d0d324fa2fb146ef203b84f0205ce0ec01807a2ad905f1081b4001b3aa16f3d3",
			},
			ArchLinuxAMD64: {
				URL:    "https://github.com/codecrafters-io/dns-server-tester/releases/download/v29/v29_linux_amd64.tar.gz",
				SHA256: "8b5370d58a765111df065a3401e530fcf2651f9833d95943ebb6813b4c6fe308",
			},
			ArchLinuxARM64: {
				URL:    "https://github.com/codecrafters-io/dns-server-tester/releases/download/v29/v29_linux_arm64.tar.gz",
				SHA256: "17cc95cc5a0b4363928246dcb3c93f4cbc5dabcc3d6751d386d6acf652aadc1f",
			},
			ArchLinux386: {
				URL:    "https://github.com/codecrafters-io/dns-server-tester/releases/download/v29/v29_linux_386.tar.gz",
				SHA256: "fc40a0f2fb577779bf77138d078b88a9610507382ddef46016e1cbd5c4863826",
			},
		},
	},
	{
		CourseName: "Build your own HTTP server",

		RepoURL:     "https://github.com/codecrafters-io/build-your-own-http-server",
		RepoCommit:  "e086ba45900544ab17cd7f5bc9c4f094c8e87157",
		RepoLicense: LicenseMIT,

		TesterURL:          "https://github.com/codecrafters-io/http-server-tester",
		TesterRelease:      "v59",
		TesterChecksumsURL: "https://github.com/codecrafters-io/http-server-tester/releases/download/v59/http-server-tester_59_checksums.txt",
		TesterLicense:      LicenseNone,

		Assets: map[string]Asset{
			ArchDarwinAMD64: {
				URL:    "https://github.com/codecrafters-io/http-server-tester/releases/download/v59/v59_darwin_amd64.tar.gz",
				SHA256: "d8f4f2d05f30fa6561465eb82b28550ce061741d1af5a44dc67e7aff379e32d2",
			},
			ArchDarwinARM64: {
				URL:    "https://github.com/codecrafters-io/http-server-tester/releases/download/v59/v59_darwin_arm64.tar.gz",
				SHA256: "bac680398f07eade64ddd2efe4b2969449dc1379986701e3a8f4b83823f5caf5",
			},
			ArchLinuxAMD64: {
				URL:    "https://github.com/codecrafters-io/http-server-tester/releases/download/v59/v59_linux_amd64.tar.gz",
				SHA256: "6af09697d080d528b81a9d366b6687db10c16c79687c1fdaf1f88a2888a2a7e3",
			},
			ArchLinuxARM64: {
				URL:    "https://github.com/codecrafters-io/http-server-tester/releases/download/v59/v59_linux_arm64.tar.gz",
				SHA256: "34c5dd24535a07dad50429763f8bce2053074cdca84f3ca1cc6c2ac2bc9aa6a8",
			},
		},
	},
	{
		CourseName: "Build your own Docker",

		RepoURL:     "https://github.com/codecrafters-io/build-your-own-docker",
		RepoCommit:  "3c07c87a45f32355a298505d7ca75372aed3ca95",
		RepoLicense: LicenseMIT,

		TesterURL:          "https://github.com/codecrafters-io/docker-tester",
		TesterRelease:      "v54",
		TesterChecksumsURL: "https://github.com/codecrafters-io/docker-tester/releases/download/v54/docker-tester_54_checksums.txt",
		TesterLicense:      LicenseNone,

		Assets: map[string]Asset{
			ArchDarwinAMD64: {
				URL:    "https://github.com/codecrafters-io/docker-tester/releases/download/v54/v54_darwin_amd64.tar.gz",
				SHA256: "1649dd6866cb38e03db68a5ef441a35c2b840418803078654b46e398b305976a",
			},
			ArchLinuxAMD64: {
				URL:    "https://github.com/codecrafters-io/docker-tester/releases/download/v54/v54_linux_amd64.tar.gz",
				SHA256: "e59d7d5ba1181a7f358f47f79d7e5805af0efddfad1488b60592541fb830ec79",
			},
			ArchLinuxARM64: {
				URL:    "https://github.com/codecrafters-io/docker-tester/releases/download/v54/v54_linux_arm64.tar.gz",
				SHA256: "0ce8080116bf77f313197bf0647aa277d1bfd2beb7b0a2a9bbc77419bd3ebe92",
			},
			ArchLinux386: {
				URL:    "https://github.com/codecrafters-io/docker-tester/releases/download/v54/v54_linux_386.tar.gz",
				SHA256: "463c4d25d8b58d00c01b2a896e4ed6d30a72614c635017007d589347b5f8868d",
			},
		},
	},
}
