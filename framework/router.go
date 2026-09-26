// Package framework는 외부 Go 모듈을 위한 작은 HTTP 서버 조립 API를 제공한다.
package framework

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/heartblast/able-rest-api/docs"
	custommw "github.com/heartblast/able-rest-api/internal/delivery/http/middleware"
)

// Module은 /api/v1 아래에 업무 경로를 등록한다.
type Module func(chi.Router)

// Router는 미들웨어, 업무 모듈, OpenAPI 문서를 조립한다.
type Router struct {
	modules    []Module
	middleware []func(http.Handler) http.Handler
	openAPI    []byte
}

// New는 포함된 OpenAPI 계약을 기본으로 사용하는 라우터를 만든다.
func New() *Router {
	return &Router{openAPI: docs.OpenAPIJSON}
}

// Register는 업무 모듈을 등록한다.
func (r *Router) Register(module Module) {
	r.modules = append(r.modules, module)
}

// Use는 모든 공개 경로에 적용할 미들웨어를 추가한다.
func (r *Router) Use(middleware func(http.Handler) http.Handler) {
	r.middleware = append(r.middleware, middleware)
}

// SetOpenAPIJSON은 /openapi.json에서 제공할 JSON 계약을 지정한다.
func (r *Router) SetOpenAPIJSON(document []byte) error {
	if !json.Valid(document) {
		return errors.New("invalid OpenAPI JSON")
	}
	r.openAPI = append([]byte(nil), document...)
	return nil
}

// Handler는 등록된 설정으로 최종 HTTP 핸들러를 만든다.
func (r *Router) Handler() http.Handler {
	h := chi.NewRouter()
	h.Use(custommw.RequestID)
	h.Use(chimiddleware.Recoverer)
	h.Use(chimiddleware.Timeout(30 * time.Second))
	h.Use(custommw.JSONContentType)
	h.Use(custommw.LimitJSONBody(30 << 20))
	for _, middleware := range r.middleware {
		h.Use(middleware)
	}
	document := append([]byte(nil), r.openAPI...)
	h.Get("/openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(document)
	})
	h.Route("/api/v1", func(api chi.Router) {
		for _, module := range r.modules {
			module(api)
		}
	})
	return h
}
