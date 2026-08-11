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
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxHeight
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
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Archive
import androidx.compose.material.icons.filled.AttachFile
import androidx.compose.material.icons.filled.CameraAlt
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Description
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Explore
import androidx.compose.material.icons.filled.FileDownload
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.InsertPhoto
import androidx.compose.material.icons.filled.Menu
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.MoreHoriz
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Palette
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Send
import androidx.compose.material.icons.filled.Slideshow
import androidx.compose.material.icons.filled.Stop
import androidx.compose.material.icons.filled.ThumbDown
import androidx.compose.material.icons.filled.ThumbUp
import androidx.compose.material.icons.filled.TravelExplore
import androidx.compose.material.icons.filled.Work
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.HorizontalDivider
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
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.foundation.text.KeyboardOptions
import cc.orbexa.ylven.identity.AuthSession
import cc.orbexa.ylven.identity.ApiException
import cc.orbexa.ylven.identity.Conversation
import cc.orbexa.ylven.identity.ConversationCache
import cc.orbexa.ylven.identity.HomeSnapshot
import cc.orbexa.ylven.identity.IdentityGateway
import cc.orbexa.ylven.identity.MessageCitation
import cc.orbexa.ylven.identity.MessageRecord
import cc.orbexa.ylven.identity.MessageRun
import cc.orbexa.ylven.identity.ModelOption
import cc.orbexa.ylven.identity.isGenerating
import cc.orbexa.ylven.ui.theme.YlvenDimensions
import cc.orbexa.ylven.ui.theme.YlvenLightColors
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private enum class P03HomeDestination {
    HOME,
    SEARCH,
}

internal fun newP03DraftConversation(
    initialDraft: String = "",
    temporary: Boolean = false,
    initialToolTrayOpen: Boolean = false,
): Conversation {
    val draftSessionId = java.util.UUID.randomUUID().toString()
    return Conversation(
        id = "draft-$draftSessionId",
        title = "新对话",
        status = "draft",
        updatedAt = "",
        draftSessionId = draftSessionId,
        initialDraft = initialDraft,
        temporary = temporary,
        initialToolTrayOpen = initialToolTrayOpen,
    )
}

@Composable
internal fun P03HomePage(
    gateway: IdentityGateway,
    session: AuthSession,
    onOpenConversation: (Conversation) -> Unit,
    onOpenAccount: () -> Unit,
) {
    var destination by rememberSaveable { mutableStateOf(P03HomeDestination.HOME) }
    var drawerOpen by rememberSaveable { mutableStateOf(false) }
    var snapshot by remember { mutableStateOf<HomeSnapshot?>(null) }
    var conversations by remember { mutableStateOf<List<Conversation>>(emptyList()) }
    var searchResults by remember { mutableStateOf<List<Conversation>>(emptyList()) }
    var cursor by remember { mutableStateOf<String?>(null) }
    var searchQuery by rememberSaveable { mutableStateOf("") }
    var loading by remember { mutableStateOf(true) }
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
                error = consumerErrorMessage(reason, "首页加载失败，请重试")
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
                error = consumerErrorMessage(reason, "会话历史加载失败，请重试")
            } finally {
                loading = false
            }
        }
    }

    fun openDraft(initialDraft: String = "", temporary: Boolean = false, openToolTray: Boolean = false) {
        drawerOpen = false
        onOpenConversation(newP03DraftConversation(initialDraft, temporary, openToolTray))
    }

    LaunchedEffect(session.bearer) { loadHome() }
    LaunchedEffect(drawerOpen) { if (drawerOpen) loadHistory(reset = true) }
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
            error = consumerErrorMessage(reason, "搜索失败，请重试")
        } finally {
            loading = false
        }
    }

    BackHandler(enabled = drawerOpen || destination != P03HomeDestination.HOME) {
        if (destination == P03HomeDestination.SEARCH) {
            destination = P03HomeDestination.HOME
            drawerOpen = true
        } else {
            drawerOpen = false
        }
    }

    when (destination) {
        P03HomeDestination.HOME -> Box(Modifier.fillMaxSize()) {
            P03HomeScreen(
                conversations = conversations,
                loading = loading,
                error = error,
                onRetry = ::loadHome,
                onNewConversation = { openDraft() },
                onHistory = { drawerOpen = true },
                onTool = { openDraft(openToolTray = true) },
                onOpenConversation = onOpenConversation,
                onOpenAccount = onOpenAccount,
            )
            if (drawerOpen) {
                P03HistoryDrawerOverlay(
                    conversations = conversations,
                    loading = loading,
                    error = error,
                    canLoadMore = cursor != null,
                    onDismiss = { drawerOpen = false },
                    onSearch = { drawerOpen = false; destination = P03HomeDestination.SEARCH },
                    onNewConversation = { openDraft() },
                    onOpenConversation = { drawerOpen = false; onOpenConversation(it) },
                    onLoadMore = { loadHistory(reset = false) },
                    onRetry = { loadHistory(reset = true) },
                )
            }
        }
        P03HomeDestination.SEARCH -> P03SearchScreen(
            query = searchQuery,
            results = searchResults,
            loading = loading,
            error = error,
            onQueryChange = { searchQuery = it },
            onBack = { destination = P03HomeDestination.HOME; drawerOpen = true },
            onOpenConversation = onOpenConversation,
        )
    }
}

