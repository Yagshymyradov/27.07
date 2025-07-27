package task

import "time"

type Status string

const (
	StatusNew        Status = "new"
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusDone       Status = "done"
	StatusError      Status = "error"
)

type ItemStatus string

const (
	ItemPending ItemStatus = "pending"
	ItemOK      ItemStatus = "ok"
	ItemError   ItemStatus = "error"
)

type Item struct {
	URL    string     `json:"url"`
	Status ItemStatus `json:"status"`
	ErrMsg string     `json:"err_msg"`
}

type Task struct {
	ID     string
	Status Status
	Items  []Item `json:"items"`

	CreatedAt time.Time
	UpdatedAt time.Time

	ResultPath string `json:"result,omitempty"`
	ErrMsg     string `json:"error,omitempty"`
}
