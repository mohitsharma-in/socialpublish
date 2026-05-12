package analytics

import "time"

// Metrics is a normalized engagement snapshot.
type Metrics struct {
	Views    int64          `json:"views"`
	Likes    int64          `json:"likes"`
	Comments int64          `json:"comments"`
	Shares   int64          `json:"shares"`
	Reach    int64          `json:"reach"`
	Extra    map[string]any `json:"extra,omitempty"`
}

// PostMetrics describes metrics for one published post.
type PostMetrics struct {
	PostID    string    `json:"post_id"`
	Metrics   Metrics   `json:"metrics"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AccountMetrics describes metrics for a connected account.
type AccountMetrics struct {
	AccountID string    `json:"account_id"`
	Metrics   Metrics   `json:"metrics"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SummaryParams filters workspace-level analytics summaries.
type SummaryParams struct {
	From *time.Time
	To   *time.Time
}

// Summary contains workspace-level analytics totals.
type Summary struct {
	Metrics Metrics `json:"metrics"`
}
