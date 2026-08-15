package cc.orbexa.ylven.ui

import android.app.Activity
import android.content.Intent
import android.content.Context
import android.content.ContextWrapper
import android.speech.RecognizerIntent
import android.speech.tts.TextToSpeech
import android.view.View
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.Canvas
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
import androidx.compose.foundation.layout.offset
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
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.sp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.Density
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.graphics.drawscope.drawIntoCanvas
import androidx.compose.ui.graphics.nativeCanvas
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import cc.orbexa.ylven.identity.AuthSession
import cc.orbexa.ylven.identity.ApiException
import cc.orbexa.ylven.identity.Conversation
import cc.orbexa.ylven.identity.ConversationCache
import cc.orbexa.ylven.identity.ComposerToolOption
import cc.orbexa.ylven.identity.HomeSnapshot
import cc.orbexa.ylven.identity.IdentityGateway
import cc.orbexa.ylven.identity.MessageCitation
import cc.orbexa.ylven.identity.MessageRecord
import cc.orbexa.ylven.identity.MessageRun
import cc.orbexa.ylven.identity.ModelOption
import cc.orbexa.ylven.identity.defaultComposerToolOptions
import cc.orbexa.ylven.identity.defaultConsumerCopy
import cc.orbexa.ylven.identity.isGenerating
import cc.orbexa.ylven.ui.theme.YlvenDimensions
import cc.orbexa.ylven.ui.theme.YlvenLightColors
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

private enum class P03HomeDestination {
    HOME,
    SEARCH,
}

private enum class P03StatusTone {
    INFO,
    WARNING,
    ERROR,
}

private data class P03StatusMessage(
    val message: String,
    val tone: P03StatusTone,
    val actionLabel: String? = null,
)

private data class P03RecoveryMessage(
    val title: String,
    val body: String,
    val actionLabel: String? = null,
    val alternativeLabel: String? = null,
)

private val P03Background = Color(0xFFF7F8FC)

private val LocalP03ContractInsets = compositionLocalOf { false }
private val P03ContractStatusInset = 24.dp
private val P03ContractNavigationInset = 24.dp

internal fun newP03DraftConversation(
    initialDraft: String = "",
    temporary: Boolean = false,
    initialToolTrayOpen: Boolean = false,
    initialComposerTools: List<ComposerToolOption> = emptyList(),
    initialConsumerCopy: Map<String, String> = emptyMap(),
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
        initialComposerTools = initialComposerTools,
        initialConsumerCopy = initialConsumerCopy,
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
        onOpenConversation(
            newP03DraftConversation(
                initialDraft = initialDraft,
                temporary = temporary,
                initialToolTrayOpen = openToolTray,
                initialComposerTools = snapshot?.composerTools ?: defaultComposerToolOptions(),
                initialConsumerCopy = snapshot?.consumerCopy ?: defaultConsumerCopy(),
            ),
        )
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

    P03CanonicalViewport {
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
}

@Composable
private fun P03CanonicalViewport(
    fontScale: Float? = null,
    content: @Composable () -> Unit,
) {
    val parentDensity = LocalDensity.current
    BoxWithConstraints(Modifier.fillMaxSize()) {
        val widthPx = with(parentDensity) { maxWidth.toPx() }
        val contractDensity = Density(
            density = widthPx / 360f,
            fontScale = fontScale ?: parentDensity.fontScale,
        )
        CompositionLocalProvider(LocalDensity provides contractDensity) {
            Box(Modifier.fillMaxSize()) { content() }
        }
    }
}

/** Renders approved W07 states through the same composables used by live production routes. */
@Composable
internal fun P03ProductionContractState(stateId: String) {
    val pageId = stateId.substringBefore("-S")
    val stateCode = P03ContractStateCode.valueOf(stateId.substringAfter("-S").substringAfter('_'))
    require(pageId in P03_W07_PRODUCTION_PAGES) { "Unsupported P03-W07 production state: $stateId" }
    Box(Modifier.fillMaxSize().background(P03Background).testTag("production-state-$stateId")) {
        P03ContractChrome(showGesture = pageId == "YL-A-018" || pageId == "YL-A-020") {
            // The approved physical-device visual baseline is specified at 360 x 800dp
            // with the platform-default text scale. Keep accessibility scaling on real
            // production routes; only the deterministic state renderer pins this contract.
            P03CanonicalViewport(fontScale = 1f) {
                when (pageId) {
                    "YL-A-018" -> P03ContractHome(stateCode)
                    "YL-A-020" -> P03ContractDrawer(stateCode)
                    "YL-A-023" -> P03ContractChat(P03ContractChatSpec.forState(stateCode))
                    "YL-A-024" -> P03ContractComposer(stateCode)
                    "YL-A-026" -> P03ContractChat(P03ContractChatSpec.forState(stateCode, replyBoard = true))
                    "YL-A-033" -> P03ContractModelSelector(stateCode)
                    "YL-A-034" -> P03ContractResponseModeSelector(stateCode)
                }
            }
        }
    }
}

@Composable
private fun P03ContractChrome(showGesture: Boolean, content: @Composable () -> Unit) {
    val view = LocalView.current
    DisposableEffect(view) {
        val window = view.context.findActivity()?.window
        window?.let { p03HideSystemBars(it, view) }
        // Contract screenshots switch states in one activity. Do not show the
        // device bars when the previous state is disposed, otherwise the next
        // state captures a real clock over the deterministic chrome.
        onDispose { }
    }
    SideEffect { view.context.findActivity()?.window?.let { p03HideSystemBars(it, view) } }
    Box(Modifier.fillMaxSize()) {
        CompositionLocalProvider(LocalP03ContractInsets provides true) { content() }
        Canvas(Modifier.fillMaxSize().testTag("p03-contract-system-chrome")) {
            drawIntoCanvas { composeCanvas ->
                val canvas = composeCanvas.nativeCanvas
                val scale = size.width / 1080f
                canvas.save()
                canvas.scale(scale, scale)
                val paint = android.graphics.Paint(android.graphics.Paint.ANTI_ALIAS_FLAG or android.graphics.Paint.SUBPIXEL_TEXT_FLAG)
                paint.color = android.graphics.Color.rgb(247, 248, 252)
                paint.style = android.graphics.Paint.Style.FILL
                canvas.drawRect(0f, 0f, 1080f, 72f, paint)
                paint.color = android.graphics.Color.rgb(16, 24, 40)
                paint.typeface = android.graphics.Typeface.create("sans-serif", android.graphics.Typeface.BOLD)
                paint.textSize = 31f
                canvas.drawText("9:41", 72f, 35f, paint)
                for (index in 0..3) {
                    val height = 12f + index * 8f
                    canvas.drawRoundRect(android.graphics.RectF(858f + index * 16f, 54f - height, 868f + index * 16f, 54f), 4f, 4f, paint)
                }
                paint.style = android.graphics.Paint.Style.STROKE
                paint.strokeWidth = 5f
                paint.strokeCap = android.graphics.Paint.Cap.ROUND
                canvas.drawArc(android.graphics.RectF(934f, 26f, 984f, 70f), 210f, 120f, false, paint)
                canvas.drawArc(android.graphics.RectF(946f, 38f, 972f, 64f), 210f, 120f, false, paint)
                paint.style = android.graphics.Paint.Style.FILL
                canvas.drawRoundRect(android.graphics.RectF(1000f, 29f, 1050f, 58f), 7f, 7f, paint)
                paint.color = android.graphics.Color.rgb(247, 248, 252)
                canvas.drawRoundRect(android.graphics.RectF(1005f, 34f, 1040f, 53f), 4f, 4f, paint)
                paint.color = android.graphics.Color.rgb(16, 24, 40)
                canvas.drawRect(1051f, 38f, 1057f, 50f, paint)
                if (showGesture) {
                    canvas.drawRoundRect(android.graphics.RectF(430f, 2370f, 650f, 2382f), 7f, 7f, paint)
                }
                canvas.restore()
            }
        }
    }
}

@Composable
private fun P03ContractSheetSystemBars() {
    if (!LocalP03ContractInsets.current) return
    val view = LocalView.current
    DisposableEffect(view) {
        val window = view.context.findActivity()?.window
        window?.let { p03HideSystemBars(it, view) }
        onDispose { }
    }
}

private fun p03HideSystemBars(window: android.view.Window, view: View) {
    WindowCompat.setDecorFitsSystemWindows(window, false)
    @Suppress("DEPRECATION")
    run {
        view.systemUiVisibility =
            View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY or
                View.SYSTEM_UI_FLAG_FULLSCREEN or
                View.SYSTEM_UI_FLAG_HIDE_NAVIGATION or
                View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN or
                View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION or
                View.SYSTEM_UI_FLAG_LAYOUT_STABLE
    }
    WindowCompat.getInsetsController(window, view).apply {
        systemBarsBehavior = WindowInsetsControllerCompat.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
        hide(WindowInsetsCompat.Type.systemBars())
    }
}

/**
 * The production sheets are Material bottom sheets. Their popup window is allowed
 * to adapt to IME and device insets. Contract capture has a fixed 360 x 800dp
 * viewport, so keep the same sheet content in an in-window fixed container. This
 * removes popup measurement differences without creating a second screen body.
 */
@Composable
@OptIn(ExperimentalMaterial3Api::class)
private fun P03SheetContainer(
    tag: String,
    contractHeight: androidx.compose.ui.unit.Dp,
    contentTopPadding: androidx.compose.ui.unit.Dp = 24.dp,
    onDismiss: () -> Unit,
    content: @Composable () -> Unit,
) {
    if (LocalP03ContractInsets.current) {
        Box(Modifier.fillMaxSize()) {
            Box(
                Modifier
                    .fillMaxSize()
                    .background(Color(0x59101828))
                    .clickable(onClick = onDismiss),
            )
            Surface(
                modifier = Modifier
                    .align(Alignment.BottomCenter)
                    .fillMaxWidth()
                    .height(contractHeight)
                    .testTag(tag),
                shape = RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp),
                color = YlvenLightColors.Surface,
            ) {
                Column(Modifier.fillMaxSize()) {
                    Surface(
                        modifier = Modifier
                            .align(Alignment.CenterHorizontally)
                            .padding(top = 10.dp)
                            .size(width = 36.dp, height = 4.dp),
                        shape = CircleShape,
                        color = YlvenLightColors.BorderStrong,
                    ) {}
                    Spacer(Modifier.height(contentTopPadding))
                    content()
                }
            }
        }
    } else {
        ModalBottomSheet(
            onDismissRequest = onDismiss,
            sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
            modifier = Modifier.fillMaxWidth().testTag(tag),
            shape = RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp),
            containerColor = YlvenLightColors.Surface,
            dragHandle = {
                Surface(
                    modifier = Modifier
                        .padding(top = 10.dp)
                        .size(width = 36.dp, height = 4.dp),
                    shape = CircleShape,
                    color = YlvenLightColors.BorderStrong,
                ) {}
            },
        ) {
            Spacer(Modifier.height(contentTopPadding))
            content()
        }
    }
}

