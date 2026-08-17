@file:OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)

package cc.orbexa.ylven.ui

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
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
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import cc.orbexa.ylven.identity.AIPreference
import cc.orbexa.ylven.identity.AuthSession
import cc.orbexa.ylven.identity.ComparisonGroup
import cc.orbexa.ylven.identity.Conversation
import cc.orbexa.ylven.identity.ConversationBranch
import cc.orbexa.ylven.identity.IdentityGateway
import cc.orbexa.ylven.identity.ModelHealthStatus
import cc.orbexa.ylven.identity.ModelOption
import kotlinx.coroutines.launch

@Composable
internal fun P04HubPage(
    gateway: IdentityGateway,
    session: AuthSession,
    onBack: () -> Unit,
    onAISettings: () -> Unit,
    onServiceStatus: () -> Unit,
    onConversationSettings: (Conversation) -> Unit,
    onBranches: (Conversation) -> Unit,
    onComparison: (Conversation) -> Unit,
) {
    var conversations by remember { mutableStateOf<List<Conversation>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    fun load() { scope.launch { loading = true; error = null; runCatching { gateway.listConversations(session.bearer).first }.onSuccess { conversations = it }.onFailure { error = it.message ?: "会话加载失败" }; loading = false } }
    LaunchedEffect(session.bearer) { load() }
    Scaffold(topBar = { TopAppBar(title = { Text("AI 工作台") }, navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "返回") } }, actions = { IconButton(onClick = ::load) { Icon(Icons.Default.Refresh, "刷新") } }) }) { padding ->
        LazyColumn(Modifier.fillMaxSize().padding(padding).padding(horizontal = 20.dp), verticalArrangement = Arrangement.spacedBy(12.dp), contentPadding = androidx.compose.foundation.layout.PaddingValues(vertical = 18.dp)) {
            item { Text("模型与运行", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold) }
            item { Text("配置会同步到当前账户，并保留版本以避免覆盖他人修改。", color = MaterialTheme.colorScheme.onSurfaceVariant) }
            item { Button(onClick = onAISettings, modifier = Modifier.fillMaxWidth().testTag("p04-open-ai-settings")) { Text("全局 AI 偏好") } }
            item { OutlinedButton(onClick = onServiceStatus, modifier = Modifier.fillMaxWidth().testTag("p04-open-service-status")) { Text("服务状态与模型健康") } }
            item { Text("选择一个会话进行会话设置、分支或多模型比较", style = MaterialTheme.typography.titleMedium) }
            error?.let { message -> item { P04InlineError(message) } }
            if (loading) item { CircularProgressIndicator(Modifier.testTag("p04-hub-loading")) }
            if (!loading && conversations.isEmpty()) item { Text("暂无可用会话", color = MaterialTheme.colorScheme.onSurfaceVariant) }
            items(conversations.take(12), key = { it.id }) { conversation ->
                Card(Modifier.fillMaxWidth().clickable { onConversationSettings(conversation) }.testTag("p04-hub-conversation-${conversation.id}"), border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface)) {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(5.dp)) {
                        Text(conversation.title.ifBlank { "新对话" }, fontWeight = FontWeight.SemiBold)
                        Text("模型：${conversation.defaultModelId.ifBlank { "自动选择" }} · 分支：${conversation.activeBranchId.take(8).ifBlank { "默认" }}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            TextButton(onClick = { onConversationSettings(conversation) }, modifier = Modifier.testTag("p04-open-conversation-settings-${conversation.id}")) { Text("会话设置") }
                            TextButton(onClick = { onBranches(conversation) }, modifier = Modifier.testTag("p04-open-branches-${conversation.id}")) { Text("分支") }
                            TextButton(onClick = { onComparison(conversation) }, modifier = Modifier.testTag("p04-open-comparison-${conversation.id}")) { Text("比较") }
                        }
                    }
                }
            }
        }
    }
}

