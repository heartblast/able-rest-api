package router

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"

	"able-rest-api/docs"
	"able-rest-api/internal/app/service"
	"able-rest-api/internal/domain/model"
	"able-rest-api/internal/domain/repository"
	"able-rest-api/internal/infra/config"
	"able-rest-api/internal/infra/persistence"
)

type contractRepo struct {
	users []model.User
	fail  bool
}

func (r *contractRepo) GetByID(_ context.Context, id int64) (*model.User, error) {
	if r.fail {
		return nil, errors.New("repository unavailable")
	}
	for _, user := range r.users {
		if user.ID == id {
			return &user, nil
		}
	}
	return nil, nil
}
func (r *contractRepo) List(_ context.Context, filter repository.UserFilter) ([]model.User, error) {
	if r.fail {
		return nil, errors.New("repository unavailable")
	}
	if filter.Offset >= len(r.users) {
		return []model.User{}, nil
	}
	end := filter.Offset + filter.Limit
	if end > len(r.users) {
		end = len(r.users)
	}
	return r.users[filter.Offset:end], nil
}
func (r *contractRepo) Create(_ context.Context, user *model.User) error {
	if r.fail {
		return errors.New("repository unavailable")
	}
	user.ID = int64(len(r.users) + 1)
	user.CreatedAt = time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	user.UpdatedAt = user.CreatedAt
	r.users = append(r.users, *user)
	return nil
}

type contractSender struct{ fail bool }

func (s contractSender) Send(context.Context, model.MailMessage) error {
	if s.fail {
		return errors.New("smtp unavailable")
	}
	return nil
}

type contractConnector struct{}

func (contractConnector) Connect(context.Context) (driver.Conn, error) { return contractConn{}, nil }
func (contractConnector) Driver() driver.Driver                        { return contractDriver{} }

type contractDriver struct{}

func (contractDriver) Open(string) (driver.Conn, error) { return contractConn{}, nil }

type contractConn struct{}

