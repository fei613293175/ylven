package cc.orbexa.ylven.ui

import android.annotation.SuppressLint
import android.net.Uri
import android.os.Handler
import android.os.Looper
import android.webkit.JavascriptInterface
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Email
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Logout
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Security
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.ui.viewinterop.AndroidView
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import cc.orbexa.ylven.identity.AuthSession
import cc.orbexa.ylven.identity.DeviceSession
import cc.orbexa.ylven.identity.IdentityGateway
import cc.orbexa.ylven.identity.OtpChallenge
import cc.orbexa.ylven.identity.PasswordPolicy
import cc.orbexa.ylven.identity.SecurityChallenge
import cc.orbexa.ylven.ui.theme.YlvenDimensions
import kotlinx.coroutines.launch

private enum class IdentityScreen {
    RESTORING,
    LOGIN,
    REGISTER,
    LOGIN_SECURITY,
    REGISTER_SECURITY,
    TURNSTILE,
    LOGIN_OTP,
    REGISTER_OTP,
    REGISTERED,
    ACCOUNT,
}

@Composable
fun YlvenApp(
    gateway: IdentityGateway,
    initialSession: AuthSession? = null,
    onSessionChange: (AuthSession?) -> Unit = {},
) {
    var screen by rememberSaveable { mutableStateOf(if (initialSession == null) IdentityScreen.LOGIN else IdentityScreen.RESTORING) }
    var session by remember { mutableStateOf(initialSession) }
    var challenge by remember { mutableStateOf<OtpChallenge?>(null) }
    var securityChallenge by remember { mutableStateOf<SecurityChallenge?>(null) }
    var pendingPassword by rememberSaveable { mutableStateOf("") }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var securityAction by remember { mutableStateOf<(() -> Unit)?>(null) }
    var passwordPolicy by remember { mutableStateOf(PasswordPolicy()) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(initialSession) {
        val local = initialSession ?: return@LaunchedEffect
        try {
            session = gateway.restore(local)
            screen = IdentityScreen.ACCOUNT
        } catch (first: Exception) {
            runCatching { gateway.refresh(local) }
                .onSuccess { refreshed ->
                    session = refreshed
                    onSessionChange(refreshed)
                    screen = IdentityScreen.ACCOUNT
                }
                .onFailure {
                    session = null
                    onSessionChange(null)
                    error = first.message ?: "登录状态已失效，请重新登录"
                    screen = IdentityScreen.LOGIN
                }
        }
    }

    LaunchedEffect(Unit) { runCatching { passwordPolicy = gateway.passwordPolicy() } }

    fun runRequest(block: suspend () -> Unit) {
        loading = true
        error = null
        scope.launch {
            try {
                block()
            } catch (reason: Exception) {
                error = reason.message ?: "请求失败，请稍后重试"
            } finally {
                loading = false
            }
        }
    }

    Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background).testTag("p01-auth-root")) {
        when (screen) {
            IdentityScreen.RESTORING -> SessionRestorePage(error)
            IdentityScreen.LOGIN -> LoginPage(
                loading = loading,
                error = error,
                onRegister = { error = null; screen = IdentityScreen.REGISTER },
                onSendCode = { email ->
                    runRequest {
                        securityChallenge = gateway.createLoginChallenge(email)
                        securityAction = {
                            securityAction = null
                            challenge = securityChallenge?.legacyOtp
                            screen = if (securityChallenge?.legacyOtp != null) IdentityScreen.LOGIN_OTP else IdentityScreen.TURNSTILE
                        }
                        screen = IdentityScreen.LOGIN_SECURITY
                    }
                },
            )
            IdentityScreen.REGISTER -> RegisterPage(
                loading = loading,
                error = error,
                policy = passwordPolicy,
                onBack = { error = null; screen = IdentityScreen.LOGIN },
                onCreate = { email, password ->
                    pendingPassword = password
                    runRequest {
                        securityChallenge = gateway.createRegistrationChallenge(email)
                        securityAction = {
                            securityAction = null
                            challenge = securityChallenge?.legacyOtp
                            screen = if (securityChallenge?.legacyOtp != null) IdentityScreen.REGISTER_OTP else IdentityScreen.TURNSTILE
                        }
                        screen = IdentityScreen.REGISTER_SECURITY
                    }
                },
            )
            IdentityScreen.LOGIN_SECURITY, IdentityScreen.REGISTER_SECURITY -> Unit
            IdentityScreen.TURNSTILE -> TurnstilePage(
                challenge = requireNotNull(securityChallenge),
                error = error,
                onCancel = {
                    val destination = if (securityChallenge?.purpose == "register") IdentityScreen.REGISTER else IdentityScreen.LOGIN
                    error = null
                    securityChallenge = null
                    screen = destination
                },
                onToken = { token ->
                    val active = requireNotNull(securityChallenge)
                    runRequest {
                        gateway.verifyTurnstile(active, token)
                        challenge = if (active.purpose == "register") gateway.requestRegistrationOtp(active) else gateway.requestLoginOtp(active)
                        screen = if (active.purpose == "register") IdentityScreen.REGISTER_OTP else IdentityScreen.LOGIN_OTP
                    }
                },
            )
            IdentityScreen.LOGIN_OTP, IdentityScreen.REGISTER_OTP -> OtpPage(
                registering = screen == IdentityScreen.REGISTER_OTP,
                challenge = requireNotNull(challenge),
                loading = loading,
                error = error,
                onBack = { error = null; screen = if (screen == IdentityScreen.REGISTER_OTP) IdentityScreen.REGISTER else IdentityScreen.LOGIN },
                onResend = {
                    val active = requireNotNull(securityChallenge)
                    runRequest {
                        challenge = if (active.purpose == "register") gateway.requestRegistrationOtp(active) else gateway.requestLoginOtp(active)
                    }
                },
                onSubmit = { code ->
                    val active = requireNotNull(challenge)
                    runRequest {
                        if (screen == IdentityScreen.REGISTER_OTP) {
                            val registeredSession = gateway.finishRegistrationSession(active, code, pendingPassword)
                            pendingPassword = ""
                            if (registeredSession != null) {
                                session = registeredSession
                                onSessionChange(registeredSession)
                                gateway.initializeWorkspace(registeredSession)
                                screen = IdentityScreen.ACCOUNT
                            } else screen = IdentityScreen.REGISTERED
                        } else {
                            session = gateway.finishLogin(active, code)
                            onSessionChange(session)
                            screen = IdentityScreen.ACCOUNT
                        }
                    }
                },
            )
            IdentityScreen.REGISTERED -> RegisteredPage { screen = IdentityScreen.LOGIN }
            IdentityScreen.ACCOUNT -> AccountPage(
                gateway = gateway,
                session = requireNotNull(session),
                onSessionUpdated = { updated -> session = updated; onSessionChange(updated) },
                onLoggedOut = { session = null; onSessionChange(null); screen = IdentityScreen.LOGIN },
            )
        }

        if (securityAction != null) {
            AlertDialog(
                modifier = Modifier.testTag("p01-security-dialog"),
                onDismissRequest = { securityAction = null },
                icon = { Icon(Icons.Default.Security, contentDescription = null) },
                title = { Text(if (screen == IdentityScreen.REGISTER_SECURITY) "注册安全验证" else "登录安全验证") },
                text = { Text("继续后将在受控安全页面完成 Turnstile 验证，验证通过后才会发送邮箱验证码。") },
                confirmButton = { Button(onClick = { securityAction?.invoke() }, modifier = Modifier.testTag("p01-security-confirm")) { Text("继续验证") } },
                dismissButton = { TextButton(onClick = { securityAction = null }) { Text("取消") } },
            )
        }
    }
}

