package instagram

// PlatformName is the canonical platform identifier.
const PlatformName = "instagram"

// Config configures the Instagram adapter.
type Config struct {
	GraphBaseURL string
	AppID        string
	AppSecret    string
}
