param(
    [Parameter(Position = 0)]
    [ValidateSet("build", "test", "run", "secretenc")]
    [string]$Target = "build",

    [string]$Config = "configs/app.yaml",
    [string]$Value = "",
    [string]$KeyEnv = "APP_MASTER_KEY"
)

$ErrorActionPreference = "Stop"

$ScriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$DistDir = Join-Path $ScriptRoot "dist"
$VendorDir = Join-Path $ScriptRoot "vendor"
$VendorModules = Join-Path $VendorDir "modules.txt"

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
        throw "'$CommandName' command not found."
    }
}

function Assert-OfflineReady {
    if (-not (Test-Path $VendorDir)) {
        throw "vendor directory not found. Run 'go mod vendor' in an online environment first."
    }

    if (-not (Test-Path $VendorModules)) {
        throw "vendor/modules.txt not found. Refresh vendor dependencies in an online environment first."
    }
}

function Invoke-GoOffline {
    param(
        [Parameter(ValueFromRemainingArguments = $true)]
        [string[]]$Arguments
    )

    & go @Arguments

    if ($LASTEXITCODE -ne 0) {
        throw "go command failed with exit code $LASTEXITCODE."
    }
}

function Invoke-GoBuild {
    if (-not (Test-Path $DistDir)) {
        New-Item -ItemType Directory -Path $DistDir | Out-Null
    }

    Invoke-GoOffline -Arguments @("build", "-o", (Join-Path $DistDir "server.exe"), "./cmd/server")
    Invoke-GoOffline -Arguments @("build", "-o", (Join-Path $DistDir "migrate.exe"), "./cmd/migrate")
    Invoke-GoOffline -Arguments @("build", "-o", (Join-Path $DistDir "secretenc.exe"), "./cmd/secretenc")
}

function Invoke-GoTest {
    Invoke-GoOffline -Arguments @("test", "./...")
}

function Invoke-GoRun {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ConfigPath
    )

    Invoke-GoOffline -Arguments @("run", "./cmd/server", $ConfigPath)
}

function Invoke-SecretEnc {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PlainValue,

        [Parameter(Mandatory = $true)]
        [string]$MasterKeyEnv
    )

    if ([string]::IsNullOrWhiteSpace($PlainValue)) {
        throw "The -Value argument is required for the secretenc target."
    }

    Invoke-GoOffline -Arguments @("run", "./cmd/secretenc", "--value", $PlainValue, "--key-env", $MasterKeyEnv)
}

Assert-CommandExists -CommandName "go"
Assert-OfflineReady

$originalEnv = @{
    GOPROXY   = $env:GOPROXY
    GOSUMDB   = $env:GOSUMDB
    GOFLAGS   = $env:GOFLAGS
    GONOSUMDB = $env:GONOSUMDB
}

try {
    Set-Location $ScriptRoot

    $env:GOPROXY = "off"
    $env:GOSUMDB = "off"
    $env:GONOSUMDB = "*"

    if ([string]::IsNullOrWhiteSpace($env:GOFLAGS)) {
        $env:GOFLAGS = "-mod=vendor"
    }
    elseif ($env:GOFLAGS -notmatch "(^|\s)-mod=vendor(\s|$)") {
        $env:GOFLAGS = "$($env:GOFLAGS) -mod=vendor"
    }

    switch ($Target) {
        "build" {
            Invoke-Step -Name "Offline Go Build" -Action { Invoke-GoBuild }
        }
        "test" {
            Invoke-Step -Name "Offline Go Test" -Action { Invoke-GoTest }
        }
        "run" {
            Invoke-Step -Name "Offline Run Server" -Action { Invoke-GoRun -ConfigPath $Config }
        }
        "secretenc" {
            Invoke-Step -Name "Offline Encrypt Secret" -Action { Invoke-SecretEnc -PlainValue $Value -MasterKeyEnv $KeyEnv }
        }
        default {
            throw "Unsupported target: $Target"
        }
    }
}
finally {
    foreach ($name in $originalEnv.Keys) {
        if ($null -eq $originalEnv[$name]) {
            Remove-Item "Env:$name" -ErrorAction SilentlyContinue
        }
        else {
            Set-Item "Env:$name" $originalEnv[$name]
        }
    }
}
