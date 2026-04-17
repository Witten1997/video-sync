# 打包命令为 powershell scripts/build.ps1 0.1.2

$ErrorActionPreference = "Stop"

$Version = if ($args[0]) { $args[0] } else {
    $match = Select-String -Path "internal/version/version.go" -Pattern 'Version\s*=\s*"([^"]+)"'
    $match.Matches[0].Groups[1].Value
}

$BuildTime = (Get-Date -Format "yyyy-MM-dd HH:mm:ss")
$LdFlags = "-w -s -X 'bili-download/internal/version.Version=$Version' -X 'bili-download/internal/version.GitTag=$Version' -X 'bili-download/internal/version.BuildTime=$BuildTime'"
$OutputDir = "dist"

Write-Host "=== Building video-sync $Version ==="

# 1. 构建前端
if (Test-Path "web/dist") { Remove-Item -Recurse -Force "web/dist" }
Write-Host "--- Building frontend ---"
Push-Location web
if (!(Test-Path "node_modules")) { npm ci }
npx vite build
Pop-Location

# 2. 清理输出目录
if (Test-Path $OutputDir) { Remove-Item -Recurse -Force $OutputDir }
New-Item -ItemType Directory -Path $OutputDir | Out-Null

# 3. 交叉编译
$targets = @(
    @{ OS="linux";   ARCH="amd64"; BIN="video-sync" },
    @{ OS="linux";   ARCH="arm64"; BIN="video-sync" },
    @{ OS="windows"; ARCH="amd64"; BIN="video-sync.exe" }
)

foreach ($t in $targets) {
    $os = $t.OS; $arch = $t.ARCH; $bin = $t.BIN
    Write-Host "--- Building $os/$arch ---"

    $env:CGO_ENABLED = "0"
    $env:GOOS = $os
    $env:GOARCH = $arch
    go build -ldflags="$LdFlags" -o "$OutputDir/$bin" ./cmd/server

    $archive = "video-sync-$Version-$os-$arch.tar.gz"
    Write-Host "--- Packaging $archive ---"
    tar -czf "$OutputDir/$archive" -C "$OutputDir" "$bin"
    Remove-Item "$OutputDir/$bin"

    Write-Host "    -> $OutputDir/$archive"
}

# 清理环境变量
Remove-Item Env:GOOS
Remove-Item Env:GOARCH
Remove-Item Env:CGO_ENABLED

Write-Host ""
Write-Host "=== Done ==="
Get-ChildItem "$OutputDir/*.tar.gz" | Format-Table Name, @{N="Size(MB)";E={[math]::Round($_.Length/1MB,2)}}
