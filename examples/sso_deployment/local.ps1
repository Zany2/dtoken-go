param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('center', 'alpha', 'beta')]
    [string]$Role
)

# Public local-only credentials; this script always forces loopback development mode.
# 公开的本地演示凭据；此脚本始终使用回环地址及开发模式。
$ErrorActionPreference = 'Stop'
$alphaExchange = 'local-alpha-exchange-' + ('a' * 32)
$alphaSigning = 'local-alpha-signing-' + ('b' * 32)
$betaExchange = 'local-beta-exchange-' + ('c' * 32)
$betaSigning = 'local-beta-signing-' + ('d' * 32)
$env:SSO_DEV = 'true'
$env:SSO_NAMESPACE = 'local-demo'
$env:SSO_REDIS_URL = ''
$env:SSO_VERIFICATION_KEY = ''
$env:SSO_COOKIE_VERIFICATION_KEY = ''
$env:SSO_CENTER_URL = 'http://localhost:9200'
$env:SSO_COOKIE_KEY = 'local-' + $Role + '-cookie-' + ('e' * 32)
$env:SSO_ENABLE_DEMO_LOGIN = 'true'
$env:SSO_DEMO_USER = 'alice'
$env:SSO_DEMO_PASSWORD = 'local-example-password'
$env:SSO_CLIENTS_JSON = @(
    @{clientId='alpha'; clientSecret=$alphaExchange; secretKey=$alphaSigning; requirePkce=$true; modes=@('ticket'); scopes=@('profile'); redirectUris=@('http://localhost:9201/sso/callback')},
    @{clientId='beta'; clientSecret=$betaExchange; secretKey=$betaSigning; requirePkce=$true; modes=@('ticket'); scopes=@('profile'); redirectUris=@('http://localhost:9202/sso/callback')}
) | ConvertTo-Json -Depth 4 -Compress

$processRole = 'client'
switch ($Role) {
    'center' {
        $processRole = 'center'
        $env:SSO_LISTEN = 'localhost:9200'
        $env:SSO_PUBLIC_URL = 'http://localhost:9200'
    }
    'alpha' {
        $env:SSO_LISTEN = 'localhost:9201'
        $env:SSO_PUBLIC_URL = 'http://localhost:9201'
        $env:SSO_CLIENT_ID = 'alpha'
        $env:SSO_CLIENT_SECRET = $alphaExchange
        $env:SSO_SIGNING_KEY = $alphaSigning
    }
    'beta' {
        $env:SSO_LISTEN = 'localhost:9202'
        $env:SSO_PUBLIC_URL = 'http://localhost:9202'
        $env:SSO_CLIENT_ID = 'beta'
        $env:SSO_CLIENT_SECRET = $betaExchange
        $env:SSO_SIGNING_KEY = $betaSigning
    }
}

Push-Location $PSScriptRoot
try {
    go run . -role $processRole
    if ($LASTEXITCODE -ne 0) { throw 'SSO example exited with an error.' }
} finally {
    Pop-Location
}