@Composable
internal fun P04AISettingsPage(gateway: IdentityGateway, session: AuthSession, onBack: () -> Unit) {
    var models by remember { mutableStateOf<List<ModelOption>>(emptyList()) }
    var preference by remember { mutableStateOf(AIPreference()) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    fun load() { scope.launch { loading = true; error = null; runCatching { models = gateway.models(session.bearer); preference = gateway.aiPreference(session.bearer) }.onFailure { error = it.message ?: "偏好加载失败" }; loading = false } }
    LaunchedEffect(session.bearer) { load() }
    Scaffold(topBar = { TopAppBar(title = { Text("全局 AI 偏好") }, navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "返回") } }, actions = { IconButton(onClick = ::load) { Icon(Icons.Default.Refresh, "刷新") } }) }) { padding ->
        LazyColumn(Modifier.fillMaxSize().padding(padding).padding(horizontal = 20.dp), verticalArrangement = Arrangement.spacedBy(12.dp), contentPadding = androidx.compose.foundation.layout.PaddingValues(vertical = 18.dp)) {
            item { Text("默认模型", style = MaterialTheme.typography.titleMedium) }
            error?.let { message -> item { P04InlineError(message) } }
            if (loading) item { CircularProgressIndicator() }
            items(models.filter { it.enabled }, key = { it.id }) { model ->
                Card(Modifier.fillMaxWidth().clickable { preference = preference.copy(modelId = model.id, reasoningProfile = preference.reasoningProfile.takeIf { it in model.reasoningProfiles } ?: "auto") }.testTag("p04-ai-model-${model.id}"), border = BorderStroke(1.dp, if (preference.modelId == model.id) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outline)) {
                    Row(Modifier.padding(14.dp), verticalAlignment = Alignment.CenterVertically) { Column(Modifier.weight(1f)) { Text(model.name, fontWeight = FontWeight.SemiBold); Text("${model.providerName.ifBlank { "供应商" }} · ${model.speedTier}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }; if (preference.modelId == model.id) Icon(Icons.Default.Check, "已选择", tint = MaterialTheme.colorScheme.primary) }
                }
            }
            item {
                P04ReasoningProfileSelector(
                    profiles = models.firstOrNull { it.id == preference.modelId }?.reasoningProfiles.orEmpty(),
                    selected = preference.reasoningProfile,
                    onSelect = { preference = preference.copy(reasoningProfile = it) },
                    tag = "p04-ai-reasoning-profile",
                )
            }
            item { Button(enabled = !saving && !loading, onClick = { scope.launch { saving = true; error = null; runCatching { preference = gateway.updateAIPreference(session.bearer, preference.modelId, preference.reasoningProfile, preference.version) }.onFailure { error = it.message ?: "保存失败" }; saving = false } }, modifier = Modifier.fillMaxWidth().testTag("p04-save-ai-preference")) { Text(if (saving) "保存中…" else "保存偏好") } }
        }
    }
}

@Composable
internal fun P04ConversationSettingsPage(gateway: IdentityGateway, session: AuthSession, conversation: Conversation, onBack: () -> Unit, onUpdated: (Conversation) -> Unit) {
    var modelId by rememberSaveable(conversation.id) { mutableStateOf(conversation.defaultModelId) }
    var profile by rememberSaveable(conversation.id) { mutableStateOf(conversation.defaultReasoningProfile.ifBlank { "auto" }) }
    var models by remember { mutableStateOf<List<ModelOption>>(emptyList()) }
    var error by remember { mutableStateOf<String?>(null) }
    var saving by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    LaunchedEffect(session.bearer) { runCatching { models = gateway.models(session.bearer) }.onFailure { error = it.message } }
    Scaffold(topBar = { TopAppBar(title = { Text("会话 AI 设置") }, navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "返回") } }) }) { padding ->
        LazyColumn(Modifier.fillMaxSize().padding(padding).padding(horizontal = 20.dp), verticalArrangement = Arrangement.spacedBy(12.dp), contentPadding = androidx.compose.foundation.layout.PaddingValues(vertical = 18.dp)) {
            item { Text(conversation.title.ifBlank { "新对话" }, style = MaterialTheme.typography.headlineSmall) }
            item { Text("会话设置会覆盖全局默认值。当前版本：${conversation.aiSettingsVersion}", color = MaterialTheme.colorScheme.onSurfaceVariant) }
            error?.let { message -> item { P04InlineError(message) } }
            items(models.filter { it.enabled }, key = { it.id }) { model ->
                OutlinedButton(onClick = { modelId = model.id }, modifier = Modifier.fillMaxWidth().testTag("p04-conversation-model-${model.id}"), border = BorderStroke(1.dp, if (modelId == model.id) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outline)) { Text(if (modelId == model.id) "✓ ${model.name}" else model.name) }
            }
            item {
                P04ReasoningProfileSelector(
                    profiles = models.firstOrNull { it.id == modelId }?.reasoningProfiles.orEmpty(),
                    selected = profile,
                    onSelect = { profile = it },
                    tag = "p04-conversation-reasoning-profile",
                )
            }
            item { Button(enabled = !saving, onClick = { scope.launch { saving = true; runCatching { gateway.updateConversationAISettings(session.bearer, conversation.id, modelId, profile, conversation.aiSettingsVersion) }.onSuccess(onUpdated).onFailure { error = it.message ?: "保存失败" }; saving = false } }, modifier = Modifier.fillMaxWidth().testTag("p04-save-conversation-settings")) { Text(if (saving) "保存中…" else "保存会话设置") } }
        }
    }
}

