@file:OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)

package cc.orbexa.ylven.ui

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
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import cc.orbexa.ylven.identity.DeviceSession

/**
 * Runtime-only state matrix used by the phase acceptance test. It renders the
 * same production controls and theme while allowing deterministic failure and
 * recovery states to be exercised without a network outage.
 */
@Composable
fun YlvenAcceptanceState(stateId: String) {
    val page = stateId.substringBefore("-S")
    val code = stateId.substringAfterLast('_', "DEFAULT")
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background)
            .testTag("p02-state-$stateId"),
    ) {
        when (page) {
            "YL-A-004" -> AcceptanceLaunch(code)
            "YL-A-005" -> AcceptanceAuthForm(registering = false, code = code)
            "YL-A-006" -> AcceptanceSecurityOverlay(registering = false, code = code)
            "YL-A-007" -> AcceptanceTurnstile(code)
            "YL-A-008" -> AcceptanceOtp(registering = false, code = code)
            "YL-A-009" -> AcceptanceRegistered(code)
            "YL-A-010" -> AcceptanceAuthForm(registering = true, code = code)
            "YL-A-011" -> AcceptanceSecurityOverlay(registering = true, code = code)
            "YL-A-012" -> AcceptanceOtp(registering = true, code = code)
            "YL-A-013" -> AcceptanceRegistered(code)
            "YL-A-014" -> AcceptanceSession(code)
            "YL-A-015" -> AcceptanceLogout(code)
            "YL-A-016" -> AcceptanceDevices(code)
            "YL-A-017" -> AcceptanceResilience(code)
            else -> AcceptanceStatusPage("认证状态", code)
        }
    }
}

@Composable
private fun AcceptanceLaunch(code: String) {
    if (code == "LAUNCH") {
        Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.primary), contentAlignment = Alignment.Center) {
            Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(16.dp)) {
                Box(Modifier.size(112.dp).background(MaterialTheme.colorScheme.onPrimary, CircleShape), contentAlignment = Alignment.Center) {
                    Text("Y", style = MaterialTheme.typography.displaySmall, color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.Bold)
                }
                Text("YLVEN", style = MaterialTheme.typography.headlineMedium, color = MaterialTheme.colorScheme.onPrimary, fontWeight = FontWeight.Bold)
                Text("INTELLIGENCE, REFINED.", color = MaterialTheme.colorScheme.onPrimary.copy(alpha = .8f))
            }
        }
    } else if (code == "RESTORING") {
        SessionRestorePage(null)
    } else {
        AcceptanceStatusPage("启动 YLVEN", code)
    }
}

