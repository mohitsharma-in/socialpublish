package schedule

import "time"

// Window describes an available publishing time window.
type Window struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// CalendarParams filters scheduled posts by time range.
type CalendarParams struct {
	From *time.Time
	To   *time.Time
}

// QueueParams filters queued scheduled work.
type QueueParams struct {
	Limit  int
	Cursor string
}

// Entry is one scheduled publishing item.
type Entry struct {
	PostID      string    `json:"post_id"`
	Status      string    `json:"status"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Platforms   []string  `json:"platforms"`
}

// CalendarResponse returns scheduled items in a range.
type CalendarResponse struct {
	Entries []Entry `json:"entries"`
}

// QueueResponse returns pending scheduled work.
type QueueResponse struct {
	Items      []Entry `json:"items"`
	NextCursor string  `json:"next_cursor,omitempty"`
	HasMore    bool    `json:"has_more"`
}
