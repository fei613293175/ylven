package cc.orbexa.ylven

import android.accessibilityservice.AccessibilityService
import android.graphics.Bitmap
import android.os.SystemClock
import androidx.activity.compose.setContent
import androidx.compose.ui.test.assertTextContains
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.hasTestTag
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.compose.ui.test.onAllNodesWithTag
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollTo
import androidx.compose.ui.test.performScrollToIndex
import androidx.compose.ui.test.performScrollToNode
import androidx.compose.ui.test.performTextClearance
import androidx.compose.ui.test.performTextInput
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.test.swipeLeft
import androidx.test.espresso.Espresso
import androidx.test.platform.app.InstrumentationRegistry
import androidx.core.view.ViewCompat
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import cc.orbexa.ylven.identity.AuthSession
import cc.orbexa.ylven.identity.Conversation
import cc.orbexa.ylven.identity.DeviceSession
import cc.orbexa.ylven.identity.FirstMessageResult
import cc.orbexa.ylven.identity.HomeSnapshot
import cc.orbexa.ylven.identity.IdentityGateway
import cc.orbexa.ylven.identity.MessageCitation
import cc.orbexa.ylven.identity.MessageRecord
import cc.orbexa.ylven.identity.MessageRun
import cc.orbexa.ylven.identity.ModelOption
import cc.orbexa.ylven.identity.OtpChallenge
import cc.orbexa.ylven.identity.RunEvent
import cc.orbexa.ylven.ui.YlvenApp
import cc.orbexa.ylven.ui.theme.YlvenTheme
import java.io.IOException
import kotlinx.coroutines.CompletableDeferred
import java.util.concurrent.CopyOnWriteArraySet
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

/**
 * Runs real Compose clicks, text input, back navigation and scrolling on the
 * connected physical device. The deterministic gateway isolates UI behavior;
 * it is not staging/API acceptance evidence.
 */
class P03RealDeviceFlowTest {
    @get:Rule
    val composeRule = createEmptyComposeRule()
    private lateinit var activity: MainActivity

