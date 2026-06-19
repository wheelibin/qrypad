package constants

const (
	AppDesc = "A simple client for quick, ad-hoc database exploration"
)

// Version is the application version. It is overridden at build time via
// -ldflags "-X github.com/wheelibin/qrypad/internal/constants.Version=<version>".
var Version = "dev" //nolint:gochecknoglobals