@Composable
internal fun P04BranchesPage(gateway: IdentityGateway, session: AuthSession, conversation: Conversation, onBack: () -> Unit, onUpdated: (Conversation) -> Unit) {
    var branches by remember { mutableStateOf<List<ConversationBranch>>(emptyList()) }
    var error by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var creating by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    fun load() { scope.launch { loading = true; runCatching { branches = gateway.conversationBranches(session.bearer, conversation.id) }.onFailure { error = it.message ?: "分支加载失败" }; loading = false } }
    fun createBranch() {
        scope.launch {
            creating = true
            error = null
            runCatching {
                val messages = gateway.conversationMessages(session.bearer, conversation.id)
                val forkedFromMessageId = (messages.lastOrNull { it.role == "assistant" } ?: messages.lastOrNull())
                    ?.id
                    .orEmpty()
                gateway.createConversationBranch(session.bearer, conversation.id, forkedFromMessageId)
            }.onSuccess { branch ->
                onUpdated(conversation.copy(activeBranchId = branch.id, updatedAt = branch.updatedAt))
                load()
            }.onFailure { error = it.message ?: "创建分支失败" }
            creating = false
        }
    }
    LaunchedEffect(conversation.id) { load() }
    Scaffold(topBar = { TopAppBar(title = { Text("会话分支") }, navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "返回") } }, actions = { IconButton(onClick = ::load) { Icon(Icons.Default.Refresh, "刷新") } }) }) { padding ->
        LazyColumn(Modifier.fillMaxSize().padding(padding).padding(horizontal = 20.dp), verticalArrangement = Arrangement.spacedBy(12.dp), contentPadding = androidx.compose.foundation.layout.PaddingValues(vertical = 18.dp)) {
            item { Button(enabled = !creating, onClick = ::createBranch, modifier = Modifier.fillMaxWidth().testTag("p04-create-branch")) { Text(if (creating) "创建中…" else "从当前上下文创建分支") } }
            error?.let { message -> item { P04InlineError(message) } }
            if (loading) item { CircularProgressIndicator() }
            items(branches, key = { it.id }) { branch ->
                Card(Modifier.fillMaxWidth().testTag("p04-branch-${branch.id}"), border = BorderStroke(1.dp, if (branch.id == conversation.activeBranchId) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outline)) {
                    Row(Modifier.padding(14.dp), verticalAlignment = Alignment.CenterVertically) { Column(Modifier.weight(1f)) { Text(if (branch.id == conversation.activeBranchId) "当前分支" else "分支 ${branch.id.take(8)}", fontWeight = FontWeight.SemiBold); Text(p04BranchStatusLabel(branch.status), style = MaterialTheme.typography.bodySmall) }; if (branch.id != conversation.activeBranchId) TextButton(onClick = { scope.launch { runCatching { gateway.activateConversationBranch(session.bearer, conversation.id, branch.id) }.onSuccess(onUpdated).onFailure { error = it.message ?: "切换失败" } } }, modifier = Modifier.testTag("p04-activate-branch-${branch.id}")) { Text("切换") } else Icon(Icons.Default.Check, "当前") }
                }
            }
        }
    }
}