    @Test
    fun conversationJourneyUsesEveryP03ControlOnDevice() {
        val gateway = FlowGateway()
        val session = AuthSession("陈平@example.invalid", "session", "access", "refresh", "physical-device")
        activity = launchP03TargetActivity()
        composeRule.mainClock.autoAdvance = true
        composeRule.runOnUiThread {
            activity.setContent {
                YlvenTheme(darkTheme = false) {
                    YlvenApp(gateway = gateway, initialSession = session)
                }
            }
        }

        waitForTag("YL-A-018-C-P03_001-01")
        captureProductionPage("YL-A-018-PRODUCTION", "YL-A-018-C-P03_001-01")
        composeRule.onNodeWithTag("CO-P03-001-HOME-COMPOSER").performClick()
        waitForTag("p03-local-draft-conversation")
        assertTrue("Opening a blank chat must not persist a conversation", "create" !in gateway.calls)
        hideKeyboardIfVisible()
        Espresso.pressBack()
        waitForTag("YL-A-018-C-P03_001-01")

        composeRule.onNodeWithTag("p03-home-list").performScrollToNode(hasTestTag("CO-P03-001-HOME-TOOL"))
        composeRule.onNodeWithTag("CO-P03-001-HOME-TOOL").performClick()
        waitForTag("YL-A-024-S06-tool-tray")
        captureProductionPage("YL-A-024-PRODUCTION", "YL-A-024-S06-tool-tray")
        composeRule.onNodeWithTag("p03-tool-0").assertIsNotEnabled()
        composeRule.onNodeWithTag("p03-tool-5").assertIsNotEnabled()
        Espresso.pressBack()
        waitForTag("p03-local-draft-conversation")
        hideKeyboardIfVisible()
        Espresso.pressBack()
        waitForTag("YL-A-018-C-P03_001-01")

        composeRule.onNodeWithTag("CO-P03-001-HOME-SEND").performClick()
        waitForTag("p03-local-draft-conversation")
        composeRule.onNodeWithTag("p03-open-conversation-menu").performClick()
        waitForTag("YL-A-031-C-P03_028-01")
        composeRule.onNodeWithTag("YL-A-031-C-P03_028-01").performClick()
        waitForTag("p03-local-draft-conversation")
        assertTemporaryConversationScope()
        assertTrue("Opening a temporary draft must not persist a conversation", "temporary" !in gateway.calls)
        hideKeyboardIfVisible()
        Espresso.pressBack()
        waitForTag("YL-A-018-C-P03_001-01")

        composeRule.onNodeWithTag("p03-open-conversation-drawer").performClick()
        waitForTag("p03-conversation-drawer")
        waitForTag("YL-A-020-root")
        captureProductionPage("YL-A-020-PRODUCTION", "YL-A-020-root")
        composeRule.onNodeWithTag("p03-drawer-scrim").performClick()
        waitForDrawerClosed()

        composeRule.onNodeWithTag("p03-open-conversation-drawer").performClick()
        waitForTag("YL-A-020-root")
        Espresso.pressBack()
        waitForDrawerClosed()

        composeRule.onNodeWithTag("p03-open-conversation-drawer").performClick()
        waitForTag("p03-conversation-drawer")
        composeRule.onNodeWithTag("p03-conversation-drawer").performTouchInput { swipeLeft() }
        waitForDrawerClosed()

        composeRule.onNodeWithTag("p03-open-conversation-drawer").performClick()
        waitForTag("p03-conversation-drawer")
        waitForTag("YL-A-020-C-P03_003-01")
        composeRule.onNodeWithTag("YL-A-020-C-P03_003-01").performScrollTo().performClick()
        composeRule.onNodeWithTag("p03-open-search").performClick()
        waitForTag("YL-A-021-root")
        composeRule.onNodeWithTag("YL-A-021-C-P03_004-01").performTextInput("Alpha")
        waitForTag("p03-conversation-alpha")
        captureProductionPage("YL-A-021-PRODUCTION", "YL-A-021-root")
        composeRule.onNodeWithTag("p03-conversation-alpha").performScrollTo().performClick()
        waitForTag("YL-A-023-C-P03_013-01")
        waitForTag("YL-A-032-C-P03_032-01")
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").assertTextContains("saved draft")
        composeRule.onNodeWithTag("CO-P03-001-CHAT-MODEL").performClick()
        waitForTag("YL-A-033-root")
        waitForModalSheetToSettle()
        captureProductionPage("YL-A-033-PRODUCTION", "YL-A-033-root")
        composeRule.onNodeWithTag("p03-model-gpt-5.6-sol").performClick()
        composeRule.onNodeWithTag("CO-P03-001-CHAT-REASONING").performClick()
        waitForTag("YL-A-034-root")
        waitForModalSheetToSettle()
        captureProductionPage("YL-A-034-PRODUCTION", "YL-A-034-root")
        composeRule.onNodeWithTag("p03-response-mode-quick").assertIsNotEnabled()
        composeRule.onNodeWithTag("p03-response-mode-auto").performClick()
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").performTextClearance()
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").performTextInput("physical flow message")
        composeRule.onNodeWithTag("YL-A-023-C-P03_009-01").performClick()
        waitForCondition { gateway.sentModel == "gpt-5.6-sol" }
        scrollChatTo("YL-A-023-C-P03_010-01")
        gateway.releaseSend()
        waitForCondition { "events-attempt" in gateway.calls }
        gateway.releaseFirstEventFailure()
        waitForCall(gateway, "events-retry")
        waitForTag("YL-A-023-C-P03_012-01")
        scrollChatTo("YL-A-023-C-P03_012-01")
        waitForTag("YL-A-024-C-P03_011-01")
        composeRule.onNodeWithTag("YL-A-024-C-P03_011-01").performClick()
        waitForCondition { "cancel" in gateway.calls }
        gateway.releaseStream()
        waitForCondition { "events" in gateway.calls }

        waitForTag("YL-A-025-C-P03_016-01")
        waitForTag("YL-A-023-C-P03_017-01")
        waitForTag("YL-A-026-C-P03_026-01")
        scrollChatTo("YL-A-026-C-P03_018-01")
        captureProductionPage("YL-A-026-PRODUCTION", "YL-A-026-C-P03_018-01")
        scrollChatTo("YL-A-027-C-P03_019-01")
        scrollChatTo("YL-A-028-C-P03_020-01")
        scrollChatTo("YL-A-029-C-P03_021-01")
        captureProductionPage(
            "YL-A-023-PRODUCTION",
            "YL-A-023-C-P03_013-01",
            scrollAnchorTag = "YL-A-025-C-P03_016-01",
        )
        scrollChatTo("YL-A-030-C-P03_022-01")
        composeRule.onNodeWithTag("YL-A-030-C-P03_022-01").performClick()
        scrollChatTo("YL-A-030-C-P03_023-01")
        composeRule.onNodeWithTag("YL-A-030-C-P03_023-01").performClick()
        waitForCall(gateway, "export-message")
        composeRule.waitForIdle()
        scrollChatTo("YL-A-030-C-P03_025-01")
        val feedbackNodes = composeRule.onAllNodesWithTag("YL-A-030-C-P03_025-01")
        check(feedbackNodes.fetchSemanticsNodes().size >= 2) { "Both feedback controls are not present" }
        feedbackNodes[0].performClick()
        waitForCondition { "feedback-up" in gateway.calls || "feedback-down" in gateway.calls }
        composeRule.waitForIdle()
        composeRule.onAllNodesWithTag("YL-A-030-C-P03_025-01")[1].performClick()
        waitForCall(gateway, "feedback-up")
        waitForCall(gateway, "feedback-down")
        composeRule.waitForIdle()
        scrollChatTo("YL-A-030-C-P03_024-01")
        composeRule.onNodeWithTag("YL-A-030-C-P03_024-01").performClick()
        waitForCall(gateway, "regenerate")
        composeRule.waitForIdle()
        scrollChatTo("YL-A-030-C-P03_030-01")
        composeRule.onNodeWithTag("YL-A-030-C-P03_030-01").performClick()
        waitForCall(gateway, "speak")
        scrollChatTo("p03-copy-code")
        composeRule.onNodeWithTag("p03-copy-code").performClick()
        gateway.failNextSend = true
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").performTextInput("retry me")
        composeRule.onNodeWithTag("YL-A-023-C-P03_009-01").performClick()
        waitForTag("YL-A-023-C-P03_027-01")
        waitForTag("CO-P03-001-CHAT-RETRY")
        composeRule.onNodeWithText("网络不可用，请检查连接后重试").assertExists()
        composeRule.onNodeWithTag("CO-P03-001-CHAT-RETRY").performClick()
        waitForCondition { gateway.sendAttempts >= 3 }
        waitForCondition {
            composeRule.onAllNodesWithTag("YL-A-023-C-P03_009-01").fetchSemanticsNodes().isNotEmpty() &&
                composeRule.onAllNodesWithTag("YL-A-023-C-P03_027-01").fetchSemanticsNodes().isEmpty() &&
                composeRule.onAllNodesWithText("retry me").fetchSemanticsNodes().size == 1
        }
        hideKeyboardIfVisible()
        Espresso.pressBack()
        waitForTag("YL-A-018-C-P03_001-01")

        composeRule.onNodeWithTag("p03-home-list")
            .performScrollToNode(hasTestTag("p03-conversation-alpha"))
        composeRule.onNodeWithTag("p03-conversation-alpha").performClick()
        waitForTag("YL-A-032-C-P03_032-01")
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").assertTextContains("retry me")

        composeRule.onNodeWithTag("YL-A-032-C-P03_031-01").performClick()
        returnFromVoiceInputIfLaunched()
        if (composeRule.onAllNodesWithTag("YL-A-023-C-P03_013-01").fetchSemanticsNodes().isEmpty()) {
            waitForTag("YL-A-018-C-P03_001-01")
            composeRule.onNodeWithTag("p03-home-list")
                .performScrollToNode(hasTestTag("p03-conversation-alpha"))
            composeRule.onNodeWithTag("p03-conversation-alpha").performClick()
        }
        waitForTag("YL-A-023-C-P03_013-01")

        composeRule.onNodeWithTag("p03-open-conversation-menu").performClick()
        waitForTag("YL-A-022-root")
        waitForTag("p03-conversation-menu")
        waitForModalSheetToSettle()
        captureProductionPage("YL-A-022-PRODUCTION", "YL-A-022-root")
        composeRule.onNodeWithTag("YL-A-022-C-P03_029-01").performClick()
        waitForCondition { "export-conversation" in gateway.calls }

        composeRule.onNodeWithTag("p03-open-conversation-menu").performClick()
        composeRule.onNodeWithTag("p03-open-rename-current").performClick()
        waitForTag("YL-A-022-C-P03_005-01")
        composeRule.onNodeWithTag("YL-A-022-C-P03_005-01").performTextClearance()
        composeRule.onNodeWithTag("YL-A-022-C-P03_005-01").performTextInput("Alpha renamed")
        Espresso.closeSoftKeyboard()
        composeRule.onNodeWithTag("CO-P03-001-TITLE-RENAME").performClick()
        waitForCondition { "rename" in gateway.calls }

        composeRule.onNodeWithTag("p03-open-conversation-menu").performClick()
        composeRule.onNodeWithTag("YL-A-022-C-P03_006-01").performClick()
        waitForCondition { "archive" in gateway.calls }
        waitForTag("YL-A-018-C-P03_001-01")

        composeRule.onNodeWithTag("p03-home-list")
            .performScrollToNode(hasTestTag("p03-conversation-beta"))
        composeRule.onNodeWithTag("p03-conversation-beta").performClick()
        waitForTag("p03-active-conversation-beta")
        composeRule.onNodeWithTag("p03-open-conversation-menu").performClick()
        composeRule.onNodeWithTag("YL-A-022-C-P03_007-01").performClick()
        waitForTag("p03-confirm-delete")
        composeRule.onNodeWithTag("p03-confirm-delete").performClick()
        waitForCondition { "delete" in gateway.calls }
        waitForTag("YL-A-018-C-P03_001-01")

        val requiredCalls = setOf(
            "home", "models", "list-page-1", "list-page-2", "search",
            "rename", "archive", "delete", "send", "events-retry", "cancel",
            "export-message", "export-conversation", "feedback-up", "feedback-down",
            "regenerate", "speak", "load-draft", "save-draft",
        )
        assertTrue("Missing gateway calls: ${requiredCalls - gateway.calls}", gateway.calls.containsAll(requiredCalls))
    }

