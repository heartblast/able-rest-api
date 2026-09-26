package router

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"

	"able-rest-api/docs"
	"able-rest-api/internal/delivery/http/dto"
	"able-rest-api/internal/infra/config"
	"able-rest-api/internal/infra/persistence"
)

type quietLogger struct{}

func (quietLogger) Info(string, ...any)  {}
func (quietLogger) Error(string, ...any) {}

func TestOpenAPIContractAndRoutes(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromData(docs.OpenAPIYAML)
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatalf("invalid OpenAPI contract: %v", err)
	}
	if !strings.HasPrefix(doc.OpenAPI, "3.") {
		t.Fatalf("expected OpenAPI 3.x, got %q", doc.OpenAPI)
	}

	r := New(&config.Config{Swagger: config.SwaggerConfig{Enabled: true}}, quietLogger{}, nil, &persistence.Repositories{}, nil)
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /openapi.json returned %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("unexpected content type %q", got)
	}
	var served, source any
	if err := json.Unmarshal(recorder.Body.Bytes(), &served); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(docs.OpenAPIJSON, &source); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(served, source) {
		t.Fatal("served OpenAPI document differs from the embedded contract")
	}

	actual := map[string]bool{}
	if err := chi.Walk(r.(chi.Routes), func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		// These two routes serve documentation; every other HTTP route is public API.
		if route != "/openapi.json" && !strings.HasPrefix(route, "/swagger/") {
			actual[method+" "+strings.TrimSuffix(route, "/")] = true
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, issue := range contractIssues(doc, actual) {
		t.Error(issue)
	}

	checkProperties(t, doc, "CreateUserRequest", dto.CreateUserRequest{})
	checkProperties(t, doc, "User", dto.UserResponse{})
	checkProperties(t, doc, "UserListData", dto.UserListResponse{})
	checkProperties(t, doc, "SendMailRequest", dto.SendMailRequest{})
	checkProperties(t, doc, "SendMailAttachmentRequest", dto.SendMailAttachmentRequest{})
	checkProperties(t, doc, "SendMailResponseData", dto.SendMailResponseData{})
	checkProperties(t, doc, "HealthData", dto.HealthData{})
	checkProperties(t, doc, "ErrorDetail", dto.ErrorDetail{})
	checkProperties(t, doc, "ErrorResponse", dto.ErrorResponse{})
}

func contractIssues(doc *openapi3.T, actual map[string]bool) []string {
	var issues []string
	ids := map[string]string{}
	contractRoutes := map[string]bool{}
	for path, item := range doc.Paths.Map() {
		for method, operation := range item.Operations() {
			route := method + " " + path
			contractRoutes[route] = true
			if !actual[route] {
				issues = append(issues, route+": OpenAPI operationId "+operation.OperationID+" has no router route")
			}
			if strings.TrimSpace(operation.OperationID) == "" {
				issues = append(issues, route+": missing operationId")
			} else if previous, ok := ids[operation.OperationID]; ok {
				issues = append(issues, route+": duplicate operationId "+operation.OperationID+" (also "+previous+")")
			}
			ids[operation.OperationID] = route
			if strings.TrimSpace(operation.Summary) == "" {
				issues = append(issues, route+" (operationId "+operation.OperationID+"): missing summary")
			}
			if len(operation.Tags) == 0 {
				issues = append(issues, route+" (operationId "+operation.OperationID+"): missing tags")
			}
		}
	}
	for route := range actual {
		if !contractRoutes[route] {
			issues = append(issues, route+": router route has no OpenAPI operation (operationId missing)")
		}
	}
	sort.Strings(issues)
	return issues
}

func TestContractIssuesDetectDrift(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromData(docs.OpenAPIYAML)
	if err != nil {
		t.Fatal(err)
	}
	actual := map[string]bool{"GET /new-route": true}
	issues := strings.Join(contractIssues(doc, actual), "\n")
	for _, fragment := range []string{"GET /new-route: router route has no OpenAPI operation", "GET /health: OpenAPI operationId getHealth has no router route"} {
		if !strings.Contains(issues, fragment) {
			t.Errorf("missing drift diagnostic %q in:\n%s", fragment, issues)
		}
	}
	doc.Paths.Find("/health").Get.OperationID = ""
	if issues := strings.Join(contractIssues(doc, actual), "\n"); !strings.Contains(issues, "GET /health: missing operationId") {
		t.Errorf("missing operationId diagnostic in:\n%s", issues)
	}
	doc.Paths.Find("/health").Get.OperationID = "getReadiness"
	if issues := strings.Join(contractIssues(doc, actual), "\n"); !strings.Contains(issues, "duplicate operationId getReadiness") {
		t.Errorf("missing duplicate operationId diagnostic in:\n%s", issues)
	}
}

func checkProperties(t *testing.T, doc *openapi3.T, schemaName string, value any) {
	t.Helper()
	schema := doc.Components.Schemas[schemaName]
	if schema == nil || schema.Value == nil {
		t.Fatalf("missing schema %s", schemaName)
	}
	want := map[string]bool{}
	typ := reflect.TypeOf(value)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		want[name] = true
		property := schema.Value.Properties[name]
		if property == nil || property.Value == nil {
			t.Errorf("%s.%s is missing", schemaName, name)
			continue
		}
		if kind := jsonType(field.Type); !property.Value.Type.Is(kind) {
			t.Errorf("%s.%s has schema type %v; DTO type is %s", schemaName, name, property.Value.Type, kind)
		}
		if field.Type.Kind() == reflect.Slice {
			items := property.Value.Items
			if items == nil || items.Value == nil || !items.Value.Type.Is(jsonType(field.Type.Elem())) {
				t.Errorf("%s.%s has array item type inconsistent with DTO", schemaName, name)
			}
		}
	}
	got := map[string]bool{}
	for name := range schema.Value.Properties {
		got[name] = true
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s schema properties %v differ from DTO properties %v", schemaName, keys(got), keys(want))
	}
}

func jsonType(typ reflect.Type) string {
	switch typ.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int64:
		return "integer"
	case reflect.String:
		return "string"
	case reflect.Slice:
		return "array"
	default:
		return "object"
	}
}

func keys[V any](values map[string]V) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func TestSwaggerUIUsesOpenAPI(t *testing.T) {
	r := New(&config.Config{Swagger: config.SwaggerConfig{Enabled: true}}, quietLogger{}, nil, &persistence.Repositories{}, nil)
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("Swagger UI returned %d: %s", recorder.Code, recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte("/openapi.json")) {
		t.Fatal("Swagger UI does not reference /openapi.json")
	}
}
