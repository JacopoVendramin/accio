// Package plugin provides plugin interfaces for extending accio.
package plugin

import (
	"github.com/jvendramin/accio/pkg/provider"
)

// Plugin represents a loadable plugin.
type Plugin interface {
	// Name returns the plugin name.
	Name() string

	// Version returns the plugin version.
	Version() string

	// Description returns a human-readable description.
	Description() string

	// Initialize initializes the plugin.
	Initialize() error

	// Shutdown shuts down the plugin.
	Shutdown() error
}

// ProviderPlugin is a plugin that provides a cloud provider.
type ProviderPlugin interface {
	Plugin

	// Provider returns the cloud provider implementation.
	Provider() provider.CloudProvider
}

// Registry manages loaded plugins.
type Registry struct {
	plugins   map[string]Plugin
	providers map[string]provider.CloudProvider
}

// NewRegistry creates a new plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins:   make(map[string]Plugin),
		providers: make(map[string]provider.CloudProvider),
	}
}

// Register registers a plugin.
func (r *Registry) Register(p Plugin) error {
	if err := p.Initialize(); err != nil {
		return err
	}

	r.plugins[p.Name()] = p

	// If it's a provider plugin, register the provider too
	if pp, ok := p.(ProviderPlugin); ok {
		provider := pp.Provider()
		r.providers[provider.Name()] = provider
	}

	return nil
}

// Unregister unregisters a plugin.
func (r *Registry) Unregister(name string) error {
	p, ok := r.plugins[name]
	if !ok {
		return nil
	}

	if err := p.Shutdown(); err != nil {
		return err
	}

	// If it's a provider plugin, unregister the provider too
	if pp, ok := p.(ProviderPlugin); ok {
		delete(r.providers, pp.Provider().Name())
	}

	delete(r.plugins, name)
	return nil
}

// GetPlugin returns a plugin by name.
func (r *Registry) GetPlugin(name string) (Plugin, bool) {
	p, ok := r.plugins[name]
	return p, ok
}

// GetProvider returns a provider by name.
func (r *Registry) GetProvider(name string) (provider.CloudProvider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// ListPlugins returns all registered plugins.
func (r *Registry) ListPlugins() []Plugin {
	plugins := make([]Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// ListProviders returns all registered providers.
func (r *Registry) ListProviders() []provider.CloudProvider {
	providers := make([]provider.CloudProvider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

// Shutdown shuts down all plugins.
func (r *Registry) Shutdown() error {
	var lastErr error
	for name := range r.plugins {
		if err := r.Unregister(name); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
