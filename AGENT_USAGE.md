# 외부 Go 프로젝트 사용 가이드

## 설치

새 공개 릴리스가 만들어진 뒤 해당 버전으로 설치한다.

```bash
go get github.com/heartblast/able-rest-api@vX.Y.Z
```

현재 소스는 Go 1.26.1을 요구한다. 기존 v1.1.0에는 이 공개 API와 새 module path가 없으므로 새 릴리스가 필요하다. 비공개 저장소라면 접근 권한과 `GOPRIVATE=github.com/heartblast/*`를 설정한다.

## 공개 패키지

| 패키지 | 용도 |
| --- | --- |
| `github.com/heartblast/able-rest-api/framework` | HTTP 라우터 생성, 미들웨어와 업무 모듈 등록, OpenAPI JSON 제공, `http.Handler` 생성 |
| `github.com/heartblast/able-rest-api/httpx` | 공통 성공·오류 JSON 응답 |
| `github.com/heartblast/able-rest-api/docs` | 본 저장소의 기본 OpenAPI YAML·JSON 계약 |

`framework.Module`은 `func(chi.Router)`이므로 경로 등록에는 `github.com/go-chi/chi/v5`를 사용한다. 공개 라우터는 `/api/v1` 아래에 모듈 경로를 등록한다.

## 최소 실행 예제

```go
package main

import (
    "log"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/heartblast/able-rest-api/framework"
    "github.com/heartblast/able-rest-api/httpx"
)

func main() {
    router := framework.New()
    router.Register(func(api chi.Router) {
        api.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
            httpx.Success(w, r, http.StatusOK, map[string]string{"message": "hello"})
        })
    })
    if err := router.SetOpenAPIJSON([]byte(`{"openapi":"3.0.3","info":{"title":"Hello API","version":"1.0.0"},"paths":{"/api/v1/hello":{"get":{"responses":{"200":{"description":"OK"}}}}}}`)); err != nil {
        log.Fatal(err)
    }
    log.Fatal(http.ListenAndServe(":8080", router.Handler()))
}
```

`GET /api/v1/hello`는 `{"success":true,"data":{"message":"hello"}}` 형식의 200 응답을 반환한다. 응답에는 요청 ID가 추가될 수 있다. `GET /openapi.json`은 지정한 JSON 문서를 200으로 제공한다.

## 경로·미들웨어·OpenAPI

`Register`에 업무별 등록 함수를 추가한다. 등록 함수에는 `/api/v1`을 제외한 상대 경로를 쓴다. `Use(func(http.Handler) http.Handler)`는 전체 공개 라우트에 미들웨어를 적용한다. 등록과 설정을 마친 뒤 `Handler()`를 호출한다. `Handler()`는 호출 시점의 설정으로 라우터를 조립한다.

`framework.New()`의 기본 문서는 이 저장소의 `docs/openapi.yaml`이다. 자체 경로를 추가했다면 자체 OpenAPI JSON을 `SetOpenAPIJSON`으로 지정하여 실제 API와 계약을 일치시킨다. 잘못된 JSON은 오류를 반환한다. 공개 API는 요청의 JSON Content-Type과 본문 크기 제한을 적용한다. 인증이 필요한 서비스에서는 `Use`로 인증 미들웨어를 등록한다.

## DO / DO NOT

| DO | DO NOT |
| --- | --- |
| 공개 `framework`, `httpx`, `docs` 패키지만 import | `internal/*` import |
| 각 서비스의 경로·응답을 OpenAPI에 기록 | 기본 OpenAPI 문서를 자체 경로의 계약으로 간주 |
| 필요한 인증 미들웨어를 `Use`로 등록 | 인증이 자동 적용된다고 가정 |
| 서버에서 `Handler()`를 `http.Server`에 전달 | 프레임워크 소스를 복사 |

기존 `cmd/server`와 `cmd/scheduler`의 설정, DB, SMTP, 예약 메일 조립은 별도 실행 프로그램이다. 공개 API가 이 실행 프로그램의 서비스나 인프라를 자동 생성하지 않는다.

## 검증

본 저장소:

```bash
go mod tidy
go test ./...
go test -race ./...
go vet ./...
go build ./...
make openapi-check
git diff --check
```

외부 프로젝트:

```bash
go mod tidy
go test ./...
go vet ./...
go build ./...
```

새 릴리스 후에는 임시 로컬 프록시나 `replace` 없이 `go get github.com/heartblast/able-rest-api@vX.Y.Z`를 실행하고 외부 프로젝트 검증 명령을 반복한다.
