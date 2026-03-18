package api

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"google.golang.org/api/drive/v3"
)

// DriveService wraps Drive API operations.
type DriveService struct {
	client *Client
}

// NewDriveService creates a Drive service wrapper.
func NewDriveService(client *Client) *DriveService {
	return &DriveService{client: client}
}

// FileSummary is a simplified file representation.
type FileSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	MimeType     string `json:"mime_type"`
	Size         int64  `json:"size"`
	ModifiedTime string `json:"modified_time"`
	CreatedTime  string `json:"created_time,omitempty"`
	WebViewLink  string `json:"web_view_link,omitempty"`
	Parents      []string `json:"parents,omitempty"`
	Shared       bool   `json:"shared"`
	Trashed      bool   `json:"trashed"`
}

// ListFiles lists files in a folder or root.
func (ds *DriveService) ListFiles(ctx context.Context, folderID string, maxResults int64) ([]FileSummary, error) {
	if !ds.client.NoCache {
		key := cacheKey("drive", "ListFiles", folderID, maxResults)
		if cached, ok := ds.client.cache.Get(key); ok {
			return cached.([]FileSummary), nil
		}
	}

	if err := ds.client.WaitRate(ctx, "drive"); err != nil {
		return nil, err
	}

	opts, err := ds.client.ClientOptions(ctx, "drive")
	if err != nil {
		return nil, err
	}

	svc, err := drive.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}

	query := "trashed = false"
	if folderID != "" {
		if !isValidDriveID(folderID) {
			return nil, fmt.Errorf("invalid folder ID: %q", folderID)
		}
		query = fmt.Sprintf("'%s' in parents and trashed = false", folderID)
	}

	call := svc.Files.List().
		Q(query).
		Fields("files(id,name,mimeType,size,modifiedTime,createdTime,webViewLink,parents,shared,trashed)").
		OrderBy("modifiedTime desc")

	if maxResults > 0 {
		call = call.PageSize(maxResults)
	}

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	var files []FileSummary
	for _, f := range resp.Files {
		files = append(files, fileToSummary(f))
	}

	if !ds.client.NoCache {
		key := cacheKey("drive", "ListFiles", folderID, maxResults)
		ds.client.cache.Set(key, files, 5*time.Minute)
	}
	return files, nil
}

// SearchFiles searches files by query.
func (ds *DriveService) SearchFiles(ctx context.Context, query string, maxResults int64) ([]FileSummary, error) {
	if !ds.client.NoCache {
		key := cacheKey("drive", "SearchFiles", query, maxResults)
		if cached, ok := ds.client.cache.Get(key); ok {
			return cached.([]FileSummary), nil
		}
	}

	if err := ds.client.WaitRate(ctx, "drive"); err != nil {
		return nil, err
	}

	opts, err := ds.client.ClientOptions(ctx, "drive")
	if err != nil {
		return nil, err
	}

	svc, err := drive.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}

	// Wrap user query with trashed filter
	fullQuery := fmt.Sprintf("(%s) and trashed = false", query)

	call := svc.Files.List().
		Q(fullQuery).
		Fields("files(id,name,mimeType,size,modifiedTime,createdTime,webViewLink,parents,shared,trashed)").
		OrderBy("modifiedTime desc")

	if maxResults > 0 {
		call = call.PageSize(maxResults)
	}

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("search files: %w", err)
	}

	var files []FileSummary
	for _, f := range resp.Files {
		files = append(files, fileToSummary(f))
	}

	if !ds.client.NoCache {
		key := cacheKey("drive", "SearchFiles", query, maxResults)
		ds.client.cache.Set(key, files, 5*time.Minute)
	}
	return files, nil
}

// UploadFile uploads a file to Drive.
func (ds *DriveService) UploadFile(ctx context.Context, localPath string, folderID string, name string) (*FileSummary, error) {
	if err := ds.client.WaitRate(ctx, "drive"); err != nil {
		return nil, err
	}

	opts, err := ds.client.ClientOptions(ctx, "drive")
	if err != nil {
		return nil, err
	}

	svc, err := drive.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}

	f, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	if name == "" {
		name = filepath.Base(localPath)
	}

	meta := &drive.File{Name: name}
	if folderID != "" {
		meta.Parents = []string{folderID}
	}

	created, err := svc.Files.Create(meta).Media(f).
		Fields("id,name,mimeType,size,modifiedTime,webViewLink,parents,shared").Do()
	if err != nil {
		return nil, fmt.Errorf("upload file: %w", err)
	}

	summary := fileToSummary(created)
	if !ds.client.NoCache {
		ds.client.cache.InvalidatePrefix("drive:")
	}
	return &summary, nil
}

