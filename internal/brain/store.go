package brain

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	appconfig "github.com/sanja/octocli_cg/internal/config"
)

const (
	brainDirName         = "brain"
	metadataFileName     = ".metadata.json"
	taskFileName         = "task.md"
	implementationPlanMD = "implementation_plan.md"
)

type Metadata struct {
	ID                 string    `json:"id"`
	Goal               string    `json:"goal"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	TaskFile           string    `json:"task_file"`
	ImplementationPlan string    `json:"implementation_plan_file"`
	Checklist          []string  `json:"checklist"`
	Completed          []bool    `json:"completed"`
}

type Task struct {
	Metadata           Metadata
	TaskMarkdown       string
	ImplementationPlan string
	Dir                string
}

type Store struct {
	WorkspaceRoot string
}

func (s Store) Create(goal string, checklist []string, implementationPlan string) (*Task, error) {
	if strings.TrimSpace(goal) == "" {
		return nil, fmt.Errorf("goal is required")
	}
	id, err := newID()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(s.brainRoot(), id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create task directory: %w", err)
	}

	now := time.Now().UTC()
	meta := Metadata{
		ID:                 id,
		Goal:               goal,
		Status:             "in_progress",
		CreatedAt:          now,
		UpdatedAt:          now,
		TaskFile:           taskFileName,
		ImplementationPlan: implementationPlanMD,
		Checklist:          checklist,
		Completed:          make([]bool, len(checklist)),
	}

	taskMarkdown := renderChecklist(goal, checklist, meta.Completed)
	if err := os.WriteFile(filepath.Join(dir, taskFileName), []byte(taskMarkdown), 0o644); err != nil {
		return nil, fmt.Errorf("write task file: %w", err)
	}
	if strings.TrimSpace(implementationPlan) == "" {
		implementationPlan = "# Implementation Plan\n\nTBD\n"
	}
	if err := os.WriteFile(filepath.Join(dir, implementationPlanMD), []byte(implementationPlan), 0o644); err != nil {
		return nil, fmt.Errorf("write implementation plan: %w", err)
	}
	if err := writeMetadata(filepath.Join(dir, metadataFileName), meta); err != nil {
		return nil, err
	}

	return &Task{Metadata: meta, TaskMarkdown: taskMarkdown, ImplementationPlan: implementationPlan, Dir: dir}, nil
}

func (s Store) List() ([]Metadata, error) {
	if err := os.MkdirAll(s.brainRoot(), 0o755); err != nil {
		return nil, fmt.Errorf("ensure brain root: %w", err)
	}
	entries, err := os.ReadDir(s.brainRoot())
	if err != nil {
		return nil, fmt.Errorf("read brain root: %w", err)
	}
	var items []Metadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		meta, err := readMetadata(filepath.Join(s.brainRoot(), entry.Name(), metadataFileName))
		if err != nil {
			return nil, err
		}
		items = append(items, meta)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s Store) Get(id string) (*Task, error) {
	dir := filepath.Join(s.brainRoot(), id)
	meta, err := readMetadata(filepath.Join(dir, metadataFileName))
	if err != nil {
		return nil, err
	}
	taskBytes, err := os.ReadFile(filepath.Join(dir, meta.TaskFile))
	if err != nil {
		return nil, fmt.Errorf("read task file: %w", err)
	}
	planBytes, err := os.ReadFile(filepath.Join(dir, meta.ImplementationPlan))
	if err != nil {
		return nil, fmt.Errorf("read implementation plan: %w", err)
	}
	return &Task{Metadata: meta, TaskMarkdown: string(taskBytes), ImplementationPlan: string(planBytes), Dir: dir}, nil
}

func (s Store) SetChecklistItem(id string, index int, completed bool) (*Task, error) {
	task, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if index < 0 || index >= len(task.Metadata.Completed) {
		return nil, fmt.Errorf("checklist index %d out of range", index)
	}
	task.Metadata.Completed[index] = completed
	task.Metadata.UpdatedAt = time.Now().UTC()
	task.Metadata.Status = deriveStatus(task.Metadata.Completed)
	task.TaskMarkdown = renderChecklist(task.Metadata.Goal, task.Metadata.Checklist, task.Metadata.Completed)

	if err := os.WriteFile(filepath.Join(task.Dir, task.Metadata.TaskFile), []byte(task.TaskMarkdown), 0o644); err != nil {
		return nil, fmt.Errorf("write task file: %w", err)
	}
	if err := writeMetadata(filepath.Join(task.Dir, metadataFileName), task.Metadata); err != nil {
		return nil, err
	}
	return task, nil
}

func (s Store) brainRoot() string {
	return filepath.Join(s.WorkspaceRoot, ".agents", brainDirName)
}

func renderChecklist(goal string, items []string, completed []bool) string {
	var b strings.Builder
	b.WriteString("# Task\n\n")
	b.WriteString(goal)
	b.WriteString("\n\n## Checklist\n")
	for i, item := range items {
		check := " "
		if i < len(completed) && completed[i] {
			check = "x"
		}
		fmt.Fprintf(&b, "- [%s] %s\n", check, item)
	}
	return b.String()
}

func deriveStatus(completed []bool) string {
	if len(completed) == 0 {
		return "in_progress"
	}
	for _, item := range completed {
		if !item {
			return "in_progress"
		}
	}
	return "completed"
}

func writeMetadata(path string, meta Metadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	return nil
}

func readMetadata(path string) (Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Metadata{}, fmt.Errorf("read metadata %s: %w", path, err)
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return Metadata{}, fmt.Errorf("parse metadata %s: %w", path, err)
	}
	return meta, nil
}

func newID() (string, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate task id: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}

func DefaultGlobalBrainRoot() (string, error) {
	appHome, err := appconfig.AppHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appHome, brainDirName), nil
}
