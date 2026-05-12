package platform

// Registry holds all registered adapters.
type Registry struct {
	adapters map[string]PlatformAdapter
}

// NewRegistry builds a registry from the provided adapters.
func NewRegistry(adapters ...PlatformAdapter) Registry {
	m := make(map[string]PlatformAdapter, len(adapters))
	for _, adapter := range adapters {
		m[adapter.Platform()] = adapter
	}
	return Registry{adapters: m}
}

// Get returns the adapter for the given platform.
func (r Registry) Get(platform string) (PlatformAdapter, bool) {
	adapter, ok := r.adapters[platform]
	return adapter, ok
}

// Platforms returns registered platform names.
func (r Registry) Platforms() []string {
	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}