func (contractConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (contractConn) Close() error                        { return nil }
func (contractConn) Begin() (driver.Tx, error)           { return nil, errors.New("unused") }
func (contractConn) Ping(context.Context) error          { return nil }

func TestRuntimeOpenAPIContract(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromData(docs.OpenAPIYAML)
	if err != nil {
		t.Fatal(err)
	}
	contractRouter, err := legacy.NewRouter(doc)
	if err != nil {
		t.Fatal(err)
	}
	const userJSON = `{"name":"Alice","email":"alice@example.com"}`
	const mailJSON = `{"to":["user@example.com"],"subject":"Welcome","body":"Hello","is_html":false}`
	cases := []struct {
		name, method, path, body, contentType      string
		status                                     int
		code                                       string
		invalidRequest                             bool
		repoFail, mailFail, mailDisabled, dbClosed bool
	}{
		{name: "health", method: "GET", path: "/health", status: 200},
		{name: "ready", method: "GET", path: "/ready", status: 200},
		{name: "ready unavailable", method: "GET", path: "/ready", status: 503, code: "DB_NOT_READY", dbClosed: true},
		{name: "list users", method: "GET", path: "/api/v1/users?limit=1&offset=0", status: 200},
		{name: "list users invalid limit is normalized", method: "GET", path: "/api/v1/users?limit=not-an-integer", status: 200, invalidRequest: true},
		{name: "list users invalid offset is normalized", method: "GET", path: "/api/v1/users?offset=not-an-integer", status: 200, invalidRequest: true},
		{name: "list users repository error", method: "GET", path: "/api/v1/users", status: 500, code: "INTERNAL_ERROR", repoFail: true},
		{name: "create user", method: "POST", path: "/api/v1/users", body: userJSON, contentType: "application/json", status: 201},
		{name: "create user invalid body", method: "POST", path: "/api/v1/users", body: `{"name":"","email":"bad"}`, contentType: "application/json", status: 400, code: "VALIDATION_ERROR", invalidRequest: true},
		{name: "create user missing required field", method: "POST", path: "/api/v1/users", body: `{"name":"Alice"}`, contentType: "application/json", status: 400, code: "VALIDATION_ERROR", invalidRequest: true},
		{name: "create user missing required body", method: "POST", path: "/api/v1/users", contentType: "application/json", status: 400, code: "INVALID_JSON", invalidRequest: true},
		{name: "create user invalid json", method: "POST", path: "/api/v1/users", body: `{`, contentType: "application/json", status: 400, code: "INVALID_JSON", invalidRequest: true},
		{name: "create user unsupported media", method: "POST", path: "/api/v1/users", body: userJSON, contentType: "text/plain", status: 415, code: "UNSUPPORTED_MEDIA_TYPE", invalidRequest: true},
		{name: "create user repository error", method: "POST", path: "/api/v1/users", body: userJSON, contentType: "application/json", status: 500, code: "INTERNAL_ERROR", repoFail: true},
		{name: "get user", method: "GET", path: "/api/v1/users/1", status: 200},
		{name: "get user missing", method: "GET", path: "/api/v1/users/42", status: 404, code: "NOT_FOUND"},
		{name: "get user invalid path type", method: "GET", path: "/api/v1/users/nope", status: 400, code: "INVALID_ID", invalidRequest: true},
		{name: "get user invalid path minimum", method: "GET", path: "/api/v1/users/0", status: 400, code: "VALIDATION_ERROR", invalidRequest: true},
		{name: "get user repository error", method: "GET", path: "/api/v1/users/1", status: 500, code: "INTERNAL_ERROR", repoFail: true},
		{name: "send mail", method: "POST", path: "/api/v1/mail/send", body: mailJSON, contentType: "application/json", status: 202},
		{name: "send mail invalid body", method: "POST", path: "/api/v1/mail/send", body: `{"to":[],"subject":"","body":""}`, contentType: "application/json", status: 400, code: "VALIDATION_ERROR", invalidRequest: true},
		{name: "send mail missing required field", method: "POST", path: "/api/v1/mail/send", body: `{"subject":"Hello","body":"Hello"}`, contentType: "application/json", status: 400, code: "VALIDATION_ERROR", invalidRequest: true},
		{name: "send mail unsupported media", method: "POST", path: "/api/v1/mail/send", body: mailJSON, contentType: "text/plain", status: 415, code: "UNSUPPORTED_MEDIA_TYPE", invalidRequest: true},
		{name: "send mail disabled", method: "POST", path: "/api/v1/mail/send", body: mailJSON, contentType: "application/json", status: 503, code: "MAIL_DISABLED", mailDisabled: true},
		{name: "send mail sender error", method: "POST", path: "/api/v1/mail/send", body: mailJSON, contentType: "application/json", status: 500, code: "INTERNAL_ERROR", mailFail: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
			repo := &contractRepo{users: []model.User{{ID: 1, Name: "Alice", Email: "alice@example.com", CreatedAt: now, UpdatedAt: now}}, fail: tc.repoFail}
			db := sql.OpenDB(contractConnector{})
			if tc.dbClosed {
				_ = db.Close()
			} else {
				t.Cleanup(func() { _ = db.Close() })
			}
			handler := New(&config.Config{}, quietLogger{}, db, &persistence.Repositories{UserRepository: repo}, service.NewMailService(!tc.mailDisabled, contractSender{fail: tc.mailFail}))
			request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.contentType != "" {
				request.Header.Set("Content-Type", tc.contentType)
			}
			route, pathParams, err := contractRouter.FindRoute(request)
			if err != nil {
				t.Fatalf("%s %s: OpenAPI route: %v", tc.method, tc.path, err)
			}
			input := &openapi3filter.RequestValidationInput{Request: request, Route: route, PathParams: pathParams}
			requestErr := openapi3filter.ValidateRequest(context.Background(), input)
			if tc.invalidRequest && requestErr == nil {
				t.Fatalf("%s %s: expected request parameter/body validation to fail", tc.method, tc.path)
			}
			if !tc.invalidRequest && requestErr != nil {
				t.Fatalf("%s %s: request contract: %v", tc.method, tc.path, requestErr)
			}
			// Validation reads the body; restore it even when validation rejects the request.
			request.Body = io.NopCloser(bytes.NewReader([]byte(tc.body)))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			response := recorder.Result()
			defer response.Body.Close()
			payload, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			label := fmt.Sprintf("%s %s -> %d", tc.method, route.Path, response.StatusCode)
			if response.StatusCode != tc.status {
				t.Fatalf("%s: status want %d; body %s", label, tc.status, payload)
			}
			validation := &openapi3filter.ResponseValidationInput{
				RequestValidationInput: input, Status: response.StatusCode, Header: response.Header,
				Options: &openapi3filter.Options{IncludeResponseStatus: true},
			}
			validation.SetBodyBytes(payload)
			if err := openapi3filter.ValidateResponse(context.Background(), validation); err != nil {
				t.Fatalf("%s: response status/content-type/schema: %v; body %s", label, err, payload)
			}
			if got := response.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
				t.Fatalf("%s: Content-Type = %q", label, got)
			}
			var envelope map[string]json.RawMessage
			if err := json.Unmarshal(payload, &envelope); err != nil {
				t.Fatalf("%s: invalid JSON: %v", label, err)
			}
			keys := []string{"success", "request_id", "data"}
			if tc.code != "" {
				keys = []string{"success", "request_id", "error"}
			}
			if tc.status == 415 {
				keys = []string{"success", "error"}
			}
			if !sameKeys(envelope, keys) {
				t.Fatalf("%s: envelope fields %v, want %v", label, mapKeys(envelope), keys)
			}
			var success bool
			if err := json.Unmarshal(envelope["success"], &success); err != nil || success != (tc.code == "") {
				t.Fatalf("%s: success flag %s", label, envelope["success"])
			}
			if tc.status != 415 {
				var requestID string
				if err := json.Unmarshal(envelope["request_id"], &requestID); err != nil || requestID == "" {
					t.Fatalf("%s: missing request_id", label)
				}
			}
			if tc.code != "" {
				var detail struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				}
				if err := json.Unmarshal(envelope["error"], &detail); err != nil || detail.Code != tc.code || detail.Message == "" {
					t.Fatalf("%s: error details %s, want code %s", label, envelope["error"], tc.code)
				}
			}
		})
	}
}

