param(
    [Parameter(Mandatory)]
    [ValidatePattern('^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$')]
    [string]$Tag,
    [string]$Compiler = "${env:ProgramFiles(x86)}/Inno Setup 6/ISCC.exe"
)

$ErrorActionPreference = 'Stop'
$projectRoot = (Resolve-Path "$PSScriptRoot/../..").Path
$version = $Tag.Substring(1)
$numericVersion = $version.Split('-')[0]
foreach ($part in $numericVersion.Split('.')) {
    if ([long]$part -gt 65535) { throw 'Version components must be at most 65535 for Windows version information.' }
}
if (-not (Test-Path -LiteralPath $Compiler -PathType Leaf)) { throw "Inno Setup 6 compiler not found: $Compiler" }
foreach ($file in @('build/bin/cli-bridger.exe', 'build/bin/demo-cli.exe', 'README.md', 'LICENSE')) {
    if (-not (Test-Path -LiteralPath "$projectRoot/$file" -PathType Leaf)) { throw "Missing $file; run ./build.ps1 first." }
}

$priorModuleCache = $env:GOMODCACHE
$priorBuildCache = $env:GOCACHE
Push-Location $projectRoot
try {
    $env:GOMODCACHE = Join-Path $projectRoot '.cache/gomod'
    $env:GOCACHE = Join-Path $projectRoot '.cache/go-build'
    # Reuse the bootstrapper bundled with our pinned Wails version.
    go run github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.28 generate webview2bootstrapper -dir build/bin
    if ($LASTEXITCODE) { throw 'WebView2 bootstrapper generation failed.' }

    $output = Join-Path $projectRoot "build/bin/release/$Tag"
    New-Item -ItemType Directory -Force -Path $output | Out-Null
    & $Compiler "/DAppVersion=$version" "/DNumericVersion=$numericVersion" "/O$output" "$PSScriptRoot/installer.iss"
    if ($LASTEXITCODE) { throw 'Installer compilation failed.' }

    $installer = Join-Path $output "CLI-Bridger-$version-windows-x64-setup.exe"
    if (-not (Test-Path -LiteralPath $installer)) { throw 'Installer output is missing.' }
    $zip = Join-Path $output "CLI-Bridger-$version-windows-x64-portable.zip"
    Compress-Archive -Path build/bin/cli-bridger.exe, build/bin/demo-cli.exe, README.md, LICENSE -DestinationPath $zip -Force
    @($installer, $zip) | ForEach-Object {
        $hash = Get-FileHash -LiteralPath $_ -Algorithm SHA256
        '{0}  {1}' -f $hash.Hash.ToLowerInvariant(), (Split-Path $_ -Leaf)
    } | Set-Content -LiteralPath (Join-Path $output 'SHA256SUMS.txt') -Encoding ascii
    Get-ChildItem -LiteralPath $output -File | Select-Object Name, Length
} finally {
    $env:GOMODCACHE = $priorModuleCache
    $env:GOCACHE = $priorBuildCache
    Pop-Location
}
