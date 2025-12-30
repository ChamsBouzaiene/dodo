package prompts

import (
	"fmt"
	"sync"
)

// PromptRegistry manages versioned prompts.
type PromptRegistry struct {
	mu      sync.RWMutex
	prompts map[string]map[PromptVersion]*Prompt // ID -> Version -> Prompt
}

var defaultRegistry *PromptRegistry
var defaultRegistryOnce sync.Once

// DefaultRegistry returns the default global prompt registry.
func DefaultRegistry() *PromptRegistry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewPromptRegistry()
	})
	return defaultRegistry
}

// NewPromptRegistry creates a new prompt registry.
func NewPromptRegistry() *PromptRegistry {
	return &PromptRegistry{
		prompts: make(map[string]map[PromptVersion]*Prompt),
	}
}

// Register registers a prompt in the registry.
func (r *PromptRegistry) Register(p *Prompt) {
	if p == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.prompts[p.ID] == nil {
		r.prompts[p.ID] = make(map[PromptVersion]*Prompt)
	}
	r.prompts[p.ID][p.Version] = p
}

// Get retrieves a specific version of a prompt.
func (r *PromptRegistry) Get(id string, version PromptVersion) (*Prompt, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	versions, ok := r.prompts[id]
	if !ok {
		return nil, fmt.Errorf("prompt not found: %s", id)
	}

	prompt, ok := versions[version]
	if !ok {
		return nil, fmt.Errorf("prompt %s version %s not found", id, version)
	}

	return prompt, nil
}

// GetLatest retrieves the latest (non-deprecated) version of a prompt.
// If all versions are deprecated, returns the most recent version.
func (r *PromptRegistry) GetLatest(id string) (*Prompt, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	versions, ok := r.prompts[id]
	if !ok {
		return nil, fmt.Errorf("prompt not found: %s", id)
	}

	// Find latest non-deprecated version
	var latest *Prompt
	var latestVersion PromptVersion

	for version, prompt := range versions {
		if !prompt.Deprecated {
			if latest == nil || version > latestVersion {
				latest = prompt
				latestVersion = version
			}
		}
	}

	// If all are deprecated, return the most recent deprecated version
	if latest == nil {
		for version, prompt := range versions {
			if latest == nil || version > latestVersion {
				latest = prompt
				latestVersion = version
			}
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no versions found for prompt: %s", id)
	}

	return latest, nil
}

// List returns all prompt IDs in the registry.
func (r *PromptRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.prompts))
	for id := range r.prompts {
		ids = append(ids, id)
	}
	return ids
}

// Versions returns all versions for a given prompt ID.
func (r *PromptRegistry) Versions(id string) []PromptVersion {
	r.mu.RLock()
	defer r.mu.RUnlock()

	versions, ok := r.prompts[id]
	if !ok {
		return nil
	}

	result := make([]PromptVersion, 0, len(versions))
	for version := range versions {
		result = append(result, version)
	}
	return result
}
