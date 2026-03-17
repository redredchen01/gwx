package cmd

import (
	"github.com/user/gwx/internal/api"
	"github.com/user/gwx/internal/exitcode"
)

// TasksCmd groups Tasks operations.
type TasksCmd struct {
	List     TasksListCmd     `cmd:"" help:"List tasks"`
	Lists    TasksListsCmd    `cmd:"" help:"List task lists"`
	Create   TasksCreateCmd   `cmd:"" help:"Create a task"`
	Complete TasksCompleteCmd `cmd:"" help:"Mark a task as completed"`
	Delete   TasksDeleteCmd   `cmd:"" help:"Delete a task"`
}

// TasksListCmd lists tasks.
type TasksListCmd struct {
	ListID        string `help:"Task list ID (default: primary)" name:"list"`
	ShowCompleted bool   `help:"Include completed tasks" name:"show-completed"`
}

func (c *TasksListCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "tasks.list"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"tasks"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "tasks.list"})
		return nil
	}

	tasksSvc := api.NewTasksService(rctx.APIClient)
	items, err := tasksSvc.ListTasks(rctx.Context, c.ListID, c.ShowCompleted)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"tasks": items,
		"count": len(items),
	})
	return nil
}

// TasksListsCmd lists task lists.
type TasksListsCmd struct{}

func (c *TasksListsCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "tasks.lists"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"tasks"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	tasksSvc := api.NewTasksService(rctx.APIClient)
	lists, err := tasksSvc.ListTaskLists(rctx.Context)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"task_lists": lists,
		"count":      len(lists),
	})
	return nil
}

// TasksCreateCmd creates a task.
type TasksCreateCmd struct {
	Title  string `help:"Task title" required:""`
	Notes  string `help:"Task notes" short:"n"`
	Due    string `help:"Due date (YYYY-MM-DD)" short:"d"`
	ListID string `help:"Task list ID" name:"list"`
}

func (c *TasksCreateCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "tasks.create"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"tasks"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "tasks.create", "title": c.Title})
		return nil
	}

	tasksSvc := api.NewTasksService(rctx.APIClient)
	item, err := tasksSvc.CreateTask(rctx.Context, c.ListID, c.Title, c.Notes, c.Due)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"created": true,
		"task":    item,
	})
	return nil
}

// TasksCompleteCmd completes a task.
type TasksCompleteCmd struct {
	TaskID string `arg:"" help:"Task ID to complete"`
	ListID string `help:"Task list ID" name:"list"`
}

func (c *TasksCompleteCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "tasks.complete"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"tasks"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "tasks.complete", "task_id": c.TaskID})
		return nil
	}

	tasksSvc := api.NewTasksService(rctx.APIClient)
	item, err := tasksSvc.CompleteTask(rctx.Context, c.ListID, c.TaskID)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"completed": true,
		"task":      item,
	})
	return nil
}

// TasksDeleteCmd deletes a task.
type TasksDeleteCmd struct {
	TaskID string `arg:"" help:"Task ID to delete"`
	ListID string `help:"Task list ID" name:"list"`
}

func (c *TasksDeleteCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "tasks.delete"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"tasks"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "tasks.delete", "task_id": c.TaskID})
		return nil
	}

	tasksSvc := api.NewTasksService(rctx.APIClient)
	if err := tasksSvc.DeleteTask(rctx.Context, c.ListID, c.TaskID); err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"deleted": true,
		"task_id": c.TaskID,
	})
	return nil
}
