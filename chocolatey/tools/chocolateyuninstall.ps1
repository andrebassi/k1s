$ErrorActionPreference = 'Stop'

$packageName = 'k1s'
$toolsDir = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"
$exePath = Join-Path $toolsDir 'k1s.exe'

Uninstall-BinFile -Name 'k1s' -Path $exePath

if (Test-Path $exePath) {
  Remove-Item $exePath -Force
}
