param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string] $InputPath,

    [string] $Output = "D:\Meetings\Processed",
    [string] $Model = (Join-Path $PSScriptRoot "models\ggml-small.en.bin"),
    [string] $Language = "en",

    [ValidateSet("none", "heuristic", "ollama")]
    [string] $NotesProvider = "none",

    [string] $OllamaModel = ""
)

$ErrorActionPreference = "Stop"

$exe = Join-Path $PSScriptRoot "meeting-transcriber.exe"
if (-not (Test-Path -LiteralPath $exe)) {
    throw "meeting-transcriber.exe was not found at $exe. Build it with: go build -o .\meeting-transcriber.exe .\cmd\meeting-transcriber"
}

$argsList = @(
    "run",
    "--input", $InputPath,
    "--output", $Output,
    "--model", $Model,
    "--language", $Language,
    "--notes-provider", $NotesProvider
)

if ($NotesProvider -eq "ollama") {
    if ([string]::IsNullOrWhiteSpace($OllamaModel)) {
        throw "--OllamaModel is required when -NotesProvider ollama"
    }
    $argsList += @("--ollama-model", $OllamaModel)
}

& $exe @argsList
exit $LASTEXITCODE
