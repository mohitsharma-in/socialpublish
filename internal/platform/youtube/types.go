package youtube

// PlatformName is the canonical platform identifier.
const PlatformName = "youtube"

// Config configures the YouTube adapter.
type Config struct {
	APIBaseURL   string
	UploadURL    string
	ClientID     string
	ClientSecret string
}