@Composable
private fun SessionRestorePage(error: String?) {
    Column(
        Modifier.fillMaxSize().padding(horizontal = 24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        CircularProgressIndicator(modifier = Modifier.testTag("p02-session-restoring"))
        Spacer(Modifier.height(20.dp))
        Text("正在恢复登录状态", style = MaterialTheme.typography.titleLarge)
        if (!error.isNullOrBlank()) InlineError(error)
    }
}

@Composable
private fun TurnstilePage(
    challenge: SecurityChallenge,
    error: String?,
    onCancel: () -> Unit,
    onToken: (String) -> Unit,
) {
    var submitted by remember(challenge.id) { mutableStateOf(false) }
    Column(
        Modifier.fillMaxSize().padding(horizontal = 16.dp, vertical = 24.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = onCancel, modifier = Modifier.testTag("p02-turnstile-cancel")) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "返回")
            }
            Text("安全验证", style = MaterialTheme.typography.titleLarge)
        }
        Text("请在受控安全页面完成验证。验证令牌仅用于当前登录请求。", color = MaterialTheme.colorScheme.onSurfaceVariant)
        ControlledTurnstileWebView(
            challenge = challenge,
            modifier = Modifier.fillMaxWidth().weight(1f).testTag("p02-turnstile-webview"),
            onToken = { token ->
                if (!submitted) {
                    submitted = true
                    onToken(token)
                }
            },
        )
        InlineError(error)
        OutlinedButton(onClick = onCancel, modifier = Modifier.fillMaxWidth().height(52.dp)) { Text("取消验证") }
    }
}

