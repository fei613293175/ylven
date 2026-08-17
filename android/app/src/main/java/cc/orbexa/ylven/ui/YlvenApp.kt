package cc.orbexa.ylven.ui

import android.content.Intent
import android.speech.RecognizerIntent
import android.speech.tts.TextToSpeech

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
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
import androidx.compose.foundation.rememberScrollState
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
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Archive
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.Person
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
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.compose.BackHandler
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import cc.orbexa.ylven.identity.AuthSession
import cc.orbexa.ylven.identity.Conversation
import cc.orbexa.ylven.identity.HomeSnapshot
import cc.orbexa.ylven.identity.DeviceSession
import cc.orbexa.ylven.identity.IdentityGateway
import cc.orbexa.ylven.identity.OtpChallenge
import cc.orbexa.ylven.identity.PasswordPolicy
import cc.orbexa.ylven.identity.SecurityChallenge
import cc.orbexa.ylven.identity.MessageRecord
import cc.orbexa.ylven.identity.MessageCitation
import cc.orbexa.ylven.identity.ConversationCache
import cc.orbexa.ylven.identity.consumerStatusText
import cc.orbexa.ylven.identity.isGenerating
import cc.orbexa.ylven.ui.theme.YlvenDimensions
import cc.orbexa.ylven.ui.theme.YlvenLightColors
import kotlinx.coroutines.launch

private enum class IdentityScreen {
    RESTORING,
    LOGIN,
    REGISTER,
    LOGIN_SECURITY,
    REGISTER_SECURITY,
    LOGIN_OTP,
    REGISTER_OTP,
    REGISTERED,
    ACCOUNT,
    P04_HUB,
    P04_AI_SETTINGS,
    P04_CONVERSATION_SETTINGS,
    P04_BRANCHES,
    P04_COMPARISON,
    P04_SERVICE_STATUS,
    HOME,
    CHAT,
}

