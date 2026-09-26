param(
    [Parameter(Position = 0)]
    [ValidateSet("run", "build", "test", "openapi-check", "secretenc")]
    [string]$Target = "build",

    [string]$Config = "configs/app.yaml",
    [string]$Value = "",
    [string]$KeyEnv = "APP_MASTER_KEY"
)

$ErrorActionPreference = "Stop"
$DistDir = "dist"

function Invoke-Step {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [Parameter(Mandatory = $true)]
        [scriptblock]$Action
    )

    Write-Host "==> $Name" -ForegroundColor Cyan
    & $Action
}

function Assert-CommandExists {
    param(
        [Parameter(Mandatory = $true)]
        [string]$CommandName
    )

    if (-not (Get-Command $CommandName -ErrorAction SilentlyContinue)) {
        throw "'$CommandName' 명령을 찾을 수 없습니다."
    }
}

function Invoke-GoBuild {
    if (-not (Test-Path $DistDir)) {
        New-Item -ItemType Directory -Path $DistDir | Out-Null
    }

    go build -o ".\$DistDir\server.exe" ./cmd/server
    go build -o ".\$DistDir\migrate.exe" ./cmd/migrate
    go build -o ".\$DistDir\secretenc.exe" ./cmd/secretenc
}

function Invoke-GoTest {
    go test ./...
}

function Invoke-GoRun {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ConfigPath
    )

    go run ./cmd/server $ConfigPath
}

function Invoke-SecretEnc {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PlainValue,

        [Parameter(Mandatory = $true)]
        [string]$MasterKeyEnv
    )

    if ([string]::IsNullOrWhiteSpace($PlainValue)) {
        throw "secretenc 타깃은 -Value 인자가 필요합니다."
    }

    go run ./cmd/secretenc --value $PlainValue --key-env $MasterKeyEnv
}

Assert-CommandExists -CommandName "go"

switch ($Target) {
    "build" {
        Invoke-Step -Name "Go Build" -Action { Invoke-GoBuild }
    }
    "test" {
        Invoke-Step -Name "Go Test" -Action { Invoke-GoTest }
    }
    "run" {
        Invoke-Step -Name "Run Server" -Action { Invoke-GoRun -ConfigPath $Config }
    }
    "openapi-check" {
        Invoke-Step -Name "Validate OpenAPI" -Action { go test ./docs ./internal/delivery/http/router }
    }
    "secretenc" {
        Invoke-Step -Name "Encrypt Secret" -Action { Invoke-SecretEnc -PlainValue $Value -MasterKeyEnv $KeyEnv }
    }
    default {
        throw "지원하지 않는 타깃입니다: $Target"
    }
}
