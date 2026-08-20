// Package ordering is the single code path that reads and writes the
// `ordering` list in alaws.toml, and creates the book/chapter/section
// source files that participate in it. Both the CLI (`alaws books`/
// `chapter`/`section`, PLAN1 §32) and the future drag-and-drop UI (PLAN1
// §29) call this package rather than editing TOML themselves, so there is
// exactly one place that mutates ordering (PLAN1 §30, §52).
package ordering

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"

	"github.com/shrsv/AgentLaws/internal/parser"
)

// Placement describes where a new or moved entry should be inserted
// relative to the existing ordering, checked in this order:
//
//  1. Position > 0: the entry is placed at that 1-based absolute position.
//  2. Before is set: the entry is placed immediately before Before's node
//     (ahead of its entire subtree, if it has one).
//  3. After is set: the entry is placed immediately after After's entire
//     subtree (so adding a section under a chapter appends it as that
//     chapter's last child by default).
//  4. None set: the entry is appended at the end.
type Placement struct {
	After    string
	Before   string
	Position int
}

// Node is one entry in the ordering, resolved to its level and derived
// parent, per the outline rule in docs/PLAN1.md §32.
type Node struct {
	Path     string
	ID       string
	Level    int
	ParentID string // "" for a top-level chapter
}

type tomlConfig struct {
	Title           string   `toml:"title"`
	Ordering        []string `toml:"ordering"`
	PromptTemplates []string `toml:"promptTemplates"`
}

func loadConfig(configPath string) (tomlConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return tomlConfig{}, err
	}
	var cfg tomlConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return tomlConfig{}, fmt.Errorf("invalid-metadata: %s: %w", configPath, err)
	}
	return cfg, nil
}

func saveConfig(configPath string, cfg tomlConfig) error {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

// treeFromOrdering resolves each entry in ordering (relative to dir) to a
// Node, deriving Level from frontmatter (default 1) and ParentID from the
// outline rule: the nearest preceding entry with a lower level.
func treeFromOrdering(dir string, entries []string) ([]Node, error) {
	nodes := make([]Node, 0, len(entries))
	levels := make([]int, 0, len(entries))

	for _, entry := range entries {
		ps, err := parser.ParseSection(filepath.Join(dir, entry))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entry, err)
		}
		level := parser.ResolveLevel(entry, ps.Level)
		nodes = append(nodes, Node{Path: entry, ID: ps.ID, Level: level})
		levels = append(levels, level)
	}

	for i := range nodes {
		if p := parentIndex(levels, i); p >= 0 {
			nodes[i].ParentID = nodes[p].ID
		}
	}
	return nodes, nil
}

func parentIndex(levels []int, i int) int {
	for j := i - 1; j >= 0; j-- {
		if levels[j] < levels[i] {
			return j
		}
	}
	return -1
}

// Tree computes the chapter/section parent-child structure implied by a
// flat ordering list plus each entry's Level.
func Tree(configPath string) ([]Node, error) {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}
	return treeFromOrdering(filepath.Dir(configPath), cfg.Ordering)
}

// lastDescendantIndex returns the index of the last entry belonging to
// nodes[i]'s subtree (nodes[i] itself if it has no children with a higher
// level immediately following it).
func lastDescendantIndex(nodes []Node, i int) int {
	end := i
	for j := i + 1; j < len(nodes) && nodes[j].Level > nodes[i].Level; j++ {
		end = j
	}
	return end
}

func indexOfID(nodes []Node, id string) int {
	for i, n := range nodes {
		if n.ID == id {
			return i
		}
	}
	return -1
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// resolveInsertIndex computes the 0-based index in ordering at which a new
// entry should be inserted, given placement and the already-resolved nodes
// for that ordering.
func resolveInsertIndex(nodes []Node, placement Placement) (int, error) {
	if placement.Position > 0 {
		return clamp(placement.Position-1, 0, len(nodes)), nil
	}
	if placement.Before != "" {
		target := indexOfID(nodes, placement.Before)
		if target == -1 {
			return 0, fmt.Errorf("ordering: %q not found", placement.Before)
		}
		return target, nil
	}
	if placement.After != "" {
		target := indexOfID(nodes, placement.After)
		if target == -1 {
			return 0, fmt.Errorf("ordering: %q not found", placement.After)
		}
		return lastDescendantIndex(nodes, target) + 1, nil
	}
	return len(nodes), nil
}

// Insert adds a new ordering entry at the position described by placement
// and rewrites alaws.toml in place.
func Insert(configPath string, entryPath string, placement Placement) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	nodes, err := treeFromOrdering(filepath.Dir(configPath), cfg.Ordering)
	if err != nil {
		return err
	}
	idx, err := resolveInsertIndex(nodes, placement)
	if err != nil {
		return err
	}

	ordering := make([]string, 0, len(cfg.Ordering)+1)
	ordering = append(ordering, cfg.Ordering[:idx]...)
	ordering = append(ordering, entryPath)
	ordering = append(ordering, cfg.Ordering[idx:]...)
	cfg.Ordering = ordering
	return saveConfig(configPath, cfg)
}

