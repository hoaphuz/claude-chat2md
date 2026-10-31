<#
.SYNOPSIS
    Build claude-chat2md và đóng gói bản portable.

.DESCRIPTION
    Bản chạy được ghi đè vào -OutDir (không tách thư mục theo phiên bản; thư mục
    'logs' có sẵn ở đó được giữ nguyên). File .zip kèm số phiên bản vẫn nằm trong dist\.

.EXAMPLE
    .\build.ps1                 # go vet + go test + build + chép vào OutDir + zip vào dist\
    .\build.ps1 -SkipBuild      # chỉ đóng gói lại từ exe đã build
    .\build.ps1 -NoZip          # không tạo file .zip
    .\build.ps1 -OutDir D:\Apps\claude-chat2md   # chép bản chạy vào thư mục khác
#>
[CmdletBinding()]
param(
    [string]$Version,
    [string]$OutDir = 'D:\Program\Portable\claude-chat2md\build',
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

$ReadmeText = @"
$AppName - phiên bản $Version
================================
- Chạy $AppName.exe, chọn thư mục dữ liệu chat (mặc định %USERPROFILE%\.claude\projects)
  và thư mục lưu file Markdown, rồi bấm "Chuyển đổi".
- Dữ liệu gốc chỉ được đọc, không bị sửa hay xóa.
- File Markdown xuất ra chứa nguyên nội dung chat: xem lại trước khi chia sẻ.
- Nhật ký ở thư mục 'logs' cạnh file exe (hoặc %LOCALAPPDATA%\$AppName\logs nếu không có quyền ghi),
  tự xóa sau 7 ngày. Nút "Mở thư mục log" trong ứng dụng mở thẳng thư mục này.
"@

# Chép exe + hướng dẫn vào một thư mục (ghi đè file cũ, không xóa thư mục)
function Write-Package($dir) {
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
    Copy-Item $ExeSrc $dir -Force
    $ReadmeText | Set-Content -Path (Join-Path $dir 'DOC-TRUOC.txt') -Encoding UTF8
}

Step "Chép bản chạy vào: $OutDir"
Write-Package $OutDir

if (-not $NoZip) {
    $Zip = Join-Path $DistDir "$PkgName.zip"
    Step "Nén: $Zip"
    # Chỉ lấy exe + hướng dẫn từ OutDir (bỏ qua logs), đặt trong thư mục tên có phiên bản
    Add-Type -AssemblyName System.IO.Compression, System.IO.Compression.FileSystem
    New-Item -ItemType Directory -Path $DistDir -Force | Out-Null
    if (Test-Path $Zip) { Remove-Item $Zip -Force }
    $Archive = [IO.Compression.ZipFile]::Open($Zip, [IO.Compression.ZipArchiveMode]::Create)
    try {
        foreach ($name in "$AppName.exe", 'DOC-TRUOC.txt') {
            [IO.Compression.ZipFileExtensions]::CreateEntryFromFile(
                $Archive, (Join-Path $OutDir $name), "$PkgName/$name",
                [IO.Compression.CompressionLevel]::Optimal) | Out-Null
        }
    }
    finally {
        $Archive.Dispose()
    }
}

Step "Hoàn tất: $OutDir"