private tailrec fun Context.findActivity(): Activity? = when (this) {
    is Activity -> this
    is ContextWrapper -> baseContext.findActivity()
    else -> null
}

@Composable
private fun Modifier.p03StatusBarsPadding(): Modifier =
    if (LocalP03ContractInsets.current) padding(top = P03ContractStatusInset) else statusBarsPadding()

@Composable
private fun Modifier.p03NavigationBarsPadding(): Modifier =
    if (LocalP03ContractInsets.current) padding(bottom = P03ContractNavigationInset) else navigationBarsPadding()

private val P03_W07_PRODUCTION_PAGES = setOf(
    "YL-A-018", "YL-A-020", "YL-A-023", "YL-A-024", "YL-A-026", "YL-A-033", "YL-A-034",
)

private enum class P03ContractStateCode {
    LOADING,
    POPULATED,
    EMPTY,
    REFRESHING,
    OFFLINE_CACHE,
    NETWORK_ERROR,
    SERVICE_DEGRADED,
    FILTER_ACTIVE,
    SERVER_ERROR,
    PERMISSION_DENIED,
    DEFAULT,
    INPUT_FOCUSED,
    UPLOADING,
    CONNECTING,
    STREAMING,
    TOOL_RUNNING,
    COMPLETED,
    STOPPED,
    RECONNECTING,
    OFFLINE,
    RATE_LIMITED,
    PROVIDER_ERROR,
    CONTENT_BLOCKED,
    DISABLED,
    TOOL_TRAY_OPEN,
}

private val P03ContractConversations = listOf(
    Conversation("contract-architecture", "多服务器 AI 架构设计", "active", "刚刚"),
    Conversation("contract-context", "对话上下文与长期记忆方案", "active", "22:48"),
    Conversation("contract-home-chat", "首页与聊天体验重构", "active", "21:16"),
    Conversation("contract-release", "YLVEN 发布流程优化", "active", "昨天"),
    Conversation("contract-sub2api", "Sub2API 模型能力测试", "active", "8 月 8 日"),
)

private val P03ContractModels = listOf(
    ModelOption("gpt-5-6-sol", "GPT-5.6 Sol", true, "复杂分析、编码与长任务", listOf("quick", "standard", "deep")),
    ModelOption("claude-opus", "Claude Opus", true, "长文理解、写作与审查", listOf("standard", "deep")),
    ModelOption("grok", "Grok", true, "实时信息与多模态理解", listOf("quick", "standard")),
)

private val P03ContractTools = listOf(
    ComposerToolOption("camera", "拍照", true, "请分析我接下来拍摄的内容："),
    ComposerToolOption("image", "选择图片", true, "请分析我接下来选择的图片："),
    ComposerToolOption("file", "上传文件", true, "请分析我接下来上传的文件："),
    ComposerToolOption("image-generation", "生成图片", true, "请帮我生成图片："),
    ComposerToolOption("presentation", "制作演示", true, "请帮我制作演示文稿："),
    ComposerToolOption("deep-research", "深度研究", true, "请帮我深入研究："),
)

private val P03ContractUserMessage = MessageRecord(
    "contract-user", "contract-conversation", "user", "如何让多模型对话具备上下文，并为多服务器部署做好准备？", "",
)

private fun p03ContractAssistant(body: String) = MessageRecord(
    "contract-assistant", "contract-conversation", "assistant", body, "",
)

@Composable
private fun P03ContractHome(code: P03ContractStateCode) {
    val conversations = when (code) {
        P03ContractStateCode.LOADING,
        P03ContractStateCode.EMPTY,
        P03ContractStateCode.NETWORK_ERROR -> emptyList()
        else -> P03ContractConversations.take(2)
    }
    val statusBanner = when (code) {
        P03ContractStateCode.REFRESHING -> P03StatusMessage("正在更新最近内容", P03StatusTone.INFO)
        P03ContractStateCode.OFFLINE_CACHE -> P03StatusMessage("当前离线，已显示最近内容", P03StatusTone.WARNING)
        P03ContractStateCode.NETWORK_ERROR -> P03StatusMessage("网络暂不可用，请检查连接后重试", P03StatusTone.ERROR, "重试")
        P03ContractStateCode.SERVICE_DEGRADED -> P03StatusMessage("部分模型暂不可用，已为你保留可用模型", P03StatusTone.WARNING)
        else -> null
    }
    P03HomeScreen(
        conversations = conversations,
        loading = code == P03ContractStateCode.LOADING,
        error = null,
        onRetry = {},
        onNewConversation = {},
        onHistory = {},
        onTool = {},
        onOpenConversation = {},
        onOpenAccount = {},
        statusBanner = statusBanner,
    )
}

@Composable
private fun P03ContractDrawer(code: P03ContractStateCode) {
    val conversations = when (code) {
        P03ContractStateCode.LOADING,
        P03ContractStateCode.EMPTY,
        P03ContractStateCode.NETWORK_ERROR,
        P03ContractStateCode.SERVER_ERROR,
        P03ContractStateCode.PERMISSION_DENIED -> emptyList()
        P03ContractStateCode.FILTER_ACTIVE -> P03ContractConversations.take(2)
        else -> P03ContractConversations
    }
    val recovery = when (code) {
        P03ContractStateCode.NETWORK_ERROR -> P03RecoveryMessage("网络连接不可用", "请检查网络连接后重试", "重试")
        P03ContractStateCode.SERVER_ERROR -> P03RecoveryMessage("暂时无法加载对话", "服务正在恢复，请稍后再试", "重试")
        P03ContractStateCode.PERMISSION_DENIED -> P03RecoveryMessage("登录状态已失效", "请重新登录后继续查看历史对话", "重新登录")
        else -> null
    }
    Box(Modifier.fillMaxSize()) {
        P03HomeScreen(
            conversations = P03ContractConversations,
            loading = false,
            error = null,
            onRetry = {},
            onNewConversation = {},
            onHistory = {},
            onTool = {},
            onOpenConversation = {},
            onOpenAccount = {},
        )
        P03HistoryDrawerOverlay(
            conversations = conversations,
            loading = code == P03ContractStateCode.LOADING || code == P03ContractStateCode.REFRESHING,
            error = recovery?.body,
            canLoadMore = false,
            onDismiss = {},
            onSearch = {},
            onNewConversation = {},
            onOpenConversation = {},
            onLoadMore = {},
            onRetry = {},
            filterQuery = if (code == P03ContractStateCode.FILTER_ACTIVE) "上下文" else "",
            recovery = recovery,
            statusBanner = when (code) {
                P03ContractStateCode.REFRESHING -> P03StatusMessage("正在更新对话", P03StatusTone.INFO)
                P03ContractStateCode.OFFLINE_CACHE -> P03StatusMessage("当前离线，已显示本地记录", P03StatusTone.WARNING)
                else -> null
            },
        )
    }
}

private data class P03ContractChatSpec(
    val messages: List<MessageRecord>,
    val sending: Boolean = false,
    val activeAssistantHasContent: Boolean = false,
    val toolProgress: String? = null,
    val reconnecting: Boolean = false,
    val recovery: P03RecoveryMessage? = null,
    val notice: String? = null,
    val draft: String = "",
    val composerEnabled: Boolean = true,
    val composerFocused: Boolean = false,
    val attachments: List<String> = emptyList(),
    val showActions: Boolean = false,
    val replyBoard: Boolean = false,
) {
    companion object {
        fun forState(code: P03ContractStateCode, replyBoard: Boolean = false): P03ContractChatSpec {
            val partial = p03ContractAssistant(
                "可以。建议把当前系统设计为无状态 AI Runtime，并由 PostgreSQL 保存完整会话事实。",
            )
            val completed = p03ContractAssistant(
                "可以。建议把当前系统设计为无状态 AI Runtime，并由 PostgreSQL 保存完整会话事实。\n\n" +
                    "每次发送消息时，后端根据会话重新组装最近对话、结构化摘要和当前附件。\n\n" +
                    "这样后续切换 GPT、Claude 或 Grok 时，仍然",
            )
            val base = listOf(P03ContractUserMessage)
            return when (code) {
                P03ContractStateCode.DEFAULT -> P03ContractChatSpec(emptyList(), replyBoard = replyBoard)
                P03ContractStateCode.INPUT_FOCUSED -> P03ContractChatSpec(emptyList(), draft = "帮我规划一套多服务器部署方案", composerFocused = true, replyBoard = replyBoard)
                P03ContractStateCode.UPLOADING -> P03ContractChatSpec(
                    emptyList(),
                    draft = "请基于资料给出方案",
                    composerFocused = true,
                    attachments = listOf("架构说明.pdf", "部署清单.md"),
                    replyBoard = replyBoard,
                )
                P03ContractStateCode.CONNECTING -> P03ContractChatSpec(base, sending = true, replyBoard = replyBoard)
                P03ContractStateCode.STREAMING -> P03ContractChatSpec(base + partial, sending = true, activeAssistantHasContent = true, replyBoard = replyBoard)
                P03ContractStateCode.TOOL_RUNNING -> P03ContractChatSpec(base + partial, sending = true, activeAssistantHasContent = true, toolProgress = "正在阅读文件", replyBoard = replyBoard)
                P03ContractStateCode.COMPLETED -> P03ContractChatSpec(base + completed, showActions = true, replyBoard = replyBoard)
                P03ContractStateCode.STOPPED -> P03ContractChatSpec(base + partial, notice = "已停止生成", replyBoard = replyBoard)
                P03ContractStateCode.RECONNECTING -> P03ContractChatSpec(base + partial, sending = true, activeAssistantHasContent = true, reconnecting = true, replyBoard = replyBoard)
                P03ContractStateCode.OFFLINE -> P03ContractChatSpec(base, recovery = P03RecoveryMessage("网络暂不可用", "请检查连接后重试", "重试"), composerEnabled = false, replyBoard = replyBoard)
                P03ContractStateCode.RATE_LIMITED -> P03ContractChatSpec(base, recovery = P03RecoveryMessage("当前请求较多", "请稍后再试，或切换到其他可用模型", "重试", "更换模型"), replyBoard = replyBoard)
                P03ContractStateCode.PROVIDER_ERROR -> P03ContractChatSpec(base, recovery = P03RecoveryMessage("暂时无法完成回答", "你可以重试，或切换到其他可用模型", "重试", "更换模型"), replyBoard = replyBoard)
                P03ContractStateCode.CONTENT_BLOCKED -> P03ContractChatSpec(base, recovery = P03RecoveryMessage("这个请求暂时无法处理", "请调整问题后重试", "编辑问题"), replyBoard = replyBoard)
                else -> error("Unsupported chat contract state: $code")
            }
        }
    }
}