@SuppressLint("SetJavaScriptEnabled")
@Composable
private fun ControlledTurnstileWebView(
    challenge: SecurityChallenge,
    modifier: Modifier,
    onToken: (String) -> Unit,
) {
    var webView: WebView? = null
    AndroidView(
        modifier = modifier,
        factory = { context ->
            WebView(context).apply {
                webView = this
                settings.javaScriptEnabled = true
                settings.domStorageEnabled = true
                settings.allowFileAccess = false
                settings.allowContentAccess = false
                settings.setSupportMultipleWindows(false)
                webViewClient = object : WebViewClient() {
                    override fun shouldOverrideUrlLoading(view: WebView, request: WebResourceRequest): Boolean {
                        return !isAllowedSecurityUrl(request.url)
                    }

                    @Deprecated("Deprecated in API 24")
                    override fun shouldOverrideUrlLoading(view: WebView, url: String): Boolean {
                        return !isAllowedSecurityUrl(Uri.parse(url))
                    }
                }
                addJavascriptInterface(object {
                    @JavascriptInterface
                    fun onTurnstileToken(token: String) {
                        val trimmed = token.trim()
                        if (trimmed.isNotEmpty()) Handler(Looper.getMainLooper()).post { onToken(trimmed) }
                    }
                }, "YlvenSecurity")
                val target = Uri.Builder()
                    .scheme("https")
                    .authority("auth.orbexa.cc")
                    .path("/security/turnstile")
                    .appendQueryParameter("challenge_id", challenge.id)
                    .appendQueryParameter("action", challenge.purpose)
                    .build()
                loadUrl(target.toString())
            }
        },
    )
    DisposableEffect(challenge.id) {
        onDispose {
            webView?.apply {
                stopLoading()
                removeJavascriptInterface("YlvenSecurity")
                destroy()
            }
            webView = null
        }
    }
}

private fun isAllowedSecurityUrl(uri: Uri): Boolean {
    if (uri.scheme != "https") return false
    return uri.host.equals("auth.orbexa.cc", ignoreCase = true) ||
        uri.host.equals("challenges.cloudflare.com", ignoreCase = true)
}