@Composable
private fun P03HomeScreen(
    conversations: List<Conversation>,
    loading: Boolean,
    error: String?,
    onRetry: () -> Unit,
    onNewConversation: () -> Unit,
    onHistory: () -> Unit,
    onTool: () -> Unit,
    onOpenConversation: (Conversation) -> Unit,
    onOpenAccount: () -> Unit,
) {
    Scaffold(
        modifier = Modifier.testTag("YL-A-018-C-P03_001-01"),
        containerColor = YlvenLightColors.Background,
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        topBar = {
            P03BrandTopBar(onHistory, onNewConversation)
        },
        bottomBar = { P03BottomNavigation(onOpenAccount) },
    ) { padding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .testTag("p03-home-list"),
            contentPadding = PaddingValues(horizontal = YlvenDimensions.PageHorizontal, vertical = 30.dp),
            verticalArrangement = Arrangement.spacedBy(18.dp),
        ) {
            item {
                Column(horizontalAlignment = Alignment.CenterHorizontally, modifier = Modifier.fillMaxWidth()) {
                    Text("✦", color = YlvenLightColors.Primary, style = MaterialTheme.typography.headlineLarge)
                    Spacer(Modifier.height(12.dp))
                    Text("今天想完成什么？", style = MaterialTheme.typography.headlineMedium, fontWeight = FontWeight.Bold)
                    Text("多模型协同，完成更复杂的事。", style = MaterialTheme.typography.bodyLarge, color = YlvenLightColors.TextTertiary)
                }
            }
            item {
                P03HomeComposer(onNewConversation)
            }
            item {
                P03ShortcutRow(onTool)
            }
            item {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text("最近对话", modifier = Modifier.weight(1f), style = MaterialTheme.typography.titleLarge)
                    TextButton(onClick = onHistory, modifier = Modifier.testTag("p03-open-all-conversations")) {
                        Text("查看全部")
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
        }
    }
}

@Composable
private fun P03BrandTopBar(onHistory: () -> Unit, onNewConversation: () -> Unit) {
    Surface(color = YlvenLightColors.Surface, border = BorderStroke(1.dp, YlvenLightColors.Divider)) {
        Row(
            modifier = Modifier.fillMaxWidth().statusBarsPadding().height(56.dp).padding(horizontal = 6.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onHistory, modifier = Modifier.size(48.dp).testTag("p03-open-conversation-drawer")) {
                Icon(Icons.Default.Menu, "会话历史", tint = YlvenLightColors.TextPrimary)
            }
            Text("YLVEN", modifier = Modifier.weight(1f), style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold, textAlign = androidx.compose.ui.text.style.TextAlign.Center)
            IconButton(onClick = onNewConversation, modifier = Modifier.size(48.dp).testTag("CO-P03-001-HOME-SEND")) {
                Icon(Icons.Default.Add, "新对话", tint = YlvenLightColors.TextPrimary)
            }
        }
    }
}

@Composable
private fun P03HomeComposer(onOpen: () -> Unit) {
    Surface(
        modifier = Modifier.fillMaxWidth().heightIn(min = 100.dp).clickable(onClick = onOpen).testTag("CO-P03-001-HOME-COMPOSER"),
        shape = RoundedCornerShape(24.dp),
        color = YlvenLightColors.Surface,
        border = BorderStroke(1.dp, YlvenLightColors.Border),
        shadowElevation = 8.dp,
    ) {
        Column(Modifier.padding(horizontal = 16.dp, vertical = 14.dp), verticalArrangement = Arrangement.SpaceBetween) {
            Text("问问 YLVEN…", style = MaterialTheme.typography.titleMedium, color = YlvenLightColors.TextDisabled)
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Default.Add, "添加内容", tint = YlvenLightColors.TextSecondary, modifier = Modifier.size(28.dp))
                Spacer(Modifier.width(12.dp))
                Surface(shape = RoundedCornerShape(18.dp), color = YlvenLightColors.SurfaceBrandSoft) {
                    Text("自动选择", Modifier.padding(horizontal = 14.dp, vertical = 7.dp), color = YlvenLightColors.Primary)
                }
                Spacer(Modifier.weight(1f))
                Icon(Icons.Default.Mic, "语音输入", tint = YlvenLightColors.TextSecondary, modifier = Modifier.size(28.dp))
                Spacer(Modifier.width(10.dp))
                Surface(shape = CircleShape, color = YlvenLightColors.Primary, modifier = Modifier.size(46.dp)) {
                    Icon(Icons.Default.Send, "开始对话", Modifier.padding(11.dp), tint = Color.White)
                }
            }
        }
    }
}

@Composable
private fun P03ShortcutRow(onTool: () -> Unit) {
    val shortcuts = listOf(
        "分析文件" to "文",
        "生成图片" to "图",
        "制作演示" to "P",
    )
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        shortcuts.forEachIndexed { index, (label, symbol) ->
            Surface(
                modifier = Modifier.weight(1f).height(42.dp).clickable(onClick = onTool)
                    .testTag(if (index == 0) "CO-P03-001-HOME-TOOL" else "p03-home-tool-$index"),
                shape = RoundedCornerShape(21.dp),
                color = YlvenLightColors.Surface,
                border = BorderStroke(1.dp, YlvenLightColors.Border),
            ) {
                Row(Modifier.padding(horizontal = 10.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.Center) {
                    Surface(shape = CircleShape, color = YlvenLightColors.SurfaceBrandSoft, modifier = Modifier.size(26.dp)) {
                        Box(contentAlignment = Alignment.Center) {
                            Text(symbol, color = YlvenLightColors.Primary, fontWeight = FontWeight.Bold)
                        }
                    }
                    Spacer(Modifier.width(6.dp))
                    Text(label, style = MaterialTheme.typography.labelLarge, maxLines = 1)
                }
            }
        }
    }
}

