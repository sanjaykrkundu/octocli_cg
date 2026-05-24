package async

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sanja/octocli_cg/internal/brain"
)

type Runner struct {
	Store brain.Store
}

func (r Runner) StartTask(goal string, checklist []string, plan string) (*brain.Task, error) {
	task, err := r.Store.Create(goal, checklist, plan)
	if err != nil {
		return nil, err
	}
	if err := r.StartExistingTask(task.Metadata.ID); err != nil {
		return nil, err
	}
	return task, nil
}

func (r Runner) StartExistingTask(taskID string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	exePath, err = r.ensureStableExecutable(exePath)
	if err != nil {
		return err
	}
	cmd := exec.Command(exePath, "task", "worker", taskID)
	cmd.Dir = r.Store.WorkspaceRoot
	logPath := filepath.Join(r.Store.WorkspaceRoot, ".agents", "brain", taskID, "worker.stdout.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open worker stdout log: %w", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("start background worker: %w", err)
	}
	go func() {
		_ = cmd.Wait()
		_ = logFile.Close()
	}()
	return nil
}

func (r Runner) ensureStableExecutable(exePath string) (string, error) {
	cleanPath := strings.ToLower(filepath.Clean(exePath))
	if !strings.Contains(cleanPath, filepath.Clean(strings.ToLower(os.TempDir()))) && !strings.Contains(cleanPath, "go-build") {
		return exePath, nil
	}

	binDir := filepath.Join(r.Store.WorkspaceRoot, ".agents", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", fmt.Errorf("create stable bin directory: %w", err)
	}
	stablePath := filepath.Join(binDir, "octocli_cg-worker.exe")
	if err := copyFile(exePath, stablePath); err != nil {
		return "", fmt.Errorf("copy stable worker executable: %w", err)
	}
	return stablePath, nil
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func (r Runner) RunTask(taskID string) {
	_, _ = r.Store.UpdateStatus(taskID, "running")
	_ = r.Store.AppendLog(taskID, fmt.Sprintf("[%s] task started\n", time.Now().Format(time.RFC3339)))

	task, err := r.Store.Get(taskID)
	if err != nil {
		_ = r.Store.AppendLog(taskID, fmt.Sprintf("[%s] failed to load task: %v\n", time.Now().Format(time.RFC3339), err))
		_, _ = r.Store.UpdateStatus(taskID, "failed")
		return
	}
	for i, item := range task.Metadata.Checklist {
		_ = r.Store.AppendLog(taskID, fmt.Sprintf("[%s] processing checklist item %d: %s\n", time.Now().Format(time.RFC3339), i, item))
		time.Sleep(700 * time.Millisecond)
		_, _ = r.Store.SetChecklistItem(taskID, i, true)
	}

	_ = r.Store.AppendLog(taskID, fmt.Sprintf("[%s] task completed\n", time.Now().Format(time.RFC3339)))
	_, _ = r.Store.UpdateStatus(taskID, "completed")
}
