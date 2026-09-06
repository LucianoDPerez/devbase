# DevBase installer — Windows (PowerShell)
#   irm https://raw.githubusercontent.com/LucianoDPerez/devbase/main/install.ps1 | iex
$ErrorActionPreference = "Stop"

$Repo = "LucianoDPerez/devbase"
$Version = if ($env:DEVBASE_VERSION) { $env:DEVBASE_VERSION } else { "latest" }
$BinDir = if ($env:DEVBIN) { $env:DEVBIN } else { "$env:USERPROFILE\.local\bin" }

if ($Version -eq "latest") {
  $Version = (Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest").tag_name
}
if (-not $Version) { throw "could not resolve version" }

$Zip = "devbase_$($Version.TrimStart('v'))_windows_amd64.zip"
$Base = "https://github.com/$Repo/releases/download/$Version"
$Tmp = Join-Path $env:TEMP ("devbase-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $Tmp | Out-Null
try {
  Write-Host "downloading $Base/$Zip"
  Invoke-WebRequest "$Base/$Zip" -OutFile (Join-Path $Tmp $Zip)
  Invoke-WebRequest "$Base/checksums.txt" -OutFile (Join-Path $Tmp "checksums.txt")
  $Expected = (Select-String -Path (Join-Path $Tmp "checksums.txt") -Pattern "  $([regex]::Escape($Zip))$" | ForEach-Object { $_.Line.Split(" ")[0] })
  $Actual = (Get-FileHash (Join-Path $Tmp $Zip) -Algorithm SHA256).Hash.ToLower()
  if ($Expected -ne $Actual) { throw "checksum mismatch for $Zip" }
  Expand-Archive (Join-Path $Tmp $Zip) -DestinationPath $Tmp -Force
  New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
  Move-Item (Join-Path $Tmp "devbase.exe") (Join-Path $BinDir "devbase.exe") -Force
  Write-Host "installed to $BinDir\devbase.exe"
  if (($env:PATH -split ";") -notcontains $BinDir) {
    Write-Host "NOTE: $BinDir is not in PATH. Add it via System settings or: setx PATH `"$env:PATH;$BinDir`""
  }
  Write-Host ""
  Write-Host "--- your setup ---"
  & (Join-Path $BinDir "devbase.exe") doctor
  Write-Host ""
  Write-Host "next steps - full bootstrap (installs deps, writes rules, wires IDEs):"
  Write-Host "  cd C:\path\to\your\project"
  Write-Host "  devbase setup --dir ."
  Write-Host ""
  Write-Host "Or step by step:"
  Write-Host "  devbase doctor                   # read-only diagnosis"
  Write-Host "  devbase init --dir . --wire      # rules + IDE wiring"
  Write-Host "  devbase gate --dir .             # verify with evidence"
  Write-Host ""
  Write-Host "Each detected IDE (Cursor, VS Code, Claude Code, OpenCode...) is wired"
  Write-Host "automatically. Restart your IDE afterwards so it picks up the config."
} finally {
  Remove-Item $Tmp -Recurse -Force -ErrorAction SilentlyContinue
}
