package models

// FileInfo represents a single file or directory entry
type FileInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "file", "dir", "link"
	Size        int64  `json:"size"`
	ModTime     int64  `json:"mod_time"`
	Permissions string `json:"permissions"`
}

// ListFilesResponse is the response for listing files in a directory
type ListFilesResponse struct {
	Path  string     `json:"path"`
	Files []FileInfo `json:"files"`
}

// ReadFileResponse is the response for reading a file's content
type ReadFileResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
}

// WriteFileRequest is the request for writing content to a file
type WriteFileRequest struct {
	Path    string `json:"path" binding:"required"`
	Content string `json:"content"`
}

// MkdirRequest is the request for creating a directory
type MkdirRequest struct {
	Path string `json:"path" binding:"required"`
}

// DeleteFileRequest is the request for deleting a file or directory
type DeleteFileRequest struct {
	Path string `json:"path" binding:"required"`
}

// MoveFileRequest is the request for moving/renaming a file
type MoveFileRequest struct {
	Source      string `json:"source" binding:"required"`
	Destination string `json:"destination" binding:"required"`
}

// CopyFileRequest is the request for copying a file
type CopyFileRequest struct {
	Source      string `json:"source" binding:"required"`
	Destination string `json:"destination" binding:"required"`
}

// CompressFilesRequest is the request for compressing files inside the container
type CompressFilesRequest struct {
	BaseDir   string   `json:"base_dir" binding:"required"`
	FileNames []string `json:"file_names" binding:"required"`
	DestPath  string   `json:"dest_path" binding:"required"`
}

// CompressAndDownloadRequest is the request for compressing files and downloading
type CompressAndDownloadRequest struct {
	BaseDir     string   `json:"base_dir" binding:"required"`
	FileNames   []string `json:"file_names" binding:"required"`
	ArchiveName string   `json:"archive_name"`
}
