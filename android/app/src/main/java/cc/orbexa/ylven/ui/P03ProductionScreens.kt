package cc.orbexa.ylven.ui

import android.content.Intent
import android.speech.RecognizerIntent
import android.speech.tts.TextToSpeech
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Archive
import androidx.compose.material.icons.filled.AttachFile
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Explore
import androidx.compose.material.icons.filled.FileDownload
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.MoreHoriz
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Send
import androidx.compose.material.icons.filled.Stop
import androidx.compose.material.icons.filled.ThumbDown
import androidx.compose.material.icons.filled.ThumbUp
import androidx.compose.material.icons.filled.Work
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import cc.orbexa.ylven.identity.AuthSession
import cc.orbexa.ylven.identity.Conversation
import cc.orbexa.ylven.identity.ConversationCache
import cc.orbexa.ylven.identity.HomeSnapshot
import cc.orbexa.ylven.identity.IdentityGateway
import cc.orbexa.ylven.identity.MessageCitation
import cc.orbexa.ylven.identity.MessageRecord
import cc.orbexa.ylven.identity.MessageRun
import cc.orbexa.ylven.ui.theme.YlvenDimensions
import cc.orbexa.ylven.ui.theme.YlvenLightColors
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private enum class P03HomeDestination {
    HOME,
    NEW_CONVERSATION,
    HISTORY,
    SEARCH,
    TEMPORARY_CONVERSATION,
}

@Composable
internal fun P03HomePage(
    gateway: IdentityGateway,
    session: AuthSession,
    onOpenConversation: (Conversation) -> Unit,
    onOpenAccount: () -> Unit,
) {
    var destination by rememberSaveable { mutableStateOf(P03HomeDestination.HOME) }
    var snapshot by remember { mutableStateOf<HomeSnapshot?>(null) }
    var conversations by remember { mutableStateOf<List<Conversation>>(emptyList()) }
    var searchResults by remember { mutableStateOf<List<Conversation>>(emptyList()) }
    var cursor by remember { mutableStateOf<String?>(null) }
    var searchQuery by rememberSaveable { mutableStateOf("") }
    var loading by remember { mutableStateOf(true) }
    var operationLoading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun loadHome() {
        scope.launch {
            loading = true
            error = null
            try {
                snapshot = gateway.home(session.bearer)
                conversations = snapshot?.conversations.orEmpty()
            } catch (reason: Exception) {
                error = reason.message ?: "首页加载失败，请重试"
            } finally {
                loading = false
            }
        }
    }

    fun loadHistory(reset: Boolean) {
        scope.launch {
            loading = true
            error = null
            try {
                val page = gateway.listConversations(session.bearer, if (reset) null else cursor)
                conversations = if (reset) page.first else conversations + page.first
                cursor = page.second
            } catch (reason: Exception) {
                error = reason.message ?: "会话历史加载失败，请重试"
            } finally {
                loading = false
            }
        }
    }

    fun createConversation(temporary: Boolean, title: String) {
        scope.launch {
            operationLoading = true
            error = null
            try {
                val created = if (temporary) {
                    gateway.createTemporaryConversation(session.bearer, title)
                } else {
                    gateway.createConversation(session.bearer, title)
                }
                conversations = listOf(created) + conversations.filterNot { it.id == created.id }
                onOpenConversation(created)
            } catch (reason: Exception) {
                error = reason.message ?: if (temporary) "无法创建临时会话" else "无法新建会话"
            } finally {
                operationLoading = false
            }
        }
    }

    LaunchedEffect(session.bearer) { loadHome() }
    LaunchedEffect(destination) {
        if (destination == P03HomeDestination.HISTORY) loadHistory(reset = true)
    }
    LaunchedEffect(searchQuery, destination) {
        if (destination != P03HomeDestination.SEARCH) return@LaunchedEffect
        if (searchQuery.isBlank()) {
            searchResults = emptyList()
            return@LaunchedEffect
        }
        delay(250)
        loading = true
        error = null
        try {
            searchResults = gateway.searchConversations(session.bearer, searchQuery.trim())
        } catch (reason: Exception) {
            error = reason.message ?: "搜索失败，请重试"
        } finally {
            loading = false
        }
    }

    BackHandler(enabled = destination != P03HomeDestination.HOME) {
        destination = when (destination) {
            P03HomeDestination.SEARCH -> P03HomeDestination.HISTORY
            else -> P03HomeDestination.HOME
        }
    }

    when (destination) {
        P03HomeDestination.HOME -> P03HomeScreen(
            session = session,
            snapshot = snapshot,
            conversations = conversations,
            loading = loading,
            error = error,
            onRetry = ::loadHome,
            onNewConversation = { destination = P03HomeDestination.NEW_CONVERSATION },
            onHistory = { destination = P03HomeDestination.HISTORY },
            onTemporaryConversation = { destination = P03HomeDestination.TEMPORARY_CONVERSATION },
            onOpenConversation = onOpenConversation,
            onOpenAccount = onOpenAccount,
        )
        P03HomeDestination.NEW_CONVERSATION -> P03NewConversationScreen(
            models = snapshot?.modelCatalog.orEmpty(),
            loading = operationLoading,
            error = error,
            onBack = { destination = P03HomeDestination.HOME },
            onCreate = { prompt -> createConversation(temporary = false, title = prompt) },
        )
        P03HomeDestination.HISTORY -> P03HistoryScreen(
            conversations = conversations,
            loading = loading,
            error = error,
            canLoadMore = cursor != null,
            onBack = { destination = P03HomeDestination.HOME },
            onSearch = { destination = P03HomeDestination.SEARCH },
            onNewConversation = { destination = P03HomeDestination.NEW_CONVERSATION },
            onOpenConversation = onOpenConversation,
            onLoadMore = { loadHistory(reset = false) },
            onRetry = { loadHistory(reset = true) },
        )
        P03HomeDestination.SEARCH -> P03SearchScreen(
            query = searchQuery,
            results = searchResults,
            loading = loading,
            error = error,
            onQueryChange = { searchQuery = it },
            onBack = { destination = P03HomeDestination.HISTORY },
            onOpenConversation = onOpenConversation,
        )
        P03HomeDestination.TEMPORARY_CONVERSATION -> P03TemporaryConversationScreen(
            models = snapshot?.modelCatalog.orEmpty(),
            loading = operationLoading,
            error = error,
            onBack = { destination = P03HomeDestination.HOME },
            onCreate = { title -> createConversation(temporary = true, title = title) },
        )
    }
}

