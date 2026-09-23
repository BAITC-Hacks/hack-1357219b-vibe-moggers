param([string]$Model = 'gpt-4.1-mini')
$ErrorActionPreference = 'Stop'
if ($Model -notmatch '^[a-zA-Z0-9][a-zA-Z0-9._:-]+$') { throw 'Use a model ID, not a URL or display name.' }
$backend = Split-Path -Parent $PSScriptRoot
$envFile = Join-Path $backend '.env'
git -C $backend check-ignore -q -- .env
if ($LASTEXITCODE -ne 0) { throw 'Refusing to write a secret: backend/.env must be Git-ignored.' }
if (!(Test-Path -LiteralPath $envFile)) { Copy-Item -LiteralPath (Join-Path $backend '.env.example') -Destination $envFile }
$secret = Read-Host 'OpenAI API key (hidden; never paste it into chat)' -AsSecureString
$ptr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secret)
try {
    $key = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($ptr).Trim()
    if ($key.Length -lt 10 -or $key -match '[\s"''#=]') { throw 'The API key is empty or contains invalid characters.' }
    $updates = [ordered]@{ OPENAI_API_KEY = $key; OPENAI_MODEL = $Model; AI_MODE = 'live' }
    $lines = [System.Collections.Generic.List[string]]::new()
    foreach ($line in [IO.File]::ReadAllLines($envFile)) {
        if ($line -match '^\s*(OPENAI_API_KEY|OPENAI_MODEL|AI_MODE)\s*=') { continue }
        $lines.Add($line)
    }
    foreach ($name in $updates.Keys) { $lines.Add("$name=$($updates[$name])") }
    [IO.File]::WriteAllLines($envFile, $lines, [Text.UTF8Encoding]::new($false))
    Write-Host "Updated ignored backend/.env. Model: $Model. Restart the API to use live mode."
    Write-Host 'From backend, run: go run ./cmd/check-ai (one small billable request).'
} finally {
    [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($ptr)
    $key = $null
    $updates = $null
    $secret.Dispose()
}
