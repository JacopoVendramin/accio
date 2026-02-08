package config

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"

	domainErrors "github.com/jvendramin/accio/internal/domain/errors"
	"github.com/jvendramin/accio/internal/domain/integration"
)

// IntegrationRepository implements integration.Repository using YAML file storage.
type IntegrationRepository struct {
	filePath string
	mu       sync.RWMutex
	cache    map[string]*integration.Integration
}

// integrationData represents the YAML structure for integrations.
type integrationData struct {
	Integrations []*integration.Integration `yaml:"integrations"`
}

// NewIntegrationRepository creates a new file-based integration repository.
func NewIntegrationRepository(filePath string) (*IntegrationRepository, error) {
	repo := &IntegrationRepository{
		filePath: filePath,
		cache:    make(map[string]*integration.Integration),
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
		return nil, err
	}

	// Load existing integrations
	if err := repo.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return repo, nil
}

// Save persists an integration.
func (r *IntegrationRepository) Save(ctx context.Context, integ *integration.Integration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cache[integ.ID] = integ
	return r.persist()
}

// Get retrieves an integration by ID.
func (r *IntegrationRepository) Get(ctx context.Context, id string) (*integration.Integration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	integ, ok := r.cache[id]
	if !ok {
		return nil, domainErrors.NewDomainError("IntegrationRepository.Get", domainErrors.ErrIntegrationNotFound, nil).
			WithContext("id", id)
	}

	return integ, nil
}

// List returns all integrations.
func (r *IntegrationRepository) List(ctx context.Context) ([]*integration.Integration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	integrations := make([]*integration.Integration, 0, len(r.cache))
	for _, integ := range r.cache {
		integrations = append(integrations, integ)
	}

	return integrations, nil
}

// ListByType returns integrations of a specific type.
func (r *IntegrationRepository) ListByType(ctx context.Context, integrationType integration.IntegrationType) ([]*integration.Integration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var integrations []*integration.Integration
	for _, integ := range r.cache {
		if integ.Type == integrationType {
			integrations = append(integrations, integ)
		}
	}

	return integrations, nil
}

// Delete removes an integration.
func (r *IntegrationRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.cache[id]; !ok {
		return domainErrors.NewDomainError("IntegrationRepository.Delete", domainErrors.ErrIntegrationNotFound, nil).
			WithContext("id", id)
	}

	delete(r.cache, id)
	return r.persist()
}

// load reads integrations from the file.
func (r *IntegrationRepository) load() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return err
	}

	var id integrationData
	if err := yaml.Unmarshal(data, &id); err != nil {
		return err
	}

	r.cache = make(map[string]*integration.Integration)
	for _, integ := range id.Integrations {
		r.cache[integ.ID] = integ
	}

	return nil
}

// persist writes integrations to the file.
func (r *IntegrationRepository) persist() error {
	integrations := make([]*integration.Integration, 0, len(r.cache))
	for _, integ := range r.cache {
		integrations = append(integrations, integ)
	}

	id := integrationData{Integrations: integrations}
	data, err := yaml.Marshal(&id)
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, data, 0600)
}

// GetByStartURL returns an integration by SSO start URL.
func (r *IntegrationRepository) GetByStartURL(ctx context.Context, startURL string) (*integration.Integration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, integ := range r.cache {
		if integ.Type == integration.IntegrationTypeAWSSSO &&
			integ.Config.AWSSSO != nil &&
			integ.Config.AWSSSO.StartURL == startURL {
			return integ, nil
		}
	}

	return nil, domainErrors.NewDomainError("IntegrationRepository.GetByStartURL", domainErrors.ErrIntegrationNotFound, nil).
		WithContext("startURL", startURL)
}
