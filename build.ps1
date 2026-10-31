<#
.SYNOPSIS
    Build claude-chat2md và đóng gói bản portable (thư mục + file .zip).

.EXAMPLE
    .\build.ps1                 # go vet + go test + build + đóng gói vào dist\
    .\build.ps1 -SkipBuild      # chỉ đóng gói lại từ exe đã build
    .\build.ps1 -NoZip          # không tạo file .zip
#>
[CmdletBinding()]
param(
    [string]$Version,
    [switch]$SkipBuild,
    [switch]$NoZip
)

$ErrorActionPreference = 'Stop'
$Root    = $PSScriptRoot
$AppName = 'claude-chat2md'

# Phiên bản mặc định: file VERSION (nguồn duy nhất)
if (-not $Version) {
    $Version = (Get-Content (Join-Path $Root 'VERSION') -Raw -Encoding UTF8).Trim()
    if (-not $Version) { throw 'VERSION file is empty.' }
}
$ExeSrc  = Join-Path $Root "build\$AppName.exe"
$DistDir = Join-Path $Root 'dist'
$PkgName = "$AppName-portable-$Version"
$PkgDir  = Join-Path $DistDir $PkgName

function Step($msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }

if (-not $SkipBuild) {
    Push-Location $Root
    try {
        Step 'Sinh tài nguyên icon (rsrc_windows_amd64.syso)'
        go run ./tools/mkrsrc
        if ($LASTEXITCODE -ne 0) { throw 'mkrsrc failed.' }

        Step 'go vet + go test'
        go vet ./...
        if ($LASTEXITCODE -ne 0) { throw 'go vet failed.' }
        go test ./...
        if ($LASTEXITCODE -ne 0) { throw 'go test failed.' }

        Step 'go build (windows/amd64, GUI)'
        $env:GOOS = 'windows'; $env:GOARCH = 'amd64'
        go build -trimpath -ldflags "-H=windowsgui -s" -o $ExeSrc .
        if ($LASTEXITCODE -ne 0) { throw 'go build failed.' }
    }
    finally {
        Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue
        Pop-Location
    }
}

if (-not (Test-Path $ExeSrc)) { throw "Exe not found: $ExeSrc" }

Step "Đóng gói: $PkgDir"
if (Test-Path $PkgDir) { Remove-Item $PkgDir -Recurse -Force }
New-Item -ItemType Directory -Path $PkgDir | Out-Null
Copy-Item $ExeSrc $PkgDir

@"
$AppName - phiên bản $Version
================================
- Chạy $AppName.exe, chọn thư mục dữ liệu chat (mặc định %USERPROFILE%\.claude\projects)
  và thư mục lưu file Markdown, rồi bấm "Chuyển đổi".
- Dữ liệu gốc chỉ được đọc, không bị sửa hay xóa.
- File Markdown xuất ra chứa nguyên nội dung chat: xem lại trước khi chia sẻ.
- Nhật ký ở thư mục 'logs' cạnh file exe (hoặc %LOCALAPPDATA%\$AppName\logs nếu không có quyền ghi),
  tự xóa sau 7 ngày. Nút "Mở thư mục log" trong ứng dụng mở thẳng thư mục này.
"@ | Set-Content -Path (Join-Path $PkgDir 'DOC-TRUOC.txt') -Encoding UTF8

if (-not $NoZip) {
    $Zip = Join-Path $DistDir "$PkgName.zip"
    Step "Nén: $Zip"
    if (Test-Path $Zip) { Remove-Item $Zip -Force }
    Compress-Archive -Path $PkgDir -DestinationPath $Zip -CompressionLevel Optimal
}

Step "Hoàn tất: $PkgDir"