@Composable
private fun P03HomeScreen(
    session: AuthSession,
    snapshot: HomeSnapshot?,
    conversations: List<Conversation>,
    loading: Boolean,
    error: String?,
    onRetry: () -> Unit,
    onNewConversation: () -> Unit,
    onHistory: () -> Unit,
    onTemporaryConversation: () -> Unit,
    onOpenConversation: (Conversation) -> Unit,
    onOpenAccount: () -> Unit,
) {
    val displayName = session.email.substringBefore('@').ifBlank { "YLVEN 用户" }
    Scaffold(
        modifier = Modifier.testTag("YL-A-018-C-P03_001-01"),
        containerColor = YlvenLightColors.Background,
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        topBar = {
            P03TopBar(
                title = "AI 首页",
                subtitle = "统一多模型对话入口",
                actionIcon = Icons.Default.Add,
                actionDescription = "新建对话",
                actionTag = "p03-open-new-conversation",
                onAction = onNewConversation,
            )
        },
        bottomBar = { P03BottomNavigation(onOpenAccount) },
    ) { padding ->
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(padding),
            contentPadding = PaddingValues(horizontal = YlvenDimensions.PageHorizontal, vertical = 24.dp),
            verticalArrangement = Arrangement.spacedBy(YlvenDimensions.CardGap),
        ) {
            item {
                Text("你好，$displayName", style = MaterialTheme.typography.headlineSmall)
                Text(
                    "今天准备完成什么？",
                    style = MaterialTheme.typography.bodyLarge,
                    color = YlvenLightColors.TextTertiary,
                )
            }
            item {
                Card(
                    onClick = onNewConversation,
                    modifier = Modifier.fillMaxWidth().testTag("p03-home-composer"),
                    shape = RoundedCornerShape(YlvenDimensions.CardRadius),
                    colors = CardDefaults.cardColors(containerColor = YlvenLightColors.Surface),
                    border = BorderStroke(YlvenDimensions.Border, YlvenLightColors.Border),
                ) {
                    Column(Modifier.padding(YlvenDimensions.CardPadding), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                "输入问题或上传文件",
                                modifier = Modifier.weight(1f),
                                color = YlvenLightColors.TextDisabled,
                                style = MaterialTheme.typography.bodyLarge,
                            )
                            Surface(shape = CircleShape, color = YlvenLightColors.Primary, modifier = Modifier.size(48.dp)) {
                                Icon(Icons.Default.Send, "开始新对话", Modifier.padding(12.dp), tint = Color.White)
                            }
                        }
                        Text(
                            snapshot?.modelCatalog?.firstOrNull()?.let { "$it · 深度" } ?: "模型将在会话中自动匹配",
                            modifier = Modifier.background(YlvenLightColors.SurfaceBrandSoft, RoundedCornerShape(18.dp)).padding(horizontal = 12.dp, vertical = 6.dp),
                            color = YlvenLightColors.Primary,
                            style = MaterialTheme.typography.labelLarge,
                        )
                    }
                }
            }
            item {
                P03ShortcutGrid()
            }
            item {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text("最近对话", modifier = Modifier.weight(1f), style = MaterialTheme.typography.titleLarge)
                    TextButton(onClick = onHistory, modifier = Modifier.testTag("p03-open-conversation-drawer")) {
                        Text("全部")
                    }
                }
            }
            if (loading && conversations.isEmpty()) {
                item { P03LoadingRows("yl-a-018-loading") }
            } else if (error != null && conversations.isEmpty()) {
                item { P03ErrorState(error, onRetry) }
            } else if (conversations.isEmpty()) {
                item { P03EmptyState("还没有会话", "新建第一条对话后会显示在这里", onNewConversation) }
            } else {
                items(conversations.take(3), key = { it.id }) { conversation ->
                    P03ConversationRow(conversation, onOpenConversation)
                }
            }
            if (error != null && conversations.isNotEmpty()) item { P03InlineError(error, onRetry) }
            item {
                OutlinedButton(
                    onClick = onTemporaryConversation,
                    modifier = Modifier.fillMaxWidth().height(52.dp).testTag("p03-open-temporary-conversation"),
                    shape = RoundedCornerShape(14.dp),
                ) {
                    Text("创建临时对话")
                }
            }
        }
    }
}

