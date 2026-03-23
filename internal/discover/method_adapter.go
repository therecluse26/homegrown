package discover

import "context"

// methodAdapter implements MethodologyServiceForDiscover using a raw function.
// This adapter pattern allows discover:: to consume method:: without importing
// the method package directly, avoiding circular dependencies. [ARCH §4.2]
//
// Wired in cmd/server/main.go via NewMethodAdapter.
type methodAdapter struct {
	getDisplayName func(ctx context.Context, slug string) (string, error)
}

// NewMethodAdapter creates a MethodologyServiceForDiscover adapter from a raw function.
// The function is provided at wiring time in cmd/server/main.go.
//
// Example wiring:
//
//	discover.NewMethodAdapter(func(ctx context.Context, slug string) (string, error) {
//	    all, err := methodSvc.ListMethodologies(ctx)
//	    if err != nil { return slug, nil } // graceful fallback
//	    for _, m := range all {
//	        if string(m.Slug) == slug { return m.DisplayName, nil }
//	    }
//	    return slug, nil
//	})
func NewMethodAdapter(fn func(ctx context.Context, slug string) (string, error)) MethodologyServiceForDiscover {
	return &methodAdapter{getDisplayName: fn}
}

func (a *methodAdapter) GetMethodologyDisplayName(ctx context.Context, slug string) (string, error) {
	return a.getDisplayName(ctx, slug)
}