@Composable
internal fun P04ComparisonPage(gateway: IdentityGateway, session: AuthSession, conversation: Conversation, onBack: () -> Unit) {
    var models by remember { mutableStateOf<List<ModelOption>>(emptyList()) }
    var selected by remember { mutableStateOf<Set<String>>(emptySet()) }
    var prompt by rememberSaveable { mutableStateOf("") }
    var group by remember { mutableStateOf<ComparisonGroup?>(null) }
    var selectedCandidateRunId by rememberSaveable { mutableStateOf("") }
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    LaunchedEffect(session.bearer) { runCatching { models = gateway.models(session.bearer) }.onFailure { error = it.message } }
    fun refreshGroup() { val id = group?.id ?: return; scope.launch { runCatching { group = gateway.comparison(session.bearer, id) }.onFailure { error = it.message ?: "比较状态加载失败" } } }
    Scaffold(topBar = { TopAppBar(title = { Text("多模型比较") }, navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "返回") } }, actions = { IconButton(onClick = ::refreshGroup, modifier = Modifier.testTag("p04-refresh-comparison")) { Icon(Icons.Default.Refresh, "刷新") } }) }) { padding ->
        LazyColumn(Modifier.fillMaxSize().padding(padding).padding(horizontal = 20.dp), verticalArrangement = Arrangement.spacedBy(12.dp), contentPadding = androidx.compose.foundation.layout.PaddingValues(vertical = 18.dp)) {
            item { OutlinedTextField(value = prompt, onValueChange = { prompt = it }, label = { Text("比较问题") }, minLines = 3, modifier = Modifier.fillMaxWidth().testTag("p04-comparison-prompt")) }
            item { Text("选择模型（至少两个）", style = MaterialTheme.typography.titleMedium) }
            items(models.filter { it.enabled }, key = { it.id }) { model -> Row(Modifier.fillMaxWidth().clickable { selected = if (model.id in selected) selected - model.id else selected + model.id }.testTag("p04-comparison-model-${model.id}").padding(vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) { Checkbox(checked = model.id in selected, onCheckedChange = { selected = if (it) selected + model.id else selected - model.id }); Text(model.name) } }
            error?.let { message -> item { P04InlineError(message) } }
            item { Button(enabled = !busy && prompt.isNotBlank() && selected.size >= 2, onClick = { scope.launch { busy = true; runCatching { group = gateway.createComparison(session.bearer, conversation.id, prompt, selected.toList()) }.onFailure { error = it.message ?: "比较创建失败" }; busy = false } }, modifier = Modifier.fillMaxWidth().testTag("p04-create-comparison")) { Text(if (busy) "提交中…" else "开始比较") } }
            group?.let { current ->
                item { Text("状态：${p04ComparisonStatusLabel(current.status)}", style = MaterialTheme.typography.titleMedium, modifier = Modifier.testTag("p04-comparison-status")) }
                val selectedCandidate = current.candidates.firstOrNull { it.runId == selectedCandidateRunId } ?: current.candidates.firstOrNull()
                if (selectedCandidate != null) {
                    item {
                        TabRow(selectedTabIndex = current.candidates.indexOf(selectedCandidate).coerceAtLeast(0)) {
                            current.candidates.forEach { candidate ->
                                Tab(
                                    selected = candidate.runId == selectedCandidate.runId,
                                    onClick = { selectedCandidateRunId = candidate.runId },
                                    text = { Text(candidate.modelId) },
                                    modifier = Modifier.testTag("p04-comparison-tab-${candidate.runId}"),
                                )
                            }
                        }
                    }
                    item(key = selectedCandidate.runId) {
                        Card(Modifier.fillMaxWidth(), border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline)) {
                            Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                                Text(selectedCandidate.modelId, fontWeight = FontWeight.SemiBold)
                                Text(p04ComparisonStatusLabel(selectedCandidate.status))
                                if (selectedCandidate.body.isNotBlank()) Text(selectedCandidate.body)
                                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                    TextButton(enabled = selectedCandidate.status == "completed", onClick = { scope.launch { runCatching { group = gateway.adoptComparison(session.bearer, current.id, selectedCandidate.runId) }.onFailure { error = it.message } } }, modifier = Modifier.testTag("p04-adopt-${selectedCandidate.runId}")) { Text("采纳") }
                                    TextButton(enabled = selectedCandidate.status == "completed", onClick = { scope.launch { runCatching { group = gateway.synthesizeComparison(session.bearer, current.id, listOf(selectedCandidate.runId)) }.onFailure { error = it.message } } }, modifier = Modifier.testTag("p04-synthesize-${selectedCandidate.runId}")) { Text("以此综合") }
                                }
                            }
                        }
                    }
                }
                item { Text("已采纳：${current.adoptedRunId.ifBlank { "无" }} · 综合：${current.synthesisRunId.ifBlank { "无" }}", style = MaterialTheme.typography.bodySmall) }
            }
        }
    }
}