@Composable
private fun P03ShortcutGrid() {
    val shortcuts = listOf(
        "分析文件" to "文",
        "生成图片" to "图",
        "制作 PPT" to "P",
        "多模型对比" to "比",
    )
    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        shortcuts.chunked(2).forEach { row ->
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                row.forEach { (label, symbol) ->
                    Surface(
                        modifier = Modifier.weight(1f).height(56.dp),
                        shape = RoundedCornerShape(YlvenDimensions.CardRadius),
                        color = YlvenLightColors.Surface,
                        border = BorderStroke(1.dp, YlvenLightColors.Border),
                    ) {
                        Row(Modifier.padding(horizontal = 12.dp), verticalAlignment = Alignment.CenterVertically) {
                            Surface(shape = CircleShape, color = YlvenLightColors.SurfaceBrandSoft, modifier = Modifier.size(40.dp)) {
                                Box(contentAlignment = Alignment.Center) {
                                    Text(symbol, color = YlvenLightColors.Primary, fontWeight = FontWeight.Bold)
                                }
                            }
                            Spacer(Modifier.width(10.dp))
                            Text(label, style = MaterialTheme.typography.titleMedium, maxLines = 1)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun P03NewConversationScreen(
    models: List<String>,
    loading: Boolean,
    error: String?,
    onBack: () -> Unit,
    onCreate: (String) -> Unit,
) {
    var prompt by rememberSaveable { mutableStateOf("") }
    val choices = listOf("智能推荐") + models.take(3).let { available ->
        if (available.isEmpty()) listOf("GPT", "Claude", "Grok") else available
    }
    P03FormScaffold(
        rootTag = "YL-A-019-root",
        title = "开始新对话",
        subtitle = "选择模型后输入你的问题",
        onBack = onBack,
    ) {
        choices.take(4).forEachIndexed { index, label ->
            P03LabeledField(
                label = label,
                value = if (index == 0) prompt else "",
                onValueChange = { if (index == 0) prompt = it },
                placeholder = "请输入$label",
                readOnly = index != 0,
                tag = if (index == 0) "p03-new-conversation-prompt" else null,
            )
        }
        P03PrimaryButton(
            label = "输入问题",
            loading = loading,
            enabled = !loading,
            tag = "YL-A-019-C-P03_002-01",
            onClick = { onCreate(prompt.trim()) },
        )
        if (error != null) P03InlineError(error)
    }
}

@Composable
private fun P03HistoryScreen(
    conversations: List<Conversation>,
    loading: Boolean,
    error: String?,
    canLoadMore: Boolean,
    onBack: () -> Unit,
    onSearch: () -> Unit,
    onNewConversation: () -> Unit,
    onOpenConversation: (Conversation) -> Unit,
    onLoadMore: () -> Unit,
    onRetry: () -> Unit,
) {
    Scaffold(
        modifier = Modifier.testTag("YL-A-020-root"),
        containerColor = YlvenLightColors.Background,
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        topBar = {
            P03TopBar(
                title = "会话历史",
                subtitle = "搜索、归档和管理全部对话",
                onBack = onBack,
                actionIcon = Icons.Default.Search,
                actionDescription = "搜索会话",
                actionTag = "p03-open-search",
                onAction = onSearch,
            )
        },
    ) { padding ->
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(padding).testTag("p03-conversation-drawer"),
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 24.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            item {
                Row(Modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    P03FilterChip("今天", true) {}
                    P03FilterChip("昨天", false) {}
                    P03FilterChip("最近 7 天", false) {}
                    P03FilterChip("新建对话", false, onNewConversation)
                }
            }
            when {
                loading && conversations.isEmpty() -> item { P03LoadingRows("yl-a-020-loading") }
                error != null && conversations.isEmpty() -> item { P03ErrorState(error, onRetry) }
                conversations.isEmpty() -> item { P03EmptyState("暂无会话", "新建会话后会显示在这里", onNewConversation) }
                else -> items(conversations, key = { it.id }) { conversation ->
                    P03ConversationRow(conversation, onOpenConversation, showMore = true)
                }
            }
            if (canLoadMore) {
                item {
                    OutlinedButton(
                        onClick = onLoadMore,
                        modifier = Modifier.fillMaxWidth().height(52.dp).testTag("YL-A-020-C-P03_003-01"),
                        shape = RoundedCornerShape(14.dp),
                    ) { Text("加载更多") }
                }
            }
            if (error != null && conversations.isNotEmpty()) item { P03InlineError(error, onRetry) }
        }
    }
}

@Composable
private fun P03SearchScreen(
    query: String,
    results: List<Conversation>,
    loading: Boolean,
    error: String?,
    onQueryChange: (String) -> Unit,
    onBack: () -> Unit,
    onOpenConversation: (Conversation) -> Unit,
) {
    Scaffold(
        modifier = Modifier.testTag("YL-A-021-root"),
        containerColor = YlvenLightColors.Background,
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        topBar = { P03TopBar("搜索会话", "按标题和消息内容查找历史记录", onBack = onBack) },
    ) { padding ->
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(padding),
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 16.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            item {
                Text("搜索", style = MaterialTheme.typography.titleMedium, color = YlvenLightColors.TextSecondary)
                OutlinedTextField(
                    value = query,
                    onValueChange = onQueryChange,
                    modifier = Modifier.fillMaxWidth().heightIn(min = 52.dp).testTag("YL-A-021-C-P03_004-01"),
                    placeholder = { Text("按标题和消息内容查找历史记录") },
                    leadingIcon = { Icon(Icons.Default.Search, null) },
                    singleLine = true,
                    shape = RoundedCornerShape(14.dp),
                )
            }
            item {
                Row(Modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    P03FilterChip("搜索输入框", true) {}
                    P03FilterChip("筛选项目", false) {}
                    P03FilterChip("搜索结果", false) {}
                }
            }
            when {
                loading && query.isNotBlank() -> item { P03LoadingRows("yl-a-021-loading") }
                error != null -> item { P03InlineError(error) }
                query.isBlank() -> item { P03EmptyState("输入关键词开始搜索", "可搜索会话标题和消息正文") }
                results.isEmpty() -> item { P03EmptyState("没有找到结果", "请尝试其他关键词") }
                else -> items(results, key = { it.id }) { conversation ->
                    P03ConversationRow(conversation, onOpenConversation, showMore = true)
                }
            }
        }
    }
}

@Composable
private fun P03TemporaryConversationScreen(
    models: List<String>,
    loading: Boolean,
    error: String?,
    onBack: () -> Unit,
    onCreate: (String) -> Unit,
) {
    var title by rememberSaveable { mutableStateOf("") }
    var project by rememberSaveable { mutableStateOf("") }
    var model by rememberSaveable { mutableStateOf(models.firstOrNull().orEmpty()) }
    var instruction by rememberSaveable { mutableStateOf("") }
    P03FormScaffold(
        rootTag = "YL-A-031-root",
        title = "新建对话",
        subtitle = "设置会话名称、项目和默认模型",
        onBack = onBack,
    ) {
        P03LabeledField("会话名称", title, { title = it }, "请输入会话名称", tag = "p03-temporary-title")
        P03LabeledField("所属项目", project, { project = it }, "请输入所属项目")
        P03LabeledField("默认模型", model, { model = it }, "请输入默认模型")
        P03LabeledField("会话指令", instruction, { instruction = it }, "请输入会话指令", singleLine = false)
        P03PrimaryButton(
            label = "创建临时对话",
            loading = loading,
            enabled = !loading,
            tag = "YL-A-031-C-P03_028-01",
            onClick = { onCreate(title.trim()) },
        )
        if (error != null) P03InlineError(error)
    }
}

@Composable
private fun P03FormScaffold(
    rootTag: String,
    title: String,
    subtitle: String,
    onBack: () -> Unit,
    content: @Composable () -> Unit,
) {
    Scaffold(
        modifier = Modifier.testTag(rootTag),
        containerColor = YlvenLightColors.Background,
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        topBar = { P03TopBar(title, subtitle, onBack = onBack) },
    ) { padding ->
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(padding).imePadding(),
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 24.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) { item { Column(verticalArrangement = Arrangement.spacedBy(16.dp), content = { content() }) } }
    }
}

@Composable
internal fun P03ChatPage(
    gateway: IdentityGateway,
    session: AuthSession,
    conversation: Conversation,
    onBack: () -> Unit,
) {
    var title by remember(conversation.id) { mutableStateOf(conversation.title.ifBlank { "新对话" }) }
    var draft by rememberSaveable(conversation.id) { mutableStateOf("") }
    var draftLoaded by remember(conversation.id) { mutableStateOf(false) }
    var messages by remember(conversation.id) { mutableStateOf<List<MessageRecord>>(emptyList()) }
    var run by remember(conversation.id) { mutableStateOf<MessageRun?>(null) }
    var error by remember(conversation.id) { mutableStateOf<String?>(null) }
    var sending by remember(conversation.id) { mutableStateOf(false) }
    var reconnecting by remember(conversation.id) { mutableStateOf(false) }
    var citations by remember(conversation.id) { mutableStateOf<Map<String, List<MessageCitation>>>(emptyMap()) }
    var retryBody by remember(conversation.id) { mutableStateOf<String?>(null) }
    var menuOpen by rememberSaveable(conversation.id) { mutableStateOf(false) }
    var renameMode by rememberSaveable(conversation.id) { mutableStateOf(false) }
    var renameText by rememberSaveable(conversation.id) { mutableStateOf(title) }
    var confirmDelete by rememberSaveable(conversation.id) { mutableStateOf(false) }
    var menuBusy by remember(conversation.id) { mutableStateOf(false) }
    var menuMessage by remember(conversation.id) { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val clipboard = LocalClipboardManager.current
    val context = LocalContext.current
    val cache = remember { ConversationCache(context) }
    val voiceLauncher = rememberLauncherForActivityResult(ActivityResultContracts.StartActivityForResult()) { result ->
        result.data?.getStringArrayListExtra(RecognizerIntent.EXTRA_RESULTS)?.firstOrNull()?.let { draft = it }
    }
    val tts = remember(context) { TextToSpeech(context, null) }
    DisposableEffect(tts) { onDispose { tts.shutdown() } }

    LaunchedEffect(conversation.id) {
        messages = cache.load(conversation.id)
        runCatching { gateway.loadDraft(session.bearer, conversation.id) }.onSuccess { draft = it }
        draftLoaded = true
    }
    LaunchedEffect(draftLoaded, draft) {
        if (!draftLoaded) return@LaunchedEffect
        delay(500)
        runCatching { gateway.saveDraft(session.bearer, conversation.id, draft) }
    }
    LaunchedEffect(messages) {
        messages.filter { it.role == "assistant" && !citations.containsKey(it.id) }.forEach { message ->
            runCatching { gateway.messageCitations(session.bearer, message.id) }
                .getOrNull()
                ?.let { loaded -> citations = citations + (message.id to loaded) }
        }
    }

    suspend fun observeRun(runId: String) {
        var cursor = run?.cursor ?: 0L
        var consecutiveFailures = 0
        while (true) {
            try {
                val stream = gateway.runEvents(session.bearer, runId, cursor)
                cursor = maxOf(cursor, stream.first.cursor, stream.second.maxOfOrNull { it.id } ?: 0L)
                run = stream.first
                val snapshot = gateway.runStatus(session.bearer, runId)
                run = snapshot.first
                messages = snapshot.second
                cache.save(conversation.id, messages)
                reconnecting = false
                consecutiveFailures = 0
                if (snapshot.first.status.lowercase() in setOf("completed", "cancelled", "failed", "content_blocked")) return
                delay(350)
            } catch (reason: Exception) {
                consecutiveFailures += 1
                reconnecting = true
                if (consecutiveFailures >= 3) throw reason
                delay(500L * consecutiveFailures)
            }
        }
    }

    fun leaveChat() {
        scope.launch {
            if (draftLoaded) runCatching { gateway.saveDraft(session.bearer, conversation.id, draft) }
            onBack()
        }
    }

    fun openVoiceInput() {
        runCatching {
            voiceLauncher.launch(Intent(RecognizerIntent.ACTION_RECOGNIZE_SPEECH).apply {
                putExtra(RecognizerIntent.EXTRA_LANGUAGE_MODEL, RecognizerIntent.LANGUAGE_MODEL_FREE_FORM)
            })
        }.onFailure { error = "当前设备没有可用的语音输入服务" }
    }

    fun send() {
        val body = draft.trim()
        if (body.isEmpty() || sending) return
        scope.launch {
            sending = true
            error = null
            messages = messages + MessageRecord("optimistic-${System.currentTimeMillis()}", conversation.id, "user", body, "")
            try {
                val created = gateway.sendMessage(session.bearer, conversation.id, body)
                draft = ""
                run = created
                observeRun(created.id)
                cache.save(conversation.id, messages)
            } catch (reason: Exception) {
                error = reason.message ?: "发送失败，请重试"
                retryBody = body
            } finally {
                runCatching { gateway.saveDraft(session.bearer, conversation.id, draft) }
                sending = false
            }
        }
    }

    BackHandler(onBack = ::leaveChat)
    Scaffold(
        modifier = Modifier.testTag("YL-A-023-C-P03_013-01"),
        containerColor = YlvenLightColors.Background,
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        topBar = {
            P03TopBar(
                title = title,
                subtitle = "GPT-5.6 Sol · 深度推理",
                onBack = ::leaveChat,
                actionIcon = Icons.Default.MoreHoriz,
                actionDescription = "会话操作",
                actionTag = "p03-open-conversation-menu",
                onAction = {
                    renameMode = false
                    confirmDelete = false
                    menuMessage = null
                    menuOpen = true
                },
            )
        },
        bottomBar = {
            P03Composer(
                draft = draft,
                onDraftChange = { draft = it },
                sending = sending || run?.status.equals("streaming", ignoreCase = true),
                onSend = ::send,
                onStop = {
                    run?.let { active ->
                        scope.launch {
                            runCatching { gateway.cancelRun(session.bearer, active.id) }
                                .onSuccess { run = it }
                                .onFailure { error = it.message ?: "停止失败" }
                        }
                    }
                },
                onVoice = ::openVoiceInput,
            )
        },
    ) { padding ->
        Box(Modifier.fillMaxSize().padding(padding).testTag("p03-active-conversation-${conversation.id}")) {
            Box(Modifier.fillMaxSize().testTag("YL-A-023-C-P03_017-01")) {
            LazyColumn(
                modifier = Modifier.fillMaxSize().testTag("YL-A-025-C-P03_016-01"),
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 18.dp),
                verticalArrangement = Arrangement.spacedBy(20.dp),
            ) {
                item {
                    Text(
                        "GPT-5.6 Sol · 深度",
                        modifier = Modifier.background(YlvenLightColors.SurfaceBrandSoft, RoundedCornerShape(18.dp)).padding(horizontal = 12.dp, vertical = 6.dp),
                        color = YlvenLightColors.Primary,
                        style = MaterialTheme.typography.labelLarge,
                    )
                }
                if (messages.isEmpty() && !sending) {
                    item { P03EmptyState("开始这次对话", "在下方输入问题，回答会流式显示") }
                }
                items(messages, key = { it.id }) { message ->
                    if (message.role == "user") {
                        Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.CenterEnd) {
                            Surface(
                                modifier = Modifier.widthIn(max = 300.dp),
                                shape = RoundedCornerShape(18.dp),
                                color = YlvenLightColors.Primary,
                            ) {
                                Text(message.body, Modifier.padding(horizontal = 16.dp, vertical = 12.dp), color = Color.White)
                            }
                        }
                    } else {
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text("GPT-5.6 Sol · 深度推理", color = YlvenLightColors.Primary, style = MaterialTheme.typography.labelLarge)
                            MessageContent(message.body, Modifier.fillMaxWidth()) { code -> clipboard.setText(AnnotatedString(code)) }
                            citations[message.id]?.forEach { citation ->
                                Surface(
                                    modifier = Modifier.fillMaxWidth().testTag("YL-A-029-C-P03_021-01"),
                                    shape = RoundedCornerShape(12.dp),
                                    color = YlvenLightColors.SurfaceSubtle,
                                    border = BorderStroke(1.dp, YlvenLightColors.Border),
                                ) {
                                    Column(Modifier.padding(12.dp)) {
                                        Text("来源", style = MaterialTheme.typography.labelMedium, color = YlvenLightColors.TextTertiary)
                                        Text(citation.title, style = MaterialTheme.typography.bodyMedium)
                                    }
                                }
                            }
                            P03MessageActions(
                                message = message,
                                gateway = gateway,
                                session = session,
                                clipboardCopy = { clipboard.setText(AnnotatedString(it)) },
                                tts = tts,
                                onRun = { nextRun ->
                                    run = nextRun
                                    scope.launch {
                                        runCatching { observeRun(nextRun.id) }.onFailure { error = it.message ?: "重答失败" }
                                    }
                                },
                                onError = { error = it },
                            )
                        }
                    }
                }
                if (sending) item {
                    Text("正在连接并接收回答…", color = YlvenLightColors.Primary, modifier = Modifier.testTag("YL-A-023-C-P03_010-01"))
                }
                if (reconnecting) item {
                    Text("连接中断，正在按游标恢复…", color = YlvenLightColors.Warning, modifier = Modifier.testTag("YL-A-023-C-P03_012-01"))
                }
                run?.let { activeRun ->
                    item {
                        Text(
                            "生成状态：${activeRun.status}",
                            style = MaterialTheme.typography.bodySmall,
                            color = YlvenLightColors.TextTertiary,
                            modifier = Modifier.testTag("YL-A-026-C-P03_026-01"),
                        )
                    }
                }
                error?.let { message -> item { P03InlineError(message) } }
                retryBody?.let { body ->
                    item {
                        TextButton(
                            onClick = { draft = body; retryBody = null },
                            modifier = Modifier.testTag("YL-A-023-C-P03_027-01"),
                        ) { Text("将失败内容放回输入框") }
                    }
                }
            }
            }
        }
    }

    if (menuOpen) {
        P03ConversationMenu(
            title = title,
            renameText = renameText,
            renameMode = renameMode,
            confirmDelete = confirmDelete,
            busy = menuBusy,
            message = menuMessage,
            onDismiss = { if (!menuBusy) menuOpen = false },
            onStartRename = { renameText = title; renameMode = true; confirmDelete = false },
            onRenameTextChange = { renameText = it },
            onRename = {
                scope.launch {
                    menuBusy = true
                    menuMessage = null
                    runCatching { gateway.renameConversation(session.bearer, conversation.id, renameText.trim()) }
                        .onSuccess { updated -> title = updated.title; renameMode = false; menuOpen = false }
                        .onFailure { menuMessage = it.message ?: "重命名失败" }
                    menuBusy = false
                }
            },
            onArchive = {
                scope.launch {
                    menuBusy = true
                    runCatching { gateway.archiveConversation(session.bearer, conversation.id) }
                        .onSuccess { menuOpen = false; leaveChat() }
                        .onFailure { menuMessage = it.message ?: "归档失败" }
                    menuBusy = false
                }
            },
            onExport = {
                scope.launch {
                    menuBusy = true
                    runCatching { gateway.exportConversation(session.bearer, conversation.id) }
                        .onSuccess { markdown -> clipboard.setText(AnnotatedString(markdown)); menuOpen = false }
                        .onFailure { menuMessage = it.message ?: "会话导出失败" }
                    menuBusy = false
                }
            },
            onRequestDelete = { confirmDelete = true; renameMode = false },
            onDelete = {
                scope.launch {
                    menuBusy = true
                    runCatching { gateway.deleteConversation(session.bearer, conversation.id) }
                        .onSuccess { menuOpen = false; leaveChat() }
                        .onFailure { menuMessage = it.message ?: "删除失败" }
                    menuBusy = false
                }
            },
        )
    }
}

