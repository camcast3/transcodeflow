package model

// Job represents a transcoding job.
type Job struct {
	InputFilePath  string `json:"input_file_path"`
	OutputFilePath string `json:"output_file_path"`
	ContainerType  string `json:"container_type"`
	Flags          string `json:"flags"`
}