@Composable
fun YlvenApp(
    gateway: IdentityGateway,
    initialSession: AuthSession? = null,
    onSessionChange: (AuthSession?) -> Unit = {},
) {
    var screen by rememberSaveable { mutableStateOf(if (initialSession == null) IdentityScreen.LOGIN else IdentityScreen.RESTORING) }
    var session by remember { mutableStateOf(initialSession) }
    var activeConversation by remember { mutableStateOf<Conversation?>(null) }
    var p04ReturnsToChat by rememberSaveable { mutableStateOf(false) }
    var challenge by remember { mutableStateOf<OtpChallenge?>(null) }
    var securityChallenge by remember { mutableStateOf<SecurityChallenge?>(null) }
    var pendingPassword by rememberSaveable { mutableStateOf("") }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var securityAction by remember { mutableStateOf<(() -> Unit)?>(null) }
    var securityAnswer by rememberSaveable { mutableStateOf("") }
    var passwordPolicy by remember { mutableStateOf(PasswordPolicy()) }
    var splashVisible by remember { mutableStateOf(true) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(initialSession) {
        val local = initialSession ?: return@LaunchedEffect
        try {
            session = gateway.restore(local)
            screen = IdentityScreen.HOME
        } catch (first: Exception) {
            runCatching { gateway.refresh(local) }
                .onSuccess { refreshed ->
                    session = refreshed
                    onSessionChange(refreshed)
                    screen = IdentityScreen.HOME
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
    LaunchedEffect(Unit) {
        kotlinx.coroutines.delay(3_000)
        splashVisible = false
    }

    val backTarget = when (screen) {
        IdentityScreen.ACCOUNT -> IdentityScreen.HOME
        IdentityScreen.P04_HUB -> IdentityScreen.ACCOUNT
        IdentityScreen.P04_AI_SETTINGS -> IdentityScreen.P04_HUB
        IdentityScreen.P04_CONVERSATION_SETTINGS,
        IdentityScreen.P04_BRANCHES,
        IdentityScreen.P04_COMPARISON -> if (p04ReturnsToChat) IdentityScreen.CHAT else IdentityScreen.P04_HUB
        IdentityScreen.P04_SERVICE_STATUS -> IdentityScreen.P04_HUB
        else -> null
    }
    BackHandler(enabled = !splashVisible && backTarget != null) {
        screen = requireNotNull(backTarget)
    }

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
        if (splashVisible) SplashPage() else when (screen) {
            IdentityScreen.RESTORING -> SessionRestorePage(error)
            IdentityScreen.LOGIN -> LoginPage(
                loading = loading,
                error = error,
                onRegister = { error = null; screen = IdentityScreen.REGISTER },
                onSendCode = { email ->
                    runRequest {
                        securityChallenge = gateway.createLoginChallenge(email)
                        securityAction = {
                            val active = requireNotNull(securityChallenge)
                            runRequest {
                                gateway.verifyAnswer(active, securityAnswer)
                                challenge = gateway.requestLoginOtp(active)
                                securityAction = null
                                securityAnswer = ""
                                screen = IdentityScreen.LOGIN_OTP
                            }
                        }
                        securityAnswer = ""
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
                            val active = requireNotNull(securityChallenge)
                            runRequest {
                                gateway.verifyAnswer(active, securityAnswer)
                                challenge = gateway.requestRegistrationOtp(active)
                                securityAction = null
                                securityAnswer = ""
                                screen = IdentityScreen.REGISTER_OTP
                            }
                        }
                        securityAnswer = ""
                        screen = IdentityScreen.REGISTER_SECURITY
                    }
                },
            )
            IdentityScreen.LOGIN_SECURITY, IdentityScreen.REGISTER_SECURITY -> Unit
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
                                screen = IdentityScreen.HOME
                            } else screen = IdentityScreen.REGISTERED
                        } else {
                            session = gateway.finishLogin(active, code)
                            onSessionChange(session)
                            screen = IdentityScreen.HOME
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
                onBack = { screen = IdentityScreen.HOME },
                onOpenP04 = { p04ReturnsToChat = false; screen = IdentityScreen.P04_HUB },
            )
            IdentityScreen.P04_HUB -> P04HubPage(
                gateway = gateway,
                session = requireNotNull(session),
                onBack = { screen = IdentityScreen.ACCOUNT },
                onAISettings = { screen = IdentityScreen.P04_AI_SETTINGS },
                onServiceStatus = { screen = IdentityScreen.P04_SERVICE_STATUS },
                onConversationSettings = { activeConversation = it; p04ReturnsToChat = false; screen = IdentityScreen.P04_CONVERSATION_SETTINGS },
                onBranches = { activeConversation = it; p04ReturnsToChat = false; screen = IdentityScreen.P04_BRANCHES },
                onComparison = { activeConversation = it; p04ReturnsToChat = false; screen = IdentityScreen.P04_COMPARISON },
            )
            IdentityScreen.P04_AI_SETTINGS -> P04AISettingsPage(
                gateway = gateway,
                session = requireNotNull(session),
                onBack = { screen = IdentityScreen.P04_HUB },
            )
            IdentityScreen.P04_CONVERSATION_SETTINGS -> P04ConversationSettingsPage(
                gateway = gateway,
                session = requireNotNull(session),
                conversation = requireNotNull(activeConversation),
                onBack = { screen = if (p04ReturnsToChat) IdentityScreen.CHAT else IdentityScreen.P04_HUB },
                onUpdated = { activeConversation = it; screen = if (p04ReturnsToChat) IdentityScreen.CHAT else IdentityScreen.P04_HUB },
            )
            IdentityScreen.P04_BRANCHES -> P04BranchesPage(
                gateway = gateway,
                session = requireNotNull(session),
                conversation = requireNotNull(activeConversation),
                onBack = { screen = if (p04ReturnsToChat) IdentityScreen.CHAT else IdentityScreen.P04_HUB },
                onUpdated = { activeConversation = it },
            )
            IdentityScreen.P04_COMPARISON -> P04ComparisonPage(
                gateway = gateway,
                session = requireNotNull(session),
                conversation = requireNotNull(activeConversation),
                onBack = { screen = if (p04ReturnsToChat) IdentityScreen.CHAT else IdentityScreen.P04_HUB },
            )
            IdentityScreen.P04_SERVICE_STATUS -> P04ServiceStatusPage(
                gateway = gateway,
                session = requireNotNull(session),
                onBack = { screen = IdentityScreen.P04_HUB },
            )
            IdentityScreen.HOME -> P03HomePage(
                gateway = gateway,
                session = requireNotNull(session),
                onOpenConversation = { conversation ->
                    activeConversation = conversation
                    screen = IdentityScreen.CHAT
                },
                onOpenAccount = { screen = IdentityScreen.ACCOUNT },
            )
            IdentityScreen.CHAT -> P03ChatPage(
                gateway = gateway,
                session = requireNotNull(session),
                conversation = requireNotNull(activeConversation),
                onBack = { screen = IdentityScreen.HOME },
                onOpenConversation = { conversation ->
                    activeConversation = conversation
                    screen = IdentityScreen.CHAT
                },
                onOpenConversationSettings = { conversation -> activeConversation = conversation; p04ReturnsToChat = true; screen = IdentityScreen.P04_CONVERSATION_SETTINGS },
                onOpenBranches = { conversation -> activeConversation = conversation; p04ReturnsToChat = true; screen = IdentityScreen.P04_BRANCHES },
                onOpenComparison = { conversation -> activeConversation = conversation; p04ReturnsToChat = true; screen = IdentityScreen.P04_COMPARISON },
            )
        }

        if (securityAction != null) {
            AlertDialog(
                modifier = Modifier.testTag("p01-security-dialog"),
                onDismissRequest = {
                    securityAction = null
                    error = null
                    screen = if (securityChallenge?.purpose == "register") IdentityScreen.REGISTER else IdentityScreen.LOGIN
                },
                icon = { Icon(Icons.Default.Security, contentDescription = null) },
                title = { Text("请完成小验证") },
                text = {
                    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                        Text("为了确认是你本人，请算一算下面这道题。")
                        Text(securityChallenge?.question ?: "请按提示完成验证", style = MaterialTheme.typography.titleMedium)
                        OutlinedTextField(
                            value = securityAnswer,
                            onValueChange = { securityAnswer = it.filter(Char::isDigit).take(3) },
                            label = { Text("请输入答案") },
                            singleLine = true,
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                            modifier = Modifier.fillMaxWidth().testTag("p02-security-answer"),
                        )
                        InlineError(error)
                    }
                },
                confirmButton = { Button(onClick = { securityAction?.invoke() }, enabled = securityAnswer.isNotBlank() && !loading, modifier = Modifier.testTag("p01-security-confirm")) { Text(if (loading) "正在验证…" else "确认") } },
                dismissButton = { TextButton(onClick = {
                    securityAction = null
                    error = null
                    screen = if (securityChallenge?.purpose == "register") IdentityScreen.REGISTER else IdentityScreen.LOGIN
                }) { Text("取消") } },
            )
        }
    }
}

