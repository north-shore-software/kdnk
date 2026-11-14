package llm

import (
	"context"

	"github.com/north-shore-software/agent-smith/providers"
)

var (
	providerFactory func(model string) providers.Provider
	activeSet       = SetLocal
)

// SetProviderFactory registers f, which SelectedProvider calls to build a providers.Provider for
// a model name. It replaces any earlier factory and is not safe to call concurrently with
// SelectedProvider.
func SetProviderFactory(f func(model string) providers.Provider) {
	providerFactory = f
}

// Ready reports whether a provider factory has been registered.
func Ready() bool {
	return providerFactory != nil
}

// SetActiveModelSet sets the process-wide ModelSet used by ResolveRole. The default is SetLocal.
func SetActiveModelSet(s ModelSet) {
	activeSet = s
}

// ActiveModelSet returns the ModelSet used by ResolveRole.
func ActiveModelSet() ModelSet {
	return activeSet
}

// ResolveRole returns the model the active ModelSet uses for r.
func ResolveRole(r Role) (string, error) {
	return ModelFor(activeSet, r)
}

// ErrorProvider returns a providers.Provider whose Stream always returns err. A provider factory
// can return it for a model it cannot serve.
func ErrorProvider(err error) providers.Provider {
	return errProvider{err: err}
}

type errProvider struct{ err error }

func (p errProvider) Stream(context.Context, []providers.Message, []providers.Tool) (*providers.Completion, error) {
	return nil, p.err
}

// SelectedProvider returns the providers.Provider the registered factory builds for model. It
// panics if no factory is registered.
func SelectedProvider(model string) providers.Provider {
	if providerFactory == nil {
		panic("llm: no provider factory registered; call llm.SetProviderFactory at startup")
	}
	return providerFactory(model)
}
