$ErrorActionPreference = 'Stop'

$packageName = 'k1s'
$version = '0.1.4'
$url64 = "https://github.com/andrebassi/k1s/releases/download/v$version/k1s-windows-amd64.exe"

$toolsDir = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"
$exePath = Join-Path $toolsDir 'k1s.exe'

$packageArgs = @{
  packageName   = $packageName
  fileFullPath  = $exePath
  url64bit      = $url64
  checksum64    = '' # Will be filled by CI
  checksumType64= 'sha256'
}

Get-ChocolateyWebFile @packageArgs

# Create shim
Install-BinFile -Name 'k1s' -Path $exePath
