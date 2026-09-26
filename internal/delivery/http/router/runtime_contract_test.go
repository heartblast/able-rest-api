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
	"sort"
	"strconv"
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

type contractCaseKind string

const (
	successCase contractCaseKind = "success"
	errorCase   contractCaseKind = "error"
)

type runtimeContractCase struct {
	name, method, contractPath, operationID, path, body, contentType string
	kind                                                             contractCaseKind
	status                                                           int
	code                                                             string
	assertResponse                                                   func(json.RawMessage) error
	invalidRequest                                                   bool
	repoFail, mailFail, mailDisabled, dbClosed                       bool
}

type contractUserData struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func assertHealthStatus(want string) func(json.RawMessage) error {
	return func(raw json.RawMessage) error {
		var data struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			return fmt.Errorf("data.status: decode: %w", err)
		}
		if data.Status != want {
			return fmt.Errorf("data.status = %q, want %q", data.Status, want)
		}
		return nil
	}
}

func assertUserData(got contractUserData, id int64, name, email string) error {
	if got.ID != id || got.Name != name || got.Email != email {
		return fmt.Errorf("user id/name/email = %d/%q/%q, want %d/%q/%q", got.ID, got.Name, got.Email, id, name, email)
	}
	const fixtureTime = "2026-04-06T10:00:00Z"
	if got.CreatedAt != fixtureTime || got.UpdatedAt != fixtureTime {
		return fmt.Errorf("user created_at/updated_at = %q/%q, want %q", got.CreatedAt, got.UpdatedAt, fixtureTime)
	}
	return nil
}

func assertUser(id int64, name, email string) func(json.RawMessage) error {
	return func(raw json.RawMessage) error {
		var data contractUserData
		if err := json.Unmarshal(raw, &data); err != nil {
			return fmt.Errorf("data.user: decode: %w", err)
		}
		return assertUserData(data, id, name, email)
	}
}

func assertUserList(names ...string) func(json.RawMessage) error {
	return func(raw json.RawMessage) error {
		var data struct {
			Items []contractUserData `json:"items"`
			Count int                `json:"count"`
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			return fmt.Errorf("data.items/count: decode: %w", err)
		}
		if data.Count != len(data.Items) || len(data.Items) != len(names) {
			return fmt.Errorf("data.count/items length = %d/%d, want %d", data.Count, len(data.Items), len(names))
		}
		for i, name := range names {
			id := int64(1)
			if name == "Bob" {
				id = 2
			}
			if err := assertUserData(data.Items[i], id, name, strings.ToLower(name)+"@example.com"); err != nil {
				return fmt.Errorf("data.items[%d]: %w", i, err)
			}
		}
		return nil
	}
}

func assertAcceptedRecipients(want int) func(json.RawMessage) error {
	return func(raw json.RawMessage) error {
		var data struct {
			AcceptedRecipients int `json:"accepted_recipients"`
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			return fmt.Errorf("data.accepted_recipients: decode: %w", err)
		}
		if data.AcceptedRecipients != want {
			return fmt.Errorf("data.accepted_recipients = %d, want %d", data.AcceptedRecipients, want)
		}
		return nil
	}
}

