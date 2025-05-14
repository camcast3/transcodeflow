package model

type JobResult struct {
	Job    `json:"job,omitempty"`
	Output string `json:"output,omitempty"`
	Error  error  `json:"error,omitempty"`
}
