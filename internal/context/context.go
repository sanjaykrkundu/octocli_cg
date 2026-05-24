package context

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	appconfig "github.com/sanja/octocli_cg/internal/config"
)

type Workflow struct {
	Name    string
	Source  string
	Path    string
	Content string
}

type Bundle struct {
	Rules     string
	Workflows map[string]Workflow
}

func Load(workspaceRoot string) (Bundle, error) {
	globalRulesDir, err := appconfig.GlobalRulesDir()
	if err != nil {
		return Bundle{}, err
	}
	globalWorkflowsDir, err := appconfig.GlobalWorkflowsDir()
	if err != nil {
		return Bundle{}, err
	}

	rules, err := loadRules(globalRulesDir, appconfig.WorkspaceRulesDir(workspaceRoot))
	if err != nil {
		return Bundle{}, err
	}
	workflows, err := loadWorkflows(globalWorkflowsDir, appconfig.WorkspaceWorkflowsDir(workspaceRoot))
	if err != nil {
		return Bundle{}, err
	}

	return Bundle{Rules: rules, Workflows: workflows}, nil
}

func loadRules(dirs ...string) (string, error) {
	var sections []string
	for _, dir := range dirs {
		files, err := markdownFiles(dir)
		if err != nil {
			return "", err
		}
		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				return "", fmt.Errorf("read rules file %s: %w", file, err)
			}
			sections = append(sections, fmt.Sprintf("## Rules from %s\n%s", file, strings.TrimSpace(string(data))))
		}
	}
	return strings.TrimSpace(strings.Join(sections, "\n\n")), nil
}

func loadWorkflows(dirs ...string) (map[string]Workflow, error) {
	workflows := map[string]Workflow{}
	for idx, dir := range dirs {
		files, err := markdownFiles(dir)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				return nil, fmt.Errorf("read workflow file %s: %w", file, err)
			}
			name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
			source := "global"
			if idx > 0 {
				source = "workspace"
			}
			workflows[name] = Workflow{
				Name:    name,
				Source:  source,
				Path:    file,
				Content: strings.TrimSpace(string(data)),
			}
		}
	}
	return workflows, nil
}

func markdownFiles(dir string) ([]string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("inspect directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", dir, err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".md" && ext != ".markdown" {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}