@Composable
private fun BrandHeader() {
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        Box(Modifier.size(40.dp).background(MaterialTheme.colorScheme.primary, CircleShape), contentAlignment = Alignment.Center) {
            Text("Y", color = MaterialTheme.colorScheme.onPrimary, fontWeight = FontWeight.Bold)
        }
        Column {
            Text("YLVEN", style = MaterialTheme.typography.titleLarge)
            Text("安全、统一的多模型 AI 工作台", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@Composable
private fun AuthLayout(
    title: String,
    subtitle: String,
    back: (() -> Unit)? = null,
    content: @Composable () -> Unit,
) {
    LazyColumn(
        modifier = Modifier.fillMaxSize().padding(horizontal = 24.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        item { Spacer(Modifier.height(36.dp)) }
        item { if (back != null) IconButton(onClick = back, modifier = Modifier.testTag("auth-back")) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "返回") } }
        item { BrandHeader() }
        item { Spacer(Modifier.height(44.dp)) }
        item { Text(title, style = MaterialTheme.typography.headlineSmall) }
        item { Text(subtitle, style = MaterialTheme.typography.bodyLarge, color = MaterialTheme.colorScheme.onSurfaceVariant) }
        item { Spacer(Modifier.height(8.dp)) }
        item { content() }
        item { Spacer(Modifier.height(48.dp)) }
    }
}

@Composable
private fun LoginPage(
    loading: Boolean,
    error: String?,
    onRegister: () -> Unit,
    onSendCode: (String) -> Unit,
) {
    var email by rememberSaveable { mutableStateOf("") }
    var validation by remember { mutableStateOf<String?>(null) }
    AuthLayout("登录 YLVEN", "使用邮箱验证码安全登录") {
        Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
            OutlinedTextField(
                value = email,
                onValueChange = { email = it; validation = null },
                modifier = Modifier.fillMaxWidth().testTag("p01-login-email"),
                label = { Text("邮箱地址") },
                leadingIcon = { Icon(Icons.Default.Email, null) },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email),
                singleLine = true,
            )
            InlineError(validation ?: error)
            PrimaryAction("获取登录验证码", loading, "p01-login-send") {
                if (!email.looksLikeEmail()) validation = "请输入有效的邮箱地址" else onSendCode(email.trim())
            }
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically) {
                Text("还没有账户？", color = MaterialTheme.colorScheme.onSurfaceVariant)
                TextButton(onClick = onRegister, modifier = Modifier.testTag("p01-go-register")) { Text("创建账户") }
            }
        }
    }
}