    private fun waitForTag(tag: String) = waitForCondition {
        composeRule.onAllNodesWithTag(tag).fetchSemanticsNodes().isNotEmpty()
    }

    private fun waitForTagAbsent(tag: String) = waitForCondition {
        composeRule.onAllNodesWithTag(tag).fetchSemanticsNodes().isEmpty()
    }

    private fun waitForDrawerClosed() {
        waitForTagAbsent("YL-A-020-root")
        // Older physical devices can remove drawer semantics before the close
        // transition releases its gesture state and restores the opener.
        composeRule.waitForIdle()
        SystemClock.sleep(350)
        waitForTag("p03-open-conversation-drawer")
    }

    private fun assertTemporaryConversationScope() {
        listOf("所属项目", "默认模型", "会话指令").forEach { label ->
            assertTrue(
                "Temporary conversation must not expose unsupported $label control",
                composeRule.onAllNodesWithText(label).fetchSemanticsNodes().isEmpty(),
            )
        }
    }

    private fun waitForCall(gateway: FlowGateway, call: String) {
        try {
            waitForCondition { call in gateway.calls }
        } catch (failure: Throwable) {
            throw AssertionError("Missing gateway call $call; calls=${gateway.calls}", failure)
        }
    }

    private fun scrollChatTo(tag: String) {
        composeRule.onNodeWithTag("YL-A-025-C-P03_016-01")
            .performScrollToNode(hasTestTag(tag))
        waitForTag(tag)
    }

