[CmdletBinding()]
param([string]$JsonText = '{"name":"demo","enabled":true}')

$data = $JsonText | ConvertFrom-Json
[pscustomobject]@{
    Name = $data.name
    Enabled = $data.enabled
}