func sameKeys(value map[string]json.RawMessage, expected []string) bool {
	if len(value) != len(expected) {
		return false
	}
	for _, key := range expected {
		if _, ok := value[key]; !ok {
			return false
		}
	}
	return true
}
func mapKeys(value map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	return keys
}

// Assert the validator actually detects the regressions this suite is intended to catch.
func TestRuntimeValidatorRejectsContractDrift(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromData(docs.OpenAPIYAML)
	if err != nil {
		t.Fatal(err)
	}
	router, err := legacy.NewRouter(doc)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/health", nil)
	route, params, err := router.FindRoute(request)
	if err != nil {
		t.Fatal(err)
	}
	input := &openapi3filter.RequestValidationInput{Request: request, Route: route, PathParams: params}
	for _, tc := range []struct {
		name              string
		status            int
		contentType, body string
	}{
		{"unknown status", 418, "application/json", `{"success":true,"data":{"status":"ok"}}`},
		{"wrong content type", 200, "text/plain", `{"success":true,"data":{"status":"ok"}}`},
		{"wrong schema", 200, "application/json", `{"success":true,"data":{"status":12}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := &openapi3filter.ResponseValidationInput{RequestValidationInput: input, Status: tc.status, Header: http.Header{"Content-Type": []string{tc.contentType}}, Options: &openapi3filter.Options{IncludeResponseStatus: true}}
			result.SetBodyBytes([]byte(tc.body))
			if err := openapi3filter.ValidateResponse(context.Background(), result); err == nil {
				t.Fatal("validator accepted contract drift")
			}
		})
	}
}