func runtimeContractCases() []runtimeContractCase {
	const userJSON = `{"name":"  Carol  ","email":" CAROL@Example.com "}`
	const mailJSON = `{"to":["user@example.com"],"cc":["team@example.com"],"bcc":["team@example.com","audit@example.com"],"subject":"Welcome","body":"Hello","is_html":false}`
	return []runtimeContractCase{
		{operationID: "getHealth", contractPath: "/health", kind: successCase, name: "health", method: "GET", path: "/health", status: 200, assertResponse: assertHealthStatus("ok")},
		{operationID: "getReadiness", contractPath: "/ready", kind: successCase, name: "ready", method: "GET", path: "/ready", status: 200, assertResponse: assertHealthStatus("ready")},
		{operationID: "getReadiness", contractPath: "/ready", kind: errorCase, name: "ready unavailable", method: "GET", path: "/ready", status: 503, code: "DB_NOT_READY", dbClosed: true},
		{operationID: "listUsers", contractPath: "/api/v1/users", kind: successCase, name: "list users", method: "GET", path: "/api/v1/users?limit=1&offset=0", status: 200, assertResponse: assertUserList("Alice")},
		{operationID: "listUsers", contractPath: "/api/v1/users", kind: successCase, name: "list users offset", method: "GET", path: "/api/v1/users?limit=1&offset=1", status: 200, assertResponse: assertUserList("Bob")},
		{operationID: "listUsers", contractPath: "/api/v1/users", kind: successCase, name: "list users empty page", method: "GET", path: "/api/v1/users?offset=99", status: 200, assertResponse: assertUserList()},
		{operationID: "listUsers", contractPath: "/api/v1/users", kind: successCase, name: "list users invalid limit is normalized", method: "GET", path: "/api/v1/users?limit=not-an-integer", status: 200, invalidRequest: true, assertResponse: assertUserList("Alice", "Bob")},
		{operationID: "listUsers", contractPath: "/api/v1/users", kind: successCase, name: "list users invalid offset is normalized", method: "GET", path: "/api/v1/users?offset=not-an-integer", status: 200, invalidRequest: true, assertResponse: assertUserList("Alice", "Bob")},
		{operationID: "listUsers", contractPath: "/api/v1/users", kind: errorCase, name: "list users repository error", method: "GET", path: "/api/v1/users", status: 500, code: "INTERNAL_ERROR", repoFail: true},
		{operationID: "createUser", contractPath: "/api/v1/users", kind: successCase, name: "create user", method: "POST", path: "/api/v1/users", body: userJSON, contentType: "application/json", status: 201, assertResponse: assertUser(3, "Carol", "carol@example.com")},
		{operationID: "createUser", contractPath: "/api/v1/users", kind: errorCase, name: "create user invalid body", method: "POST", path: "/api/v1/users", body: `{"name":"","email":"bad"}`, contentType: "application/json", status: 400, code: "VALIDATION_ERROR", invalidRequest: true},
		{operationID: "createUser", contractPath: "/api/v1/users", kind: errorCase, name: "create user missing required field", method: "POST", path: "/api/v1/users", body: `{"name":"Alice"}`, contentType: "application/json", status: 400, code: "VALIDATION_ERROR", invalidRequest: true},
		{operationID: "createUser", contractPath: "/api/v1/users", kind: errorCase, name: "create user missing required body", method: "POST", path: "/api/v1/users", contentType: "application/json", status: 400, code: "INVALID_JSON", invalidRequest: true},
		{operationID: "createUser", contractPath: "/api/v1/users", kind: errorCase, name: "create user invalid json", method: "POST", path: "/api/v1/users", body: `{`, contentType: "application/json", status: 400, code: "INVALID_JSON", invalidRequest: true},
		{operationID: "createUser", contractPath: "/api/v1/users", kind: errorCase, name: "create user unsupported media", method: "POST", path: "/api/v1/users", body: userJSON, contentType: "text/plain", status: 415, code: "UNSUPPORTED_MEDIA_TYPE", invalidRequest: true},
		{operationID: "createUser", contractPath: "/api/v1/users", kind: errorCase, name: "create user repository error", method: "POST", path: "/api/v1/users", body: userJSON, contentType: "application/json", status: 500, code: "INTERNAL_ERROR", repoFail: true},
		{operationID: "getUser", contractPath: "/api/v1/users/{id}", kind: successCase, name: "get user", method: "GET", path: "/api/v1/users/1", status: 200, assertResponse: assertUser(1, "Alice", "alice@example.com")},
		{operationID: "getUser", contractPath: "/api/v1/users/{id}", kind: errorCase, name: "get user missing", method: "GET", path: "/api/v1/users/42", status: 404, code: "NOT_FOUND"},
		{operationID: "getUser", contractPath: "/api/v1/users/{id}", kind: errorCase, name: "get user invalid path type", method: "GET", path: "/api/v1/users/nope", status: 400, code: "INVALID_ID", invalidRequest: true},
		{operationID: "getUser", contractPath: "/api/v1/users/{id}", kind: errorCase, name: "get user invalid path minimum", method: "GET", path: "/api/v1/users/0", status: 400, code: "VALIDATION_ERROR", invalidRequest: true},
		{operationID: "getUser", contractPath: "/api/v1/users/{id}", kind: errorCase, name: "get user repository error", method: "GET", path: "/api/v1/users/1", status: 500, code: "INTERNAL_ERROR", repoFail: true},
		{operationID: "sendMail", contractPath: "/api/v1/mail/send", kind: successCase, name: "send mail", method: "POST", path: "/api/v1/mail/send", body: mailJSON, contentType: "application/json", status: 202, assertResponse: assertAcceptedRecipients(3)},
		{operationID: "sendMail", contractPath: "/api/v1/mail/send", kind: errorCase, name: "send mail invalid body", method: "POST", path: "/api/v1/mail/send", body: `{"to":[],"subject":"","body":""}`, contentType: "application/json", status: 400, code: "VALIDATION_ERROR", invalidRequest: true},
		{operationID: "sendMail", contractPath: "/api/v1/mail/send", kind: errorCase, name: "send mail missing required field", method: "POST", path: "/api/v1/mail/send", body: `{"subject":"Hello","body":"Hello"}`, contentType: "application/json", status: 400, code: "VALIDATION_ERROR", invalidRequest: true},
		{operationID: "sendMail", contractPath: "/api/v1/mail/send", kind: errorCase, name: "send mail unsupported media", method: "POST", path: "/api/v1/mail/send", body: mailJSON, contentType: "text/plain", status: 415, code: "UNSUPPORTED_MEDIA_TYPE", invalidRequest: true},
		{operationID: "sendMail", contractPath: "/api/v1/mail/send", kind: errorCase, name: "send mail disabled", method: "POST", path: "/api/v1/mail/send", body: mailJSON, contentType: "application/json", status: 503, code: "MAIL_DISABLED", mailDisabled: true},
		{operationID: "sendMail", contractPath: "/api/v1/mail/send", kind: errorCase, name: "send mail sender error", method: "POST", path: "/api/v1/mail/send", body: mailJSON, contentType: "application/json", status: 500, code: "INTERNAL_ERROR", mailFail: true},
	}
}