@Composable
private fun P03MessageActions(
    message: MessageRecord,
    gateway: IdentityGateway,
    session: AuthSession,
    clipboardCopy: (String) -> Unit,
    tts: TextToSpeech,
    onRun: (MessageRun) -> Unit,
    onError: (String) -> Unit,
) {
    val scope = rememberCoroutineScope()
    Row(Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(2.dp)) {
        P03SmallAction(Icons.Default.ContentCopy, "复制", "YL-A-030-C-P03_022-01") { clipboardCopy(message.body) }
        P03SmallAction(Icons.Default.FileDownload, "导出", "YL-A-030-C-P03_023-01") {
            scope.launch {
                runCatching { gateway.exportMessage(session.bearer, message.id) }
                    .onSuccess(clipboardCopy)
                    .onFailure { onError(it.message ?: "回答导出失败") }
            }
        }
        P03SmallAction(Icons.Default.ThumbUp, "赞", "YL-A-030-C-P03_025-01") {
            scope.launch { runCatching { gateway.submitFeedback(session.bearer, message.id, "up") }.onFailure { onError(it.message ?: "反馈失败") } }
        }
        P03SmallAction(Icons.Default.ThumbDown, "踩", "YL-A-030-C-P03_025-01") {
            scope.launch { runCatching { gateway.submitFeedback(session.bearer, message.id, "down") }.onFailure { onError(it.message ?: "反馈失败") } }
        }
        P03SmallAction(Icons.Default.Refresh, "重答", "YL-A-030-C-P03_024-01") {
            scope.launch {
                runCatching { gateway.regenerate(session.bearer, message.id) }
                    .onSuccess(onRun)
                    .onFailure { onError(it.message ?: "重答失败") }
            }
        }
        P03SmallAction(Icons.Default.PlayArrow, "朗读", "YL-A-030-C-P03_030-01") {
            scope.launch {
                runCatching {
                    gateway.speak(session.bearer, message.id)
                    tts.speak(message.body, TextToSpeech.QUEUE_FLUSH, null, message.id)
                }.onFailure { onError(it.message ?: "朗读失败") }
            }
        }
    }
}

