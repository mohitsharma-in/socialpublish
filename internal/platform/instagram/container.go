package instagram

import "time"

// ContainerStatus is an Instagram media container processing status.
type ContainerStatus string

const (
	// ContainerInProgress means Instagram is still processing the container.
	ContainerInProgress ContainerStatus = "IN_PROGRESS"
	// ContainerFinished means the container can be published.
	ContainerFinished ContainerStatus = "FINISHED"
	// ContainerError means Instagram failed the container.
	ContainerError ContainerStatus = "ERROR"
)

// PollConfig controls Instagram container polling.
type PollConfig struct {
	Interval time.Duration
	Timeout  time.Duration
}