    private fun waitForCondition(predicate: () -> Boolean) {
        composeRule.waitUntil(timeoutMillis = 8_000, condition = predicate)
    }

    private fun captureProductionPage(
        name: String,
        expectedRootTag: String,
        scrollAnchorTag: String? = null,
    ) {
        val instrumentation = InstrumentationRegistry.getInstrumentation()
        val context = instrumentation.targetContext
        waitForTag(expectedRootTag)
        check(composeRule.onAllNodesWithTag(expectedRootTag).fetchSemanticsNodes().isNotEmpty()) {
            "Expected page root $expectedRootTag is absent before capturing $name"
        }
        hideKeyboardIfVisible()
        composeRule.waitForIdle()
        instrumentation.waitForIdleSync()
        if (scrollAnchorTag != null) {
            composeRule.onNodeWithTag(scrollAnchorTag).performScrollToIndex(0)
            composeRule.waitForIdle()
            instrumentation.waitForIdleSync()
        }
        val bitmap = requireNotNull(instrumentation.uiAutomation.takeScreenshot()) {
            "Could not capture the physical device display for $name"
        }
        check(bitmap.width > 0 && bitmap.height > 0) {
            "P03 production screenshot is empty for $name"
        }
        try {
            P03ScreenshotStorage.writePng(
                context = context,
                bitmap = bitmap,
                legacyDirectory = "production-screenshots",
                scopedDownloadDirectory = "ylven-p03-production",
                fileName = "$name.png",
                subject = "P03 production screenshot",
            )
        } finally {
            bitmap.recycle()
        }
    }

