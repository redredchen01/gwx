package cmd

import (
	"github.com/user/gwx/internal/api"
	"github.com/user/gwx/internal/exitcode"
)

// DriveCmd groups Drive operations.
type DriveCmd struct {
	List     DriveListCmd     `cmd:"" help:"List files"`
	Search   DriveSearchCmd   `cmd:"" help:"Search files"`
	Upload   DriveUploadCmd   `cmd:"" help:"Upload a file"`
	Download DriveDownloadCmd `cmd:"" help:"Download a file"`
	Share    DriveShareCmd    `cmd:"" help:"Share a file"`
	Mkdir    DriveMkdirCmd    `cmd:"" help:"Create a folder"`
}

// DriveListCmd lists files.
type DriveListCmd struct {
	Folder string `help:"Folder ID to list" short:"d"`
	Limit  int64  `help:"Max files to return" default:"20" short:"n"`
}

func (c *DriveListCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "drive.list"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"drive"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "drive.list", "folder": c.Folder})
		return nil
	}

	drvSvc := api.NewDriveService(rctx.APIClient)
	files, err := drvSvc.ListFiles(rctx.Context, c.Folder, c.Limit)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"files": files,
		"count": len(files),
	})
	return nil
}

// DriveSearchCmd searches files.
type DriveSearchCmd struct {
	Query string `arg:"" help:"Drive search query (e.g. name contains 'report')"`
	Limit int64  `help:"Max results" default:"20" short:"n"`
}

func (c *DriveSearchCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "drive.search"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"drive"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "drive.search", "query": c.Query})
		return nil
	}

	drvSvc := api.NewDriveService(rctx.APIClient)
	files, err := drvSvc.SearchFiles(rctx.Context, c.Query, c.Limit)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"query": c.Query,
		"files": files,
		"count": len(files),
	})
	return nil
}

// DriveUploadCmd uploads a file.
type DriveUploadCmd struct {
	File   string `arg:"" help:"Local file path to upload"`
	Folder string `help:"Destination folder ID" short:"d"`
	Name   string `help:"Override file name"`
}

func (c *DriveUploadCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "drive.upload"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"drive"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run": "drive.upload",
			"file":    c.File,
			"folder":  c.Folder,
			"name":    c.Name,
		})
		return nil
	}

	drvSvc := api.NewDriveService(rctx.APIClient)
	result, err := drvSvc.UploadFile(rctx.Context, c.File, c.Folder, c.Name)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"uploaded": true,
		"file":     result,
	})
	return nil
}

// DriveDownloadCmd downloads a file.
type DriveDownloadCmd struct {
	FileID string `arg:"" help:"File ID to download"`
	Output string `help:"Output path" short:"o"`
}

func (c *DriveDownloadCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "drive.download"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"drive"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "drive.download", "file_id": c.FileID})
		return nil
	}

	drvSvc := api.NewDriveService(rctx.APIClient)
	outputPath, err := drvSvc.DownloadFile(rctx.Context, c.FileID, c.Output)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"downloaded": true,
		"path":       outputPath,
	})
	return nil
}

// DriveShareCmd shares a file.
type DriveShareCmd struct {
	FileID string `arg:"" help:"File ID to share"`
	Email  string `help:"Email to share with" required:""`
	Role   string `help:"Permission role: reader, writer, commenter" default:"reader" enum:"reader,writer,commenter"`
}

func (c *DriveShareCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "drive.share"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"drive"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run": "drive.share",
			"file_id": c.FileID,
			"email":   c.Email,
			"role":    c.Role,
		})
		return nil
	}

	drvSvc := api.NewDriveService(rctx.APIClient)
	if err := drvSvc.ShareFile(rctx.Context, c.FileID, c.Email, c.Role); err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"shared":  true,
		"file_id": c.FileID,
		"email":   c.Email,
		"role":    c.Role,
	})
	return nil
}

// DriveMkdirCmd creates a folder.
type DriveMkdirCmd struct {
	Name   string `arg:"" help:"Folder name"`
	Parent string `help:"Parent folder ID" short:"p"`
}

func (c *DriveMkdirCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "drive.mkdir"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"drive"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run": "drive.mkdir",
			"name":    c.Name,
			"parent":  c.Parent,
		})
		return nil
	}

	drvSvc := api.NewDriveService(rctx.APIClient)
	folder, err := drvSvc.CreateFolder(rctx.Context, c.Name, c.Parent)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"created": true,
		"folder":  folder,
	})
	return nil
}