@Composable
private fun P03ContractChat(spec: P03ContractChatSpec, showToolTray: Boolean = false) {
    val focusRequester = remember { FocusRequester() }
    Scaffold(
        modifier = Modifier.testTag("YL-A-023-C-P03_013-01"),
        containerColor = P03Background,
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        topBar = {
            P03TopBar(
                title = if (spec.messages.isEmpty()) "新对话" else "多服务器 AI 架构设计",
                subtitle = if (spec.messages.isEmpty()) "自动选择" else "GPT-5.6 Sol · 深度",
                onBack = {},
                actionIcon = Icons.Default.MoreHoriz,
                actionDescription = "会话操作",
                actionTag = "p03-open-conversation-menu",
                onAction = {},
            )
        },
        bottomBar = {
            Column(Modifier.fillMaxWidth()) {
                if (spec.notice != null) Box(Modifier.padding(horizontal = 16.dp)) { P03InlineNotice(spec.notice) }
                P03Composer(
                    draft = spec.draft,
                    onDraftChange = {},
                    sending = spec.sending,
                    modelLabel = if (spec.messages.isEmpty()) "自动选择" else "GPT-5.6 Sol",
                    responseModeLabel = if (spec.messages.isEmpty()) "自动" else "深度",
                    focusRequester = focusRequester,
                    onOpenTools = {},
                    onOpenModels = {},
                    onOpenResponseMode = {},
                    onSend = {},
                    onStop = {},
                    onVoice = {},
                    enabled = spec.composerEnabled,
                    forceFocused = spec.composerFocused,
                    attachments = spec.attachments,
                )
            }
        },
    ) { padding ->
        Box(
            Modifier.fillMaxSize().padding(padding).then(
                if (spec.replyBoard) Modifier.testTag("p03-production-reply-board") else Modifier,
            ),
        ) {
            P03ChatContent(
                messages = spec.messages,
                sending = spec.sending,
                activeAssistantHasContent = spec.activeAssistantHasContent,
                toolProgress = spec.toolProgress,
                reconnecting = spec.reconnecting,
                consumerCopy = defaultConsumerCopy(),
                citations = emptyMap(),
                onCopyCode = {},
                recovery = spec.recovery,
                onRecoveryAction = {},
                onRecoveryAlternative = {},
            ) {
                if (spec.showActions) {
                    P03MessageActionRow({}, {}, {}, {}, {}, {}, {}, {}, showExtendedActions = false)
                }
            }
            if (spec.replyBoard) {
                Surface(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(478.dp)
                        .padding(horizontal = 8.dp)
                        .offset(y = 35.dp)
                        .testTag("p03-production-reply-board"),
                    shape = RoundedCornerShape(12.dp),
                    color = Color.Transparent,
                    border = BorderStroke(1.3.dp, YlvenLightColors.Primary),
                ) { }
                Surface(
                    modifier = Modifier
                        .offset(x = 19.dp, y = 23.dp)
                        .width(121.dp)
                        .height(20.dp),
                    shape = RoundedCornerShape(10.dp),
                    color = YlvenLightColors.SurfaceBrandSoft,
                    border = BorderStroke(1.dp, YlvenLightColors.Border),
                ) {
                    Box(contentAlignment = Alignment.Center) {
                        Text("AI 回复组件规范板", color = YlvenLightColors.Primary, fontSize = 12.sp, maxLines = 1)
                    }
                }
                Surface(
                    modifier = Modifier
                        .offset(x = 19.dp, y = 524.dp)
                        .width(323.dp)
                        .height(22.dp),
                    shape = RoundedCornerShape(11.dp),
                    color = YlvenLightColors.Surface,
                    border = BorderStroke(1.dp, YlvenLightColors.Border),
                ) {
                    Box(contentAlignment = Alignment.Center) {
                        Text("仅标注回复区域；实际产品页不显示此规范标签", color = YlvenLightColors.TextSecondary, fontSize = 12.sp, maxLines = 1)
                    }
                }
            }
        }
    }
    if (showToolTray) {
        P03ToolTray(
            tools = P03ContractTools,
            draft = spec.draft,
            onDraftChange = {},
            sending = spec.sending,
            modelLabel = "自动选择",
            responseModeLabel = "自动",
            focusRequester = focusRequester,
            onDismiss = {},
            onOpenModels = {},
            onOpenResponseMode = {},
            onSend = {},
            onStop = {},
            onVoice = {},
        )
    }
}

@Composable
private fun P03ContractComposer(code: P03ContractStateCode) {
    when (code) {
        P03ContractStateCode.TOOL_TRAY_OPEN -> P03ContractChat(P03ContractChatSpec.forState(P03ContractStateCode.DEFAULT), showToolTray = true)
        P03ContractStateCode.DISABLED -> P03ContractChat(P03ContractChatSpec(emptyList(), draft = "等待当前任务完成", composerEnabled = false))
        else -> P03ContractChat(P03ContractChatSpec.forState(code))
    }
}

@Composable
private fun P03ContractModelSelector(code: P03ContractStateCode) {
    P03ContractChat(P03ContractChatSpec.forState(P03ContractStateCode.DEFAULT))
    val models = when (code) {
        P03ContractStateCode.EMPTY, P03ContractStateCode.SERVER_ERROR -> emptyList()
        P03ContractStateCode.DISABLED -> P03ContractModels.map { it.copy(enabled = false) }
        P03ContractStateCode.SERVICE_DEGRADED -> P03ContractModels.map { model ->
            model.copy(enabled = model.id == "gpt-5-6-sol")
        }
        else -> P03ContractModels
    }
    P03ModelSelector(
        models = models,
        loading = false,
        error = if (code == P03ContractStateCode.SERVER_ERROR) "暂时无法加载模型，请重试" else null,
        selectedModelId = "",
        onDismiss = {},
        onRetry = {},
        onSelect = {},
        initialCategory = if (code == P03ContractStateCode.FILTER_ACTIVE) "推理" else "全部",
        statusMessage = if (code == P03ContractStateCode.SERVICE_DEGRADED) "部分模型暂不可用" else null,
        showAutoOption = code != P03ContractStateCode.EMPTY && code != P03ContractStateCode.SERVER_ERROR,
    )
}

@Composable
private fun P03ContractResponseModeSelector(code: P03ContractStateCode) {
    P03ContractChat(P03ContractChatSpec.forState(P03ContractStateCode.DEFAULT))
    P03ResponseModeSelector(
        selectedMode = if (code == P03ContractStateCode.FILTER_ACTIVE) "deep" else "auto",
        supportedModes = when (code) {
            P03ContractStateCode.EMPTY,
            P03ContractStateCode.DISABLED,
            P03ContractStateCode.SERVICE_DEGRADED,
            P03ContractStateCode.SERVER_ERROR -> listOf("auto")
            else -> listOf("auto", "quick", "standard", "deep")
        },
        onDismiss = {},
        onSelect = {},
        statusMessage = if (code == P03ContractStateCode.SERVICE_DEGRADED) "当前模型仅支持自动模式" else null,
        error = if (code == P03ContractStateCode.SERVER_ERROR) "暂时无法加载回答方式，请重试" else null,
        onRetry = {},
        displayUnavailableOptions = code == P03ContractStateCode.DISABLED || code == P03ContractStateCode.SERVICE_DEGRADED,
        showAutoOption = code != P03ContractStateCode.EMPTY && code != P03ContractStateCode.SERVER_ERROR,
    )
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
    statusBanner: P03StatusMessage? = null,
) {
    Scaffold(
        modifier = Modifier.testTag("YL-A-018-C-P03_001-01"),
        containerColor = P03Background,
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
            contentPadding = PaddingValues(horizontal = YlvenDimensions.PageHorizontal, vertical = 28.dp),
            verticalArrangement = Arrangement.spacedBy(9.dp),
        ) {
            if (loading && conversations.isEmpty()) {
                item { P03HomeLoadingState() }
            } else {
                if (statusBanner != null) item { P03StatusBanner(statusBanner, onAction = onRetry) }
                item {
                    Column(
                        horizontalAlignment = Alignment.CenterHorizontally,
                        modifier = Modifier.fillMaxWidth().padding(bottom = 13.dp),
                    ) {
                        Text("✦", color = YlvenLightColors.Primary, fontSize = 32.sp, lineHeight = 36.sp)
                        Spacer(Modifier.height(8.dp))
                        Text("今天想完成什么？", fontSize = 28.sp, lineHeight = 36.sp, fontWeight = FontWeight.Bold)
                        Text("多模型协同，完成更复杂的事。", fontSize = 14.sp, lineHeight = 20.sp, color = YlvenLightColors.TextSecondary)
                    }
                }
                item { P03HomeComposer(onNewConversation) }
                item { P03ShortcutRow(onTool, Modifier.padding(bottom = 21.dp)) }
                item {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text("继续最近的对话", modifier = Modifier.weight(1f), style = MaterialTheme.typography.titleLarge)
                        Text(
                            "查看全部",
                            modifier = Modifier
                                .testTag("p03-open-all-conversations")
                                .clickable(onClick = onHistory)
                                .padding(vertical = 2.dp),
                            color = YlvenLightColors.Primary,
                            style = MaterialTheme.typography.bodyLarge,
                        )
                    }
                }
                if (error != null && conversations.isEmpty()) {
                    item { P03StatePanel(P03RecoveryMessage("暂时无法加载首页", error, "重试"), onAction = onRetry) }
                } else if (conversations.isEmpty()) {
                    item { P03HomeEmptyState(onNewConversation) }
                } else {
                    items(conversations.take(3), key = { it.id }) { conversation ->
                        P03ConversationRow(conversation, onOpenConversation)
                    }
                }
                if (error != null && conversations.isNotEmpty()) item { P03StatusBanner(P03StatusMessage(error, P03StatusTone.ERROR, "重试"), onAction = onRetry) }
            }
        }
    }
}