func TestRuntimeOpenAPIContract(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromData(docs.OpenAPIYAML)
	if err != nil {
		t.Fatal(err)
	}
	contractRouter, err := legacy.NewRouter(doc)
	if err != nil {
		t.Fatal(err)
	}
	cases := runtimeContractCases()
	for _, issue := range runtimeCoverageIssues(doc, cases) {
		t.Error(issue)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
			repo := &contractRepo{users: []model.User{
				{ID: 1, Name: "Alice", Email: "alice@example.com", CreatedAt: now, UpdatedAt: now},
				{ID: 2, Name: "Bob", Email: "bob@example.com", CreatedAt: now, UpdatedAt: now},
			}, fail: tc.repoFail}
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
			label := fmt.Sprintf("%s %s (operationId %s)", tc.method, tc.contractPath, tc.operationID)
			if route.Path != tc.contractPath || route.Operation.OperationID != tc.operationID {
				t.Fatalf("%s: request %s resolved to %s (operationId %s)", label, tc.path, route.Path, route.Operation.OperationID)
			}
			input := &openapi3filter.RequestValidationInput{Request: request, Route: route, PathParams: pathParams}
			requestErr := openapi3filter.ValidateRequest(context.Background(), input)
			if tc.invalidRequest && requestErr == nil {
				t.Fatalf("%s: expected request parameter/body validation to fail", label)
			}
			if !tc.invalidRequest && requestErr != nil {
				t.Fatalf("%s: request contract: %v", label, requestErr)
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
			label = fmt.Sprintf("%s -> %d", label, response.StatusCode)
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
				if err := json.Unmarshal(envelope["request_id"], &requestID); err != nil || strings.TrimSpace(requestID) == "" {
					t.Fatalf("%s: missing request_id", label)
				}
			}
			if tc.code != "" {
				var detail struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				}
				if err := json.Unmarshal(envelope["error"], &detail); err != nil || detail.Code != tc.code || strings.TrimSpace(detail.Message) == "" {
					t.Fatalf("%s: error details %s, want code %s", label, envelope["error"], tc.code)
				}
			}
			if tc.kind == successCase {
				if err := assertSemanticResponse(tc, envelope["data"]); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func assertSemanticResponse(tc runtimeContractCase, data json.RawMessage) error {
	label := fmt.Sprintf("%s %s (operationId %s)", tc.method, tc.contractPath, tc.operationID)
	if tc.assertResponse == nil {
		return fmt.Errorf("%s: missing semantic assertion", label)
	}
	if err := tc.assertResponse(data); err != nil {
		return fmt.Errorf("%s: semantic assertion: %w", label, err)
	}
	return nil
}

func TestSemanticAssertionsReportOperationAndField(t *testing.T) {
	for _, tc := range []struct {
		operationID string
		data        string
		field       string
	}{
		{"getHealth", `{"status":"ready"}`, "data.status"},
		{"listUsers", `{"items":[],"count":2}`, "data.count/items length"},
		{"createUser", `{"id":3,"name":"Wrong","email":"carol@example.com"}`, "user id/name/email"},
		{"getUser", `{"id":2,"name":"Alice","email":"alice@example.com"}`, "user id/name/email"},
		{"sendMail", `{"accepted_recipients":1}`, "data.accepted_recipients"},
	} {
		t.Run(tc.operationID, func(t *testing.T) {
			for _, contractCase := range runtimeContractCases() {
				if contractCase.operationID != tc.operationID || contractCase.kind != successCase {
					continue
				}
				err := assertSemanticResponse(contractCase, json.RawMessage(tc.data))
				if err == nil || !strings.Contains(err.Error(), "operationId "+tc.operationID) || !strings.Contains(err.Error(), tc.field) {
					t.Fatalf("semantic regression diagnostic = %v, want operationId %s and %s", err, tc.operationID, tc.field)
				}
				return
			}
			t.Fatalf("missing successful case for %s", tc.operationID)
		})
	}
}

func runtimeCoverageIssues(doc *openapi3.T, cases []runtimeContractCase) []string {
	var issues []string
	covered := map[string]map[int]bool{}
	for _, tc := range cases {
		label := fmt.Sprintf("%s %s (operationId %s)", tc.method, tc.contractPath, tc.operationID)
		item := doc.Paths.Find(tc.contractPath)
		var operation *openapi3.Operation
		if item != nil {
			operation = item.GetOperation(tc.method)
		}
		if operation == nil {
			issues = append(issues, label+": contract case has no OpenAPI operation")
			continue
		}
		if tc.operationID != operation.OperationID {
			issues = append(issues, label+": operationId differs from OpenAPI operationId "+operation.OperationID)
		}
		status := strconv.Itoa(tc.status)
		if operation.Responses.Value(status) == nil {
			issues = append(issues, label+": contract case status "+status+" is not declared in OpenAPI")
		}
		if tc.kind != successCase && tc.kind != errorCase {
			issues = append(issues, label+": contract case has no valid success/error kind")
		} else if (tc.status < 400) != (tc.kind == successCase) || (tc.code == "") != (tc.kind == successCase) {
			issues = append(issues, label+": contract case status/error code disagrees with "+string(tc.kind)+" kind")
		}
		if tc.kind == successCase && tc.assertResponse == nil {
			issues = append(issues, label+": missing semantic assertion for successful response")
		}
		key := tc.method + " " + tc.contractPath + " " + tc.operationID
		if covered[key] == nil {
			covered[key] = map[int]bool{}
		}
		covered[key][tc.status] = true
	}
	for path, item := range doc.Paths.Map() {
		for method, operation := range item.Operations() {
			key := method + " " + path + " " + operation.OperationID
			label := fmt.Sprintf("%s %s (operationId %s)", method, path, operation.OperationID)
			if len(covered[key]) == 0 {
				issues = append(issues, label+": missing runtime contract cases")
			}
			var hasSuccess, hasError bool
			for status := range operation.Responses.Map() {
				code, err := strconv.Atoi(status)
				if err != nil || !covered[key][code] {
					issues = append(issues, label+": missing runtime contract case for declared response status "+status)
				}
				if err == nil {
					if code < 400 {
						hasSuccess = true
					} else {
						hasError = true
					}
				}
			}
			if !hasSuccess || !hasCoveredKind(cases, method, path, operation.OperationID, successCase) {
				issues = append(issues, label+": missing successful response contract case")
			}
			if hasError && !hasCoveredKind(cases, method, path, operation.OperationID, errorCase) {
				issues = append(issues, label+": missing error response contract case")
			}
		}
	}
	sort.Strings(issues)
	return issues
}

func hasCoveredKind(cases []runtimeContractCase, method, path, operationID string, kind contractCaseKind) bool {
	for _, tc := range cases {
		if tc.method == method && tc.contractPath == path && tc.operationID == operationID && tc.kind == kind &&
			((kind == successCase && tc.status < 400 && tc.code == "") || (kind == errorCase && tc.status >= 400 && tc.code != "")) {
			return true
		}
	}
	return false
}

func TestRuntimeCoverageDetectsMissingOperationsAndCases(t *testing.T) {
	load := func(t *testing.T) *openapi3.T {
		t.Helper()
		doc, err := openapi3.NewLoader().LoadFromData(docs.OpenAPIYAML)
		if err != nil {
			t.Fatal(err)
		}
		return doc
	}
	assertIssue := func(t *testing.T, doc *openapi3.T, cases []runtimeContractCase, fragment string) {
		t.Helper()
		issues := strings.Join(runtimeCoverageIssues(doc, cases), "\n")
		if !strings.Contains(issues, fragment) {
			t.Fatalf("missing diagnostic %q in:\n%s", fragment, issues)
		}
	}
	if issues := runtimeCoverageIssues(load(t), runtimeContractCases()); len(issues) != 0 {
		t.Fatalf("baseline contract coverage: %v", issues)
	}

	t.Run("new OpenAPI operation without cases", func(t *testing.T) {
		doc := load(t)
		description := "ok"
		doc.Paths.Set("/new-operation", &openapi3.PathItem{Get: &openapi3.Operation{
			OperationID: "getNewOperation",
			Responses:   openapi3.NewResponses(openapi3.WithStatus(200, &openapi3.ResponseRef{Value: &openapi3.Response{Description: &description}})),
		}})
		assertIssue(t, doc, runtimeContractCases(), "GET /new-operation (operationId getNewOperation): missing runtime contract cases")
	})

	t.Run("removed success and error cases", func(t *testing.T) {
		var cases []runtimeContractCase
		for _, tc := range runtimeContractCases() {
			if tc.operationID != "getReadiness" {
				cases = append(cases, tc)
			}
		}
		assertIssue(t, load(t), cases, "GET /ready (operationId getReadiness): missing successful response contract case")
		assertIssue(t, load(t), cases, "GET /ready (operationId getReadiness): missing error response contract case")
		assertIssue(t, load(t), cases, "GET /ready (operationId getReadiness): missing runtime contract case for declared response status 503")
	})

	t.Run("mismatched operation id", func(t *testing.T) {
		cases := runtimeContractCases()
		cases[0].operationID = "wrongId"
		assertIssue(t, load(t), cases, "GET /health (operationId wrongId): operationId differs from OpenAPI operationId getHealth")
	})

	t.Run("missing case kind", func(t *testing.T) {
		cases := runtimeContractCases()
		cases[0].kind = ""
		assertIssue(t, load(t), cases, "GET /health (operationId getHealth): contract case has no valid success/error kind")
		assertIssue(t, load(t), cases, "GET /health (operationId getHealth): missing successful response contract case")
	})

	t.Run("missing semantic assertion", func(t *testing.T) {
		cases := runtimeContractCases()
		cases[0].assertResponse = nil
		assertIssue(t, load(t), cases, "GET /health (operationId getHealth): missing semantic assertion for successful response")
	})
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