@Composable
private fun P03SmallAction(icon: androidx.compose.ui.graphics.vector.ImageVector, description: String, tag: String, onClick: () -> Unit) {
    IconButton(onClick = onClick, modifier = Modifier.size(44.dp).testTag(tag)) {
        Icon(icon, description, tint = YlvenLightColors.TextSecondary)
    }
}

@Composable
private fun P03Composer(
    draft: String,
    onDraftChange: (String) -> Unit,
    sending: Boolean,
    onSend: () -> Unit,
    onStop: () -> Unit,
    onVoice: () -> Unit,
) {
    Surface(color = YlvenLightColors.Background) {
        Surface(
            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 10.dp).navigationBarsPadding(),
            shape = RoundedCornerShape(24.dp),
            color = YlvenLightColors.Surface,
            border = BorderStroke(1.dp, YlvenLightColors.BorderStrong),
        ) {
            Row(Modifier.heightIn(min = 52.dp, max = 148.dp).padding(horizontal = 6.dp), verticalAlignment = Alignment.CenterVertically) {
                IconButton(onClick = {}, enabled = false, modifier = Modifier.size(44.dp)) {
                    Icon(Icons.Default.AttachFile, "添加附件", tint = YlvenLightColors.TextSecondary)
                }
                Box(Modifier.weight(1f).padding(vertical = 14.dp)) {
                    if (draft.isEmpty()) Text("继续追问…", color = YlvenLightColors.TextDisabled)
                    BasicTextField(
                        value = draft,
                        onValueChange = onDraftChange,
                        modifier = Modifier.fillMaxWidth().testTag("YL-A-032-C-P03_032-01"),
                        textStyle = MaterialTheme.typography.bodyLarge.copy(color = YlvenLightColors.TextPrimary),
                        maxLines = 6,
                    )
                }
                IconButton(onClick = onVoice, modifier = Modifier.size(44.dp).testTag("YL-A-032-C-P03_031-01")) {
                    Icon(Icons.Default.Mic, "语音输入", tint = YlvenLightColors.TextSecondary)
                }
                Surface(shape = CircleShape, color = YlvenLightColors.Primary, modifier = Modifier.size(44.dp)) {
                    IconButton(
                        onClick = if (sending) onStop else onSend,
                        enabled = sending || draft.isNotBlank(),
                        modifier = Modifier.testTag(if (sending) "YL-A-024-C-P03_011-01" else "YL-A-023-C-P03_009-01"),
                    ) {
                        Icon(if (sending) Icons.Default.Stop else Icons.Default.Send, if (sending) "停止" else "发送", tint = Color.White)
                    }
                }
            }
        }
    }
}

