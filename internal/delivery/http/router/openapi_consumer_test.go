package router

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"able-rest-api/docs"
)

// This projects the contract into the inputs a tool generator would expose.
// Response schemas are walked separately so errors cannot become tool inputs.
func TestOpenAPIConsumerInputs(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromData(docs.OpenAPIYAML)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := map[string]map[string]bool{
		"getHealth":    {},
		"getReadiness": {},
		"listUsers":    {"query.limit": false, "query.offset": false},
		"createUser":   {"body.name": true, "body.email": true},
		"getUser":      {"path.id": true},
		"sendMail": {
			"body.to": true, "body.to[]": false,
			"body.cc": false, "body.cc[]": false,
			"body.bcc": false, "body.bcc[]": false,
			"body.subject": true, "body.body": true, "body.is_html": false,
			"body.attachments": false, "body.attachments[]": false,
			"body.attachments[].filename":       true,
			"body.attachments[].content_type":   false,
			"body.attachments[].content_base64": true,
		},
	}
	gotIDs := map[string]bool{}
	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			id := op.OperationID
			if gotIDs[id] {
				t.Errorf("duplicate tool name %q", id)
			}
			gotIDs[id] = true
			if _, ok := want[id]; !ok {
				t.Errorf("unexpected tool name %q", id)
			}
			inputs := map[string]bool{}
			for _, parameter := range append(append(openapi3.Parameters{}, item.Parameters...), op.Parameters...) {
				if parameter == nil || parameter.Value == nil {
					t.Fatalf("%s: unresolved parameter", id)
				}
				p := parameter.Value
				walkConsumerSchema(t, doc, p.Schema, p.In+"."+p.Name, inputs, p.Required, map[*openapi3.Schema]bool{})
			}
			if op.RequestBody != nil {
				if op.RequestBody.Value == nil || !op.RequestBody.Value.Required {
					t.Fatalf("%s: unresolved or optional request body", id)
				}
				media := op.RequestBody.Value.Content["application/json"]
				if media == nil {
					t.Fatalf("%s: missing JSON request body", id)
				}
				walkConsumerSchema(t, doc, media.Schema, "body", inputs, false, map[*openapi3.Schema]bool{})
			}
			if expected, ok := want[id]; ok && !reflect.DeepEqual(inputs, expected) {
				t.Errorf("%s %s (%s): tool inputs %v; want %v", method, path, id, inputs, expected)
			}
			for status, response := range op.Responses.Map() {
				if response == nil || response.Value == nil {
					t.Fatalf("%s response %s: unresolved response", id, status)
				}
				media := response.Value.Content["application/json"]
				if media == nil {
					t.Fatalf("%s response %s: missing JSON schema", id, status)
				}
				walkConsumerSchema(t, doc, media.Schema, id+" response "+status, nil, false, map[*openapi3.Schema]bool{})
			}
		}
	}
	if len(gotIDs) != len(want) {
		t.Errorf("got %d tool names, want %d: %v", len(gotIDs), len(want), gotIDs)
	}
	names := map[string]string{}
	for name, ref := range doc.Components.Schemas {
		folded := strings.ToLower(name)
		if previous, exists := names[folded]; exists {
			t.Errorf("SDK schema name collision: %s and %s", previous, name)
		}
		names[folded] = name
		walkConsumerSchema(t, doc, ref, "schema "+name, nil, false, map[*openapi3.Schema]bool{})
	}
}

func walkConsumerSchema(t *testing.T, doc *openapi3.T, ref *openapi3.SchemaRef, path string, fields map[string]bool, required bool, visiting map[*openapi3.Schema]bool) {
	t.Helper()
	if ref == nil || ref.Value == nil {
		t.Fatalf("%s: unresolved schema", path)
	}
	if ref.Ref != "" {
		name := strings.TrimPrefix(ref.Ref, "#/components/schemas/")
		if name == ref.Ref || doc.Components.Schemas[name] == nil {
			t.Fatalf("%s: unsupported schema reference %q", path, ref.Ref)
		}
	}
	s := ref.Value
	if visiting[s] {
		t.Fatalf("%s: circular schema reference", path)
	}
	visiting[s] = true
	defer delete(visiting, s)
	if fields != nil && path != "body" {
		fields[path] = required
	}
	if len(s.OneOf) != 0 || len(s.AnyOf) != 0 || s.Not != nil || s.AdditionalProperties.Schema != nil || (s.AdditionalProperties.Has != nil && *s.AdditionalProperties.Has) {
		t.Fatalf("%s: ambiguous or unsupported schema construct", path)
	}
	if len(s.AllOf) != 0 {
		if s.Type != nil || len(s.Properties) != 0 {
			t.Fatalf("%s: mixed allOf and inline properties", path)
		}
		for _, part := range s.AllOf {
			walkConsumerSchema(t, doc, part, path, fields, false, visiting)
		}
		return
	}
	switch {
	case s.Type.Is("object"):
		if len(s.Properties) == 0 {
			t.Fatalf("%s: free-form object", path)
		}
		for _, name := range s.Required {
			if s.Properties[name] == nil {
				t.Fatalf("%s: required property %q missing", path, name)
			}
		}
		for name, property := range s.Properties {
			walkConsumerSchema(t, doc, property, path+"."+name, fields, containsName(s.Required, name), visiting)
		}
	case s.Type.Is("array"):
		walkConsumerSchema(t, doc, s.Items, path+"[]", fields, false, visiting)
	case s.Type.Is("string"), s.Type.Is("integer"), s.Type.Is("number"), s.Type.Is("boolean"):
	default:
		t.Fatalf("%s: missing or unsupported schema type %v", path, s.Type)
	}
}

func containsName(names []string, name string) bool {
	for _, candidate := range names {
		if candidate == name {
			return true
		}
	}
	return false
}