@Composable
private fun RegisterPage(
    loading: Boolean,
    error: String?,
    policy: PasswordPolicy,
    onBack: () -> Unit,
    onCreate: (String, String) -> Unit,
) {
    var email by rememberSaveable { mutableStateOf("") }
    var password by rememberSaveable { mutableStateOf("") }
    var confirmation by rememberSaveable { mutableStateOf("") }
    var validation by remember { mutableStateOf<String?>(null) }
    AuthLayout("创建 YLVEN 账户", "使用邮箱和登录密码建立个人工作区", onBack) {
        Column(verticalArrangement = Arrangement.spacedBy(14.dp)) {
            AuthField(email, { email = it; validation = null }, "邮箱地址", "p01-register-email", KeyboardType.Email, false)
            AuthField(password, { password = it; validation = null }, "登录密码", "p01-register-password", KeyboardType.Password, true)
            AuthField(confirmation, { confirmation = it; validation = null }, "确认登录密码", "p01-register-confirm", KeyboardType.Password, true)
            Column(verticalArrangement = Arrangement.spacedBy(6.dp), modifier = Modifier.testTag("p02-password-policy")) {
                val strength = passwordStrength(password, policy)
                Text("密码强度：$strength", style = MaterialTheme.typography.bodySmall, color = if (strength == "强") MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant)
                Text("· 至少 ${policy.minLength} 个字符", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                if (policy.requiresLetter) Text("· 包含字母", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                if (policy.requiresDigit) Text("· 包含数字", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                Text("· 两次密码一致", style = MaterialTheme.typography.bodySmall, color = if (confirmation.isNotEmpty() && password != confirmation) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurfaceVariant)
            }
            InlineError(validation ?: error)
            PrimaryAction("创建账户", loading, "p01-register-submit") {
                validation = when {
                    !email.looksLikeEmail() -> "请输入有效的邮箱地址"
                    password.length < policy.minLength || (policy.requiresLetter && password.none(Char::isLetter)) || (policy.requiresDigit && password.none(Char::isDigit)) -> "密码不符合安全策略"
                    password != confirmation -> "两次输入的密码不一致"
                    else -> null
                }
                if (validation == null) onCreate(email.trim(), password)
            }
        }
    }
}

@Composable
private fun AuthField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    tag: String,
    keyboardType: KeyboardType,
    password: Boolean,
) {
    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        modifier = Modifier.fillMaxWidth().testTag(tag),
        label = { Text(label) },
        leadingIcon = { Icon(if (password) Icons.Default.Lock else Icons.Default.Email, null) },
        keyboardOptions = KeyboardOptions(keyboardType = keyboardType, imeAction = ImeAction.Next),
        visualTransformation = if (password) PasswordVisualTransformation() else androidx.compose.ui.text.input.VisualTransformation.None,
        singleLine = true,
    )
}

@Composable
private fun OtpPage(
    registering: Boolean,
    challenge: OtpChallenge,
    loading: Boolean,
    error: String?,
    onBack: () -> Unit,
    onResend: () -> Unit,
    onSubmit: (String) -> Unit,
) {
    var code by rememberSaveable(challenge.id) { mutableStateOf(challenge.debugCode.orEmpty()) }
    var secondsRemaining by remember(challenge.id) { mutableStateOf(challenge.resendAfterSeconds) }
    LaunchedEffect(challenge.id, secondsRemaining) {
        if (secondsRemaining > 0) {
            kotlinx.coroutines.delay(1000)
            secondsRemaining -= 1
        }
    }
    val masked = challenge.email.maskEmail()
    AuthLayout(if (registering) "验证注册邮箱" else "输入登录验证码", "输入发送至 $masked 的六位验证码", onBack) {
        Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
            Card(
                modifier = Modifier.fillMaxWidth(),
                border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
            ) {
                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("验证码已发送至", color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Text(masked, style = MaterialTheme.typography.titleMedium)
                    challenge.debugCode?.let { Text("联调验证码：$it", color = MaterialTheme.colorScheme.primary, modifier = Modifier.testTag("p01-debug-otp")) }
                }
            }
            OutlinedTextField(
                value = code,
                onValueChange = { code = it.filter(Char::isDigit).take(6) },
                modifier = Modifier.fillMaxWidth().testTag("p01-otp-code"),
                label = { Text("六位验证码") },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword, imeAction = ImeAction.Done, capitalization = KeyboardCapitalization.None),
                keyboardActions = androidx.compose.foundation.text.KeyboardActions(onDone = { if (code.length == 6) onSubmit(code) }),
                singleLine = true,
            )
            Text(
                if (secondsRemaining > 0) "${secondsRemaining}s 后可重新发送" else "可以重新发送验证码",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.testTag("p02-otp-countdown"),
            )
            TextButton(
                onClick = { secondsRemaining = challenge.resendAfterSeconds; onResend() },
                enabled = secondsRemaining == 0 && !loading,
                modifier = Modifier.testTag("p02-otp-resend"),
            ) { Text("重新发送验证码") }
            InlineError(error)
            PrimaryAction(if (registering) "确认并创建账户" else "验证并登录", loading, "p01-otp-submit") {
                if (code.length == 6) onSubmit(code)
            }
        }
    }
}