@Composable
@OptIn(ExperimentalMaterial3Api::class)
private fun P03ConversationMenu(
    title: String,
    renameText: String,
    renameMode: Boolean,
    confirmDelete: Boolean,
    busy: Boolean,
    message: String?,
    onDismiss: () -> Unit,
    onStartRename: () -> Unit,
    onRenameTextChange: (String) -> Unit,
    onRename: () -> Unit,
    onArchive: () -> Unit,
    onExport: () -> Unit,
    onRequestDelete: () -> Unit,
    onDelete: () -> Unit,
) {
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        modifier = Modifier.testTag("YL-A-022-root"),
        shape = RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp),
        containerColor = YlvenLightColors.Surface,
        dragHandle = {
            Surface(Modifier.padding(top = 10.dp).size(width = 60.dp, height = 5.dp), shape = CircleShape, color = YlvenLightColors.BorderStrong) {}
        },
    ) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp).navigationBarsPadding().testTag("p03-conversation-menu"), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text("会话操作", style = MaterialTheme.typography.headlineSmall)
            Text(title, color = YlvenLightColors.TextTertiary, style = MaterialTheme.typography.bodySmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
            if (renameMode) {
                OutlinedTextField(
                    value = renameText,
                    onValueChange = onRenameTextChange,
                    modifier = Modifier.fillMaxWidth().testTag("p03-rename-input"),
                    singleLine = true,
                    shape = RoundedCornerShape(14.dp),
                    label = { Text("会话名称") },
                )
                Button(
                    onClick = onRename,
                    enabled = renameText.isNotBlank() && !busy,
                    modifier = Modifier.fillMaxWidth().height(52.dp).testTag("YL-A-022-C-P03_005-01"),
                    shape = RoundedCornerShape(14.dp),
                ) { Text(if (busy) "正在保存" else "保存名称") }
            } else if (confirmDelete) {
                Text("删除后会话将进入回收策略，是否继续？", color = YlvenLightColors.Error)
                Button(
                    onClick = onDelete,
                    enabled = !busy,
                    modifier = Modifier.fillMaxWidth().height(52.dp).testTag("p03-confirm-delete"),
                    colors = ButtonDefaults.buttonColors(containerColor = YlvenLightColors.Error),
                    shape = RoundedCornerShape(14.dp),
                ) { Text(if (busy) "正在删除" else "确认删除") }
            } else {
                P03MenuRow(Icons.Default.Edit, "重命名", "p03-open-rename-current", !busy, onStartRename)
                P03MenuRow(Icons.Default.Archive, "归档", "YL-A-022-C-P03_006-01", !busy, onArchive)
                P03MenuRow(Icons.Default.FileDownload, "导出", "YL-A-022-C-P03_029-01", !busy, onExport)
                P03MenuRow(Icons.Default.Folder, "移入项目", null, enabled = false, onClick = {})
                P03MenuRow(Icons.Default.Delete, "删除", "YL-A-022-C-P03_007-01", !busy, onRequestDelete, destructive = true)
            }
            if (busy) CircularProgressIndicator(Modifier.align(Alignment.CenterHorizontally).size(28.dp))
            if (message != null) Text(message, color = YlvenLightColors.Error, style = MaterialTheme.typography.bodySmall)
            Spacer(Modifier.height(16.dp))
        }
    }
}

