$content = 'abc123'
$result = $content -creplace '([a-z]+)(\d+)', "$1-$2"
Write-Output $result
