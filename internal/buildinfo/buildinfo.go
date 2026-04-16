package buildinfo

var (
	version string
	build   string
	date    string
)

func VersionInfo() string {
	return version
}

func BuildInfo() string {
	return build
}

func BuildDate() string {
	return date
}