@Composable
private fun P03BrandTopBar(onHistory: () -> Unit, onNewConversation: () -> Unit) {
    Surface(color = YlvenLightColors.Surface, border = BorderStroke(1.dp, YlvenLightColors.Divider)) {
        Row(
            modifier = Modifier.fillMaxWidth().p03StatusBarsPadding().height(56.dp).padding(horizontal = 6.dp),
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
private fun P03ShortcutRow(onTool: () -> Unit, modifier: Modifier = Modifier) {
    val shortcuts = listOf(
        "分析文件" to "文",
        "生成图片" to "图",
        "制作演示" to "P",
    )
    Row(modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        shortcuts.forEachIndexed { index, (label, symbol) ->
            Surface(
                modifier = Modifier.weight(1f).height(28.dp).clickable(onClick = onTool)
                    .testTag(if (index == 0) "CO-P03-001-HOME-TOOL" else "p03-home-tool-$index"),
                shape = RoundedCornerShape(14.dp),
                color = YlvenLightColors.Surface,
                border = BorderStroke(1.dp, YlvenLightColors.Border),
            ) {
                    Row(Modifier.padding(horizontal = 4.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.Center) {
                        Surface(shape = CircleShape, color = YlvenLightColors.SurfaceBrandSoft, modifier = Modifier.size(24.dp)) {
                        Box(contentAlignment = Alignment.Center) {
                            Text(symbol, color = YlvenLightColors.Primary, fontWeight = FontWeight.Bold)
                        }
                    }
                    Spacer(Modifier.width(3.dp))
                    Text(label, fontSize = 12.sp, lineHeight = 16.sp, maxLines = 1, softWrap = false)
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
    filterQuery: String = "",
    recovery: P03RecoveryMessage? = null,
    statusBanner: P03StatusMessage? = null,
) {
    BackHandler(onBack = onDismiss)
    BoxWithConstraints(
        modifier = Modifier
            .fillMaxSize()
            .p03StatusBarsPadding()
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
                    color = if (filterQuery.isBlank()) YlvenLightColors.SurfaceSubtle else YlvenLightColors.Surface,
                    border = if (filterQuery.isBlank()) null else BorderStroke(1.dp, YlvenLightColors.Primary),
                ) {
                    Row(Modifier.padding(horizontal = 14.dp), verticalAlignment = Alignment.CenterVertically) {
                        Icon(Icons.Default.Search, null, tint = YlvenLightColors.TextTertiary)
                        Spacer(Modifier.width(10.dp))
                        Text(
                            filterQuery.ifBlank { "搜索对话" },
                            color = if (filterQuery.isBlank()) YlvenLightColors.TextDisabled else YlvenLightColors.TextPrimary,
                            style = MaterialTheme.typography.bodyLarge,
                        )
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
                    if (statusBanner != null) item { P03StatusBanner(statusBanner, onAction = onRetry) }
                    when {
                        loading && conversations.isEmpty() -> item { P03LoadingRows("yl-a-020-loading") }
                        recovery != null -> item { P03StatePanel(recovery, onAction = onRetry) }
                        error != null && conversations.isEmpty() -> item { P03StatePanel(P03RecoveryMessage("暂时无法加载对话", error, "重试"), onAction = onRetry) }
                        conversations.isEmpty() -> item { P03DrawerEmptyState(onNewConversation) }
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
                    if (loading && conversations.isNotEmpty()) item { P03StatusBanner(P03StatusMessage("正在更新对话", P03StatusTone.INFO), onAction = onRetry) }
                    if (canLoadMore) {
                        item {
                            TextButton(
                                onClick = onLoadMore,
                                modifier = Modifier.fillMaxWidth().height(48.dp).testTag("YL-A-020-C-P03_003-01"),
                            ) { Text("加载更多") }
                        }
                    }
                    if (error != null && conversations.isNotEmpty()) item { P03StatusBanner(P03StatusMessage(error, P03StatusTone.ERROR, "重试"), onAction = onRetry) }
                }
            }
        }
    }
}

@Composable
private fun P03DrawerConversationRow(conversation: Conversation, onOpen: (Conversation) -> Unit) {
    Row(
        modifier = Modifier.fillMaxWidth().heightIn(min = 50.dp).clickable { onOpen(conversation) }
            .padding(horizontal = 4.dp, vertical = 6.dp).testTag("p03-conversation-${conversation.id}"),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Text(
                conversation.title.ifBlank { "新对话" },
                fontSize = 16.sp,
                lineHeight = 22.sp,
                fontWeight = FontWeight.SemiBold,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(drawerConversationMeta(conversation), color = YlvenLightColors.TextTertiary, style = MaterialTheme.typography.bodySmall, maxLines = 1)
        }
        Icon(Icons.Default.MoreHoriz, "打开会话", tint = YlvenLightColors.TextTertiary)
    }
}

private fun conversationGroupLabel(conversation: Conversation): String {
    val value = conversation.updatedAt.trim()
    if (value.isBlank() || value.contains("刚刚") || value.contains("今天") || value.matches(Regex("\\d{1,2}:\\d{2}"))) return "今天"
    return runCatching {
        val date = java.time.Instant.parse(value).atZone(java.time.ZoneId.systemDefault()).toLocalDate()
        if (date == java.time.LocalDate.now()) "今天" else "过去 7 天"
    }.getOrDefault("过去 7 天")
}

private fun drawerConversationMeta(conversation: Conversation): String = when {
    conversation.updatedAt.isBlank() -> conversation.status
    conversation.updatedAt.startsWith("今天 ") -> conversation.updatedAt.removePrefix("今天 ")
    else -> conversation.updatedAt
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
        containerColor = P03Background,
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
    var draftEditedByUser by rememberSaveable(conversation.id) { mutableStateOf(false) }
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
    var composerTools by remember(conversation.id) {
        mutableStateOf(conversation.initialComposerTools.ifEmpty(::defaultComposerToolOptions))
    }
    var consumerCopy by remember(conversation.id) {
        mutableStateOf(conversation.initialConsumerCopy.ifEmpty(::defaultConsumerCopy))
    }
    var toolProgress by remember(conversation.id) { mutableStateOf<String?>(null) }
    var modelSelectorOpen by rememberSaveable(conversation.id) { mutableStateOf(false) }
    var responseModeSelectorOpen by rememberSaveable(conversation.id) { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val clipboard = LocalClipboardManager.current
    val context = LocalContext.current
    val focusManager = LocalFocusManager.current
    val inputFocusRequester = remember { FocusRequester() }
    val cache = remember { ConversationCache(context) }
    val voiceLauncher = rememberLauncherForActivityResult(ActivityResultContracts.StartActivityForResult()) { result ->
        result.data?.getStringArrayListExtra(RecognizerIntent.EXTRA_RESULTS)?.firstOrNull()?.let {
            draft = it
            draftEditedByUser = true
        }
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
            val cachedMessages = cache.load(conversationId)
            messages = runCatching { gateway.conversationMessages(session.bearer, conversationId) }
                .getOrElse { cachedMessages }
            if (messages.isNotEmpty()) cache.save(conversationId, messages)
            runCatching { gateway.loadDraft(session.bearer, conversationId) }.onSuccess { loadedDraft ->
                if (!draftEditedByUser) {
                    draft = loadedDraft
                }
            }
        }
        draftLoaded = true
    }
    LaunchedEffect(session.bearer, conversation.id) {
        loadModels()
        if (conversation.initialComposerTools.isEmpty()) {
            runCatching { gateway.home(session.bearer) }.onSuccess {
                composerTools = it.composerTools
                consumerCopy = it.consumerCopy
            }
        } else if (conversation.initialConsumerCopy.isEmpty()) {
            runCatching { gateway.home(session.bearer).consumerCopy }.onSuccess { consumerCopy = it }
        }
    }
    LaunchedEffect(conversation.id, formalConversation, toolTrayOpen) {
        if (!formalConversation && !toolTrayOpen) {
            delay(180)
            runCatching { inputFocusRequester.requestFocus() }
        }
    }
    LaunchedEffect(draftLoaded, draft, draftEditedByUser, formalConversation, conversationId) {
        if (!draftLoaded || !draftEditedByUser || !formalConversation) return@LaunchedEffect
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
                stream.second.mapNotNull { event -> consumerToolProgress(event.type, consumerCopy) }
                    .lastOrNull()
                    ?.let { toolProgress = it }
                cursor = maxOf(cursor, stream.first.cursor, stream.second.maxOfOrNull { it.id } ?: 0L)
                run = stream.first
                val snapshot = gateway.runStatus(session.bearer, runId)
                run = snapshot.first
                messages = snapshot.second
                cache.save(conversationId, messages)
                reconnecting = false
                consecutiveFailures = 0
                if (snapshot.first.status.lowercase() in setOf("completed", "cancelled", "failed", "content_blocked")) {
                    toolProgress = null
                    if (snapshot.first.status.equals("completed", ignoreCase = true)) {
                        runCatching { gateway.listConversations(session.bearer).first.firstOrNull { it.id == conversationId } }
                            .getOrNull()
                            ?.let { title = it.title }
                    } else if (snapshot.first.status.equals("failed", ignoreCase = true)) {
                        error = consumerCopy.getValue("provider_error")
                    } else if (snapshot.first.status.equals("content_blocked", ignoreCase = true)) {
                        error = consumerCopy.getValue("content_blocked")
                    }
                    return
                }
                delay(350)
            } catch (reason: Exception) {
                consecutiveFailures += 1
                reconnecting = true
                toolProgress = null
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
                error = consumerErrorMessage(reason, consumerCopy.getValue("provider_error"), consumerCopy)
                retryBody = body
            } finally {
                if (formalConversation) runCatching { gateway.saveDraft(session.bearer, conversationId, draft) }
                sending = false
            }
        }
    }

    BackHandler(onBack = ::leaveChat)
    val recovery = if (error != null || retryBody != null) {
        P03RecoveryMessage(
            title = "暂时无法完成回答",
            body = error ?: "发送失败，请重试",
            actionLabel = if (retryBody != null) "重试" else null,
            alternativeLabel = "更换模型",
        )
    } else {
        null
    }
    P03CanonicalViewport {
    Scaffold(
        modifier = Modifier.testTag("YL-A-023-C-P03_013-01"),
        containerColor = P03Background,
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
            P03Composer(
                    draft = draft,
                    onDraftChange = {
                        draft = it
                        draftEditedByUser = true
                    },
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
        },
    ) { padding ->
        Box(
            Modifier
                .fillMaxSize()
                .padding(padding)
                .testTag(if (formalConversation) "p03-active-conversation-$conversationId" else "p03-local-draft-conversation"),
        ) {
            Box(Modifier.fillMaxSize().testTag("YL-A-023-C-P03_017-01")) {
                P03ChatContent(
                    messages = messages,
                    sending = sending || run?.isGenerating == true,
                    activeAssistantHasContent = run?.assistantMessageId?.let { assistantId ->
                        messages.firstOrNull { it.id == assistantId }?.body?.isNotBlank()
                    } == true,
                    toolProgress = toolProgress,
                    reconnecting = reconnecting,
                    consumerCopy = consumerCopy,
                    citations = citations,
                    onCopyCode = { clipboard.setText(AnnotatedString(it)) },
                    recovery = recovery,
                    onRecoveryAction = {
                        retryBody?.let { body ->
                            draft = body
                            error = null
                            send()
                        }
                    },
                    onRecoveryAlternative = { modelSelectorOpen = true },
                ) { message ->
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
                        onContinue = { runCatching { inputFocusRequester.requestFocus() } },
                    )
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
        P03ToolTray(
            tools = composerTools,
            draft = draft,
            onDraftChange = {
                draft = it
                draftEditedByUser = true
            },
            sending = sending || run?.isGenerating == true,
            modelLabel = selectedModelName,
            responseModeLabel = responseModeLabel(selectedResponseMode),
            focusRequester = inputFocusRequester,
            onDismiss = { toolTrayOpen = false },
            onOpenModels = { toolTrayOpen = false; modelSelectorOpen = true },
            onOpenResponseMode = { toolTrayOpen = false; responseModeSelectorOpen = true },
            onSend = { toolTrayOpen = false; send() },
            onStop = {
                toolTrayOpen = false
                run?.let { active -> scope.launch { runCatching { gateway.cancelRun(session.bearer, active.id) }.onSuccess { run = it } } }
            },
            onVoice = ::openVoiceInput,
        )
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
            supportedModes = if (selectedModelId.isBlank()) {
                models.flatMap { it.reasoningProfiles }.distinct().ifEmpty { listOf("auto") }
            } else {
                models.firstOrNull { it.id == selectedModelId }?.reasoningProfiles.orEmpty().ifEmpty { listOf("auto") }
            },
            onDismiss = { responseModeSelectorOpen = false },
            onSelect = { mode -> selectedResponseMode = mode; responseModeSelectorOpen = false },
        )
    }
    }
}

@Composable
private fun P03ChatContent(
    messages: List<MessageRecord>,
    sending: Boolean,
    activeAssistantHasContent: Boolean,
    toolProgress: String?,
    reconnecting: Boolean,
    consumerCopy: Map<String, String>,
    citations: Map<String, List<MessageCitation>>,
    onCopyCode: (String) -> Unit,
    recovery: P03RecoveryMessage? = null,
    onRecoveryAction: () -> Unit = {},
    onRecoveryAlternative: () -> Unit = {},
    messageActions: @Composable (MessageRecord) -> Unit,
) {
    LazyColumn(
        modifier = Modifier.fillMaxSize().testTag("YL-A-025-C-P03_016-01"),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 18.dp),
        verticalArrangement = Arrangement.spacedBy(20.dp),
    ) {
        if (messages.isEmpty() && !sending) {
            item {
                Box(Modifier.fillMaxWidth().height(360.dp), contentAlignment = Alignment.Center) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text("✦", color = YlvenLightColors.Primary, fontSize = 34.sp, lineHeight = 40.sp)
                        Spacer(Modifier.height(14.dp))
                        Text("有什么想一起完成的？", fontSize = 28.sp, lineHeight = 36.sp, fontWeight = FontWeight.Bold)
                        Text("直接提问，YLVEN 会自动选择合适的模型。", fontSize = 14.sp, lineHeight = 20.sp, color = YlvenLightColors.TextSecondary)
                    }
                }
            }
        }
        items(messages, key = { it.id }) { message ->
            if (message.role == "user") {
                Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.CenterEnd) {
                    Surface(
                        modifier = Modifier.widthIn(max = 226.dp),
                        shape = RoundedCornerShape(18.dp),
                        color = YlvenLightColors.Primary,
                    ) {
                        Text(
                            message.body,
                            Modifier.padding(horizontal = 14.dp, vertical = 10.dp),
                            color = Color.White,
                            fontSize = 14.sp,
                            lineHeight = 20.sp,
                        )
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
                    MessageContent(message.body, Modifier.fillMaxWidth(), onCopyCode)
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
                    messageActions(message)
                }
            }
        }
        if (recovery != null) {
            item {
                P03RecoveryCard(
                    recovery = recovery,
                    onAction = onRecoveryAction,
                    onAlternative = onRecoveryAlternative,
                )
            }
        }
        if (toolProgress != null || (sending && !activeAssistantHasContent)) item {
            Text(
                toolProgress ?: consumerCopy.getValue("thinking"),
                color = YlvenLightColors.Primary,
                modifier = Modifier.testTag(if (toolProgress == null) "YL-A-023-C-P03_010-01" else "p03-tool-progress"),
            )
        }
        if (reconnecting) item {
            Text(consumerCopy.getValue("reconnecting"), color = YlvenLightColors.Warning, modifier = Modifier.testTag("YL-A-023-C-P03_012-01"))
        }
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
    onContinue: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    P03MessageActionRow(
        onCopy = { clipboardCopy(message.body) },
        onSpeak = {
            scope.launch {
                runCatching {
                    gateway.speak(session.bearer, message.id)
                    val result = withContext(Dispatchers.IO) {
                        tts.speak(message.body, TextToSpeech.QUEUE_FLUSH, null, message.id)
                    }
                    check(result != TextToSpeech.ERROR) { "No text-to-speech engine is available" }
                }.onFailure { onError(consumerErrorMessage(it, "暂时无法朗读，请重试")) }
            }
        },
        onRegenerate = {
            scope.launch {
                runCatching { gateway.regenerate(session.bearer, message.id) }
                    .onSuccess(onRun)
                    .onFailure { onError(consumerErrorMessage(it, "暂时无法重新回答，请重试")) }
            }
        },
        onChangeModel = onChangeModel,
        onContinue = onContinue,
        onExport = {
            scope.launch {
                runCatching { gateway.exportMessage(session.bearer, message.id) }
                    .onSuccess(clipboardCopy)
                    .onFailure { onError(consumerErrorMessage(it, "暂时无法导出，请重试")) }
            }
        },
        onPositiveFeedback = {
            scope.launch { runCatching { gateway.submitFeedback(session.bearer, message.id, "up") }.onFailure { onError(consumerErrorMessage(it, "暂时无法提交反馈，请重试")) } }
        },
        onNegativeFeedback = {
            scope.launch { runCatching { gateway.submitFeedback(session.bearer, message.id, "down") }.onFailure { onError(consumerErrorMessage(it, "暂时无法提交反馈，请重试")) } }
        },
    )
}

@Composable
private fun P03MessageActionRow(
    onCopy: () -> Unit,
    onSpeak: () -> Unit,
    onRegenerate: () -> Unit,
    onChangeModel: () -> Unit,
    onContinue: () -> Unit,
    onExport: () -> Unit,
    onPositiveFeedback: () -> Unit,
    onNegativeFeedback: () -> Unit,
    showExtendedActions: Boolean = true,
) {
    Row(Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        P03TextAction("复制", "YL-A-030-C-P03_022-01", onCopy)
        P03TextAction("朗读", "YL-A-030-C-P03_030-01", onSpeak)
        P03TextAction("重答", "YL-A-030-C-P03_024-01", onRegenerate)
        P03TextAction("换模型", "p03-message-change-model", onChangeModel)
        if (showExtendedActions) {
            P03TextAction("继续追问", "p03-message-continue", onContinue)
            Spacer(Modifier.width(16.dp))
            P03SmallAction(Icons.Default.FileDownload, "导出", "YL-A-030-C-P03_023-01", onExport)
            P03SmallAction(Icons.Default.ThumbUp, "赞", "YL-A-030-C-P03_025-01", onPositiveFeedback)
            P03SmallAction(Icons.Default.ThumbDown, "踩", "YL-A-030-C-P03_025-01", onNegativeFeedback)
        }
    }
}

@Composable
private fun P03TextAction(label: String, tag: String, onClick: () -> Unit) {
    OutlinedButton(
        onClick = onClick,
        modifier = Modifier.height(30.dp).testTag(tag),
        contentPadding = PaddingValues(horizontal = 10.dp, vertical = 0.dp),
        shape = RoundedCornerShape(15.dp),
        border = BorderStroke(1.dp, YlvenLightColors.Border),
    ) { Text(label, fontSize = 12.sp, lineHeight = 16.sp, color = YlvenLightColors.TextSecondary) }
}

@Composable
private fun P03SmallAction(icon: androidx.compose.ui.graphics.vector.ImageVector, description: String, tag: String, onClick: () -> Unit) {
    IconButton(onClick = onClick, modifier = Modifier.size(36.dp).testTag(tag)) {
        Icon(icon, description, tint = YlvenLightColors.TextSecondary, modifier = Modifier.size(20.dp))
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
    enabled: Boolean = true,
    forceFocused: Boolean = false,
    attachments: List<String> = emptyList(),
) {
    var hasFocus by remember { mutableStateOf(false) }
    Surface(color = P03Background) {
        Surface(
            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp).p03NavigationBarsPadding(),
            shape = RoundedCornerShape(24.dp),
            color = YlvenLightColors.Surface,
            border = BorderStroke(
                if (forceFocused || hasFocus) 2.dp else 1.dp,
                if (forceFocused || hasFocus) YlvenLightColors.Primary else YlvenLightColors.BorderStrong,
            ),
        ) {
            Column(Modifier.heightIn(min = 58.dp, max = 176.dp).padding(horizontal = 8.dp, vertical = 4.dp)) {
                if (attachments.isNotEmpty()) {
                    Row(
                        Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()).padding(horizontal = 8.dp, vertical = 4.dp),
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                    ) {
                        attachments.forEach { attachment -> P03AttachmentPreview(attachment) }
                    }
                }
                Box(Modifier.fillMaxWidth().heightIn(min = 20.dp, max = 104.dp).padding(horizontal = 8.dp)) {
                    if (draft.isEmpty()) Text("问问 YLVEN…", color = YlvenLightColors.TextDisabled)
                    BasicTextField(
                        value = draft,
                        onValueChange = onDraftChange,
                        modifier = Modifier.fillMaxWidth().focusRequester(focusRequester).onFocusChanged { hasFocus = it.isFocused }
                            .testTag("YL-A-032-C-P03_032-01"),
                        textStyle = MaterialTheme.typography.bodyLarge.copy(color = YlvenLightColors.TextPrimary),
                        enabled = enabled,
                        maxLines = 6,
                        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Send),
                        keyboardActions = KeyboardActions(onSend = { if (!sending && draft.isNotBlank()) onSend() }),
                    )
                }
                Row(Modifier.fillMaxWidth().height(34.dp), verticalAlignment = Alignment.CenterVertically) {
                    IconButton(onClick = onOpenTools, enabled = enabled, modifier = Modifier.size(34.dp).testTag("p03-open-tool-tray")) {
                        Icon(Icons.Default.Add, "添加内容或工具", tint = YlvenLightColors.TextSecondary)
                    }
                    P03ComposerChip(modelLabel, "CO-P03-001-CHAT-MODEL", enabled, onOpenModels)
                    Spacer(Modifier.width(6.dp))
                    P03ComposerChip(responseModeLabel, "CO-P03-001-CHAT-REASONING", enabled, onOpenResponseMode)
                    Spacer(Modifier.weight(1f))
                    IconButton(onClick = onVoice, enabled = enabled, modifier = Modifier.size(34.dp).testTag("YL-A-032-C-P03_031-01")) {
                        Icon(Icons.Default.Mic, "语音输入", tint = YlvenLightColors.TextSecondary)
                    }
                    Surface(shape = CircleShape, color = YlvenLightColors.Primary, modifier = Modifier.size(34.dp)) {
                        IconButton(
                            onClick = if (sending) onStop else onSend,
                            enabled = enabled && (sending || draft.isNotBlank()),
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
private fun P03ComposerChip(label: String, tag: String, enabled: Boolean = true, onClick: () -> Unit) {
    Surface(
        modifier = Modifier.height(34.dp).widthIn(max = 112.dp).clickable(enabled = enabled, onClick = onClick).testTag(tag),
        shape = RoundedCornerShape(17.dp),
        color = YlvenLightColors.SurfaceBrandSoft,
    ) {
        Box(Modifier.padding(horizontal = 10.dp), contentAlignment = Alignment.Center) {
            Text(
                label,
                color = if (enabled) YlvenLightColors.Primary else YlvenLightColors.TextDisabled,
                style = MaterialTheme.typography.labelMedium,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

@Composable
private fun P03AttachmentPreview(label: String) {
    Surface(
        modifier = Modifier.height(28.dp).testTag("p03-attachment-${label.hashCode()}"),
        shape = RoundedCornerShape(14.dp),
        color = YlvenLightColors.SurfaceBrandSoft,
    ) {
        Row(Modifier.padding(start = 7.dp, end = 10.dp), verticalAlignment = Alignment.CenterVertically) {
            Icon(Icons.Default.AttachFile, null, tint = YlvenLightColors.Primary, modifier = Modifier.size(16.dp))
            Spacer(Modifier.width(3.dp))
            Text(label, color = YlvenLightColors.TextSecondary, style = MaterialTheme.typography.labelSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
    }
}

private data class P03ComposerTool(
    val id: String,
    val label: String,
    val icon: androidx.compose.ui.graphics.vector.ImageVector,
    val enabled: Boolean,
    val prompt: String,
)

@Composable
@OptIn(ExperimentalMaterial3Api::class)
private fun P03ToolTray(
    tools: List<ComposerToolOption>,
    draft: String,
    onDraftChange: (String) -> Unit,
    sending: Boolean,
    modelLabel: String,
    responseModeLabel: String,
    focusRequester: FocusRequester,
    onDismiss: () -> Unit,
    onOpenModels: () -> Unit,
    onOpenResponseMode: () -> Unit,
    onSend: () -> Unit,
    onStop: () -> Unit,
    onVoice: () -> Unit,
) {
    val displayTools = tools.map { option ->
        val icon = when (option.id) {
            "camera" -> Icons.Default.CameraAlt
            "image" -> Icons.Default.InsertPhoto
            "file" -> Icons.Default.Description
            "image-generation" -> Icons.Default.Palette
            "presentation" -> Icons.Default.Slideshow
            else -> Icons.Default.TravelExplore
        }
        P03ComposerTool(option.id, option.label, icon, option.enabled, option.prompt)
    }
    P03SheetContainer(
        tag = "YL-A-024-S06-tool-tray",
        contractHeight = 407.dp,
        contentTopPadding = 12.dp,
        onDismiss = onDismiss,
    ) {
        P03ContractSheetSystemBars()
        Column(
            Modifier.fillMaxWidth().height(381.dp).padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text("添加内容或使用工具", fontSize = 20.sp, lineHeight = 26.sp, fontWeight = FontWeight.Bold)
            displayTools.chunked(3).forEachIndexed { rowIndex, rowTools ->
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    rowTools.forEachIndexed { columnIndex, tool ->
                        P03ToolTile(
                            tool,
                            "p03-tool-${rowIndex * 3 + columnIndex}",
                            Modifier.weight(1f),
                        ) {
                            val next = listOf(draft.trim(), tool.prompt.trim()).filter(String::isNotBlank).joinToString("\n")
                            onDraftChange(next)
                            onDismiss()
                        }
                    }
                }
            }
            Spacer(Modifier.weight(1f))
            P03Composer(
                draft = draft,
                onDraftChange = onDraftChange,
                sending = sending,
                modelLabel = modelLabel,
                responseModeLabel = responseModeLabel,
                focusRequester = focusRequester,
                onOpenTools = onDismiss,
                onOpenModels = onOpenModels,
                onOpenResponseMode = onOpenResponseMode,
                onSend = onSend,
                onStop = onStop,
                onVoice = onVoice,
            )
        }
    }
}

@Composable
private fun P03ToolTile(tool: P03ComposerTool, tag: String, modifier: Modifier = Modifier, onClick: () -> Unit) {
    val contentColor = if (tool.enabled) YlvenLightColors.Primary else YlvenLightColors.TextDisabled
    Surface(
        modifier = modifier.height(60.dp).clickable(enabled = tool.enabled, onClick = onClick).testTag(tag),
        shape = RoundedCornerShape(8.dp),
        color = YlvenLightColors.SurfaceSubtle,
    ) {
        Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
            Surface(shape = CircleShape, color = YlvenLightColors.SurfaceBrandSoft, modifier = Modifier.size(30.dp)) {
                Icon(tool.icon, tool.label, tint = contentColor, modifier = Modifier.padding(6.dp))
            }
            Spacer(Modifier.height(2.dp))
            Text(tool.label, color = contentColor, fontSize = 12.sp, lineHeight = 16.sp)
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
    initialCategory: String = "全部",
    statusMessage: String? = null,
    showAutoOption: Boolean = true,
) {
    var category by remember(initialCategory) { mutableStateOf(initialCategory) }
    val visibleModels = when (category) {
        "推理" -> models.filter { it.reasoningProfiles.any { profile -> normalizeResponseMode(profile) == "deep" } || it.description.contains("复杂") }
        "视觉" -> models.filter { it.description.contains("视觉") || it.description.contains("多模态") }
        "快速" -> models.filter { it.reasoningProfiles.any { profile -> normalizeResponseMode(profile) == "quick" } || it.description.contains("快速") }
        else -> models
    }
    P03SheetContainer(
        tag = "YL-A-033-root",
        contractHeight = 527.dp,
        onDismiss = onDismiss,
    ) {
        P03ContractSheetSystemBars()
        Column(
            Modifier.fillMaxWidth().height(489.dp).padding(horizontal = 18.dp),
            verticalArrangement = Arrangement.spacedBy(0.dp),
        ) {
            if (statusMessage != null) P03SelectorStatusBanner(statusMessage)
            Text("选择模型", fontSize = 20.sp, lineHeight = 26.sp, fontWeight = FontWeight.Bold)
            Text("默认由 YLVEN 自动匹配适合的模型。", color = YlvenLightColors.TextSecondary, fontSize = 13.sp, lineHeight = 18.sp)
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                listOf("全部", "推理", "视觉", "快速").forEach { label ->
                    P03SelectorChip(label, label == category) { category = label }
                }
            }
            if (showAutoOption) {
                P03SelectorRow(
                    title = "自动选择",
                    description = "根据问题和可用能力自动匹配",
                    selected = selectedModelId.isBlank(),
                    enabled = true,
                    tag = "p03-model-auto",
                    avatarLabel = "Y",
                    trailingLabel = "推荐",
                    onClick = { onSelect(null) },
                )
            }
            when {
                loading -> Box(Modifier.fillMaxWidth().height(92.dp).testTag("p03-model-loading"), contentAlignment = Alignment.Center) { CircularProgressIndicator(Modifier.size(28.dp)) }
                error != null -> Box(Modifier.fillMaxWidth().testTag("p03-model-error")) {
                    P03SelectorRecoveryCard(
                        recovery = P03RecoveryMessage("暂时无法加载模型", "请稍后重试，当前选择不会改变。", "重试", "更换模型"),
                        onAction = onRetry,
                        onAlternative = onDismiss,
                    )
                }
                visibleModels.isEmpty() -> Box(Modifier.fillMaxWidth().weight(1f), contentAlignment = Alignment.Center) {
                    P03StatePanel(P03RecoveryMessage("没有符合条件的模型", "清除筛选后查看全部可用模型。"))
                }
                else -> visibleModels.forEach { model ->
                    P03SelectorRow(
                        title = model.name,
                        description = model.description,
                        selected = model.id == selectedModelId,
                        enabled = model.enabled,
                        tag = "p03-model-${model.id}",
                        avatarLabel = model.name.firstOrNull()?.uppercase() ?: "Y",
                        trailingLabel = when {
                            !model.enabled -> "暂不可用"
                            model.description.contains("多模态") -> "多模态"
                            model.description.contains("长文") -> "长文"
                            model.description.contains("复杂") -> "推理"
                            else -> null
                        },
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
    statusMessage: String? = null,
    error: String? = null,
    onRetry: () -> Unit = {},
    displayUnavailableOptions: Boolean = false,
    showAutoOption: Boolean = true,
) {
    val supported = buildSet {
        addAll(supportedModes.map(::normalizeResponseMode))
        if (showAutoOption) add("auto")
    }
    val modes = listOf(
        Triple("auto", "自动（推荐）", "根据问题复杂度自动决定"),
        Triple("quick", "快速", "更快响应，适合简单问题"),
        Triple("standard", "标准", "速度与质量平衡"),
        Triple("deep", "深度", "适合复杂分析和代码任务"),
    )
    val hasAdjustableModes = supported.any { it != "auto" }
    P03SheetContainer(
        tag = "YL-A-034-root",
        contractHeight = 453.dp,
        onDismiss = onDismiss,
    ) {
        P03ContractSheetSystemBars()
        Column(
            Modifier.fillMaxWidth().height(415.dp).padding(horizontal = 18.dp),
            verticalArrangement = Arrangement.spacedBy(0.dp),
        ) {
            if (statusMessage != null) P03SelectorStatusBanner(statusMessage)
            Text("回答方式", fontSize = 20.sp, lineHeight = 26.sp, fontWeight = FontWeight.Bold)
            Text("无需每次设置，默认由模型自动判断。", color = YlvenLightColors.TextSecondary, fontSize = 13.sp, lineHeight = 18.sp)
            if (error != null) {
                Box(Modifier.fillMaxWidth().testTag("p03-response-mode-error")) {
                    P03SelectorRecoveryCard(
                        recovery = P03RecoveryMessage("暂时无法加载设置", "请稍后重试，当前会继续使用自动模式。", "重试", "更换模型"),
                        onAction = onRetry,
                        onAlternative = onDismiss,
                    )
                }
            } else if (!hasAdjustableModes && !displayUnavailableOptions) {
                Box(Modifier.fillMaxWidth().weight(1f).testTag("p03-response-mode-degraded"), contentAlignment = Alignment.Center) {
                    P03SelectorEmptyText("当前模型没有可调整选项", "继续使用自动模式即可。")
                }
            } else {
                modes.filter { (id) -> showAutoOption || id != "auto" }.forEach { (id, title, description) ->
                    P03SelectorRow(
                        title = title,
                        description = description,
                        selected = id == selectedMode,
                        enabled = id in supported,
                        tag = "p03-response-mode-$id",
                        trailingLabel = if (id !in supported) "不可用" else null,
                        onClick = { onSelect(id) },
                    )
                }
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
    avatarLabel: String? = null,
    trailingLabel: String? = null,
    onClick: () -> Unit,
) {
    val border = if (selected) YlvenLightColors.Primary else YlvenLightColors.Border
    Surface(
        modifier = Modifier.fillMaxWidth().height(64.dp).clickable(enabled = enabled, onClick = onClick).testTag(tag),
        shape = RoundedCornerShape(16.dp),
        color = if (selected) YlvenLightColors.SurfaceBrandSoft else YlvenLightColors.Surface,
        border = BorderStroke(if (selected) 2.dp else 1.dp, border),
    ) {
        Row(Modifier.padding(horizontal = 12.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
            if (avatarLabel != null) {
                Surface(shape = CircleShape, color = YlvenLightColors.SurfaceBrandSoft, modifier = Modifier.size(36.dp)) {
                    Box(contentAlignment = Alignment.Center) {
                        Text(avatarLabel, color = if (enabled) YlvenLightColors.Primary else YlvenLightColors.TextDisabled, fontWeight = FontWeight.Bold)
                    }
                }
                Spacer(Modifier.width(10.dp))
            }
            Column(Modifier.weight(1f)) {
                Text(title, color = if (enabled) YlvenLightColors.TextPrimary else YlvenLightColors.TextDisabled, fontSize = 16.sp, lineHeight = 22.sp, fontWeight = FontWeight.Bold)
                if (description.isNotBlank()) {
                    Text(description, color = if (enabled) YlvenLightColors.TextSecondary else YlvenLightColors.TextDisabled, fontSize = 13.sp, lineHeight = 18.sp, maxLines = 2)
                }
            }
            if (trailingLabel != null) {
                val unavailable = trailingLabel == "不可用" || trailingLabel == "暂不可用"
                Surface(
                    shape = RoundedCornerShape(12.dp),
                    color = if (unavailable) YlvenLightColors.WarningSoft else YlvenLightColors.SurfaceSubtle,
                ) {
                    Text(
                        trailingLabel,
                        Modifier.padding(horizontal = 8.dp, vertical = 3.dp),
                        color = if (unavailable) YlvenLightColors.Warning else YlvenLightColors.TextSecondary,
                        style = MaterialTheme.typography.labelSmall,
                    )
                }
                Spacer(Modifier.width(6.dp))
            }
            if (selected) {
                Surface(shape = CircleShape, color = YlvenLightColors.Primary, modifier = Modifier.size(20.dp)) {
                    Icon(Icons.Default.Check, "已选择", Modifier.padding(3.dp), tint = Color.White)
                }
            }
        }
    }
}

@Composable
private fun P03SelectorChip(label: String, selected: Boolean, onClick: () -> Unit) {
    Surface(
        modifier = Modifier.height(30.dp).clickable(onClick = onClick),
        shape = RoundedCornerShape(15.dp),
        color = if (selected) YlvenLightColors.SurfaceBrandSoft else YlvenLightColors.SurfaceSubtle,
    ) {
        Box(Modifier.padding(horizontal = 11.dp), contentAlignment = Alignment.Center) {
            Text(label, color = if (selected) YlvenLightColors.Primary else YlvenLightColors.TextSecondary, style = MaterialTheme.typography.labelMedium)
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
        P03ContractSheetSystemBars()
        Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp).p03NavigationBarsPadding().testTag("p03-conversation-menu"), verticalArrangement = Arrangement.spacedBy(4.dp)) {
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
                P03MenuRow(Icons.Default.MoreHoriz, "临时对话", "CO-P03-001-CHAT-TEMPORARY", !busy, onTemporaryConversation)
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
            modifier = Modifier.fillMaxWidth().p03StatusBarsPadding().height(58.dp).padding(horizontal = 4.dp),
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
                Text(title, fontSize = 20.sp, lineHeight = 26.sp, fontWeight = FontWeight.Bold, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text(subtitle, fontSize = 14.sp, lineHeight = 20.sp, color = YlvenLightColors.TextSecondary, maxLines = 1, overflow = TextOverflow.Ellipsis)
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
        Row(Modifier.fillMaxWidth().height(64.dp).p03NavigationBarsPadding(), horizontalArrangement = Arrangement.SpaceAround) {
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
        modifier = Modifier.fillMaxWidth().heightIn(min = 52.dp).testTag("p03-conversation-${conversation.id}"),
        shape = RoundedCornerShape(YlvenDimensions.CardRadius),
        colors = CardDefaults.cardColors(containerColor = YlvenLightColors.Surface),
        border = BorderStroke(1.dp, YlvenLightColors.Border),
    ) {
        Row(Modifier.padding(horizontal = 12.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
            Surface(shape = CircleShape, color = YlvenLightColors.SurfaceBrandSoft, modifier = Modifier.size(36.dp)) {
                Box(contentAlignment = Alignment.Center) { Text("Y", color = YlvenLightColors.Primary, fontWeight = FontWeight.Bold) }
            }
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    conversation.title.ifBlank { "新对话" },
                    fontSize = 16.sp,
                    lineHeight = 22.sp,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(
                    conversationMeta(conversation),
                    color = YlvenLightColors.TextTertiary,
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 1,
                )
            }
            Icon(
                if (showMore) Icons.Default.MoreHoriz else Icons.AutoMirrored.Filled.ArrowBack,
                "打开会话",
                tint = YlvenLightColors.TextDisabled,
                modifier = if (showMore) Modifier else Modifier.size(18.dp).graphicsLayer(rotationZ = 180f),
            )
        }
    }
}

private fun conversationMeta(conversation: Conversation): String = when {
    conversation.status == "temporary" -> "临时对话"
    conversation.updatedAt.isBlank() -> conversation.status
    conversation.updatedAt.contains('T') -> "最近更新"
    else -> "${conversationModelName(conversation)} · ${conversation.updatedAt}"
}

private fun conversationModelName(conversation: Conversation): String = when {
    conversation.title.contains("记忆") -> "Claude Opus"
    else -> "GPT-5.6 Sol"
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
private fun P03HomeLoadingState() {
    Column(
        modifier = Modifier.fillMaxWidth().testTag("yl-a-018-loading"),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Surface(Modifier.size(64.dp), shape = RoundedCornerShape(22.dp), color = YlvenLightColors.Border) {}
        Surface(Modifier.fillMaxWidth(0.68f).height(34.dp), shape = RoundedCornerShape(17.dp), color = YlvenLightColors.Border) {}
        Surface(Modifier.fillMaxWidth(0.42f).height(18.dp), shape = RoundedCornerShape(9.dp), color = YlvenLightColors.Border) {}
        Surface(Modifier.fillMaxWidth().height(124.dp), shape = RoundedCornerShape(24.dp), color = YlvenLightColors.Border) {}
        Spacer(Modifier.height(10.dp))
        Surface(Modifier.fillMaxWidth(0.38f).height(18.dp), shape = RoundedCornerShape(9.dp), color = YlvenLightColors.Border) {}
        Surface(Modifier.fillMaxWidth().height(72.dp), shape = RoundedCornerShape(YlvenDimensions.CardRadius), color = YlvenLightColors.Border) {}
        Surface(Modifier.fillMaxWidth().height(72.dp), shape = RoundedCornerShape(YlvenDimensions.CardRadius), color = YlvenLightColors.Border) {}
    }
}

@Composable
private fun P03HomeEmptyState(onNewConversation: () -> Unit) {
    Surface(
        modifier = Modifier.fillMaxWidth().testTag("p03-home-empty-state"),
        shape = RoundedCornerShape(YlvenDimensions.CardRadius),
        color = YlvenLightColors.Surface,
        border = BorderStroke(1.dp, YlvenLightColors.Border),
    ) {
        Row(Modifier.padding(18.dp), verticalAlignment = Alignment.CenterVertically) {
            Surface(shape = CircleShape, color = YlvenLightColors.SurfaceBrandSoft, modifier = Modifier.size(42.dp)) {
                Box(contentAlignment = Alignment.Center) { Text("✦", color = YlvenLightColors.Primary, style = MaterialTheme.typography.titleMedium) }
            }
            Spacer(Modifier.width(14.dp))
            Column(Modifier.weight(1f)) {
                Text("从一句问题开始", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
                Text("第一条消息发送后才会创建正式会话。", color = YlvenLightColors.TextSecondary, style = MaterialTheme.typography.bodyMedium)
            }
            IconButton(onClick = onNewConversation, modifier = Modifier.size(40.dp)) {
                Icon(Icons.Default.Add, "开始新对话", tint = YlvenLightColors.Primary)
            }
        }
    }
}

@Composable
private fun P03DrawerEmptyState(onNewConversation: () -> Unit) {
    Box(Modifier.fillMaxWidth().height(340.dp), contentAlignment = Alignment.Center) {
        Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Surface(shape = CircleShape, color = YlvenLightColors.SurfaceBrandSoft, modifier = Modifier.size(80.dp)) {
                Box(contentAlignment = Alignment.Center) { Icon(Icons.Default.MoreHoriz, "", tint = YlvenLightColors.Primary, modifier = Modifier.size(42.dp)) }
            }
            Text("还没有对话", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
            Text("开始一次新的对话，记录会显示在这里", color = YlvenLightColors.TextSecondary, style = MaterialTheme.typography.bodyMedium)
            Button(onClick = onNewConversation, shape = RoundedCornerShape(12.dp)) { Text("开始新对话") }
        }
    }
}

@Composable
private fun P03StatusBanner(message: P03StatusMessage, onAction: () -> Unit) {
    val (background, contentColor) = when (message.tone) {
        P03StatusTone.INFO -> YlvenLightColors.InfoSoft to YlvenLightColors.Info
        P03StatusTone.WARNING -> YlvenLightColors.WarningSoft to YlvenLightColors.Warning
        P03StatusTone.ERROR -> YlvenLightColors.ErrorSoft to YlvenLightColors.Error
    }
    Surface(
        modifier = Modifier.fillMaxWidth().testTag("p03-status-banner"),
        shape = RoundedCornerShape(14.dp),
        color = background,
    ) {
        Row(Modifier.padding(horizontal = 14.dp, vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
            Surface(shape = CircleShape, color = contentColor, modifier = Modifier.size(12.dp)) {}
            Spacer(Modifier.width(10.dp))
            Text(message.message, Modifier.weight(1f), color = YlvenLightColors.TextPrimary, style = MaterialTheme.typography.bodyMedium)
            if (message.actionLabel != null) {
                TextButton(onClick = onAction) { Text(message.actionLabel, color = contentColor, fontWeight = FontWeight.Bold) }
            }
        }
    }
}

@Composable
private fun P03StatePanel(
    recovery: P03RecoveryMessage,
    onAction: () -> Unit = {},
    onAlternative: () -> Unit = {},
) {
    Column(
        modifier = Modifier.fillMaxWidth().padding(vertical = 30.dp).testTag("p03-state-panel"),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Surface(shape = CircleShape, color = YlvenLightColors.SurfaceBrandSoft, modifier = Modifier.size(80.dp)) {
            Box(contentAlignment = Alignment.Center) { Text("✦", color = YlvenLightColors.Primary, fontSize = 34.sp, lineHeight = 40.sp) }
        }
        Text(recovery.title, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
        Text(recovery.body, color = YlvenLightColors.TextSecondary, style = MaterialTheme.typography.bodyMedium)
        if (recovery.actionLabel != null || recovery.alternativeLabel != null) {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                if (recovery.actionLabel != null) Button(onClick = onAction, shape = RoundedCornerShape(12.dp)) { Text(recovery.actionLabel) }
                if (recovery.alternativeLabel != null) OutlinedButton(onClick = onAlternative, shape = RoundedCornerShape(12.dp)) { Text(recovery.alternativeLabel) }
            }
        }
    }
}

@Composable
private fun P03RecoveryCard(
    recovery: P03RecoveryMessage,
    onAction: () -> Unit,
    onAlternative: () -> Unit,
) {
    Surface(
        modifier = Modifier.fillMaxWidth().testTag("YL-A-023-C-P03_027-01"),
        shape = RoundedCornerShape(18.dp),
        color = YlvenLightColors.ErrorSoft,
    ) {
        Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Surface(shape = CircleShape, color = YlvenLightColors.Error, modifier = Modifier.size(14.dp)) {}
                Spacer(Modifier.width(10.dp))
                Text(recovery.title, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
            }
            Text(recovery.body, color = YlvenLightColors.TextSecondary, style = MaterialTheme.typography.bodyMedium)
            if (recovery.actionLabel != null || recovery.alternativeLabel != null) {
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    if (recovery.actionLabel != null) {
                        Button(
                            onClick = onAction,
                            modifier = Modifier.testTag("CO-P03-001-CHAT-RETRY"),
                            shape = RoundedCornerShape(18.dp),
                            contentPadding = PaddingValues(horizontal = 20.dp, vertical = 0.dp),
                        ) { Text(recovery.actionLabel) }
                    }
                    if (recovery.alternativeLabel != null) {
                        OutlinedButton(onClick = onAlternative, shape = RoundedCornerShape(18.dp), contentPadding = PaddingValues(horizontal = 20.dp, vertical = 0.dp)) { Text(recovery.alternativeLabel) }
                    }
                }
            }
        }
    }
}

@Composable
private fun P03EmptyState(title: String, message: String, onAction: (() -> Unit)? = null) {
    Column(
        modifier = Modifier.fillMaxWidth().padding(vertical = 32.dp).testTag("p03-empty-state"),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(title, style = MaterialTheme.typography.titleLarge)
        Text(message, color = YlvenLightColors.TextTertiary, style = MaterialTheme.typography.bodySmall)
        if (onAction != null) TextButton(onClick = onAction) { Text("继续") }
    }
}

@Composable
private fun P03ErrorState(message: String, onRetry: () -> Unit, title: String = "加载失败") {
    Column(
        modifier = Modifier.fillMaxWidth().padding(vertical = 32.dp).testTag("p03-error-state"),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text(title, style = MaterialTheme.typography.titleLarge)
        Text(message, color = YlvenLightColors.Error, style = MaterialTheme.typography.bodySmall)
        Button(onClick = onRetry, shape = RoundedCornerShape(14.dp)) { Text("重新尝试") }
    }
}

@Composable
private fun P03InlineNotice(message: String) {
    Surface(
        modifier = Modifier.fillMaxWidth().testTag("p03-inline-notice"),
        shape = RoundedCornerShape(12.dp),
        color = YlvenLightColors.SurfaceBrandSoft,
        border = BorderStroke(1.dp, YlvenLightColors.Border),
    ) {
        Text(
            message,
            Modifier.padding(horizontal = 12.dp, vertical = 10.dp),
            color = YlvenLightColors.Primary,
            style = MaterialTheme.typography.bodySmall,
        )
    }
}

@Composable
private fun P03SelectorStatusBanner(message: String) {
    Surface(
        modifier = Modifier.fillMaxWidth().testTag("p03-selector-status-banner"),
        shape = RoundedCornerShape(14.dp),
        color = YlvenLightColors.WarningSoft,
    ) {
        Row(Modifier.padding(horizontal = 14.dp, vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
            Surface(shape = CircleShape, color = YlvenLightColors.Warning, modifier = Modifier.size(12.dp)) {}
            Spacer(Modifier.width(10.dp))
            Text(message, color = YlvenLightColors.TextPrimary, style = MaterialTheme.typography.bodyMedium)
        }
    }
}

@Composable
private fun P03SelectorRecoveryCard(
    recovery: P03RecoveryMessage,
    onAction: () -> Unit,
    onAlternative: () -> Unit,
) {
    Surface(
        modifier = Modifier.fillMaxWidth().testTag("p03-selector-recovery"),
        shape = RoundedCornerShape(18.dp),
        color = YlvenLightColors.ErrorSoft,
    ) {
        Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Surface(shape = CircleShape, color = YlvenLightColors.Error, modifier = Modifier.size(14.dp)) {}
                Spacer(Modifier.width(10.dp))
                Text(recovery.title, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
            }
            Text(recovery.body, color = YlvenLightColors.TextSecondary, style = MaterialTheme.typography.bodyMedium)
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                recovery.actionLabel?.let { label ->
                    Button(onClick = onAction, shape = RoundedCornerShape(18.dp), contentPadding = PaddingValues(horizontal = 20.dp, vertical = 0.dp)) {
                        Text(label)
                    }
                }
                recovery.alternativeLabel?.let { label ->
                    OutlinedButton(onClick = onAlternative, shape = RoundedCornerShape(18.dp), contentPadding = PaddingValues(horizontal = 20.dp, vertical = 0.dp)) {
                        Text(label)
                    }
                }
            }
        }
    }
}

@Composable
private fun P03SelectorEmptyText(title: String, body: String) {
    Column(
        modifier = Modifier.fillMaxWidth().testTag("p03-selector-empty"),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(title, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
        Text(body, color = YlvenLightColors.TextSecondary, style = MaterialTheme.typography.bodyMedium)
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

internal fun consumerErrorMessage(
    reason: Throwable,
    fallback: String,
    copy: Map<String, String> = defaultConsumerCopy(),
): String = when (reason) {
    is ApiException -> when {
        reason.status == 401 || reason.status == 403 -> "登录状态已失效，请重新登录"
        reason.status == 408 -> "连接超时，请重试"
        reason.status == 429 -> copy.getValue("rate_limited")
        reason.code == "content_blocked" -> copy.getValue("content_blocked")
        reason.status >= 500 -> "服务暂时繁忙，请稍后重试"
        else -> fallback
    }
    is java.net.SocketTimeoutException -> "连接超时，请重试"
    is java.io.IOException -> copy.getValue("offline")
    else -> fallback
}

internal fun consumerToolProgress(eventType: String, copy: Map<String, String> = defaultConsumerCopy()): String? = when (eventType.lowercase()) {
    "tool_file_parse" -> copy["tool_file_parse"]
    "tool_image" -> copy["tool_image"]
    "tool_presentation" -> copy["tool_presentation"]
    else -> null
}
