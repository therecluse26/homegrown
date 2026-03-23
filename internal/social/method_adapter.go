package social

import "context"

// methodAdapter implements MethodServiceForSocial using raw functions.
// The adapter pattern allows social:: to consume method:: without importing
// the method package directly. Wired in cmd/server/main.go. [ARCH §4.2]
type methodAdapter struct {
	getMethodologyDisplayName func(ctx context.Context, slug string) (string, error)
}

func (a *methodAdapter) GetMethodologyDisplayName(ctx context.Context, slug string) (string, error) {
	return a.getMethodologyDisplayName(ctx, slug)
}

// NewMethodAdapter creates a MethodServiceForSocial adapter from raw functions.
func NewMethodAdapter(
	getMethodologyDisplayName func(ctx context.Context, slug string) (string, error),
) MethodServiceForSocial {
	return &methodAdapter{
		getMethodologyDisplayName: getMethodologyDisplayName,
	}
}
