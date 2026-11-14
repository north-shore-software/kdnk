package agentsmith

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/pocketbase/pocketbase/core"

	"github.com/north-shore-software/agent-smith/providers"
)

// Tool adds a custom tool named name, described to the model by description, to r. When the model
// calls it, its arguments are decoded into an In and passed to handler, and handler's result is
// returned to the model as JSON. Arguments that cannot be decoded, and an error from handler, are
// returned to the model as a tool error.
//
// The tool's JSON Schema comes from In, which must be a struct; struct{} takes no arguments. Each
// exported field is an argument named by its json tag, or by the Go field name if the tag has no
// name; fields tagged json:"-" are skipped. A jsonschema tag gives the argument's description.
// Every argument is required unless its json tag has omitempty. Supported field types are
// strings, bools, integers, floats and time.Time, including named types with those underlying
// kinds. Embedded fields, other types, and two fields with the same name make Run return an
// error.
func Tool[In any](r *Registry, name, description string, handler func(ctx context.Context, app core.App, s Session, in In) (any, error)) {
	fields, err := structFields(reflect.TypeFor[In](), nil)
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("agentsmith: tool %q: %w", name, err))
	}
	props, _ := fieldProperties(fields)
	var required []string
	for _, f := range fields {
		if !f.omitEmpty {
			required = append(required, f.name)
		}
	}
	r.tools = append(r.tools, tool{
		def: providers.Tool{Name: name, Description: description, Parameters: objectSchema(props, required)},
		run: func(ctx context.Context, app core.App, s Session, args json.RawMessage) (any, error) {
			var in In
			if err := decodeArgs(args, &in); err != nil {
				return nil, err
			}
			return handler(ctx, app, s, in)
		},
	})
}