// DownloadFile downloads a file from Drive.
func (ds *DriveService) DownloadFile(ctx context.Context, fileID string, outputPath string) (string, error) {
	if err := ds.client.WaitRate(ctx, "drive"); err != nil {
		return "", err
	}

	opts, err := ds.client.ClientOptions(ctx, "drive")
	if err != nil {
		return "", err
	}

	svc, err := drive.NewService(ctx, opts...)
	if err != nil {
		return "", fmt.Errorf("create drive service: %w", err)
	}

	// Get file metadata for name and size
	meta, err := svc.Files.Get(fileID).Fields("name,mimeType,size").Do()
	if err != nil {
		return "", fmt.Errorf("get file metadata: %w", err)
	}

	if outputPath == "" {
		outputPath = meta.Name
	}

	resp, err := svc.Files.Get(fileID).Download()
	if err != nil {
		return "", fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	out, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("create output file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return outputPath, nil
}

// ShareFile shares a file with a user.
func (ds *DriveService) ShareFile(ctx context.Context, fileID string, email string, role string) error {
	if err := ds.client.WaitRate(ctx, "drive"); err != nil {
		return err
	}

	opts, err := ds.client.ClientOptions(ctx, "drive")
	if err != nil {
		return err
	}

	svc, err := drive.NewService(ctx, opts...)
	if err != nil {
		return fmt.Errorf("create drive service: %w", err)
	}

	perm := &drive.Permission{
		Type:         "user",
		Role:         role,
		EmailAddress: email,
	}

	if _, err := svc.Permissions.Create(fileID, perm).Do(); err != nil {
		return fmt.Errorf("share file: %w", err)
	}
	return nil
}

// CreateFolder creates a folder in Drive.
func (ds *DriveService) CreateFolder(ctx context.Context, name string, parentID string) (*FileSummary, error) {
	if err := ds.client.WaitRate(ctx, "drive"); err != nil {
		return nil, err
	}

	opts, err := ds.client.ClientOptions(ctx, "drive")
	if err != nil {
		return nil, err
	}

	svc, err := drive.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}

	meta := &drive.File{
		Name:     name,
		MimeType: "application/vnd.google-apps.folder",
	}
	if parentID != "" {
		meta.Parents = []string{parentID}
	}

	created, err := svc.Files.Create(meta).
		Fields("id,name,mimeType,modifiedTime,webViewLink,parents").Do()
	if err != nil {
		return nil, fmt.Errorf("create folder: %w", err)
	}

	summary := fileToSummary(created)
	if !ds.client.NoCache {
		ds.client.cache.InvalidatePrefix("drive:")
	}
	return &summary, nil
}

// driveIDPattern validates Google Drive file/folder IDs (alphanumeric + dash + underscore).
var driveIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func isValidDriveID(id string) bool {
	return driveIDPattern.MatchString(id)
}

// CheckDownloadSize fetches file metadata and returns an error if size exceeds maxBytes.
func (ds *DriveService) CheckDownloadSize(ctx context.Context, fileID string, maxBytes int64) error {
	if err := ds.client.WaitRate(ctx, "drive"); err != nil {
		return err
	}

	opts, err := ds.client.ClientOptions(ctx, "drive")
	if err != nil {
		return err
	}

	svc, err := drive.NewService(ctx, opts...)
	if err != nil {
		return fmt.Errorf("create drive service: %w", err)
	}

	meta, err := svc.Files.Get(fileID).Fields("name,size").Do()
	if err != nil {
		return fmt.Errorf("get file metadata: %w", err)
	}

	if meta.Size > maxBytes {
		return fmt.Errorf("file %q is %.1f MB, exceeds the %.0f MB download limit",
			meta.Name,
			float64(meta.Size)/1024/1024,
			float64(maxBytes)/1024/1024,
		)
	}
	return nil
}

func fileToSummary(f *drive.File) FileSummary {
	return FileSummary{
		ID:           f.Id,
		Name:         f.Name,
		MimeType:     f.MimeType,
		Size:         f.Size,
		ModifiedTime: f.ModifiedTime,
		CreatedTime:  f.CreatedTime,
		WebViewLink:  f.WebViewLink,
		Parents:      f.Parents,
		Shared:       f.Shared,
		Trashed:      f.Trashed,
	}
}
