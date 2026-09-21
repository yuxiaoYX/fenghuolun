package version

// Filled at link time. Local `go run` stays "dev".
var (
	Version = "dev"
	Commit  = ""
)

func Display() string {
	if Version != "" {
		return Version
	}
	return "dev"
}