@Composable
private fun P03HistoryDrawerOverlay(
    conversations: List<Conversation>,
    loading: Boolean,
    error: String?,
    canLoadMore: Boolean,
    onDismiss: () -> Unit,
    onSearch: () -> Unit,
    onNewConversation: () -> Unit,
    onOpenConversation: (Conversation) -> Unit,
    onLoadMore: () -> Unit,
    onRetry: () -> Unit,
) {
    BackHandler(onBack = onDismiss)
    BoxWithConstraints(
        modifier = Modifier
            .fillMaxSize()
            .statusBarsPadding()
            .testTag("YL-A-020-root"),
    ) {
        val drawerWidth = if (maxWidth >= 360.dp) 304.dp else if (maxWidth > 56.dp) maxWidth - 56.dp else maxWidth
        Box(
            Modifier
                .width(maxWidth - drawerWidth)
                .fillMaxHeight()
                .align(Alignment.CenterEnd)
                .background(Color.Black.copy(alpha = 0.32f))
                .clickable(onClick = onDismiss)
                .testTag("p03-drawer-scrim"),
        )
        var dragDistance by remember { mutableStateOf(0f) }
        Surface(
            modifier = Modifier
                .width(drawerWidth)
                .fillMaxHeight()
                .pointerInput(onDismiss) {
                    detectHorizontalDragGestures(
                        onDragStart = { dragDistance = 0f },
                        onHorizontalDrag = { change, amount ->
                            change.consume()
                            dragDistance += amount
                        },
                        onDragEnd = {
                            if (dragDistance < -64f) onDismiss()
                            dragDistance = 0f
                        },
                    )
                }
                .testTag("p03-conversation-drawer"),
            shape = RoundedCornerShape(topEnd = 16.dp, bottomEnd = 16.dp),
            color = YlvenLightColors.Surface,
            shadowElevation = 12.dp,
        ) {
            Column(Modifier.fillMaxSize()) {
                Row(
                    Modifier.fillMaxWidth().height(56.dp).padding(start = 16.dp, end = 8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text("YLVEN", Modifier.weight(1f), style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
                    IconButton(onClick = onNewConversation, modifier = Modifier.testTag("p03-drawer-new-conversation")) {
                        Icon(Icons.Default.Add, "新对话")
                    }
                }
                Surface(
                    modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 6.dp).height(48.dp)
                        .clickable(onClick = onSearch).testTag("p03-open-search"),
                    shape = RoundedCornerShape(24.dp),
                    color = YlvenLightColors.SurfaceSubtle,
                ) {
                    Row(Modifier.padding(horizontal = 14.dp), verticalAlignment = Alignment.CenterVertically) {
                        Icon(Icons.Default.Search, null, tint = YlvenLightColors.TextTertiary)
                        Spacer(Modifier.width(10.dp))
                        Text("搜索对话", color = YlvenLightColors.TextDisabled, style = MaterialTheme.typography.bodyLarge)
                    }
                }
                Surface(
                    modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 6.dp).height(48.dp)
                        .clickable(onClick = onNewConversation).testTag("p03-drawer-start-conversation"),
                    shape = RoundedCornerShape(12.dp),
                    color = YlvenLightColors.SurfaceBrandSoft,
                ) {
                    Row(Modifier.padding(horizontal = 14.dp), verticalAlignment = Alignment.CenterVertically) {
                        Icon(Icons.Default.Add, null, tint = YlvenLightColors.Primary)
                        Spacer(Modifier.width(10.dp))
                        Text("开始新对话", color = YlvenLightColors.Primary, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                    }
                }
                LazyColumn(
                    modifier = Modifier.fillMaxWidth().weight(1f),
                    contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
                    verticalArrangement = Arrangement.spacedBy(2.dp),
                ) {
                    when {
                        loading && conversations.isEmpty() -> item { P03LoadingRows("yl-a-020-loading") }
                        error != null && conversations.isEmpty() -> item { P03ErrorState(error, onRetry) }
                        conversations.isEmpty() -> item { P03EmptyState("暂无会话", "新建会话后会显示在这里", onNewConversation) }
                        else -> {
                            val grouped = conversations.groupBy(::conversationGroupLabel)
                            grouped.forEach { (group, groupItems) ->
                                item(group) {
                                    Text(group, Modifier.padding(top = 14.dp, bottom = 6.dp), color = YlvenLightColors.TextSecondary, style = MaterialTheme.typography.titleMedium)
                                }
                                items(groupItems, key = { it.id }) { conversation ->
                                    P03DrawerConversationRow(conversation, onOpenConversation)
                                }
                            }
                        }
                    }
                    if (canLoadMore) {
                        item {
                            TextButton(
                                onClick = onLoadMore,
                                modifier = Modifier.fillMaxWidth().height(48.dp).testTag("YL-A-020-C-P03_003-01"),
                            ) { Text("加载更多") }
                        }
                    }
                    if (error != null && conversations.isNotEmpty()) item { P03InlineError(error, onRetry) }
                }
            }
        }
    }
}

@Composable
private fun P03DrawerConversationRow(conversation: Conversation, onOpen: (Conversation) -> Unit) {
    Row(
        modifier = Modifier.fillMaxWidth().heightIn(min = 58.dp).clickable { onOpen(conversation) }
            .padding(horizontal = 4.dp, vertical = 7.dp).testTag("p03-conversation-${conversation.id}"),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Text(conversation.title.ifBlank { "新对话" }, style = MaterialTheme.typography.titleMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
            Text(conversationMeta(conversation), color = YlvenLightColors.TextTertiary, style = MaterialTheme.typography.bodySmall, maxLines = 1)
        }
        Icon(Icons.Default.MoreHoriz, "打开会话", tint = YlvenLightColors.TextTertiary)
    }
}

private fun conversationGroupLabel(conversation: Conversation): String {
    val value = conversation.updatedAt.trim()
    if (value.isBlank() || value.contains("刚刚") || value.contains("今天")) return "今天"
    return runCatching {
        val date = java.time.Instant.parse(value).atZone(java.time.ZoneId.systemDefault()).toLocalDate()
        if (date == java.time.LocalDate.now()) "今天" else "过去 7 天"
    }.getOrDefault("过去 7 天")
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
internal fun P03ChatPage(
    gateway: IdentityGateway,
    session: AuthSession,
    conversation: Conversation,
    onBack: () -> Unit,
    onOpenConversation: (Conversation) -> Unit,
) {
    var conversationId by rememberSaveable(conversation.id) { mutableStateOf(conversation.id) }
    var draftSessionId by rememberSaveable(conversation.id) { mutableStateOf(conversation.draftSessionId.orEmpty()) }
    var formalConversation by rememberSaveable(conversation.id) { mutableStateOf(conversation.draftSessionId == null) }
    var title by remember(conversation.id) { mutableStateOf(conversation.title.ifBlank { "新对话" }) }
    var draft by rememberSaveable(conversation.id) { mutableStateOf(conversation.initialDraft) }
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
    var pendingIdempotencyKey by rememberSaveable(conversation.id) { mutableStateOf("") }
    var pendingBody by rememberSaveable(conversation.id) { mutableStateOf("") }
    var models by remember(conversation.id) { mutableStateOf<List<ModelOption>>(emptyList()) }
    var modelsLoading by remember(conversation.id) { mutableStateOf(true) }
    var modelsError by remember(conversation.id) { mutableStateOf<String?>(null) }
    var selectedModelId by rememberSaveable(conversation.id) { mutableStateOf("") }
    var selectedModelName by rememberSaveable(conversation.id) { mutableStateOf("自动选择") }
    var selectedResponseMode by rememberSaveable(conversation.id) { mutableStateOf("auto") }
    var toolTrayOpen by rememberSaveable(conversation.id) { mutableStateOf(conversation.initialToolTrayOpen) }
    var modelSelectorOpen by rememberSaveable(conversation.id) { mutableStateOf(false) }
    var responseModeSelectorOpen by rememberSaveable(conversation.id) { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val clipboard = LocalClipboardManager.current
    val context = LocalContext.current
    val focusManager = LocalFocusManager.current
    val inputFocusRequester = remember { FocusRequester() }
    val cache = remember { ConversationCache(context) }
    val voiceLauncher = rememberLauncherForActivityResult(ActivityResultContracts.StartActivityForResult()) { result ->
        result.data?.getStringArrayListExtra(RecognizerIntent.EXTRA_RESULTS)?.firstOrNull()?.let { draft = it }
    }
    val tts = remember(context) { TextToSpeech(context, null) }
    DisposableEffect(tts) { onDispose { tts.shutdown() } }

    fun loadModels() {
        scope.launch {
            modelsLoading = true
            modelsError = null
            try {
                models = gateway.models(session.bearer)
            } catch (reason: Exception) {
                modelsError = consumerErrorMessage(reason, "暂时无法加载模型，请重试")
            } finally {
                modelsLoading = false
            }
        }
    }

    LaunchedEffect(conversation.id) {
        if (formalConversation) {
            messages = cache.load(conversationId)
            runCatching { gateway.loadDraft(session.bearer, conversationId) }.onSuccess { draft = it }
        }
        draftLoaded = true
    }
    LaunchedEffect(session.bearer, conversation.id) { loadModels() }
    LaunchedEffect(conversation.id, formalConversation, toolTrayOpen) {
        if (!formalConversation && !toolTrayOpen) {
            delay(180)
            runCatching { inputFocusRequester.requestFocus() }
        }
    }
    LaunchedEffect(draftLoaded, draft, formalConversation, conversationId) {
        if (!draftLoaded || !formalConversation) return@LaunchedEffect
        delay(500)
        runCatching { gateway.saveDraft(session.bearer, conversationId, draft) }
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
                cache.save(conversationId, messages)
                reconnecting = false
                consecutiveFailures = 0
                if (snapshot.first.status.lowercase() in setOf("completed", "cancelled", "failed", "content_blocked")) {
                    if (snapshot.first.status.equals("completed", ignoreCase = true)) {
                        runCatching { gateway.listConversations(session.bearer).first.firstOrNull { it.id == conversationId } }
                            .getOrNull()
                            ?.let { title = it.title }
                    } else if (snapshot.first.status.equals("failed", ignoreCase = true)) {
                        error = "暂时无法完成回答，请重试"
                    } else if (snapshot.first.status.equals("content_blocked", ignoreCase = true)) {
                        error = "这项请求暂时无法完成，请调整后重试"
                    }
                    return
                }
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
            if (draftLoaded && formalConversation) runCatching { gateway.saveDraft(session.bearer, conversationId, draft) }
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
        if (pendingBody != body || pendingIdempotencyKey.isBlank()) {
            pendingBody = body
            pendingIdempotencyKey = java.util.UUID.randomUUID().toString()
        }
        val idempotencyKey = pendingIdempotencyKey
        val retryingFailedRequest = retryBody == body && pendingBody == body
        focusManager.clearFocus(force = true)
        scope.launch {
            sending = true
            error = null
            retryBody = null
            if (!retryingFailedRequest) {
                messages = messages + MessageRecord("optimistic-${System.currentTimeMillis()}", conversationId, "user", body, "")
            }
            try {
                val created = if (formalConversation) {
                    gateway.sendMessageIdempotent(
                        session.bearer,
                        conversationId,
                        body,
                        model = selectedModelId,
                        idempotencyKey = idempotencyKey,
                    )
                } else {
                    val first = gateway.startConversationFromFirstMessage(
                        bearer = session.bearer,
                        draftSessionId = draftSessionId,
                        body = body,
                        model = selectedModelId,
                        idempotencyKey = idempotencyKey,
                        temporary = conversation.temporary,
                    )
                    conversationId = first.conversation.id
                    title = first.conversation.title
                    formalConversation = true
                    first.run
                }
                draft = ""
                pendingBody = ""
                pendingIdempotencyKey = ""
                run = created
                observeRun(created.id)
                cache.save(conversationId, messages)
            } catch (reason: Exception) {
                error = consumerErrorMessage(reason, "暂时无法完成回答，请重试")
                retryBody = body
            } finally {
                if (formalConversation) runCatching { gateway.saveDraft(session.bearer, conversationId, draft) }
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
                subtitle = "$selectedModelName · ${responseModeLabel(selectedResponseMode)}",
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
            Column(Modifier.fillMaxWidth()) {
                val failedBody = retryBody
                if (error != null || failedBody != null) {
                    Box(Modifier.fillMaxWidth().padding(horizontal = 16.dp)) {
                        Box(Modifier.testTag("YL-A-023-C-P03_027-01")) {
                            P03InlineError(
                                message = error ?: "发送失败，请重试",
                                onRetry = failedBody?.let { body ->
                                {
                                    draft = body
                                    error = null
                                    send()
                                }
                                },
                                retryTag = if (failedBody != null) "CO-P03-001-CHAT-RETRY" else null,
                            )
                        }
                    }
                }
                P03Composer(
                    draft = draft,
                    onDraftChange = { draft = it },
                    sending = sending || run?.isGenerating == true,
                    modelLabel = selectedModelName,
                    responseModeLabel = responseModeLabel(selectedResponseMode),
                    focusRequester = inputFocusRequester,
                    onOpenTools = { toolTrayOpen = true },
                    onOpenModels = { modelSelectorOpen = true },
                    onOpenResponseMode = { responseModeSelectorOpen = true },
                    onSend = ::send,
                    onStop = {
                        run?.let { active ->
                            scope.launch {
                                runCatching { gateway.cancelRun(session.bearer, active.id) }
                                    .onSuccess { run = it }
                                    .onFailure { error = consumerErrorMessage(it, "暂时无法停止，请重试") }
                            }
                        }
                    },
                    onVoice = ::openVoiceInput,
                )
            }
        },
    ) { padding ->
        Box(
            Modifier
                .fillMaxSize()
                .padding(padding)
                .testTag(if (formalConversation) "p03-active-conversation-$conversationId" else "p03-local-draft-conversation"),
        ) {
            Box(Modifier.fillMaxSize().testTag("YL-A-023-C-P03_017-01")) {
            LazyColumn(
                modifier = Modifier.fillMaxSize().testTag("YL-A-025-C-P03_016-01"),
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 18.dp),
                verticalArrangement = Arrangement.spacedBy(20.dp),
            ) {
                item {
                    Text(
                        "$selectedModelName · ${responseModeLabel(selectedResponseMode)}",
                        modifier = Modifier
                            .background(YlvenLightColors.SurfaceBrandSoft, RoundedCornerShape(18.dp))
                            .padding(horizontal = 12.dp, vertical = 6.dp)
                            .testTag("p03-chat-start"),
                        color = YlvenLightColors.Primary,
                        style = MaterialTheme.typography.labelLarge,
                    )
                }
                if (messages.isEmpty() && !sending) {
                    item { P03EmptyState("开始这次对话", "在下方输入问题，回答会显示在这里") }
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
                        Column(
                            modifier = Modifier.testTag("YL-A-026-C-P03_026-01"),
                            verticalArrangement = Arrangement.spacedBy(8.dp),
                        ) {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Text("✦", color = YlvenLightColors.Primary, style = MaterialTheme.typography.titleMedium)
                                Spacer(Modifier.width(8.dp))
                                Text("YLVEN", color = YlvenLightColors.TextSecondary, style = MaterialTheme.typography.labelLarge)
                            }
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
                                        runCatching { observeRun(nextRun.id) }.onFailure { error = consumerErrorMessage(it, "暂时无法重新回答，请重试") }
                                    }
                                },
                                onError = { error = it },
                                onChangeModel = { modelSelectorOpen = true },
                            )
                        }
                    }
                }
                val activeAssistantHasContent = run?.assistantMessageId?.let { assistantId ->
                    messages.firstOrNull { it.id == assistantId }?.body?.isNotBlank()
                } == true
                if ((sending || run?.isGenerating == true) && !activeAssistantHasContent) item {
                    Text("正在思考", color = YlvenLightColors.Primary, modifier = Modifier.testTag("YL-A-023-C-P03_010-01"))
                }
                if (reconnecting) item {
                    Text("连接不稳定，正在恢复…", color = YlvenLightColors.Warning, modifier = Modifier.testTag("YL-A-023-C-P03_012-01"))
                }
            }
            }
        }
    }

    if (menuOpen) {
        P03ConversationMenu(
            title = title,
            formalConversation = formalConversation,
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
                    runCatching { gateway.renameConversation(session.bearer, conversationId, renameText.trim()) }
                        .onSuccess { updated -> title = updated.title; renameMode = false; menuOpen = false }
                        .onFailure { menuMessage = consumerErrorMessage(it, "暂时无法重命名，请重试") }
                    menuBusy = false
                }
            },
            onArchive = {
                scope.launch {
                    menuBusy = true
                    runCatching { gateway.archiveConversation(session.bearer, conversationId) }
                        .onSuccess { menuOpen = false; leaveChat() }
                        .onFailure { menuMessage = consumerErrorMessage(it, "暂时无法归档，请重试") }
                    menuBusy = false
                }
            },
            onExport = {
                scope.launch {
                    menuBusy = true
                    runCatching { gateway.exportConversation(session.bearer, conversationId) }
                        .onSuccess { markdown -> clipboard.setText(AnnotatedString(markdown)); menuOpen = false }
                        .onFailure { menuMessage = consumerErrorMessage(it, "暂时无法导出，请重试") }
                    menuBusy = false
                }
            },
            onRequestDelete = { confirmDelete = true; renameMode = false },
            onDelete = {
                scope.launch {
                    menuBusy = true
                    runCatching { gateway.deleteConversation(session.bearer, conversationId) }
                        .onSuccess { menuOpen = false; leaveChat() }
                        .onFailure { menuMessage = consumerErrorMessage(it, "暂时无法删除，请重试") }
                    menuBusy = false
                }
            },
            onTemporaryConversation = {
                menuOpen = false
                onOpenConversation(newP03DraftConversation(temporary = true))
            },
        )
    }

    if (toolTrayOpen) {
        P03ToolTray(onDismiss = { toolTrayOpen = false })
    }
    if (modelSelectorOpen) {
        P03ModelSelector(
            models = models,
            loading = modelsLoading,
            error = modelsError,
            selectedModelId = selectedModelId,
            onDismiss = { modelSelectorOpen = false },
            onRetry = ::loadModels,
            onSelect = { option ->
                selectedModelId = option?.id.orEmpty()
                selectedModelName = option?.name ?: "自动选择"
                val supported = option?.reasoningProfiles.orEmpty()
                if (selectedResponseMode !in supported && selectedResponseMode != "auto") selectedResponseMode = "auto"
                modelSelectorOpen = false
            },
        )
    }
    if (responseModeSelectorOpen) {
        P03ResponseModeSelector(
            selectedMode = selectedResponseMode,
            supportedModes = models.firstOrNull { it.id == selectedModelId }?.reasoningProfiles.orEmpty().ifEmpty { listOf("auto") },
            onDismiss = { responseModeSelectorOpen = false },
            onSelect = { mode -> selectedResponseMode = mode; responseModeSelectorOpen = false },
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
    onChangeModel: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    Row(Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        P03TextAction("复制", "YL-A-030-C-P03_022-01") { clipboardCopy(message.body) }
        P03TextAction("朗读", "YL-A-030-C-P03_030-01") {
            scope.launch {
                runCatching {
                    gateway.speak(session.bearer, message.id)
                    tts.speak(message.body, TextToSpeech.QUEUE_FLUSH, null, message.id)
                }.onFailure { onError(consumerErrorMessage(it, "暂时无法朗读，请重试")) }
            }
        }
        P03TextAction("重新回答", "YL-A-030-C-P03_024-01") {
            scope.launch {
                runCatching { gateway.regenerate(session.bearer, message.id) }
                    .onSuccess(onRun)
                    .onFailure { onError(consumerErrorMessage(it, "暂时无法重新回答，请重试")) }
            }
        }
        P03TextAction("换模型", "p03-message-change-model", onChangeModel)
        P03SmallAction(Icons.Default.FileDownload, "导出", "YL-A-030-C-P03_023-01") {
            scope.launch {
                runCatching { gateway.exportMessage(session.bearer, message.id) }
                    .onSuccess(clipboardCopy)
                    .onFailure { onError(consumerErrorMessage(it, "暂时无法导出，请重试")) }
            }
        }
        P03SmallAction(Icons.Default.ThumbUp, "赞", "YL-A-030-C-P03_025-01") {
            scope.launch { runCatching { gateway.submitFeedback(session.bearer, message.id, "up") }.onFailure { onError(consumerErrorMessage(it, "暂时无法提交反馈，请重试")) } }
        }
        P03SmallAction(Icons.Default.ThumbDown, "踩", "YL-A-030-C-P03_025-01") {
            scope.launch { runCatching { gateway.submitFeedback(session.bearer, message.id, "down") }.onFailure { onError(consumerErrorMessage(it, "暂时无法提交反馈，请重试")) } }
        }
    }
}

@Composable
private fun P03TextAction(label: String, tag: String, onClick: () -> Unit) {
    OutlinedButton(
        onClick = onClick,
        modifier = Modifier.height(36.dp).testTag(tag),
        contentPadding = PaddingValues(horizontal = 12.dp, vertical = 0.dp),
        shape = RoundedCornerShape(18.dp),
        border = BorderStroke(1.dp, YlvenLightColors.Border),
    ) { Text(label, style = MaterialTheme.typography.labelLarge, color = YlvenLightColors.TextSecondary) }
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
    modelLabel: String,
    responseModeLabel: String,
    focusRequester: FocusRequester,
    onOpenTools: () -> Unit,
    onOpenModels: () -> Unit,
    onOpenResponseMode: () -> Unit,
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
            Column(Modifier.heightIn(min = 72.dp, max = 164.dp).padding(horizontal = 8.dp, vertical = 6.dp)) {
                Box(Modifier.fillMaxWidth().weight(1f).padding(horizontal = 8.dp, vertical = 6.dp)) {
                    if (draft.isEmpty()) Text("问问 YLVEN…", color = YlvenLightColors.TextDisabled)
                    BasicTextField(
                        value = draft,
                        onValueChange = onDraftChange,
                        modifier = Modifier.fillMaxWidth().focusRequester(focusRequester).testTag("YL-A-032-C-P03_032-01"),
                        textStyle = MaterialTheme.typography.bodyLarge.copy(color = YlvenLightColors.TextPrimary),
                        maxLines = 6,
                        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Send),
                        keyboardActions = KeyboardActions(onSend = { if (!sending && draft.isNotBlank()) onSend() }),
                    )
                }
                Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                    IconButton(onClick = onOpenTools, modifier = Modifier.size(40.dp).testTag("p03-open-tool-tray")) {
                        Icon(Icons.Default.Add, "添加内容或工具", tint = YlvenLightColors.TextSecondary)
                    }
                    P03ComposerChip(modelLabel, "CO-P03-001-CHAT-MODEL", onOpenModels)
                    Spacer(Modifier.width(6.dp))
                    P03ComposerChip(responseModeLabel, "CO-P03-001-CHAT-REASONING", onOpenResponseMode)
                    Spacer(Modifier.weight(1f))
                    IconButton(onClick = onVoice, modifier = Modifier.size(40.dp).testTag("YL-A-032-C-P03_031-01")) {
                        Icon(Icons.Default.Mic, "语音输入", tint = YlvenLightColors.TextSecondary)
                    }
                    Surface(shape = CircleShape, color = YlvenLightColors.Primary, modifier = Modifier.size(40.dp)) {
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
}

@Composable
private fun P03ComposerChip(label: String, tag: String, onClick: () -> Unit) {
    Surface(
        modifier = Modifier.height(34.dp).widthIn(max = 112.dp).clickable(onClick = onClick).testTag(tag),
        shape = RoundedCornerShape(17.dp),
        color = YlvenLightColors.SurfaceBrandSoft,
    ) {
        Box(Modifier.padding(horizontal = 10.dp), contentAlignment = Alignment.Center) {
            Text(label, color = YlvenLightColors.Primary, style = MaterialTheme.typography.labelMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
    }
}

private data class P03ComposerTool(
    val label: String,
    val icon: androidx.compose.ui.graphics.vector.ImageVector,
    val enabled: Boolean,
)

@Composable
@OptIn(ExperimentalMaterial3Api::class)
private fun P03ToolTray(onDismiss: () -> Unit) {
    val tools = listOf(
        P03ComposerTool("拍照", Icons.Default.CameraAlt, false),
        P03ComposerTool("选择图片", Icons.Default.InsertPhoto, false),
        P03ComposerTool("上传文件", Icons.Default.Description, false),
        P03ComposerTool("生成图片", Icons.Default.Palette, false),
        P03ComposerTool("制作演示", Icons.Default.Slideshow, false),
        P03ComposerTool("深度研究", Icons.Default.TravelExplore, false),
    )
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        modifier = Modifier.testTag("YL-A-024-S06-tool-tray"),
        shape = RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp),
        containerColor = YlvenLightColors.Surface,
    ) {
        Column(
            Modifier.fillMaxWidth().padding(horizontal = 18.dp).navigationBarsPadding(),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            Text("添加内容或使用工具", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
            tools.chunked(3).forEachIndexed { rowIndex, rowTools ->
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    rowTools.forEachIndexed { columnIndex, tool ->
                        P03ToolTile(tool, "p03-tool-${rowIndex * 3 + columnIndex}", Modifier.weight(1f))
                    }
                }
            }
            Spacer(Modifier.height(18.dp))
        }
    }
}

@Composable
private fun P03ToolTile(tool: P03ComposerTool, tag: String, modifier: Modifier = Modifier) {
    val contentColor = if (tool.enabled) YlvenLightColors.Primary else YlvenLightColors.TextDisabled
    Surface(
        modifier = modifier.height(74.dp).clickable(enabled = tool.enabled, onClick = {}).testTag(tag),
        shape = RoundedCornerShape(8.dp),
        color = YlvenLightColors.SurfaceSubtle,
    ) {
        Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
            Icon(tool.icon, tool.label, tint = contentColor, modifier = Modifier.size(28.dp))
            Spacer(Modifier.height(4.dp))
            Text(tool.label, color = contentColor, style = MaterialTheme.typography.labelLarge)
        }
    }
}

@Composable
@OptIn(ExperimentalMaterial3Api::class)
private fun P03ModelSelector(
    models: List<ModelOption>,
    loading: Boolean,
    error: String?,
    selectedModelId: String,
    onDismiss: () -> Unit,
    onRetry: () -> Unit,
    onSelect: (ModelOption?) -> Unit,
) {
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        modifier = Modifier.testTag("YL-A-033-root"),
        shape = RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp),
        containerColor = YlvenLightColors.Surface,
    ) {
        Column(
            Modifier.fillMaxWidth().padding(horizontal = 18.dp).navigationBarsPadding(),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text("选择模型", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
            Text("默认由 YLVEN 自动匹配适合的模型。", color = YlvenLightColors.TextSecondary, style = MaterialTheme.typography.bodyMedium)
            P03SelectorRow(
                title = "自动选择",
                description = "根据问题和可用能力自动匹配",
                selected = selectedModelId.isBlank(),
                enabled = true,
                tag = "p03-model-auto",
                onClick = { onSelect(null) },
            )
            when {
                loading -> Box(Modifier.fillMaxWidth().height(92.dp), contentAlignment = Alignment.Center) { CircularProgressIndicator(Modifier.size(28.dp)) }
                error != null -> P03InlineError(error, onRetry)
                models.isEmpty() -> P03EmptyState("暂无可用模型", "可以继续使用自动选择")
                else -> models.forEach { model ->
                    P03SelectorRow(
                        title = model.name,
                        description = model.description,
                        selected = model.id == selectedModelId,
                        enabled = model.enabled,
                        tag = "p03-model-${model.id}",
                        onClick = { onSelect(model) },
                    )
                }
            }
            Spacer(Modifier.height(18.dp))
        }
    }
}

