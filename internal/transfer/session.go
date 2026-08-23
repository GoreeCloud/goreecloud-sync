package transfer

import "time"

type Status string

const (
	Queued Status = "queued"
	Transferring Status = "transferring"
	Verifying Status = "verifying"
	Completed Status = "completed"
	Failed Status = "failed"
)

type Session struct {
	ID string `json:"id"`
	Filename string `json:"filename"`
	Size int64 `json:"size"`
	Status Status `json:"status"`
	Created time.Time `json:"created"`
}