@Composable
private fun SplashPage() {
    Box(Modifier.fillMaxSize().testTag("p02-brand-splash")) {
        androidx.compose.foundation.Image(
            painter = androidx.compose.ui.res.painterResource(cc.orbexa.ylven.R.drawable.ylven_splash),
            contentDescription = "YLVEN",
            contentScale = androidx.compose.ui.layout.ContentScale.Crop,
            modifier = Modifier.fillMaxSize(),
        )
        Column(
            modifier = Modifier.align(Alignment.BottomCenter).padding(bottom = 72.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            CircularProgressIndicator(modifier = Modifier.size(30.dp), strokeWidth = 3.dp, color = androidx.compose.ui.graphics.Color.White)
            Text("正在准备你的 YLVEN", color = androidx.compose.ui.graphics.Color.White)
        }
    }
}

@Composable
internal fun SessionRestorePage(error: String?) {
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
internal fun BrandHeader() {
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        androidx.compose.foundation.Image(
            painter = androidx.compose.ui.res.painterResource(cc.orbexa.ylven.R.drawable.ylven_logo),
            contentDescription = "YLVEN",
            modifier = Modifier.size(40.dp),
        )
        Column {
            Text("YLVEN", style = MaterialTheme.typography.titleLarge)
            Text("轻松解决每天的小问题", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
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
    AuthLayout("创建 YLVEN 账户", "设置好密码，马上开始使用", onBack) {
        Column(verticalArrangement = Arrangement.spacedBy(14.dp)) {
            AuthField(email, { email = it; validation = null }, "邮箱地址", "p01-register-email", KeyboardType.Email, false)
            AuthField(password, { password = it; validation = null }, "登录密码", "p01-register-password", KeyboardType.Password, true)
            AuthField(confirmation, { confirmation = it; validation = null }, "确认登录密码", "p01-register-confirm", KeyboardType.Password, true)
            Column(verticalArrangement = Arrangement.spacedBy(6.dp), modifier = Modifier.testTag("p02-password-policy")) {
                val meetsLength = password.length >= policy.minLength
                val meetsLetter = !policy.requiresLetter || password.any(Char::isLetter)
                val meetsDigit = !policy.requiresDigit || password.any(Char::isDigit)
                val matches = confirmation.isNotEmpty() && password == confirmation
                Text("密码要求", style = MaterialTheme.typography.labelLarge)
                PasswordRuleRow(meetsLength, "至少 ${policy.minLength} 个字符", showPending = password.isEmpty())
                if (policy.requiresLetter) PasswordRuleRow(meetsLetter, "包含字母", showPending = password.isEmpty())
                if (policy.requiresDigit) PasswordRuleRow(meetsDigit, "包含数字", showPending = password.isEmpty())
                PasswordRuleRow(matches, "两次输入一致", showPending = confirmation.isEmpty())
            }
            InlineError(validation ?: error)
            PrimaryAction("创建账户", loading, "p01-register-submit") {
                validation = when {
                    !email.looksLikeEmail() -> "请输入有效的邮箱地址"
                    password.length < policy.minLength || (policy.requiresLetter && password.none(Char::isLetter)) || (policy.requiresDigit && password.none(Char::isDigit)) -> "密码还没有满足上面的要求"
                    password != confirmation -> "两次输入的密码不一致"
                    else -> null
                }
                if (validation == null) onCreate(email.trim(), password)
            }
        }
    }
}

@Composable
private fun PasswordRuleRow(satisfied: Boolean, label: String, showPending: Boolean = false) {
    val successColor = YlvenLightColors.Success
    val color = when { satisfied -> successColor; showPending -> MaterialTheme.colorScheme.onSurfaceVariant; else -> MaterialTheme.colorScheme.error }
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(if (satisfied) "✓" else "•", color = color, fontWeight = FontWeight.Bold)
        Text(label, style = MaterialTheme.typography.bodySmall, color = color)
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
                    challenge.debugCode?.let { Text("本次验证码：$it", color = MaterialTheme.colorScheme.primary, modifier = Modifier.testTag("p01-otp-code-preview")) }
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
        Text("现在可以开始使用 YLVEN", color = MaterialTheme.colorScheme.onSurfaceVariant)
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
    onBack: () -> Unit,
    onOpenP04: () -> Unit = {},
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
        topBar = {
            TopAppBar(
                title = { Text("账户与设备") },
                navigationIcon = {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "返回") }
                },
                actions = { IconButton(onClick = { loadDevices() }) { Icon(Icons.Default.Refresh, "刷新") } },
            )
        },
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
                        Text("登录状态正常", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            }
            item { Text("登录设备", style = MaterialTheme.typography.titleLarge) }
            if (loading) item { CircularProgressIndicator(Modifier.testTag("p01-devices-loading")) }
            error?.let { item { InlineError(it) } }
            items(devices, key = { it.id }) { device ->
                Card(Modifier.fillMaxWidth(), border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline)) {
                    Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        val isCurrent = session.deviceId.isNotBlank() && device.id == session.deviceId
                        Text(if (isCurrent) "当前设备" else "其他登录设备", fontWeight = FontWeight.SemiBold)
                        Text(if (device.revoked) "已退出" else "正在使用", color = if (device.revoked) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.primary)
                        if (!isCurrent && !device.revoked) {
                            TextButton(onClick = {
                                scope.launch {
                                    try {
                                        gateway.revokeDevice(session.bearer, device.id)
                                        loadDevices()
                                    } catch (reason: Exception) {
                                        error = reason.message ?: "无法退出这台设备，请重试"
                                    }
                                }
                            }) { Text("退出这台设备") }
                        }
                    }
                }
            }
            item {
                OutlinedButton(
                    onClick = onOpenP04,
                    modifier = Modifier.fillMaxWidth().height(52.dp).testTag("p04-open-workbench"),
                ) { Text("AI 工作台（P04）") }
            }
            item {
                OutlinedButton(
                    onClick = { scope.launch { try { onSessionUpdated(gateway.refresh(session)) } catch (reason: Exception) { error = reason.message } } },
                    modifier = Modifier.fillMaxWidth().height(52.dp).testTag("p01-refresh-session"),
                ) { Icon(Icons.Default.Refresh, null); Spacer(Modifier.size(8.dp)); Text("更新登录状态") }
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

private fun String.looksLikeEmail() = trim().matches(Regex("^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$"))
private fun String.maskEmail(): String {
    val parts = split('@', limit = 2)
    if (parts.size != 2) return this
    return parts[0].take(1) + "***@" + parts[1]
}