@Composable
@OptIn(ExperimentalMaterial3Api::class)
private fun P03ResponseModeSelector(
    selectedMode: String,
    supportedModes: List<String>,
    onDismiss: () -> Unit,
    onSelect: (String) -> Unit,
) {
    val supported = supportedModes.map(::normalizeResponseMode).toSet() + "auto"
    val modes = listOf(
        Triple("auto", "自动（推荐）", "根据问题复杂度自动决定"),
        Triple("quick", "快速", "更快响应，适合简单问题"),
        Triple("standard", "标准", "速度与质量平衡"),
        Triple("deep", "深度", "适合复杂分析和代码任务"),
    )
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        modifier = Modifier.testTag("YL-A-034-root"),
        shape = RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp),
        containerColor = YlvenLightColors.Surface,
    ) {
        Column(
            Modifier.fillMaxWidth().padding(horizontal = 18.dp).navigationBarsPadding(),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text("回答方式", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
            Text("可用方式由当前模型决定。", color = YlvenLightColors.TextSecondary, style = MaterialTheme.typography.bodyMedium)
            modes.forEach { (id, title, description) ->
                P03SelectorRow(
                    title = title,
                    description = description,
                    selected = id == selectedMode,
                    enabled = id in supported,
                    tag = "p03-response-mode-$id",
                    onClick = { onSelect(id) },
                )
            }
            Spacer(Modifier.height(18.dp))
        }
    }
}