    private fun waitForModalSheetToSettle() {
        // The semantics root is attached before Material's enter transition ends.
        // Capture only the settled production sheet, never an intermediate frame.
        Thread.sleep(450)
        composeRule.waitForIdle()
        InstrumentationRegistry.getInstrumentation().waitForIdleSync()
    }

    private fun hideKeyboardIfVisible() {
        var wasVisible = false
        composeRule.runOnUiThread {
            val decorView = activity.window.decorView
            wasVisible = ViewCompat.getRootWindowInsets(decorView)
                ?.isVisible(WindowInsetsCompat.Type.ime()) == true
            activity.currentFocus?.clearFocus()
            if (wasVisible) {
                WindowCompat.getInsetsController(activity.window, decorView)
                    .hide(WindowInsetsCompat.Type.ime())
            }
        }
        if (wasVisible) {
            composeRule.waitUntil(timeoutMillis = 5_000) { !isKeyboardVisible() }
        }
    }

    private fun isKeyboardVisible(): Boolean {
        var visible = false
        composeRule.runOnUiThread {
            visible = ViewCompat.getRootWindowInsets(activity.window.decorView)
                ?.isVisible(WindowInsetsCompat.Type.ime()) == true
        }
        return visible
    }

    private fun returnFromVoiceInputIfLaunched() {
        val launchDeadline = SystemClock.uptimeMillis() + 3_000
        while (hasAppWindowFocus() && SystemClock.uptimeMillis() < launchDeadline) {
            Thread.sleep(50)
        }
        if (hasAppWindowFocus()) return

        val instrumentation = InstrumentationRegistry.getInstrumentation()
        check(instrumentation.uiAutomation.performGlobalAction(AccessibilityService.GLOBAL_ACTION_BACK)) {
            "Could not return from the physical device voice-input activity"
        }
        composeRule.waitUntil(timeoutMillis = 5_000) { hasAppWindowFocus() }
        instrumentation.waitForIdleSync()
    }

