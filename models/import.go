package models

type ImportType string

const (
	ImportTypeCSV  ImportType = "csv"
	ImportTypeJSON ImportType = "json"
)

type Import struct {
	Id        uint64     `json:"id"`
	Name      string     `json:"name"`
	FilePath  string     `json:"file_path"`
	FileType  ImportType `json:"file_type"`
	FileSize  uint64     `json:"file_size"`
	ErrorPath string     `json:"error_path"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
}
