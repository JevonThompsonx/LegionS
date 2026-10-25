package state

import (
	"encoding/json"
	"os"
	"time"

	"gamepack/internal/cfg"
)

// Journal records pack/unpack steps so a run can be resumed idempotently and
// interrupted safely over flaky SSH.
type Journal struct {
	Path    string            `json:"-"`
	Updated map[string]string `json:"updated"` // slug -> "packed"|"unpacked"|"failed: reason"
	Steps   []JournalStep     `json:"steps"`
}

type JournalStep struct {
	Slug      string `json:"slug"`
	Action    string `json:"action"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
}

func Load(c *cfg.Config) *Journal {
	j := &Journal{Path: c.JournalPath(), Updated: map[string]string{}}
	data, err := os.ReadFile(c.JournalPath())
	if err != nil {
		return j
	}
	_ = json.Unmarshal(data, j)
	if j.Updated == nil {
		j.Updated = map[string]string{}
	}
	return j
}

func (j *Journal) Save() error {
	if j.Path == "" {
		return nil
	}
	if err := os.MkdirAll(dirOf(j.Path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(j.Path, data, 0o644)
}

func (j *Journal) Add(slug, action, status, message string) {
	j.Updated[slug] = status
	j.Steps = append(j.Steps, JournalStep{
		Slug:      slug,
		Action:    action,
		Status:    status,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	_ = j.Save()
}

func dirOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}