// Move relocates an existing ordering entry, along with its entire
// subtree (so moving a chapter takes its sections with it), to a new
// position and rewrites alaws.toml in place.
func Move(configPath string, entryID string, placement Placement) error {
	dir := filepath.Dir(configPath)
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	nodes, err := treeFromOrdering(dir, cfg.Ordering)
	if err != nil {
		return err
	}
	idx := indexOfID(nodes, entryID)
	if idx == -1 {
		return fmt.Errorf("ordering: %q not found", entryID)
	}
	end := lastDescendantIndex(nodes, idx)

	moving := append([]string{}, cfg.Ordering[idx:end+1]...)
	remaining := make([]string, 0, len(cfg.Ordering)-len(moving))
	remaining = append(remaining, cfg.Ordering[:idx]...)
	remaining = append(remaining, cfg.Ordering[end+1:]...)

	remNodes, err := treeFromOrdering(dir, remaining)
	if err != nil {
		return err
	}
	insertAt, err := resolveInsertIndex(remNodes, placement)
	if err != nil {
		return err
	}

	result := make([]string, 0, len(cfg.Ordering))
	result = append(result, remaining[:insertAt]...)
	result = append(result, moving...)
	result = append(result, remaining[insertAt:]...)
	cfg.Ordering = result
	return saveConfig(configPath, cfg)
}

// Remove deletes an ordering entry (and, for a chapter, its section
// descendants) from alaws.toml. It returns an error if the entry has
// descendants unless force is true. The underlying Markdown file(s) are
// left on disk, excluded from the lawbook (PLAN1 §21) rather than deleted.
func Remove(configPath string, entryID string, force bool) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	nodes, err := treeFromOrdering(filepath.Dir(configPath), cfg.Ordering)
	if err != nil {
		return err
	}
	idx := indexOfID(nodes, entryID)
	if idx == -1 {
		return fmt.Errorf("ordering: %q not found", entryID)
	}
	end := lastDescendantIndex(nodes, idx)
	if end > idx && !force {
		return fmt.Errorf("ordering: %q has %d descendant section(s); use --force to remove them too", entryID, end-idx)
	}

	ordering := make([]string, 0, len(cfg.Ordering)-(end-idx+1))
	ordering = append(ordering, cfg.Ordering[:idx]...)
	ordering = append(ordering, cfg.Ordering[end+1:]...)
	cfg.Ordering = ordering
	return saveConfig(configPath, cfg)
}

// SectionMeta is the frontmatter for a newly created chapter or section
// file (PLAN1 §6).
type SectionMeta struct {
	Title string
	ID    string
	Level int
}

// NewBook creates a new alaws.toml at path with the given title and an
// empty ordering, establishing a new lawbook cluster (PLAN1 §4).
func NewBook(path string, title string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	configPath := filepath.Join(path, "alaws.toml")
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("ordering: %s already exists", configPath)
	}
	return saveConfig(configPath, tomlConfig{Title: title, Ordering: []string{}, PromptTemplates: []string{}})
}

type sectionFrontmatter struct {
	Title string `yaml:"title"`
	ID    string `yaml:"id"`
	Level *int   `yaml:"level,omitempty"`
}

// NewSectionFile writes a new section Markdown file at path with meta's
// frontmatter and an empty commentary/laws skeleton (PLAN1 §6), ready to be
// added to a book's ordering via Insert.
func NewSectionFile(path string, meta SectionMeta) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("ordering: %s already exists", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	fm := sectionFrontmatter{Title: meta.Title, ID: meta.ID}
	if meta.Level > 0 {
		level := meta.Level
		fm.Level = &level
	}
	fmData, err := yaml.Marshal(fm)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.Write(fmData)
	b.WriteString("---\n\n")
	b.WriteString("<!-- alaws:commentary -->\n\n")
	b.WriteString(meta.Title)
	b.WriteString(".\n\n")
	b.WriteString("<!-- alaws:laws -->\n")

	return os.WriteFile(path, []byte(b.String()), 0644)
}