@Composable
internal fun P04ServiceStatusPage(gateway: IdentityGateway, session: AuthSession, onBack: () -> Unit) {
    var items by remember { mutableStateOf<List<ModelHealthStatus>>(emptyList()) }
    var modelNames by remember { mutableStateOf<Map<String, String>>(emptyMap()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    fun load() { scope.launch { loading = true; runCatching { items = gateway.serviceStatus(session.bearer); modelNames = gateway.models(session.bearer).associate { it.id to it.name } }.onFailure { error = it.message ?: "服务状态加载失败" }; loading = false } }
    LaunchedEffect(session.bearer) { load() }
    Scaffold(topBar = { TopAppBar(title = { Text("服务状态") }, navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "返回") } }, actions = { IconButton(onClick = ::load) { Icon(Icons.Default.Refresh, "刷新") } }) }) { padding ->
        LazyColumn(Modifier.fillMaxSize().padding(padding).padding(horizontal = 20.dp), verticalArrangement = Arrangement.spacedBy(10.dp), contentPadding = androidx.compose.foundation.layout.PaddingValues(vertical = 18.dp)) {
            error?.let { message -> item { P04InlineError(message) } }
            if (loading) item { CircularProgressIndicator() }
            items(items, key = { it.modelId }) { status -> Card(Modifier.fillMaxWidth(), border = BorderStroke(1.dp, MaterialTheme.colorScheme.outline)) { Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(5.dp)) { Text(modelNames[status.modelId] ?: "AI 服务", fontWeight = FontWeight.SemiBold); Text(p04ServiceStatusLabel(status.status), color = if (status.status == "available") MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.error); Text(p04ServiceStatusMessage(status.status), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant); Text("可用能力：${status.capabilities.map(::p04CapabilityLabel).joinToString("、").ifBlank { "暂未提供" }}", style = MaterialTheme.typography.bodySmall) } } }
        }
    }
}

@Composable
private fun P04InlineError(message: String?) {
    if (!message.isNullOrBlank()) {
        Text(message, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall, modifier = Modifier.testTag("p04-inline-error"))
    }
}

@Composable
private fun P04ReasoningProfileSelector(
    profiles: List<String>,
    selected: String,
    onSelect: (String) -> Unit,
    tag: String,
) {
    val options = profiles.ifEmpty { listOf("auto") }
    val normalized = selected.takeIf { it in options } ?: options.first()
    LaunchedEffect(options, selected) {
        if (normalized != selected) onSelect(normalized)
    }
    Column(verticalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxWidth().testTag(tag)) {
        Text("回答方式", style = MaterialTheme.typography.titleSmall)
        SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
            options.forEachIndexed { index, profile ->
                SegmentedButton(
                    selected = normalized == profile,
                    onClick = { onSelect(profile) },
                    shape = androidx.compose.material3.SegmentedButtonDefaults.itemShape(index = index, count = options.size),
                    label = { Text(p04ReasoningProfileLabel(profile)) },
                )
            }
        }
    }
}

private fun p04ReasoningProfileLabel(profile: String): String = when (profile.lowercase()) {
    "auto" -> "自动"
    "quick" -> "快速"
    "standard" -> "标准"
    "deep" -> "深入"
    else -> "默认"
}

private fun p04BranchStatusLabel(status: String): String = when (status.lowercase()) {
    "active" -> "正在使用"
    "archived" -> "已归档"
    else -> "可切换"
}

private fun p04ComparisonStatusLabel(status: String): String = when (status.lowercase()) {
    "running" -> "回答生成中"
    "completed" -> "已完成"
    "adopted" -> "已采纳"
    "synthesizing" -> "正在综合"
    "synthesized" -> "已生成综合回答"
    "synthesis_failed" -> "综合暂未完成"
    "failed" -> "暂未完成"
    else -> "处理中"
}

private fun p04ServiceStatusLabel(status: String): String = when (status.lowercase()) {
    "available" -> "服务正常"
    "degraded" -> "服务繁忙"
    "unavailable" -> "暂不可用"
    else -> "状态更新中"
}

private fun p04ServiceStatusMessage(status: String): String = when (status.lowercase()) {
    "available" -> "当前可以正常使用。"
    "degraded" -> "当前响应可能较慢，稍后会自动恢复。"
    "unavailable" -> "当前暂时无法使用，请稍后再试。"
    else -> "正在更新服务状态。"
}

private fun p04CapabilityLabel(capability: String): String = when (capability.lowercase()) {
    "text" -> "文本"
    "streaming" -> "连续回答"
    "vision" -> "图片理解"
    "tools" -> "工具调用"
    else -> "其他"
}