@Composable
private fun P03SelectorRow(
    title: String,
    description: String,
    selected: Boolean,
    enabled: Boolean,
    tag: String,
    onClick: () -> Unit,
) {
    val border = if (selected) YlvenLightColors.Primary else YlvenLightColors.Border
    Surface(
        modifier = Modifier.fillMaxWidth().heightIn(min = 64.dp).clickable(enabled = enabled, onClick = onClick).testTag(tag),
        shape = RoundedCornerShape(8.dp),
        color = if (selected) YlvenLightColors.SurfaceBrandSoft else YlvenLightColors.Surface,
        border = BorderStroke(if (selected) 2.dp else 1.dp, border),
    ) {
        Row(Modifier.padding(horizontal = 14.dp, vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(title, color = if (enabled) YlvenLightColors.TextPrimary else YlvenLightColors.TextDisabled, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                if (description.isNotBlank()) {
                    Text(description, color = if (enabled) YlvenLightColors.TextSecondary else YlvenLightColors.TextDisabled, style = MaterialTheme.typography.bodySmall, maxLines = 2)
                }
            }
            if (selected) {
                Surface(shape = CircleShape, color = YlvenLightColors.Primary, modifier = Modifier.size(24.dp)) {
                    Icon(Icons.Default.Check, "已选择", Modifier.padding(4.dp), tint = Color.White)
                }
            }
        }
    }
}

private fun normalizeResponseMode(value: String): String = when (value.trim().lowercase()) {
    "fast" -> "quick"
    "balanced" -> "standard"
    "reasoning" -> "deep"
    else -> value.trim().lowercase()
}

private fun responseModeLabel(value: String): String = when (normalizeResponseMode(value)) {
    "quick" -> "快速"
    "standard" -> "标准"
    "deep" -> "深度"
    else -> "自动"
}

@Composable
@OptIn(ExperimentalMaterial3Api::class)
private fun P03ConversationMenu(
    title: String,
    formalConversation: Boolean,
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
    onTemporaryConversation: () -> Unit,
) {
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        modifier = Modifier.fillMaxWidth().padding(horizontal = 24.dp).testTag("YL-A-022-root"),
        shape = RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp),
        containerColor = YlvenLightColors.Surface,
        dragHandle = {
            Surface(Modifier.padding(top = 10.dp).size(width = 60.dp, height = 5.dp), shape = CircleShape, color = YlvenLightColors.BorderStrong) {}
        },
    ) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp).navigationBarsPadding().testTag("p03-conversation-menu"), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text("会话操作", style = MaterialTheme.typography.headlineSmall)
            if (formalConversation) Text(title, color = YlvenLightColors.TextTertiary, style = MaterialTheme.typography.bodySmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
            if (renameMode && formalConversation) {
                OutlinedTextField(
                    value = renameText,
                    onValueChange = onRenameTextChange,
                    modifier = Modifier.fillMaxWidth().testTag("YL-A-022-C-P03_005-01"),
                    singleLine = true,
                    shape = RoundedCornerShape(14.dp),
                    label = { Text("会话名称") },
                )
                Button(
                    onClick = onRename,
                    enabled = renameText.isNotBlank() && !busy,
                    modifier = Modifier.fillMaxWidth().height(52.dp).testTag("CO-P03-001-TITLE-RENAME"),
                    shape = RoundedCornerShape(14.dp),
                ) { Text(if (busy) "正在保存" else "保存名称") }
            } else if (confirmDelete && formalConversation) {
                Text("删除后会话将进入回收策略，是否继续？", color = YlvenLightColors.Error)
                Button(
                    onClick = onDelete,
                    enabled = !busy,
                    modifier = Modifier.fillMaxWidth().height(52.dp).testTag("p03-confirm-delete"),
                    colors = ButtonDefaults.buttonColors(containerColor = YlvenLightColors.Error),
                    shape = RoundedCornerShape(14.dp),
                ) { Text(if (busy) "正在删除" else "确认删除") }
            } else {
                if (formalConversation) {
                    P03MenuRow(Icons.Default.Edit, "重命名", "p03-open-rename-current", !busy, onStartRename)
                    HorizontalDivider(color = YlvenLightColors.Divider)
                    P03MenuRow(Icons.Default.Folder, "移入项目", null, enabled = false, onClick = {})
                    HorizontalDivider(color = YlvenLightColors.Divider)
                    P03MenuRow(Icons.Default.Archive, "归档", "YL-A-022-C-P03_006-01", !busy, onArchive)
                    HorizontalDivider(color = YlvenLightColors.Divider)
                    P03MenuRow(Icons.Default.FileDownload, "导出", "YL-A-022-C-P03_029-01", !busy, onExport)
                    HorizontalDivider(color = YlvenLightColors.Divider)
                }
                P03MenuRow(Icons.Default.MoreHoriz, "临时对话", "YL-A-031-C-P03_028-01", !busy, onTemporaryConversation)
                if (formalConversation) {
                    HorizontalDivider(color = YlvenLightColors.Divider)
                    P03MenuRow(Icons.Default.Delete, "删除", "YL-A-022-C-P03_007-01", !busy, onRequestDelete, destructive = true)
                }
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
    val modifier = Modifier.fillMaxWidth().height(48.dp).clickable(enabled = enabled, onClick = onClick)
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
private fun P03InlineError(
    message: String,
    onRetry: (() -> Unit)? = null,
    retryTag: String? = null,
) {
    Surface(
        modifier = Modifier.fillMaxWidth().testTag("p01-inline-error"),
        shape = RoundedCornerShape(12.dp),
        color = Color(0xFFFEF3F2),
        border = BorderStroke(1.dp, Color(0xFFFDA29B)),
    ) {
        Row(Modifier.padding(12.dp), verticalAlignment = Alignment.CenterVertically) {
            Text(message, Modifier.weight(1f), color = YlvenLightColors.Error, style = MaterialTheme.typography.bodySmall)
            if (onRetry != null) {
                TextButton(
                    onClick = onRetry,
                    modifier = if (retryTag == null) Modifier else Modifier.testTag(retryTag),
                ) { Text("重试") }
            }
        }
    }
}

internal fun consumerErrorMessage(reason: Throwable, fallback: String): String = when (reason) {
    is ApiException -> when {
        reason.status == 401 || reason.status == 403 -> "登录状态已失效，请重新登录"
        reason.status == 408 -> "连接超时，请重试"
        reason.status == 429 -> "当前请求较多，请稍后重试"
        reason.code == "content_blocked" -> "这项请求暂时无法完成，请调整后重试"
        reason.status >= 500 -> "服务暂时繁忙，请稍后重试"
        else -> fallback
    }
    is java.net.SocketTimeoutException -> "连接超时，请重试"
    is java.io.IOException -> "网络不可用，请检查连接后重试"
    else -> fallback
}
