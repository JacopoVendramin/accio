package integration

import "context"

// Repository defines the interface for integration persistence.
type Repository interface {
	// Save persists an integration.
	Save(ctx context.Context, integration *Integration) error

	// Get retrieves an integration by ID.
	Get(ctx context.Context, id string) (*Integration, error)

	// List returns all integrations.
	List(ctx context.Context) ([]*Integration, error)

	// ListByType returns integrations of a specific type.
	ListByType(ctx context.Context, integrationType IntegrationType) ([]*Integration, error)

	// Delete removes an integration.
	Delete(ctx context.Context, id string) error
}
