package dto

import (
	"time"

	"github.com/google/uuid"
)

// File Response DTOs

// FileInfo represents basic file information
type FileInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	ContentType  string    `json:"content_type"`
	ETag         string    `json:"etag,omitempty"`
	URL          string    `json:"url"`
}

// FileMetadataResponse represents file metadata response
type FileMetadataResponse struct {
	Key           string            `json:"key"`
	Size          int64             `json:"size"`
	LastModified  time.Time         `json:"last_modified"`
	ContentType   string            `json:"content_type"`
	ETag          string            `json:"etag,omitempty"`
	StorageClass  string            `json:"storage_class,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	CacheControl  string            `json:"cache_control,omitempty"`
	URL           string            `json:"url"`
}

// FileURLResponse represents a file URL response
type FileURLResponse struct {
	Key       string     `json:"key"`
	URL       string     `json:"url"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// FileListResponse represents a file list response
type FileListResponse struct {
	Files                 []*FileInfo `json:"files"`
	Prefixes              []string    `json:"prefixes,omitempty"`
	IsTruncated           bool        `json:"is_truncated"`
	NextContinuationToken string      `json:"next_continuation_token,omitempty"`
	TotalFiles            int         `json:"total_files"`
}

// Batch File Upload DTOs

// BatchFileUploadResponse represents a batch file upload response
type BatchFileUploadResponse struct {
	Results    []*UploadResult         `json:"results"`
	Errors     []BatchFileUploadError   `json:"errors,omitempty"`
	TotalFiles int                      `json:"total_files"`
	Successful int                      `json:"successful"`
	Failed     int                      `json:"failed"`
}

// BatchFileUploadError represents an error in batch file upload
type BatchFileUploadError struct {
	Index    int    `json:"index"`
	Filename string `json:"filename"`
	Error    string `json:"error"`
}

// UpdateMetadataRequest represents a request to update file metadata
type UpdateMetadataRequest struct {
	Metadata map[string]string `json:"metadata" binding:"required"`
}

// File Search DTOs

// SearchFilesRequest represents a file search request
type SearchFilesRequest struct {
	Query        string `form:"q" binding:"required,min=1"`
	Directory    string `form:"directory,omitempty"`
	ContentType  string `form:"content_type,omitempty"`
	MinSize      int64  `form:"min_size,omitempty"`
	MaxSize      int64  `form:"max_size,omitempty"`
	Limit        int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

// SearchFilesResponse represents file search results
type SearchFilesResponse struct {
	Files []*FileInfo `json:"files"`
	Total int         `json:"total"`
	Query string      `json:"query"`
	Limit int         `json:"limit"`
}

// File Analytics DTOs

// FileStatsResponse represents file statistics
type FileStatsResponse struct {
	TotalFiles       int64            `json:"total_files"`
	TotalSize        int64            `json:"total_size"`
	FilesByType      map[string]int64 `json:"files_by_type"`
	FilesByDirectory map[string]int64 `json:"files_by_directory"`
	UploadsToday     int64            `json:"uploads_today"`
	UploadsThisWeek  int64            `json:"uploads_this_week"`
	UploadsThisMonth int64            `json:"uploads_this_month"`
	AverageSize      int64            `json:"average_size"`
	LargestFile      int64            `json:"largest_file"`
}

// File Storage Info DTOs

// FileStorageInfo represents storage information
type FileStorageInfo struct {
	UsedSpace       int64   `json:"used_space"`
	TotalSpace      int64   `json:"total_space"`
	AvailableSpace  int64   `json:"available_space"`
	UsagePercentage float64 `json:"usage_percentage"`
	FileCount       int64   `json:"file_count"`
	Provider        string  `json:"provider"`
}

// File Upload Progress DTOs

// FileUploadProgress represents upload progress
type FileUploadProgress struct {
	ID            string  `json:"id"`
	Filename      string  `json:"filename"`
	Progress      float64 `json:"progress"` // 0-100
	Status        string  `json:"status"`  // uploading, processing, completed, failed
	Speed         int64   `json:"speed"`   // bytes per second
	BytesTotal    int64   `json:"bytes_total"`
	BytesUploaded int64   `json:"bytes_uploaded"`
	ETA           int     `json:"eta,omitempty"` // estimated time remaining in seconds
	Error         string  `json:"error,omitempty"`
}

// File Upload Session DTOs

// UploadSession represents an upload session
type UploadSession struct {
	ID              string    `json:"id"`
	Filename        string    `json:"filename"`
	Size            int64     `json:"size"`
	ContentType     string    `json:"content_type"`
	Directory       string    `json:"directory"`
	ChunkSize       int       `json:"chunk_size"`
	TotalChunks     int       `json:"total_chunks"`
	UploadedChunks  int       `json:"uploaded_chunks"`
	Status          string    `json:"status"`     // initializing, active, paused, completed, failed
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	ErrorMessage    string    `json:"error_message,omitempty"`
}

// CreateUploadSessionRequest represents a request to create an upload session
type CreateUploadSessionRequest struct {
	Filename    string `json:"filename" binding:"required"`
	Size        int64  `json:"size" binding:"required,min=1"`
	ContentType string `json:"content_type" binding:"required"`
	Directory   string `json:"directory,omitempty"`
	ChunkSize   int    `json:"chunk_size,omitempty"`
}

// UploadChunkRequest represents a request to upload a chunk
type UploadChunkRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	ChunkIndex int    `json:"chunk_index" binding:"required,min=0"`
	Data       []byte `json:"data" binding:"required"`
	Checksum   string `json:"checksum,omitempty"`
}

// UploadChunkResponse represents a chunk upload response
type UploadChunkResponse struct {
	SessionID       string `json:"session_id"`
	ChunkIndex      int    `json:"chunk_index"`
	Uploaded        bool   `json:"uploaded"`
	Progress        float64 `json:"progress"`
	Completed       bool   `json:"completed"`
	FileKey         string `json:"file_key,omitempty"`
	FileURL         string `json:"file_url,omitempty"`
	ErrorMessage    string `json:"error_message,omitempty"`
}

// File Versioning DTOs

// FileVersion represents a file version
type FileVersion struct {
	ID         uuid.UUID `json:"id"`
	FileKey    string    `json:"file_key"`
	Version    int       `json:"version"`
	Size       int64     `json:"size"`
	Checksum   string    `json:"checksum"`
	UploadedBy string    `json:"uploaded_by"`
	UploadedAt time.Time `json:"uploaded_at"`
	IsLatest   bool      `json:"is_latest"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// FileVersionHistory represents version history for a file
type FileVersionHistory struct {
	FileKey    string        `json:"file_key"`
	Filename   string        `json:"filename"`
	Versions   []*FileVersion `json:"versions"`
	TotalVersions int        `json:"total_versions"`
	CurrentVersion int      `json:"current_version"`
}

// File Sharing DTOs

// CreateShareRequest represents a request to create a file share
type CreateShareRequest struct {
	FileKey     string    `json:"file_key" binding:"required"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Password    *string   `json:"password,omitempty"`
	MaxDownloads int      `json:"max_downloads,omitempty"`
	AllowUpload bool      `json:"allow_upload,omitempty"`
}

// FileShare represents a file share
type FileShare struct {
	ID          uuid.UUID `json:"id"`
	FileKey     string    `json:"file_key"`
	ShareToken  string    `json:"share_token"`
	URL         string    `json:"url"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Password    *string   `json:"password,omitempty"`
	MaxDownloads int      `json:"max_downloads,omitempty"`
	Downloads   int       `json:"downloads"`
	AllowUpload bool      `json:"allow_upload"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	LastAccess  *time.Time `json:"last_access,omitempty"`
	IsActive    bool      `json:"is_active"`
}

// File Processing DTOs

// ProcessFileRequest represents a request to process a file
type ProcessFileRequest struct {
	FileKey      string            `json:"file_key" binding:"required"`
	Operations   []string          `json:"operations" binding:"required,min=1"`
	Options      map[string]interface{} `json:"options,omitempty"`
	NotifyEmail  *string           `json:"notify_email,omitempty"`
}

// FileProcessingJob represents a file processing job
type FileProcessingJob struct {
	ID          string            `json:"id"`
	FileKey     string            `json:"file_key"`
	Operations  []string          `json:"operations"`
	Status      string            `json:"status"` // pending, processing, completed, failed
	Progress    float64           `json:"progress"`
	Options     map[string]interface{} `json:"options,omitempty"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string            `json:"error,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
}

// File Virus Scan DTOs

// VirusScanResult represents a virus scan result
type VirusScanResult struct {
	FileKey      string    `json:"file_key"`
	ScanResult   string    `json:"scan_result"` // clean, infected, suspicious, error
	ScanEngine   string    `json:"scan_engine"`
	ThreatName   *string   `json:"threat_name,omitempty"`
	ScannedAt    time.Time `json:"scanned_at"`
	Quarantined  bool      `json:"quarantined"`
	ErrorMessage *string   `json:"error_message,omitempty"`
}

// File Backup DTOs

// FileBackupRequest represents a request to backup files
type FileBackupRequest struct {
	Keys         []string `json:"keys,omitempty"`
	Prefix       string   `json:"prefix,omitempty"`
	BackupType   string   `json:"backup_type" binding:"required,oneof=full incremental differential"`
	Compression  bool     `json:"compression"`
	Encryption   bool     `json:"encryption"`
	Destination  string   `json:"destination"`
}

// FileBackupResponse represents a backup response
type FileBackupResponse struct {
	BackupID    string    `json:"backup_id"`
	BackupType  string    `json:"backup_type"`
	FileCount   int       `json:"file_count"`
	TotalSize   int64     `json:"total_size"`
	Compressed  bool      `json:"compressed"`
	Encrypted   bool      `json:"encrypted"`
	Destination string    `json:"destination"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Status      string    `json:"status"` // in_progress, completed, failed
	ErrorMessage *string  `json:"error_message,omitempty"`
}

// File Archive DTOs

// ArchiveFileRequest represents a request to archive files
type ArchiveFileRequest struct {
	Keys        []string `json:"keys" binding:"required,min=1"`
	ArchiveName string   `json:"archive_name" binding:"required"`
	Format      string   `json:"format" binding:"required,oneof=zip tar tar.gz"`
	Compression int      `json:"compression" binding:"omitempty,min=0,max=9"`
}

// ArchiveFileResponse represents an archive response
type ArchiveResponse struct {
	ArchiveID   string    `json:"archive_id"`
	ArchiveName string    `json:"archive_name"`
	Format      string    `json:"format"`
	FileCount   int       `json:"file_count"`
	TotalSize   int64     `json:"total_size"`
	URL         string    `json:"url"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}