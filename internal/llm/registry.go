// Package llm is the process-wide registry that maps model sets and roles to models and providers.
package llm

import "fmt"

// Role names the job a model is used for. The active ModelSet maps each Role to a model.
type Role string

const (
	// RoleChat is the model used to answer chat requests.
	RoleChat Role = "chat"
)

// ModelSet names a fixed group of models, one per Role.
type ModelSet string

const (
	// SetLocal uses gemma4, served by Ollama.
	SetLocal ModelSet = "local"
	// SetCloud uses gemini-3.6-flash, served by Gemini.
	SetCloud ModelSet = "cloud"
)

// ProviderID names an LLM backend.
type ProviderID string

const (
	// ProviderOllama is an Ollama server.
	ProviderOllama ProviderID = "ollama"
	// ProviderGemini is Google's Gemini API.
	ProviderGemini ProviderID = "gemini"
)

var modelsBySetRole = map[ModelSet]map[Role]string{
	SetLocal: {
		RoleChat: "gemma4",
	},
	SetCloud: {
		RoleChat: "gemini-3.6-flash",
	},
}

var providerByModel = map[string]ProviderID{
	"gemma4":                 ProviderOllama,
	"gemini-3.5-flash":       ProviderGemini,
	"gemini-3.5-flash-lite":  ProviderGemini,
	"gemini-3.6-flash":       ProviderGemini,
	"gemini-3.1-pro-preview": ProviderGemini,
}

var credentialEnv = map[ProviderID]string{
	ProviderOllama: "",
	ProviderGemini: "GEMINI_API_KEY",
}

// ParseModelSet returns the ModelSet named s, or an error if s is not "local" or "cloud".
func ParseModelSet(s string) (ModelSet, error) {
	switch ModelSet(s) {
	case SetLocal:
		return SetLocal, nil
	case SetCloud:
		return SetCloud, nil
	default:
		return "", fmt.Errorf("llm: unknown model set %q (want %q or %q)", s, SetLocal, SetCloud)
	}
}

// ModelFor returns the model that set s uses for role r, or an error if s is unknown or has no
// model for r.
func ModelFor(s ModelSet, r Role) (string, error) {
	roles, ok := modelsBySetRole[s]
	if !ok {
		return "", fmt.Errorf("llm: unknown model set %q", s)
	}
	name, ok := roles[r]
	if !ok {
		return "", fmt.Errorf("llm: model set %q has no model for role %q", s, r)
	}
	return name, nil
}

// ProviderFor returns the provider that serves model, or an error if model is not in the built-in
// table: gemma4 (Ollama), and gemini-3.5-flash, gemini-3.5-flash-lite, gemini-3.6-flash and
// gemini-3.1-pro-preview (Gemini).
func ProviderFor(model string) (ProviderID, error) {
	p, ok := providerByModel[model]
	if !ok {
		return "", fmt.Errorf("llm: no provider registered for model %q", model)
	}
	return p, nil
}

// CredentialEnv returns the name of the environment variable holding p's API key: GEMINI_API_KEY
// for Gemini, and "" for Ollama or an unknown provider.
func CredentialEnv(p ProviderID) string {
	return credentialEnv[p]
}

// Roles returns every Role. RoleChat is the only one.
func Roles() []Role {
	return []Role{RoleChat}
}