// SetLevel rewrites a section file's frontmatter to set (level > 0) or
// clear (level == 0) an explicit `level:` override, leaving its id,
// title, commentary, and laws untouched. This is used when a structural
// move changes a section's intended nesting depth without moving the
// underlying file, so the file's folder-depth default (parser.ResolveLevel)
// would otherwise go stale.
func SetLevel(path string, level int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return fmt.Errorf("ordering: %s: missing frontmatter", path)
	}
	fmEnd := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			fmEnd = i
			break
		}
	}
	if fmEnd == -1 {
		return fmt.Errorf("ordering: %s: unterminated frontmatter", path)
	}

	var fm sectionFrontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:fmEnd], "\n")), &fm); err != nil {
		return fmt.Errorf("ordering: %s: %w", path, err)
	}
	if level > 0 {
		fm.Level = &level
	} else {
		fm.Level = nil
	}

	fmData, err := yaml.Marshal(fm)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.Write(fmData)
	b.WriteString("---\n")
	b.WriteString(strings.Join(lines[fmEnd+1:], "\n"))

	return os.WriteFile(path, []byte(b.String()), 0644)
}

// --- Prompt template ordering (flat list, no hierarchy) ---

// InsertPrompt adds a new entry to the promptTemplates list in alaws.toml
// at the position described by placement (same semantics as Insert for
// ordering, but without subtree logic since prompts are flat).
func InsertPrompt(configPath string, entryPath string, placement Placement) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	idx := len(cfg.PromptTemplates) // default: append
	if placement.Position > 0 {
		idx = clamp(placement.Position-1, 0, len(cfg.PromptTemplates))
	} else if placement.After != "" {
		for i, p := range cfg.PromptTemplates {
			if p == placement.After {
				idx = i + 1
				break
			}
		}
	} else if placement.Before != "" {
		for i, p := range cfg.PromptTemplates {
			if p == placement.Before {
				idx = i
				break
			}
		}
	}

	pts := make([]string, 0, len(cfg.PromptTemplates)+1)
	pts = append(pts, cfg.PromptTemplates[:idx]...)
	pts = append(pts, entryPath)
	pts = append(pts, cfg.PromptTemplates[idx:]...)
	cfg.PromptTemplates = pts
	return saveConfig(configPath, cfg)
}

// RemovePrompt removes an entry from the promptTemplates list in alaws.toml
// by its file path. The underlying Markdown file is left on disk.
func RemovePrompt(configPath string, entryPath string) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	pts := make([]string, 0, len(cfg.PromptTemplates))
	found := false
	for _, p := range cfg.PromptTemplates {
		if p == entryPath {
			found = true
			continue
		}
		pts = append(pts, p)
	}
	if !found {
		return fmt.Errorf("ordering: prompt %q not found in promptTemplates", entryPath)
	}
	cfg.PromptTemplates = pts
	return saveConfig(configPath, cfg)
}

// MovePrompt relocates an entry in the promptTemplates list.
func MovePrompt(configPath string, entryPath string, placement Placement) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	// Find and remove
	found := false
	pts := make([]string, 0, len(cfg.PromptTemplates))
	for _, p := range cfg.PromptTemplates {
		if p == entryPath {
			found = true
			continue
		}
		pts = append(pts, p)
	}
	if !found {
		return fmt.Errorf("ordering: prompt %q not found in promptTemplates", entryPath)
	}

	idx := len(pts) // default: append
	if placement.Position > 0 {
		idx = clamp(placement.Position-1, 0, len(pts))
	} else if placement.After != "" {
		for i, p := range pts {
			if p == placement.After {
				idx = i + 1
				break
			}
		}
	} else if placement.Before != "" {
		for i, p := range pts {
			if p == placement.Before {
				idx = i
				break
			}
		}
	}

	result := make([]string, 0, len(cfg.PromptTemplates))
	result = append(result, pts[:idx]...)
	result = append(result, entryPath)
	result = append(result, pts[idx:]...)
	cfg.PromptTemplates = result
	return saveConfig(configPath, cfg)
}

// NewPromptFile writes a new prompt template Markdown file at path with
// the given frontmatter and an empty commentary/template skeleton.
func NewPromptFile(path string, title string, id string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("ordering: %s already exists", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	type promptFrontmatter struct {
		Title string `yaml:"title"`
		ID    string `yaml:"id"`
	}

	fmData, err := yaml.Marshal(promptFrontmatter{Title: title, ID: id})
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.Write(fmData)
	b.WriteString("---\n\n")
	b.WriteString("<!-- alaws:commentary -->\n\n")
	b.WriteString(title)
	b.WriteString(".\n\n")
	b.WriteString("<!-- alaws:promptTemplate -->\n")

	return os.WriteFile(path, []byte(b.String()), 0644)
}