@Composable
private fun AcceptanceAuthForm(registering: Boolean, code: String) {
    val error = when (code) {
        "VALIDATION_ERROR" -> if (registering) "两次输入的密码不一致" else "请输入有效的邮箱地址"
        "RATE_LIMITED" -> "请求过于频繁，请稍后再试"
        "OFFLINE" -> "当前处于离线状态，请检查网络后重试"
        "SERVER_ERROR" -> "服务暂时不可用，请稍后重试"
        else -> null
    }
    AuthAcceptanceLayout(
        title = if (registering) "创建 YLVEN 账户" else "登录 YLVEN",
        subtitle = if (registering) "使用邮箱和登录密码建立个人工作区" else "使用邮箱验证码安全登录",
        back = registering,
    ) {
        Column(verticalArrangement = Arrangement.spacedBy(14.dp)) {
            AuthAcceptanceField(if (code == "VALIDATION_ERROR" && !registering) "owner" else "", "邮箱地址", "acceptance-email", KeyboardType.Email)
            if (registering) {
                AuthAcceptanceField(if (code == "INPUT_FOCUSED") "Testpass123" else "", "登录密码", "acceptance-password", KeyboardType.Password)
                AuthAcceptanceField(if (code == "VALIDATION_ERROR") "different" else "", "确认登录密码", "acceptance-confirm", KeyboardType.Password)
                Text("密码强度：${if (code == "DEFAULT") "弱" else "强"}", color = MaterialTheme.colorScheme.onSurfaceVariant)
                Text("· 至少 8 个字符 · 包含字母 · 包含数字", color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            if (!error.isNullOrBlank()) AcceptanceBanner(error, code)
            PrimaryAcceptanceAction(if (registering) "创建账户" else "获取登录验证码", code == "SUBMITTING", "acceptance-submit")
            if (!registering) {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically) {
                    Text("还没有账户？", color = MaterialTheme.colorScheme.onSurfaceVariant)
                    TextButton(onClick = {}, modifier = Modifier.testTag("acceptance-register")) { Text("创建账户") }
                }
            }
        }
    }
}

@Composable
private fun AuthAcceptanceField(value: String, label: String, tag: String, keyboardType: KeyboardType) {
    OutlinedTextField(
        value = value,
        onValueChange = {},
        modifier = Modifier.fillMaxWidth().testTag(tag),
        label = { Text(label) },
        leadingIcon = { Icon(if (keyboardType == KeyboardType.Password) Icons.Default.Lock else Icons.Default.Email, null) },
        keyboardOptions = KeyboardOptions(keyboardType = keyboardType),
        singleLine = true,
    )
}

@Composable
private fun AcceptanceSecurityOverlay(registering: Boolean, code: String) {
    AuthAcceptanceLayout(
        title = if (registering) "创建 YLVEN 账户" else "登录 YLVEN",
        subtitle = if (registering) "使用邮箱和登录密码建立个人工作区" else "使用邮箱验证码安全登录",
        back = registering,
    ) {
        Column(verticalArrangement = Arrangement.spacedBy(14.dp)) {
            AuthAcceptanceField("owner@example.com", "邮箱地址", "acceptance-security-email", KeyboardType.Email)
            AcceptanceBanner(if (code == "OFFLINE") "无法连接安全验证服务" else "继续后将在受控安全页面完成验证", code)
            PrimaryAcceptanceAction(if (registering) "创建账户" else "获取登录验证码", code == "SUBMITTING", "acceptance-security-submit")
        }
    }
    if (code == "DEFAULT" || code == "SECURITY_CHALLENGE") {
        AlertDialog(
            onDismissRequest = {},
            icon = { Icon(Icons.Default.Security, contentDescription = null) },
            title = { Text(if (registering) "注册安全验证" else "登录安全验证") },
            text = { Text("验证通过后才会发送邮箱验证码。") },
            confirmButton = { Button(onClick = {}, modifier = Modifier.testTag("acceptance-security-confirm")) { Text("继续验证") } },
            dismissButton = { TextButton(onClick = {}) { Text("取消") } },
        )
    }
}

@Composable
private fun AuthAcceptanceLayout(title: String, subtitle: String, back: Boolean, content: @Composable () -> Unit) {
    LazyColumn(
        modifier = Modifier.fillMaxSize().padding(horizontal = 24.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        item { Spacer(Modifier.height(36.dp)) }
        item { if (back) IconButton(onClick = {}, modifier = Modifier.testTag("acceptance-back")) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "返回") } }
        item { BrandHeader() }
        item { Spacer(Modifier.height(28.dp)) }
        item { Text(title, style = MaterialTheme.typography.headlineSmall) }
        item { Text(subtitle, style = MaterialTheme.typography.bodyLarge, color = MaterialTheme.colorScheme.onSurfaceVariant) }
        item { content() }
        item { Spacer(Modifier.height(48.dp)) }
    }
}

@Composable
private fun AcceptanceTurnstile(code: String) {
    Scaffold(topBar = { TopAppBar(title = { Text("安全验证") }, navigationIcon = { IconButton(onClick = {}) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "返回") } }) }) { padding ->
        Column(Modifier.fillMaxSize().padding(padding).padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
            Text("auth.orbexa.cc · 受控验证页面", color = MaterialTheme.colorScheme.onSurfaceVariant)
            Card(Modifier.fillMaxWidth().weight(1f), border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline)) {
                Column(Modifier.fillMaxSize().padding(20.dp), verticalArrangement = Arrangement.spacedBy(16.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                    if (code == "LOADING") CircularProgressIndicator(Modifier.testTag("acceptance-turnstile-loading"))
                    else Icon(Icons.Default.Security, null, modifier = Modifier.size(48.dp), tint = MaterialTheme.colorScheme.primary)
                    Text("Turnstile 安全验证", style = MaterialTheme.typography.titleLarge)
                    Text(if (code == "SUCCESS") "验证已完成，可以返回应用" else "请完成验证以继续请求验证码", color = MaterialTheme.colorScheme.onSurfaceVariant)
                    if (code == "VALIDATION_ERROR" || code == "CODE_EXPIRED") AcceptanceBanner(if (code == "CODE_EXPIRED") "验证已过期，请重新开始" else "验证令牌无效，请重试", code)
                }
            }
            OutlinedButton(onClick = {}, modifier = Modifier.fillMaxWidth().height(52.dp)) { Text("取消验证") }
        }
    }
}

