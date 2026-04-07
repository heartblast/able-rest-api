#!/usr/bin/env bash

set -euo pipefail

TARGET="${1:-build}"
CONFIG="configs/app.yaml"
VALUE=""
KEY_ENV="APP_MASTER_KEY"

SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIST_DIR="$SCRIPT_ROOT/dist"
VENDOR_DIR="$SCRIPT_ROOT/vendor"
VENDOR_MODULES="$VENDOR_DIR/modules.txt"

ORIGINAL_GOPROXY="${GOPROXY-__UNSET__}"
ORIGINAL_GOSUMDB="${GOSUMDB-__UNSET__}"
ORIGINAL_GOFLAGS="${GOFLAGS-__UNSET__}"
ORIGINAL_GONOSUMDB="${GONOSUMDB-__UNSET__}"

restore_env() {
    if [[ "$ORIGINAL_GOPROXY" == "__UNSET__" ]]; then unset GOPROXY; else export GOPROXY="$ORIGINAL_GOPROXY"; fi
    if [[ "$ORIGINAL_GOSUMDB" == "__UNSET__" ]]; then unset GOSUMDB; else export GOSUMDB="$ORIGINAL_GOSUMDB"; fi
    if [[ "$ORIGINAL_GOFLAGS" == "__UNSET__" ]]; then unset GOFLAGS; else export GOFLAGS="$ORIGINAL_GOFLAGS"; fi
    if [[ "$ORIGINAL_GONOSUMDB" == "__UNSET__" ]]; then unset GONOSUMDB; else export GONOSUMDB="$ORIGINAL_GONOSUMDB"; fi
}

trap restore_env EXIT

print_usage() {
    cat <<'EOF'
Usage:
  ./build.sh [build|test|run|secretenc] [options]

Options:
  --config <path>     Config path for the run target
  --value <value>     Plain value for the secretenc target
  --key-env <name>    Environment variable name for the secretenc target
  -h, --help          Show this help message
EOF
}

log_step() {
    echo "==> $1"
}

assert_command_exists() {
    local command_name="$1"
    if ! command -v "$command_name" >/dev/null 2>&1; then
        echo "'$command_name' command not found." >&2
        exit 1
    fi
}

assert_offline_ready() {
    if [[ ! -d "$VENDOR_DIR" ]]; then
        echo "vendor directory not found. Run 'go mod vendor' in an online environment first." >&2
        exit 1
    fi

    if [[ ! -f "$VENDOR_MODULES" ]]; then
        echo "vendor/modules.txt not found. Refresh vendor dependencies in an online environment first." >&2
        exit 1
    fi
}

go_offline() {
    go "$@"
}

build_targets() {
    mkdir -p "$DIST_DIR"
    go_offline build -o "$DIST_DIR/server" ./cmd/server
    go_offline build -o "$DIST_DIR/migrate" ./cmd/migrate
    go_offline build -o "$DIST_DIR/secretenc" ./cmd/secretenc
}

run_tests() {
    go_offline test ./...
}

run_server() {
    go_offline run ./cmd/server "$CONFIG"
}

run_secretenc() {
    if [[ -z "$VALUE" ]]; then
        echo "The --value argument is required for the secretenc target." >&2
        exit 1
    fi

    go_offline run ./cmd/secretenc --value "$VALUE" --key-env "$KEY_ENV"
}

parse_args() {
    shift || true

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --config)
                [[ $# -ge 2 ]] || { echo "--config requires a value." >&2; exit 1; }
                CONFIG="$2"
                shift 2
                ;;
            --value)
                [[ $# -ge 2 ]] || { echo "--value requires a value." >&2; exit 1; }
                VALUE="$2"
                shift 2
                ;;
            --key-env)
                [[ $# -ge 2 ]] || { echo "--key-env requires a value." >&2; exit 1; }
                KEY_ENV="$2"
                shift 2
                ;;
            -h|--help)
                print_usage
                exit 0
                ;;
            *)
                echo "Unknown argument: $1" >&2
                print_usage >&2
                exit 1
                ;;
        esac
    done
}

assert_command_exists go
assert_offline_ready
parse_args "$@"

cd "$SCRIPT_ROOT"

export GOPROXY="off"
export GOSUMDB="off"
export GONOSUMDB="*"

if [[ -z "${GOFLAGS-}" ]]; then
    export GOFLAGS="-mod=vendor"
elif [[ " ${GOFLAGS} " != *" -mod=vendor "* ]]; then
    export GOFLAGS="${GOFLAGS} -mod=vendor"
fi

case "$TARGET" in
    build)
        log_step "Offline Go Build"
        build_targets
        ;;
    test)
        log_step "Offline Go Test"
        run_tests
        ;;
    run)
        log_step "Offline Run Server"
        run_server
        ;;
    secretenc)
        log_step "Offline Encrypt Secret"
        run_secretenc
        ;;
    -h|--help)
        print_usage
        ;;
    *)
        echo "Unsupported target: $TARGET" >&2
        print_usage >&2
        exit 1
        ;;
esac
