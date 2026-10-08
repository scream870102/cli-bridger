$ErrorActionPreference = 'Stop'
$priorModuleCache = $env:GOMODCACHE
$priorBuildCache = $env:GOCACHE
Push-Location $PSScriptRoot
try {
    $env:GOMODCACHE = Join-Path $PSScriptRoot '.cache/gomod'
    $env:GOCACHE = Join-Path $PSScriptRoot '.cache/go-build'
    Push-Location frontend
    try {
        npm.cmd ci --cache ../.cache/npm
        if ($LASTEXITCODE) { throw 'npm ci failed' }
        npm.cmd run build
        if ($LASTEXITCODE) { throw 'Frontend build failed' }
    } finally { Pop-Location }
    go test ./...
    if ($LASTEXITCODE) { throw 'Go tests failed' }
    New-Item -ItemType Directory -Force build/bin | Out-Null
    go build -o build/bin/demo-cli.exe ./examples/demo
    if ($LASTEXITCODE) { throw 'Demo build failed' }
    $buildArch = go env GOARCH
    go run github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.28 generate syso -manifest build/windows/app.manifest -icon build/windows/icon.ico -arch $buildArch -out "rsrc_windows_$buildArch.syso"
    if ($LASTEXITCODE) { throw 'Windows resource build failed' }
    go build -tags production -ldflags '-H windowsgui' -o build/bin/cli-bridger.exe .
    if ($LASTEXITCODE) { throw 'Desktop build failed' }
} finally {
    $env:GOMODCACHE = $priorModuleCache
    $env:GOCACHE = $priorBuildCache
    Pop-Location
}
