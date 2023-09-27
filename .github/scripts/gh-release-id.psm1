function Get-ReleaseId
{
    param
    (
        [Parameter(Mandatory = $true)]
        [string] $tag,
        [Parameter(Mandatory = $true)]
        [Int64] $waitSec
    )
    $url = "https://api.github.com/repos/openrport/openrport/releases?page=1&per_page=5"
    $headers = @{
        "Authorization" = "Bearer " + $env:GITHUB_TOKEN
    }
    $sleep = 20
    $i = 0
    while ($i -le $waitSec)
    {
        $releases = Invoke-RestMethod -Uri $url -Headers $headers
        foreach ($release in $releases)
        {
            if ($release.tag_name -eq $tag)
            {
                # Return the tag id
                $release.id
                return
            }
        }
        Write-Warning "release $($tag) not found yet. Trying again in $($sleep) seconds."
        Start-Sleep -Seconds $sleep
        $i = $i + $sleep
    }
    Write-Error "Release $($tag) not found within $($waitSec) seconds."
    $false
}