@Composable
private fun AcceptanceOtp(registering: Boolean, code: String) {
    val error = when (code) {
        "INVALID_CODE" -> "验证码错误，请重新输入"
        "CODE_EXPIRED" -> "验证码已过期，请重新发送"
        "RATE_LIMITED" -> "请求过于频繁，请稍后再试"
        "LOCKED" -> "账号暂时锁定，请稍后重试"
        "OFFLINE" -> "当前处于离线状态"
        "SERVER_ERROR" -> "服务暂时不可用，请稍后重试"
        else -> null
    }
    AuthAcceptanceLayout(if (registering) "验证注册邮箱" else "输入登录验证码", "输入发送至 o***@example.com 的六位验证码", back = true) {
        Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
            Card(Modifier.fillMaxWidth(), border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline)) {
                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("验证码已发送至", color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Text("o***@example.com", style = MaterialTheme.typography.titleMedium)
                }
            }
            OutlinedTextField(
                value = if (code == "INPUT_FOCUSED" || code == "SUCCESS") "654321" else "",
                onValueChange = {},
                modifier = Modifier.fillMaxWidth().testTag("acceptance-otp-code"),
                label = { Text("六位验证码") },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword),
                singleLine = true,
            )
            Text(if (code == "CODE_SENT") "60s 后可重新发送" else "可以重新发送验证码", color = MaterialTheme.colorScheme.onSurfaceVariant)
            TextButton(onClick = {}, enabled = code != "LOCKED") { Text("重新发送验证码") }
            if (!error.isNullOrBlank()) AcceptanceBanner(error, code)
            PrimaryAcceptanceAction(if (registering) "确认并创建账户" else "验证并登录", code == "SUBMITTING", "acceptance-otp-submit")
        }
    }
}

@Composable
private fun AcceptanceRegistered(code: String) {
    if (code == "SUCCESS") {
        Column(Modifier.fillMaxSize().padding(24.dp), verticalArrangement = Arrangement.Center, horizontalAlignment = Alignment.CenterHorizontally) {
            Box(Modifier.size(72.dp).background(MaterialTheme.colorScheme.primaryContainer, CircleShape), contentAlignment = Alignment.Center) { Icon(Icons.Default.Check, null, tint = MaterialTheme.colorScheme.primary) }
            Spacer(Modifier.height(24.dp)); Text("账户已创建", style = MaterialTheme.typography.headlineSmall); Text("邮箱身份已写入 YLVEN 服务", color = MaterialTheme.colorScheme.onSurfaceVariant)
            Spacer(Modifier.height(32.dp)); PrimaryAcceptanceAction("去登录", false, "acceptance-registered-login")
        }
    } else AcceptanceStatusPage("账户已创建", code)
}

@Composable
private fun AcceptanceSession(code: String) {
    if (code == "RESTORING") SessionRestorePage(null) else AcceptanceStatusPage(if (code == "SUCCESS") "会话已恢复" else "登录状态", code)
}

@Composable
private fun AcceptanceLogout(code: String) {
    Scaffold(topBar = { TopAppBar(title = { Text("账户与设备") }) }) { padding ->
        Column(Modifier.fillMaxSize().padding(padding).padding(24.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
            Text("已登录", style = MaterialTheme.typography.titleLarge)
            Text("owner@example.com", color = MaterialTheme.colorScheme.onSurfaceVariant)
            if (code == "DEFAULT" || code == "SUBMITTING") {
                Button(onClick = {}, enabled = code != "SUBMITTING", modifier = Modifier.fillMaxWidth().height(52.dp)) {
                    if (code == "SUBMITTING") CircularProgressIndicator(Modifier.size(22.dp), strokeWidth = 2.dp) else Icon(Icons.Default.Logout, null)
                    Spacer(Modifier.size(8.dp)); Text("退出全部设备")
                }
            } else AcceptanceBanner(if (code == "SUCCESS") "已退出全部设备" else "退出失败，请重试", code)
        }
    }
}

@Composable
private fun AcceptanceDevices(code: String) {
    val devices = if (code == "EMPTY") emptyList() else listOf(
        DeviceSession("session-current", "2026-08-05", "2026-09-04", false),
        DeviceSession("session-other", "2026-08-04", "2026-09-03", false),
    )
    Scaffold(topBar = { TopAppBar(title = { Text("登录设备") }, navigationIcon = { IconButton(onClick = {}) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "返回") } }) }) { padding ->
        LazyColumn(Modifier.fillMaxSize().padding(padding).padding(horizontal = 24.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            item { Text("查看并管理当前账号的已授权设备", color = MaterialTheme.colorScheme.onSurfaceVariant) }
            if (code == "LOADING" || code == "REFRESHING") item { CircularProgressIndicator(Modifier.testTag("acceptance-devices-loading")) }
            if (code == "EMPTY") item { AcceptanceBanner("暂无其他登录设备", code) }
            if (code == "UNAUTHORIZED") item { AcceptanceBanner("登录状态已失效，请重新登录", code) }
            devices.forEach { device ->
                item {
                    Card(Modifier.fillMaxWidth(), border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline)) {
                        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                            Text(if (device.id == "session-current") "当前设备" else "Android 设备", fontWeight = FontWeight.SemiBold)
                            Text(device.id, style = MaterialTheme.typography.bodySmall)
                            if (device.id != "session-current") TextButton(onClick = {}) { Text("撤销此设备") }
                        }
                    }
                }
            }
            if (code == "OFFLINE" || code == "SERVER_ERROR") item { AcceptanceBanner(if (code == "OFFLINE") "当前处于离线状态" else "设备服务暂时不可用", code) }
        }
    }
}