@Composable
private fun RegisteredPage(onLogin: () -> Unit) {
    Column(
        Modifier.fillMaxSize().padding(horizontal = 24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(Modifier.size(72.dp).background(MaterialTheme.colorScheme.primaryContainer, CircleShape), contentAlignment = Alignment.Center) {
            Icon(Icons.Default.Check, null, tint = MaterialTheme.colorScheme.primary, modifier = Modifier.size(40.dp))
        }
        Spacer(Modifier.height(24.dp))
        Text("账户已创建", style = MaterialTheme.typography.headlineSmall)
        Text("邮箱身份已写入 YLVEN 服务", color = MaterialTheme.colorScheme.onSurfaceVariant)
        Spacer(Modifier.height(32.dp))
        PrimaryAction("去登录", false, "p01-registered-login", onLogin)
    }
}

@Composable
@OptIn(ExperimentalMaterial3Api::class)
private fun AccountPage(
    gateway: IdentityGateway,
    session: AuthSession,
    onSessionUpdated: (AuthSession) -> Unit,
    onLoggedOut: () -> Unit,
) {
    var devices by remember { mutableStateOf<List<DeviceSession>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun loadDevices() {
        loading = true
        error = null
        scope.launch {
            try { devices = gateway.devices(session.bearer) }
            catch (reason: Exception) { error = reason.message }
            finally { loading = false }
        }
    }
    LaunchedEffect(session.bearer) { loadDevices() }

    Scaffold(
        topBar = { TopAppBar(title = { Text("账户与设备") }, actions = { IconButton(onClick = { loadDevices() }) { Icon(Icons.Default.Refresh, "刷新") } }) },
    ) { padding ->
        LazyColumn(
            Modifier.fillMaxSize().padding(padding).padding(horizontal = YlvenDimensions.PageHorizontal),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Card(Modifier.fillMaxWidth(), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer)) {
                    Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Text("已登录", style = MaterialTheme.typography.titleMedium)
                        Text(session.email)
                        Text("会话 ${session.sessionId.take(12)}…", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            }
            item { Text("登录设备", style = MaterialTheme.typography.titleLarge) }
            if (loading) item { CircularProgressIndicator(Modifier.testTag("p01-devices-loading")) }
            error?.let { item { InlineError(it) } }
            items(devices, key = { it.id }) { device ->
                Card(Modifier.fillMaxWidth(), border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline)) {
                    Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Text(if (device.id == session.sessionId) "当前设备" else "Android 设备", fontWeight = FontWeight.SemiBold)
                        Text(device.id.take(16), style = MaterialTheme.typography.bodySmall)
                        Text(if (device.revoked) "已撤销" else "会话有效", color = if (device.revoked) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.primary)
                        if (device.id != session.sessionId && !device.revoked) {
                            TextButton(onClick = {
                                scope.launch {
                                    try {
                                        gateway.revokeDevice(session.bearer, device.id)
                                        loadDevices()
                                    } catch (reason: Exception) {
                                        error = reason.message ?: "设备撤销失败，请重试"
                                    }
                                }
                            }) { Text("撤销此设备") }
                        }
                    }
                }
            }
            item {
                OutlinedButton(
                    onClick = { scope.launch { try { onSessionUpdated(gateway.refresh(session)) } catch (reason: Exception) { error = reason.message } } },
                    modifier = Modifier.fillMaxWidth().height(52.dp).testTag("p01-refresh-session"),
                ) { Icon(Icons.Default.Refresh, null); Spacer(Modifier.size(8.dp)); Text("刷新登录会话") }
            }
            item {
                OutlinedButton(
                    onClick = {
                        scope.launch {
                            try {
                                gateway.logout(session.bearer, true)
                                onLoggedOut()
                            } catch (reason: Exception) {
                                error = reason.message ?: "退出失败，请重试"
                            }
                        }
                    },
                    modifier = Modifier.fillMaxWidth().height(52.dp).testTag("p01-logout-all"),
                    colors = ButtonDefaults.outlinedButtonColors(contentColor = MaterialTheme.colorScheme.error),
                ) { Icon(Icons.Default.Logout, null); Spacer(Modifier.size(8.dp)); Text("退出全部设备") }
            }
            item { Spacer(Modifier.height(24.dp)) }
        }
    }
}

@Composable
private fun PrimaryAction(label: String, loading: Boolean, tag: String, onClick: () -> Unit) {
    Button(
        onClick = onClick,
        enabled = !loading,
        modifier = Modifier.fillMaxWidth().height(52.dp).testTag(tag),
        shape = RoundedCornerShape(14.dp),
    ) {
        if (loading) CircularProgressIndicator(Modifier.size(22.dp), strokeWidth = 2.dp, color = MaterialTheme.colorScheme.onPrimary)
        else Text(label)
    }
}

@Composable
private fun InlineError(message: String?) {
    if (!message.isNullOrBlank()) Text(message, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall, modifier = Modifier.testTag("p01-inline-error"))
}

private fun passwordStrength(password: String, policy: PasswordPolicy): String {
    val checks = listOf(
        password.length >= policy.minLength,
        !policy.requiresLetter || password.any(Char::isLetter),
        !policy.requiresDigit || password.any(Char::isDigit),
        password.any { !it.isLetterOrDigit() },
    )
    return when (checks.count { it }) { 4 -> "强"; 2, 3 -> "中"; else -> "弱" }
}

private fun String.looksLikeEmail() = trim().matches(Regex("^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$"))
private fun String.maskEmail(): String {
    val parts = split('@', limit = 2)
    if (parts.size != 2) return this
    return parts[0].take(1) + "***@" + parts[1]
}
