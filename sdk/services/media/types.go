package media

import "time"

// Type identifies a media asset type.
type Type string

const (
	// TypeVideo identifies video media.
	TypeVideo Type = "video"
	// TypeImage identifies image media.
	TypeImage Type = "image"
)

// Status represents media processing state.
type Status string

const (
	// StatusUploading indicates the original asset is being uploaded.
	StatusUploading Status = "uploading"
	// StatusProcessing indicates derived assets are being generated.
	StatusProcessing Status = "processing"
	// StatusReady indicates media is ready for publishing.
	StatusReady Status = "ready"
	// StatusFailed indicates processing failed.
	StatusFailed Status = "failed"
)

// Asset describes an uploaded media object.
type Asset struct {
	ID           string         `json:"media_id"`
	Status       Status         `json:"status"`
	MediaType    Type           `json:"media_type"`
	OriginalKey  string         `json:"original_key"`
	MimeType     string         `json:"mime_type"`
	SizeBytes    int64          `json:"size_bytes"`
	DurationMS   int            `json:"duration_ms,omitempty"`
	Formats      map[string]any `json:"formats"`
	ThumbnailKey string         `json:"thumbnail_key,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// UploadRequest describes a new media upload.
type UploadRequest struct {
	FileName  string `json:"file_name"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	MediaType Type   `json:"media_type"`
}

// UploadResponse returns the direct-upload target.
type UploadResponse struct {
	MediaID   string            `json:"media_id"`
	UploadURL string            `json:"upload_url"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// ListParams filters and paginates media listing.
type ListParams struct {
	Limit  int
	Cursor string
	Status Status
	Type   Type
}

// Page is a paginated media response.
type Page struct {
	Items      []*Asset `json:"items"`
	NextCursor string   `json:"next_cursor,omitempty"`
	HasMore    bool     `json:"has_more"`
	Total      int      `json:"total"`
}

// GetItems returns the page items.
func (p *Page) GetItems() []*Asset { return p.Items }

// GetNextCursor returns the next page cursor.
func (p *Page) GetNextCursor() string { return p.NextCursor }

// GetHasMore reports whether another page is available.
func (p *Page) GetHasMore() bool { return p.HasMore }
