param(
  [string]$BaseUrl = 'https://ai-admin.orbexa.cc'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$AdminCredential = $env:YLVEN_TEST_ADMIN_PASSWORD
$TurnstileToken = $env:YLVEN_TEST_TURNSTILE_TOKEN
if (-not $AdminCredential) { throw 'YLVEN_TEST_ADMIN_PASSWORD is required.' }
if (-not $TurnstileToken) { throw 'YLVEN_TEST_TURNSTILE_TOKEN is required.' }

function Invoke-Api {
  param(
    [string]$Method,
    [string]$Path,
    [object]$Body = $null,
    [string]$Bearer = '',
    [string]$StepUp = '',
    [int[]]$Expected = @(200)
  )
  $headers = @{ Accept = 'application/json' }
  if ($Bearer) { $headers.Authorization = "Bearer $Bearer" }
  if ($StepUp) { $headers['X-Step-Up-Token'] = $StepUp }
  $parameters = @{
    Uri = $BaseUrl.TrimEnd('/') + $Path
    Method = $Method
    Headers = $headers
    UseBasicParsing = $true
  }
  if ($null -ne $Body) {
    $parameters.ContentType = 'application/json; charset=utf-8'
    $parameters.Body = $Body | ConvertTo-Json -Depth 8 -Compress
  }
  try {
    $response = Invoke-WebRequest @parameters
    $status = [int]$response.StatusCode
    $content = [string]$response.Content
  } catch {
    if (-not $_.Exception.Response) { throw }
    $status = [int]$_.Exception.Response.StatusCode
    $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
    try { $content = $reader.ReadToEnd() } finally { $reader.Dispose() }
  }
  if ($status -notin $Expected) { throw "$Method $Path returned $status; expected $($Expected -join ',')" }
  $json = if ($content) { $content | ConvertFrom-Json } else { $null }
  [pscustomobject]@{ Status = $status; Json = $json; Raw = $content }
}

function Confirm-StepUp([string]$Token) {
  $payload = @{}
  $payload.Add('password', $AdminCredential)
  (Invoke-Api POST '/admin/v1/security/step-up' $payload $Token '' @(201)).Json.step_up_token
}

$stamp = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
$email = "p01.public.$stamp@example.com"
$userCredential = 'P01test' + 'pass123'

$health = Invoke-Api GET '/health/ready' $null '' '' @(200)
if ($health.Json.status -ne 'healthy') { throw 'Public readiness is not healthy.' }

Invoke-Api POST '/api/v1/auth/register/challenge' @{ email = $email; purpose = 'register'; turnstile_token = 'invalid-token' } '' '' @(422) | Out-Null
$registration = (Invoke-Api POST '/api/v1/auth/register/challenge' @{ email = $email; purpose = 'register'; turnstile_token = $TurnstileToken } '' '' @(201)).Json
$delivery = (Invoke-Api POST '/api/v1/auth/register/otp/send' @{ challenge_id = $registration.challenge_id } '' '' @(202)).Json
if (-not $delivery.debug_code) { throw 'Staging debug OTP was not returned.' }
Invoke-Api POST '/api/v1/auth/register/otp/verify' @{ challenge_id = $registration.challenge_id; code = '000000' } '' '' @(422) | Out-Null
Invoke-Api POST '/api/v1/auth/register/otp/verify' @{ challenge_id = $registration.challenge_id; code = $delivery.debug_code } '' '' @(200) | Out-Null
$completePayload = @{ challenge_id = $registration.challenge_id; email = $email }
$completePayload.Add('password', $userCredential)
$created = (Invoke-Api POST '/api/v1/auth/register/complete' $completePayload '' '' @(201)).Json

$duplicate = (Invoke-Api POST '/api/v1/auth/register/challenge' @{ email = $email; purpose = 'register'; turnstile_token = $TurnstileToken } '' '' @(201)).Json
$duplicateDelivery = (Invoke-Api POST '/api/v1/auth/register/otp/send' @{ challenge_id = $duplicate.challenge_id } '' '' @(202)).Json
Invoke-Api POST '/api/v1/auth/register/otp/verify' @{ challenge_id = $duplicate.challenge_id; code = $duplicateDelivery.debug_code } '' '' @(200) | Out-Null
$duplicatePayload = @{ challenge_id = $duplicate.challenge_id; email = $email }
$duplicatePayload.Add('password', $userCredential)
Invoke-Api POST '/api/v1/auth/register/complete' $duplicatePayload '' '' @(409) | Out-Null

$unknown = Invoke-Api POST '/api/v1/auth/login/challenge' @{ email = "unknown.$stamp@example.com"; turnstile_token = $TurnstileToken } '' '' @(201)
$login = (Invoke-Api POST '/api/v1/auth/login/challenge' @{ email = $email; turnstile_token = $TurnstileToken } '' '' @(201)).Json
if ($unknown.Status -ne 201) { throw 'Unknown-account response is distinguishable.' }
$loginDelivery = (Invoke-Api POST '/api/v1/auth/login/otp/send' @{ challenge_id = $login.challenge_id } '' '' @(202)).Json
Invoke-Api POST '/api/v1/auth/login/otp/verify' @{ challenge_id = $login.challenge_id; code = $loginDelivery.debug_code } '' '' @(200) | Out-Null
$session = (Invoke-Api POST '/api/v1/auth/sessions' @{ challenge_id = $login.challenge_id; email = $email } '' '' @(201)).Json
$devices = Invoke-Api GET '/api/v1/account/devices' $null $session.access_token '' @(200)
if ($devices.Raw -match 'access_digest|refresh_digest|password_hash') { throw 'Device response leaked a sensitive storage field.' }
$refreshPayload = @{}
$refreshPayload.Add('refresh_token', $session.refresh_token)
$rotated = (Invoke-Api POST '/api/v1/auth/token/refresh' $refreshPayload '' '' @(200)).Json
Invoke-Api POST '/api/v1/auth/token/refresh' $refreshPayload '' '' @(401) | Out-Null
Invoke-Api POST '/api/v1/auth/logout-all' @{} $rotated.access_token '' @(200) | Out-Null
Invoke-Api GET '/api/v1/account/devices' $null $rotated.access_token '' @(401) | Out-Null

$wrongLogin = @{ username = 'admin' }
$wrongLogin.Add('password', ('incorrect' + '-password'))
Invoke-Api POST '/admin/v1/auth/login' $wrongLogin '' '' @(401) | Out-Null
$adminLogin = @{ username = 'admin' }
$adminLogin.Add('password', $AdminCredential)
$admin = (Invoke-Api POST '/admin/v1/auth/login' $adminLogin '' '' @(201)).Json
$users = Invoke-Api GET '/admin/v1/users' $null $admin.access_token '' @(200)
if ($users.Raw -match 'password_hash|access_digest|refresh_digest') { throw 'Admin user list leaked a sensitive storage field.' }
$matched = @($users.Json.users | Where-Object { $_.email -eq $email })
if ($matched.Count -ne 1) { throw 'The newly registered user is missing or duplicated.' }
$detail = Invoke-Api GET "/admin/v1/users/$($created.user_id)" $null $admin.access_token '' @(200)
if ($detail.Raw -match 'password_hash|access_digest|refresh_digest') { throw 'Admin user detail leaked a sensitive storage field.' }

$emailSetting = (Invoke-Api GET '/admin/v1/settings/email' $null $admin.access_token '' @(200)).Json.value
$stepUp = Confirm-StepUp $admin.access_token
Invoke-Api PUT '/admin/v1/settings/email' @{ provider = 'staging-mock'; sender = 'no-reply@orbexa.cc'; secret_reference = 'secret://ylven/staging/email' } $admin.access_token $stepUp @(200) | Out-Null
$stepUp = Confirm-StepUp $admin.access_token
Invoke-Api PUT '/admin/v1/settings/email' $emailSetting $admin.access_token $stepUp @(200) | Out-Null

$stepUp = Confirm-StepUp $admin.access_token
Invoke-Api POST '/admin/v1/rbac/roles' @{ id = 'p01-public-probe'; name = 'P01 Public Probe'; permissions = @('users:read') } $admin.access_token $stepUp @(201) | Out-Null

$templates = (Invoke-Api GET '/admin/v1/notifications/email-templates' $null $admin.access_token '' @(200)).Json.templates
$template = @($templates | Where-Object { $_.key -eq 'register_otp' })[0]
$originalSubject = $template.subject
$stepUp = Confirm-StepUp $admin.access_token
$updatedTemplate = (Invoke-Api PUT '/admin/v1/notifications/email-templates' @{ key = $template.key; subject = "$originalSubject [probe]"; body = $template.body; version = $template.version } $admin.access_token $stepUp @(200)).Json.template
Invoke-Api PUT '/admin/v1/notifications/email-templates' @{ key = $template.key; subject = 'stale'; body = $template.body; version = $template.version } $admin.access_token (Confirm-StepUp $admin.access_token) @(409) | Out-Null
$stepUp = Confirm-StepUp $admin.access_token
Invoke-Api PUT '/admin/v1/notifications/email-templates' @{ key = $template.key; subject = $originalSubject; body = $template.body; version = $updatedTemplate.version } $admin.access_token $stepUp @(200) | Out-Null

$stepUp = Confirm-StepUp $admin.access_token
Invoke-Api POST "/admin/v1/users/$($created.user_id)/status" @{ status = 'disabled' } $admin.access_token $stepUp @(200) | Out-Null
$stepUp = Confirm-StepUp $admin.access_token
Invoke-Api POST "/admin/v1/users/$($created.user_id)/status" @{ status = 'active' } $admin.access_token $stepUp @(200) | Out-Null

Invoke-Api POST '/admin/v1/auth/logout' @{} $admin.access_token '' @(200) | Out-Null
Invoke-Api GET '/admin/v1/users' $null $admin.access_token '' @(403) | Out-Null

Write-Output "P01_PUBLIC_ACCEPTANCE_PASS email=$email"