    private fun hasAppWindowFocus(): Boolean {
        var focused = false
        composeRule.runOnUiThread { focused = activity.hasWindowFocus() }
        return focused
    }
}

private class FlowGateway : IdentityGateway {
    val calls = CopyOnWriteArraySet<String>()
    var failNextSend = false
    @Volatile var sentModel = ""
    @Volatile var sendAttempts = 0
    @Volatile var lastSentBody = "physical flow message"
    private var failFirstEvent = true
    private var currentRunStatus = "streaming"
    private var draft = "saved draft"
    private val sendGate = CompletableDeferred<Unit>()
    private val firstEventFailureGate = CompletableDeferred<Unit>()
    private val streamGate = CompletableDeferred<Unit>()
    private val alpha = Conversation("alpha", "安卓 AI 工具架构设计", "active", "刚刚")
    private val beta = Conversation("beta", "比较 Claude 与 GPT 的推理差异", "active", "昨天")

    override suspend fun startRegistration(email: String) = OtpChallenge("register", email, "123456")
    override suspend fun finishRegistration(challenge: OtpChallenge, code: String, password: String) = Unit
    override suspend fun startLogin(email: String) = OtpChallenge("login", email, "123456")
    override suspend fun finishLogin(challenge: OtpChallenge, code: String) = session()
    override suspend fun refresh(session: AuthSession) = session
    override suspend fun devices(bearer: String): List<DeviceSession> = emptyList()
    override suspend fun revokeDevice(bearer: String, sessionId: String) = Unit
    override suspend fun logout(bearer: String, allDevices: Boolean) = Unit

    override suspend fun home(bearer: String): HomeSnapshot {
        calls += "home"
        return HomeSnapshot(listOf(alpha, beta), modelOptions())
    }

    override suspend fun models(bearer: String): List<ModelOption> {
        calls += "models"
        return modelOptions()
    }

    override suspend fun listConversations(
        bearer: String,
        cursor: String?,
        includeArchived: Boolean,
    ): Pair<List<Conversation>, String?> {
        return if (cursor == null) {
            calls += "list-page-1"
            Pair(listOf(alpha, beta), "next")
        } else {
            calls += "list-page-2"
            Pair(listOf(Conversation("gamma", "YLVEN 品牌主视觉", "active", "7 月 30 日")), null)
        }
    }

    override suspend fun searchConversations(bearer: String, query: String): List<Conversation> {
        calls += "search"
        return listOf(alpha)
    }

    override suspend fun createConversation(bearer: String, title: String): Conversation {
        calls += "create"
        return Conversation("created", "Created", "active", "2026-08-08T00:00:00Z")
    }

    override suspend fun createTemporaryConversation(bearer: String, title: String): Conversation {
        calls += "temporary"
        return Conversation("temporary", "Temporary", "temporary", "2026-08-08T00:00:00Z")
    }

    override suspend fun renameConversation(bearer: String, id: String, title: String): Conversation {
        calls += "rename"
        return alpha.copy(title = title)
    }

    override suspend fun archiveConversation(bearer: String, id: String): Conversation {
        calls += "archive"
        return alpha.copy(status = "archived", archivedAt = "2026-08-08T00:01:00Z")
    }

    override suspend fun deleteConversation(bearer: String, id: String): Conversation {
        calls += "delete"
        return alpha.copy(status = "deleted")
    }

    override suspend fun sendMessage(
        bearer: String,
        conversationId: String,
        body: String,
        model: String,
    ): MessageRun {
        sendAttempts += 1
        calls += "send"
        sentModel = model
        lastSentBody = body
        if (failNextSend) {
            failNextSend = false
            throw IOException("controlled network failure")
        }
        sendGate.await()
        currentRunStatus = if (sendAttempts >= 3) "completed" else "streaming"
        return run("run-1")
    }

