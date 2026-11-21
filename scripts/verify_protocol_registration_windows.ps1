<#
.SYNOPSIS
    Verifies that the Windows protocol handler registration for the `erst`
    URL scheme is present and correctly configured for the current user.

.DESCRIPTION
    This script inspects the per-user (HKCU) protocol registration that the
    application installer / registration step creates for the `erst` scheme.
    It validates that the required registry keys and values exist and that the
    registered open command invokes the protocol handler (and, optionally, the
    expected binary).

    Registry paths checked:
        Registry::HKEY_CURRENT_USER\Software\Classes\erst
            - Must exist.
            - Must contain a non-null `URL Protocol` value.
        Registry::HKEY_CURRENT_USER\Software\Classes\erst\shell\open\command
            - Must exist.
            - Must contain a non-empty default value (the open command).

    The default value of the `shell\open\command` key must reference
    `protocol-handler`. If the ERST_BINARY environment variable is set, the
    command must also reference that binary path (compared after normalizing
    backslashes to forward slashes).

.PREREQUISITES
    - Windows with PowerShell available.
    - The `erst` protocol must already be registered for the current user
      (e.g. by running the application's registration step or installer).
    - No administrator privileges are required; only HKCU is inspected.

.ENVIRONMENT VARIABLES
    ERST_BINARY
        Optional. Absolute path to the expected `erst` binary. When set, the
        registered open command must reference this path. Path separators are
        normalized before comparison, so either `\` or `/` is accepted.

.USAGE
    powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify_protocol_registration_windows.ps1

    With an expected binary:
    $env:ERST_BINARY = 'C:\Program Files\erst\erst.exe'
    powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify_protocol_registration_windows.ps1

.OUTPUTS
    On success, prints exactly:
        Windows protocol registration verified

.EXIT BEHAVIOR
    The script uses `$ErrorActionPreference = 'Stop'` and throws on any
    validation failure, which results in a non-zero exit code. It throws when:
        - The protocol registry key is missing:
              Missing protocol registry key: <path>
        - The open command key is missing:
              Missing protocol open command key: <path>
        - The `URL Protocol` value is missing:
              Missing URL Protocol registry value
        - The default open command is missing or empty:
              Missing default protocol open command
        - ERST_BINARY is set but the command does not reference it:
              Protocol command does not reference expected binary: <binary> (got <command>)
        - The command does not invoke protocol-handler:
              Protocol command does not invoke protocol-handler

.EXAMPLE
    # Success: registration is present and correct.
    PS> powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify_protocol_registration_windows.ps1
    Windows protocol registration verified

.EXAMPLE
    # Failure: the protocol registry key has not been created.
    PS> powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify_protocol_registration_windows.ps1
    Exception: Missing protocol registry key: Registry::HKEY_CURRENT_USER\Software\Classes\erst
    # (non-zero exit code)
#>

$ErrorActionPreference = 'Stop'

$registryPath = 'Registry::HKEY_CURRENT_USER\Software\Classes\erst'
$commandPath = 'Registry::HKEY_CURRENT_USER\Software\Classes\erst\shell\open\command'

if (-not (Test-Path $registryPath)) {
    throw "Missing protocol registry key: $registryPath"
}

if (-not (Test-Path $commandPath)) {
    throw "Missing protocol open command key: $commandPath"
}

$rootKey = Get-Item $registryPath
$urlProtocol = $rootKey.GetValue('URL Protocol', $null)
if ($null -eq $urlProtocol) {
    throw 'Missing URL Protocol registry value'
}

$commandKey = Get-Item $commandPath
$command = $commandKey.GetValue('')
if ([string]::IsNullOrWhiteSpace($command)) {
    throw 'Missing default protocol open command'
}

if ($env:ERST_BINARY) {
    # Normalize paths to use forward slashes for cross-platform comparison
    $normalizedCommand = $command.Replace('\', '/')
    $normalizedBinary = $env:ERST_BINARY.Replace('\', '/')
    
    if ($normalizedCommand -notlike "*$normalizedBinary*") {
        throw "Protocol command does not reference expected binary: $normalizedBinary (got $normalizedCommand)"
    }
}

if ($command -notlike '*protocol-handler*') {
    throw 'Protocol command does not invoke protocol-handler'
}

Write-Host 'Windows protocol registration verified'