@Composable
private fun P03MenuRow(
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    label: String,
    tag: String?,
    enabled: Boolean,
    onClick: () -> Unit,
    destructive: Boolean = false,
) {
    val color = if (destructive) YlvenLightColors.Error else YlvenLightColors.TextPrimary
    val modifier = Modifier.fillMaxWidth().height(56.dp).clickable(enabled = enabled, onClick = onClick)
        .then(if (tag == null) Modifier else Modifier.testTag(tag))
    Row(modifier, verticalAlignment = Alignment.CenterVertically) {
        Icon(icon, label, tint = if (enabled) color else YlvenLightColors.TextDisabled)
        Spacer(Modifier.width(16.dp))
        Text(label, color = if (enabled) color else YlvenLightColors.TextDisabled, style = MaterialTheme.typography.bodyLarge)
    }
}

@Composable
private fun P03TopBar(
    title: String,
    subtitle: String,
    onBack: (() -> Unit)? = null,
    actionIcon: androidx.compose.ui.graphics.vector.ImageVector? = Icons.Default.MoreHoriz,
    actionDescription: String = "更多",
    actionTag: String? = null,
    onAction: (() -> Unit)? = null,
) {
    Surface(color = YlvenLightColors.Surface, shadowElevation = 0.dp) {
        Row(
            modifier = Modifier.fillMaxWidth().statusBarsPadding().height(58.dp).padding(horizontal = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (onBack != null) {
                IconButton(onClick = onBack, modifier = Modifier.size(48.dp).testTag("p03-page-back")) {
                    Icon(Icons.AutoMirrored.Filled.ArrowBack, "返回")
                }
            } else {
                Spacer(Modifier.width(12.dp))
            }
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.Center) {
                Text(title, style = MaterialTheme.typography.headlineSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text(subtitle, style = MaterialTheme.typography.bodySmall, color = YlvenLightColors.TextTertiary, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
            if (actionIcon != null && onAction != null) {
                IconButton(
                    onClick = onAction,
                    modifier = Modifier.size(48.dp).then(if (actionTag == null) Modifier else Modifier.testTag(actionTag)),
                ) { Icon(actionIcon, actionDescription, tint = YlvenLightColors.TextSecondary) }
            } else {
                Spacer(Modifier.width(48.dp))
            }
        }
    }
}

@Composable
private fun P03BottomNavigation(onOpenAccount: () -> Unit) {
    Surface(color = YlvenLightColors.Surface, border = BorderStroke(1.dp, YlvenLightColors.Divider)) {
        Row(Modifier.fillMaxWidth().height(64.dp).navigationBarsPadding(), horizontalArrangement = Arrangement.SpaceAround) {
            P03NavigationItem(Icons.Default.Home, "首页", selected = true) {}
            P03NavigationItem(Icons.Default.Work, "工作", selected = false) {}
            P03NavigationItem(Icons.Default.Explore, "发现", selected = false) {}
            P03NavigationItem(Icons.Default.Person, "我的", selected = false, onClick = onOpenAccount)
        }
    }
}

@Composable
private fun P03NavigationItem(icon: androidx.compose.ui.graphics.vector.ImageVector, label: String, selected: Boolean, onClick: () -> Unit) {
    Column(
        modifier = Modifier.width(72.dp).height(64.dp).clickable(onClick = onClick),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Icon(icon, label, tint = if (selected) YlvenLightColors.Primary else YlvenLightColors.TextTertiary, modifier = Modifier.size(24.dp))
        Text(label, color = if (selected) YlvenLightColors.Primary else YlvenLightColors.TextTertiary, style = MaterialTheme.typography.labelSmall, fontWeight = if (selected) FontWeight.Bold else FontWeight.Normal)
    }
}

@Composable
private fun P03ConversationRow(conversation: Conversation, onOpen: (Conversation) -> Unit, showMore: Boolean = false) {
    Card(
        onClick = { onOpen(conversation) },
        modifier = Modifier.fillMaxWidth().heightIn(min = 60.dp).testTag("p03-conversation-${conversation.id}"),
        shape = RoundedCornerShape(YlvenDimensions.CardRadius),
        colors = CardDefaults.cardColors(containerColor = YlvenLightColors.Surface),
        border = BorderStroke(1.dp, YlvenLightColors.Border),
    ) {
        Row(Modifier.padding(horizontal = 12.dp, vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
            Surface(shape = CircleShape, color = YlvenLightColors.SurfaceBrandSoft, modifier = Modifier.size(40.dp)) {
                Box(contentAlignment = Alignment.Center) { Text("AI", color = YlvenLightColors.Primary, fontWeight = FontWeight.Bold) }
            }
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                Text(conversation.title.ifBlank { "新对话" }, style = MaterialTheme.typography.titleMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text(
                    conversationMeta(conversation),
                    color = YlvenLightColors.TextTertiary,
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 1,
                )
            }
            if (showMore) Icon(Icons.Default.MoreHoriz, "打开会话", tint = YlvenLightColors.TextTertiary)
        }
    }
}

private fun conversationMeta(conversation: Conversation): String = when {
    conversation.status == "temporary" -> "临时对话"
    conversation.updatedAt.isBlank() -> conversation.status
    conversation.updatedAt.contains('T') -> "最近更新"
    else -> "更新于 ${conversation.updatedAt}"
}

@Composable
private fun P03LabeledField(
    label: String,
    value: String,
    onValueChange: (String) -> Unit,
    placeholder: String,
    readOnly: Boolean = false,
    tag: String? = null,
    singleLine: Boolean = true,
) {
    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Text(label, style = MaterialTheme.typography.titleMedium, color = YlvenLightColors.TextSecondary)
        OutlinedTextField(
            value = value,
            onValueChange = onValueChange,
            modifier = Modifier.fillMaxWidth().heightIn(min = 52.dp).then(if (tag == null) Modifier else Modifier.testTag(tag)),
            placeholder = { Text(placeholder) },
            readOnly = readOnly,
            singleLine = singleLine,
            minLines = if (singleLine) 1 else 2,
            maxLines = if (singleLine) 1 else 4,
            shape = RoundedCornerShape(14.dp),
        )
    }
}

@Composable
private fun P03PrimaryButton(label: String, loading: Boolean, enabled: Boolean, tag: String, onClick: () -> Unit) {
    Button(
        onClick = onClick,
        enabled = enabled,
        modifier = Modifier.fillMaxWidth().height(52.dp).testTag(tag),
        shape = RoundedCornerShape(14.dp),
    ) {
        if (loading) {
            CircularProgressIndicator(Modifier.size(20.dp), color = Color.White, strokeWidth = 2.dp)
            Spacer(Modifier.width(8.dp))
        }
        Text(label, style = MaterialTheme.typography.titleMedium)
    }
}

@Composable
private fun P03FilterChip(label: String, selected: Boolean, onClick: () -> Unit) {
    Surface(
        modifier = Modifier.height(40.dp).clickable(onClick = onClick),
        shape = RoundedCornerShape(20.dp),
        color = if (selected) YlvenLightColors.SurfaceBrandSoft else YlvenLightColors.Surface,
        border = BorderStroke(1.dp, if (selected) YlvenLightColors.Primary else YlvenLightColors.Border),
    ) {
        Box(Modifier.padding(horizontal = 14.dp), contentAlignment = Alignment.Center) {
            Text(label, color = if (selected) YlvenLightColors.Primary else YlvenLightColors.TextSecondary, fontWeight = FontWeight.SemiBold)
        }
    }
}

@Composable
private fun P03LoadingRows(tag: String) {
    Column(Modifier.fillMaxWidth().testTag(tag), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        repeat(3) {
            Surface(Modifier.fillMaxWidth().height(60.dp), shape = RoundedCornerShape(16.dp), color = YlvenLightColors.Border) {}
        }
    }
}

@Composable
private fun P03EmptyState(title: String, message: String, onAction: (() -> Unit)? = null) {
    Column(
        modifier = Modifier.fillMaxWidth().padding(vertical = 32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(title, style = MaterialTheme.typography.titleLarge)
        Text(message, color = YlvenLightColors.TextTertiary, style = MaterialTheme.typography.bodySmall)
        if (onAction != null) TextButton(onClick = onAction) { Text("继续") }
    }
}

@Composable
private fun P03ErrorState(message: String, onRetry: () -> Unit) {
    Column(
        modifier = Modifier.fillMaxWidth().padding(vertical = 32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text("加载失败", style = MaterialTheme.typography.titleLarge)
        Text(message, color = YlvenLightColors.Error, style = MaterialTheme.typography.bodySmall)
        Button(onClick = onRetry, shape = RoundedCornerShape(14.dp)) { Text("重新尝试") }
    }
}

@Composable
private fun P03InlineError(message: String, onRetry: (() -> Unit)? = null) {
    Surface(
        modifier = Modifier.fillMaxWidth().testTag("p01-inline-error"),
        shape = RoundedCornerShape(12.dp),
        color = Color(0xFFFEF3F2),
        border = BorderStroke(1.dp, Color(0xFFFDA29B)),
    ) {
        Row(Modifier.padding(12.dp), verticalAlignment = Alignment.CenterVertically) {
            Text(message, Modifier.weight(1f), color = YlvenLightColors.Error, style = MaterialTheme.typography.bodySmall)
            if (onRetry != null) TextButton(onClick = onRetry) { Text("重试") }
        }
    }
}