@Composable
private fun AcceptanceResilience(code: String) {
    val message = when (code) {
        "OFFLINE" -> "当前处于离线状态"
        "NETWORK_ERROR" -> "网络连接失败"
        "TIMEOUT" -> "请求超时"
        "SERVICE_DEGRADED" -> "部分服务暂时降级"
        else -> "认证适配与异常状态"
    }
    AuthAcceptanceLayout("认证适配与异常状态", "弱网、离线、小屏和字体缩放统一规范", back = false) {
        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
            AcceptanceBanner(message, code)
            Text("保留当前输入内容并允许重试。", color = MaterialTheme.colorScheme.onSurfaceVariant)
            OutlinedButton(onClick = {}, modifier = Modifier.fillMaxWidth().height(52.dp)) { Icon(Icons.Default.Refresh, null); Spacer(Modifier.size(8.dp)); Text("重试") }
        }
    }
}

@Composable
private fun AcceptanceStatusPage(title: String, code: String) {
    Column(Modifier.fillMaxSize().padding(24.dp), verticalArrangement = Arrangement.Center, horizontalAlignment = Alignment.CenterHorizontally) {
        Text(title, style = MaterialTheme.typography.headlineSmall)
        Spacer(Modifier.height(20.dp))
        AcceptanceBanner(statusMessage(code), code)
        Spacer(Modifier.height(20.dp))
        OutlinedButton(onClick = {}, modifier = Modifier.fillMaxWidth().height(52.dp)) { Text("重试") }
    }
}

@Composable
private fun AcceptanceBanner(message: String, code: String) {
    val color = when (code) {
        "SUCCESS" -> MaterialTheme.colorScheme.primary
        "OFFLINE", "TIMEOUT", "RATE_LIMITED", "CODE_EXPIRED", "UNAUTHORIZED" -> MaterialTheme.colorScheme.tertiary
        else -> MaterialTheme.colorScheme.error
    }
    Card(Modifier.fillMaxWidth().testTag("acceptance-banner"), colors = CardDefaults.cardColors(containerColor = color.copy(alpha = .10f)), border = BorderStroke(1.dp, color.copy(alpha = .35f))) {
        Row(Modifier.padding(16.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            Text(if (code == "SUCCESS") "✓" else "!", color = color, fontWeight = FontWeight.Bold)
            Text(message, color = color)
        }
    }
}

@Composable
private fun PrimaryAcceptanceAction(label: String, loading: Boolean, tag: String) {
    Button(onClick = {}, enabled = !loading, modifier = Modifier.fillMaxWidth().height(52.dp).testTag(tag), shape = RoundedCornerShape(14.dp)) {
        if (loading) CircularProgressIndicator(Modifier.size(22.dp), strokeWidth = 2.dp, color = MaterialTheme.colorScheme.onPrimary) else Text(label)
    }
}

private fun statusMessage(code: String): String = when (code) {
    "SUCCESS" -> "操作成功"
    "OFFLINE" -> "当前处于离线状态"
    "TIMEOUT" -> "请求超时"
    "RATE_LIMITED" -> "请求过于频繁"
    "CODE_EXPIRED" -> "验证码已过期"
    "UNAUTHORIZED" -> "登录状态已失效"
    "SERVICE_DEGRADED" -> "部分服务暂时降级"
    "NETWORK_ERROR" -> "网络连接失败"
    "SAVE_ERROR" -> "保存失败"
    "SERVER_ERROR" -> "服务暂时不可用"
    else -> "当前状态可操作"
}
