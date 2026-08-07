package cc.orbexa.ylven.ui

import android.content.Intent
import android.speech.RecognizerIntent
import android.speech.tts.TextToSpeech

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
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Archive
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Mic
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
            )
            IdentityScreen.HOME -> HomePage(gateway, requireNotNull(session), onSessionChange) { conversation -> activeConversation = conversation; screen = IdentityScreen.CHAT }
            IdentityScreen.CHAT -> ChatPage(gateway, requireNotNull(session), requireNotNull(activeConversation), onBack = { screen = IdentityScreen.HOME })
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
                    challenge.debugCode?.let { Text("本次验证码：$it", color = MaterialTheme.colorScheme.primary, modifier = Modifier.testTag("p01-debug-otp")) }
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
@OptIn(ExperimentalMaterial3Api::class)
private fun HomePage(gateway: IdentityGateway, session: AuthSession, onSessionChange: (AuthSession?) -> Unit, onOpenConversation: (Conversation) -> Unit) {
    var snapshot by remember { mutableStateOf<HomeSnapshot?>(null) }
    var conversations by remember { mutableStateOf<List<Conversation>>(emptyList()) }
    var cursor by remember { mutableStateOf<String?>(null) }
    var query by rememberSaveable { mutableStateOf("") }
    var loading by remember { mutableStateOf(true) }
    var drawerOpen by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var renameTarget by remember { mutableStateOf<Conversation?>(null) }
    var renameText by rememberSaveable { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    fun loadHome() {
        scope.launch {
            loading = true; error = null
            try { snapshot = gateway.home(session.bearer); conversations = snapshot?.conversations.orEmpty() }
            catch (e: Exception) { error = e.message ?: "首页加载失败，请重试" }
            finally { loading = false }
        }
    }
    fun loadDrawer(reset: Boolean = false) {
        scope.launch {
            loading = true; error = null
            try {
                val result = gateway.listConversations(session.bearer, if (reset) null else cursor)
                conversations = if (reset) result.first else conversations + result.first
                cursor = result.second
            } catch (e: Exception) { error = e.message ?: "会话历史加载失败，请重试" }
            finally { loading = false }
        }
    }
    LaunchedEffect(session.bearer) { loadHome() }
    LaunchedEffect(query, drawerOpen) {
        if (drawerOpen && query.isNotBlank()) {
            try { conversations = gateway.searchConversations(session.bearer, query) }
            catch (e: Exception) { error = e.message ?: "搜索失败" }
        }
    }
    Scaffold(topBar = {
        TopAppBar(title = { Text("YLVEN") }, actions = {
            IconButton(onClick = { drawerOpen = true; loadDrawer(true) }) { Icon(Icons.Default.Search, "会话") }
            IconButton(onClick = { loadHome() }) { Icon(Icons.Default.Refresh, "刷新") }
        })
    }) { padding ->
        LazyColumn(Modifier.fillMaxSize().padding(padding).padding(horizontal=16.dp), verticalArrangement=Arrangement.spacedBy(12.dp)) {
            item { Card(Modifier.fillMaxWidth(), colors=CardDefaults.cardColors(containerColor=MaterialTheme.colorScheme.primaryContainer)){ Column(Modifier.padding(16.dp),verticalArrangement=Arrangement.spacedBy(8.dp)){ Text("今天想解决什么？",style=MaterialTheme.typography.titleLarge); Text("你的会话和模型已准备好",color=MaterialTheme.colorScheme.onSurfaceVariant); Button(onClick={ scope.launch { loading=true; try { val item=gateway.createConversation(session.bearer); conversations=listOf(item)+conversations } catch(e:Exception){error=e.message?:"无法新建会话"} finally{loading=false} } },modifier=Modifier.fillMaxWidth().height(52.dp)){Icon(Icons.Default.Add,null);Spacer(Modifier.size(8.dp));Text("新建对话")}; OutlinedButton(onClick={ scope.launch { loading=true; try { val item=gateway.createTemporaryConversation(session.bearer); conversations=listOf(item)+conversations } catch(e:Exception){error=e.message?:"无法创建临时会话"} finally{loading=false} } },modifier=Modifier.fillMaxWidth().height(52.dp)){Text("临时对话（不长期保留）")}} } }
            snapshot?.modelCatalog?.takeIf{it.isNotEmpty()}?.let { models -> item { Text("可用模型：${models.joinToString("、")}",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant) } }
            item { Text("最近会话",style=MaterialTheme.typography.titleLarge) }
            if(loading && conversations.isEmpty()) item { CircularProgressIndicator(Modifier.testTag("yl-a-018-loading")) }
            error?.let { item { InlineError(it) } }
            items(conversations, key = { it.id }) { item ->
                Card(
                    onClick = { onOpenConversation(item) },
                    modifier = Modifier.fillMaxWidth(),
                    border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline),
                ) {
                    Row(
                        modifier = Modifier.padding(16.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Column(Modifier.weight(1f)) {
                            Text(item.title, style = MaterialTheme.typography.titleMedium)
                            Text(
                                item.status,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                        IconButton(onClick = {
                            renameTarget = item
                            renameText = item.title
                        }) {
                            Icon(Icons.Default.Edit, "重命名")
                        }
                    }
                }
            }
            item { OutlinedButton(onClick={drawerOpen=true;loadDrawer(true)},modifier=Modifier.fillMaxWidth().height(52.dp)){Text("查看全部会话") } }
        }
    }
    if(drawerOpen) AlertDialog(onDismissRequest={drawerOpen=false},title={Text("会话抽屉")},text={Column(verticalArrangement=Arrangement.spacedBy(8.dp)){OutlinedTextField(query,{query=it},label={Text("搜索会话")},singleLine=true,modifier=Modifier.fillMaxWidth()); conversations.forEach{item->Row(verticalAlignment=Alignment.CenterVertically){Text(item.title,Modifier.weight(1f));IconButton(onClick={renameTarget=item;renameText=item.title}){Icon(Icons.Default.Edit,"重命名")};IconButton(onClick={ scope.launch { try { gateway.archiveConversation(session.bearer,item.id); loadDrawer(true) } catch(e:Exception){ error=e.message ?: "归档失败" } } }){Icon(Icons.Default.Archive,"归档")};IconButton(onClick={ scope.launch { try { gateway.deleteConversation(session.bearer,item.id); loadDrawer(true) } catch(e:Exception){ error=e.message ?: "删除失败" } } }){Icon(Icons.Default.Delete,"删除")}}}; if(cursor!=null)TextButton(onClick={loadDrawer()}){Text("加载更多")} }},confirmButton={TextButton(onClick={drawerOpen=false}){Text("关闭")}})
    renameTarget?.let { target -> AlertDialog(onDismissRequest={renameTarget=null},title={Text("重命名会话")},text={OutlinedTextField(renameText,{renameText=it},singleLine=true,label={Text("标题")})},confirmButton={TextButton(onClick={ scope.launch { try { val updated=gateway.renameConversation(session.bearer,target.id,renameText); conversations=conversations.map{if(it.id==updated.id)updated else it}; renameTarget=null } catch(e:Exception){ error=e.message ?: "重命名失败" } } }){Text("保存")} },dismissButton={TextButton(onClick={renameTarget=null}){Text("取消")}}) }
}

@Composable
@OptIn(ExperimentalMaterial3Api::class)
private fun ChatPage(gateway: IdentityGateway, session: AuthSession, conversation: Conversation, onBack: () -> Unit) {
    var draft by rememberSaveable(conversation.id) { mutableStateOf("") }
    var messages by remember { mutableStateOf<List<cc.orbexa.ylven.identity.MessageRecord>>(emptyList()) }
    var run by remember { mutableStateOf<cc.orbexa.ylven.identity.MessageRun?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    var sending by remember { mutableStateOf(false) }
    var citations by remember { mutableStateOf<Map<String, List<MessageCitation>>>(emptyMap()) }
    var retryBody by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val clipboard = LocalClipboardManager.current
    val context = LocalContext.current
    val cache = remember { ConversationCache(context) }
    val voiceLauncher = rememberLauncherForActivityResult(ActivityResultContracts.StartActivityForResult()) { result ->
        result.data?.getStringArrayListExtra(RecognizerIntent.EXTRA_RESULTS)?.firstOrNull()?.let { draft = it }
    }
    val tts = remember(context) { TextToSpeech(context, null) }
    androidx.compose.runtime.DisposableEffect(tts) { onDispose { tts.shutdown() } }
    LaunchedEffect(conversation.id) { messages = cache.load(conversation.id); runCatching { val snapshot = gateway.loadDraft(session.bearer, conversation.id); draft = snapshot } }
    LaunchedEffect(messages) {
        messages.filter { it.role == "assistant" && !citations.containsKey(it.id) }.forEach { message ->
            runCatching { gateway.messageCitations(session.bearer, message.id) }.getOrNull()?.let { loaded -> citations = citations + (message.id to loaded) }
        }
    }

    fun refreshRun(runId: String) {
        scope.launch {
            try {
                val snapshot = gateway.runStatus(session.bearer, runId)
                run = snapshot.first
                messages = snapshot.second
                cache.save(conversation.id, messages)
            } catch (reason: Exception) { error = reason.message ?: "无法恢复生成状态" }
        }
    }

    Scaffold(topBar = {
        TopAppBar(
            title = { Text(conversation.title) },
            navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "返回") } },
            actions = {
                run?.takeIf { it.status == "streaming" }?.let { active ->
                    TextButton(onClick = { scope.launch { run = gateway.cancelRun(session.bearer, active.id) } }) { Text("停止") }
                }
            },
        )
    }) { padding ->
        Column(Modifier.fillMaxSize().padding(padding).padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            LazyColumn(Modifier.weight(1f).fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                items(messages, key = { it.id }) { message ->
                    Card(Modifier.fillMaxWidth(), border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline)) {
                        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(if (message.role == "user") "你" else "AI", style = MaterialTheme.typography.labelLarge)
                            MessageContent(message.body, Modifier.fillMaxWidth()) { code -> clipboard.setText(AnnotatedString(code)) }
                            if (message.role == "assistant") {
                                Row(horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                                    TextButton(onClick = { clipboard.setText(AnnotatedString(message.body)) }) { Text("复制") }
                                    TextButton(onClick = { scope.launch { try { clipboard.setText(AnnotatedString(gateway.exportConversation(session.bearer, conversation.id))) } catch (reason: Exception) { error = reason.message ?: "导出失败" } } }) { Text("导出") }
                                    TextButton(onClick = { scope.launch { try { gateway.submitFeedback(session.bearer, message.id, "up") } catch (reason: Exception) { error = reason.message ?: "反馈失败" } } }) { Text("赞") }
                                    TextButton(onClick = { scope.launch { try { gateway.submitFeedback(session.bearer, message.id, "down") } catch (reason: Exception) { error = reason.message ?: "反馈失败" } } }) { Text("踩") }
                                    TextButton(onClick = { scope.launch { try { val regenerated = gateway.regenerate(session.bearer, message.id); run = regenerated; refreshRun(regenerated.id) } catch (reason: Exception) { error = reason.message ?: "重答失败" } } }) { Text("重答") }
                                    TextButton(onClick = { scope.launch { try { gateway.speak(session.bearer, message.id); tts.speak(message.body, TextToSpeech.QUEUE_FLUSH, null, message.id) } catch (reason: Exception) { error = reason.message ?: "朗读失败" } } }) { Text("朗读") }
                                }
                                citations[message.id]?.forEach { citation -> Text("来源：${citation.title}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }
                            }
                        }
                    }
                }
                if (sending) item { Text("正在连接并接收回答…", color = MaterialTheme.colorScheme.onSurfaceVariant) }
                run?.let { item { Text("生成状态：${it.status}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) } }
                error?.let { message -> item { InlineError(message) } }
                retryBody?.let { body -> item { TextButton(onClick = { draft = body; retryBody = null }) { Text("将失败内容放回输入框") } } }
            }
            OutlinedTextField(
                value = draft,
                onValueChange = { draft = it },
                label = { Text("输入消息") },
                minLines = 1,
                maxLines = 6,
                modifier = Modifier.fillMaxWidth().height(52.dp),
            )
            OutlinedButton(onClick = { voiceLauncher.launch(Intent(RecognizerIntent.ACTION_RECOGNIZE_SPEECH).apply { putExtra(RecognizerIntent.EXTRA_LANGUAGE_MODEL, RecognizerIntent.LANGUAGE_MODEL_FREE_FORM) }) }, modifier = Modifier.fillMaxWidth().height(52.dp)) { Icon(Icons.Default.Mic, null); Spacer(Modifier.size(8.dp)); Text("语音输入") }
            Button(
                onClick = {
                    val body = draft.trim()
                    if (body.isEmpty()) return@Button
                    scope.launch {
                        sending = true; error = null
                        messages = messages + MessageRecord("optimistic-${System.currentTimeMillis()}", conversation.id, "user", body, "")
                        try {
                            val created = gateway.sendMessage(session.bearer, conversation.id, body)
                            draft = ""; run = created
                            val stream = gateway.runEvents(session.bearer, created.id)
                            run = stream.first
                            refreshRun(created.id)
                            cache.save(conversation.id, messages)
                        } catch (reason: Exception) { error = reason.message ?: "发送失败，请重试"; retryBody = body }
                        finally { runCatching { gateway.saveDraft(session.bearer, conversation.id, draft) }; sending = false }
                    }
                },
                enabled = !sending && draft.isNotBlank(),
                modifier = Modifier.fillMaxWidth().height(52.dp),
            ) { Text(if (sending) "发送中…" else "发送") }
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