    override suspend fun startConversationFromFirstMessage(
        bearer: String,
        draftSessionId: String,
        body: String,
        model: String,
        idempotencyKey: String,
        temporary: Boolean,
    ): FirstMessageResult {
        calls += "first-message"
        val created = Conversation("created", body.take(18), if (temporary) "temporary" else "active", "2026-08-08T00:00:00Z")
        val createdRun = sendMessage(bearer, created.id, body, model)
        return FirstMessageResult(created, createdRun)
    }

    override suspend fun runEvents(bearer: String, runId: String, after: Long): Pair<MessageRun, List<RunEvent>> {
        if (failFirstEvent) {
            failFirstEvent = false
            calls += "events-attempt"
            firstEventFailureGate.await()
            calls += "events-retry"
            throw IOException("controlled SSE disconnect")
        }
        streamGate.await()
        calls += "events"
        return Pair(run(runId), listOf(RunEvent(after + 1, "delta", "answer")))
    }

    fun releaseSend() {
        sendGate.complete(Unit)
    }

    fun releaseFirstEventFailure() {
        firstEventFailureGate.complete(Unit)
    }

    fun releaseStream() {
        streamGate.complete(Unit)
    }

    override suspend fun runStatus(bearer: String, runId: String): Pair<MessageRun, List<MessageRecord>> =
        Pair(run(runId), messages())

    override suspend fun cancelRun(bearer: String, runId: String): MessageRun {
        calls += "cancel"
        currentRunStatus = "cancelled"
        return run(runId)
    }

    override suspend fun exportConversation(bearer: String, conversationId: String): String {
        calls += "export-conversation"
        return "# conversation"
    }

    override suspend fun exportMessage(bearer: String, messageId: String): String {
        calls += "export-message"
        return "answer"
    }

    override suspend fun saveDraft(bearer: String, conversationId: String, body: String): String {
        calls += "save-draft"
        draft = body
        return draft
    }

    override suspend fun loadDraft(bearer: String, conversationId: String): String {
        calls += "load-draft"
        return draft
    }

    override suspend fun submitFeedback(bearer: String, messageId: String, value: String): Boolean {
        calls += "feedback-$value"
        return true
    }

    override suspend fun regenerate(bearer: String, messageId: String): MessageRun {
        calls += "regenerate"
        currentRunStatus = "completed"
        return run("run-regenerated")
    }

    override suspend fun speak(bearer: String, messageId: String): Boolean {
        calls += "speak"
        return true
    }

    override suspend fun messageCitations(bearer: String, messageId: String): List<MessageCitation> =
        listOf(MessageCitation("https://example.invalid/source", "Verified source"))

    private fun run(id: String) = MessageRun(id, alpha.id, currentRunStatus, 1, "assistant")

    private fun messages() = listOf(
        MessageRecord("user", alpha.id, "user", lastSentBody, "2026-08-08T00:00:01Z"),
        MessageRecord(
            "assistant",
            alpha.id,
            "assistant",
            "建议将系统拆分为控制平面、AI 数据平面和异步工作平面。\n" +
                "业务后端管理用户、会话、钱包和作品；AI Runtime 负责流式请求、模型路由与上下文编译。\n" +
                "```kotlin\nprintln(\"ok\")\n```\nA | B\n--- | ---\n1 | 2",
            "2026-08-08T00:00:02Z",
        ),
    )

    private fun modelOptions() = listOf(
        ModelOption("gpt-5.6-sol", "GPT-5.6 Sol", description = "复杂分析、编码与长任务"),
        ModelOption("claude-opus", "Claude Opus", description = "长文理解、写作与审查"),
        ModelOption("grok", "Grok", description = "实时信息与多模态理解"),
    )

    private fun session() = AuthSession("owner@example.invalid", "session", "access", "refresh")
}
