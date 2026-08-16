package cc.orbexa.ylven

import android.accessibilityservice.AccessibilityService
import android.graphics.Bitmap
import android.graphics.Color
import android.os.SystemClock
import androidx.activity.compose.setContent
import androidx.compose.ui.test.assertTextContains
import androidx.compose.ui.test.assertTextEquals
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.hasText
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
import androidx.compose.ui.test.swipeRight
import androidx.compose.runtime.key
import androidx.test.espresso.Espresso
import androidx.test.platform.app.InstrumentationRegistry
import androidx.core.view.ViewCompat
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import cc.orbexa.ylven.identity.AuthSession
import cc.orbexa.ylven.identity.Conversation
import cc.orbexa.ylven.identity.ComposerToolOption
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
import cc.orbexa.ylven.identity.defaultComposerToolOptions
import cc.orbexa.ylven.ui.YlvenApp
import cc.orbexa.ylven.ui.P03HomePage
import cc.orbexa.ylven.ui.consumerToolProgress
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
    fun capturesApprovedProductionPagesFromRealComposables() {
        val gateway = FlowGateway(enableAllTools = true)
        val session = AuthSession("陈平@example.invalid", "session", "access", "refresh", "physical-device")
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        P03ScreenshotStorage.resetDirectory(
            context = context,
            legacyDirectory = "production-screenshots",
            scopedDownloadDirectory = "ylven-p03-production",
        )
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

        composeRule.onNodeWithTag("p03-home-list").performScrollToNode(hasTestTag("CO-P03-001-HOME-TOOL"))
        composeRule.onNodeWithTag("CO-P03-001-HOME-TOOL").performClick()
        waitForTag("YL-A-024-S06-tool-tray")
        captureProductionPage("YL-A-024-PRODUCTION", "YL-A-024-S06-tool-tray")
        Espresso.pressBack()
        waitForTag("p03-local-draft-conversation")
        Espresso.pressBack()
        waitForTag("YL-A-018-C-P03_001-01")

        composeRule.onNodeWithTag("p03-open-conversation-drawer").performClick()
        waitForTag("YL-A-020-root")
        captureProductionPage("YL-A-020-PRODUCTION", "YL-A-020-root")
        composeRule.onNodeWithTag("p03-drawer-scrim").performClick()
        waitForDrawerClosed()

        composeRule.onNodeWithTag("CO-P03-001-HOME-SEND").performClick()
        waitForTag("p03-local-draft-conversation")
        hideKeyboardIfVisible()
        composeRule.onNodeWithTag("CO-P03-001-CHAT-MODEL").performClick()
        waitForTag("YL-A-033-root")
        waitForModalSheetToSettle()
        captureProductionPage("YL-A-033-PRODUCTION", "YL-A-033-root")
        // Capture response-mode choices for a model that actually supports them.
        // Auto-selection intentionally exposes only the automatic profile.
        composeRule.onNodeWithTag("p03-model-gpt-5.6-sol").performClick()

        composeRule.onNodeWithTag("CO-P03-001-CHAT-REASONING").performClick()
        waitForTag("YL-A-034-root")
        waitForModalSheetToSettle()
        composeRule.onNodeWithTag("p03-response-mode-quick").assertIsEnabled()
        captureProductionPage("YL-A-034-PRODUCTION", "YL-A-034-root")
        composeRule.onNodeWithTag("p03-response-mode-auto").performClick()
        Espresso.pressBack()
        waitForTag("YL-A-018-C-P03_001-01")

        composeRule.onNodeWithTag("p03-home-list").performScrollToNode(hasTestTag("p03-conversation-alpha"))
        composeRule.onNodeWithTag("p03-conversation-alpha").performClick()
        waitForTag("YL-A-023-C-P03_013-01")
        waitForCall(gateway, "messages")

        composeRule.onNodeWithTag("CO-P03-001-CHAT-MODEL").performClick()
        waitForTag("YL-A-033-root")
        waitForModalSheetToSettle()
        composeRule.onNodeWithTag("p03-model-gpt-5.6-sol").performClick()

        composeRule.onNodeWithTag("CO-P03-001-CHAT-REASONING").performClick()
        waitForTag("YL-A-034-root")
        waitForModalSheetToSettle()
        composeRule.onNodeWithTag("p03-response-mode-deep").performClick()

        scrollChatTo("YL-A-026-C-P03_018-01")
        captureProductionPage("YL-A-026-PRODUCTION", "YL-A-026-C-P03_018-01")
        captureProductionPage(
            "YL-A-023-PRODUCTION",
            "YL-A-023-C-P03_013-01",
            scrollAnchorTag = "YL-A-025-C-P03_016-01",
        )
    }

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
        waitForTagAbsent("p02-brand-splash")
        composeRule.onNodeWithTag("CO-P03-001-HOME-COMPOSER").performClick()
        waitForTag("p03-local-draft-conversation")
        assertTrue("Opening a blank chat must not persist a conversation", "create" !in gateway.calls)
        hideKeyboardIfVisible()
        leaveChatForHome()

        composeRule.onNodeWithTag("p03-home-list").performScrollToNode(hasTestTag("CO-P03-001-HOME-TOOL"))
        composeRule.onNodeWithTag("CO-P03-001-HOME-TOOL").performClick()
        waitForTag("YL-A-024-S06-tool-tray")
        composeRule.onNodeWithTag("p03-tool-0").performClick()
        waitForTagAbsent("YL-A-024-S06-tool-tray")
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01")
            .assertTextContains("请分析我接下来拍摄的内容：")
        composeRule.onNodeWithTag("p03-open-tool-tray").performClick()
        waitForTag("YL-A-024-S06-tool-tray")
        composeRule.onNodeWithTag("p03-tool-5").assertIsNotEnabled()
        Espresso.pressBack()
        waitForTag("p03-local-draft-conversation")
        hideKeyboardIfVisible()
        leaveChatForHome()

        composeRule.onNodeWithTag("CO-P03-001-HOME-SEND").performClick()
        waitForTag("p03-local-draft-conversation")
        composeRule.onNodeWithTag("p03-open-conversation-menu").performClick()
        waitForTag("CO-P03-001-CHAT-TEMPORARY")
        composeRule.onNodeWithTag("CO-P03-001-CHAT-TEMPORARY").performClick()
        waitForTag("p03-local-draft-conversation")
        assertTemporaryConversationScope()
        assertTrue("Opening a temporary draft must not persist a conversation", "temporary" !in gateway.calls)
        hideKeyboardIfVisible()
        leaveChatForHome()

        composeRule.onNodeWithTag("p03-open-conversation-drawer").performClick()
        waitForTag("p03-conversation-drawer")
        waitForTag("YL-A-020-root")
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
        composeRule.onNodeWithTag("p03-conversation-alpha").performScrollTo().performClick()
        waitForTag("YL-A-023-C-P03_013-01")
        waitForCall(gateway, "messages")
        waitForTag("YL-A-032-C-P03_032-01")
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").assertTextContains("saved draft")
        composeRule.onNodeWithTag("CO-P03-001-CHAT-MODEL").performClick()
        waitForTag("YL-A-033-root")
        waitForModalSheetToSettle()
        composeRule.onNodeWithTag("p03-model-gpt-5.6-sol").performClick()
        composeRule.onNodeWithTag("CO-P03-001-CHAT-REASONING").performClick()
        waitForTag("YL-A-034-root")
        waitForModalSheetToSettle()
        composeRule.onNodeWithTag("p03-response-mode-quick").assertIsEnabled().performClick()
        composeRule.onNodeWithTag("CO-P03-001-CHAT-REASONING").assertTextContains("快速")
        composeRule.onNodeWithTag("CO-P03-001-CHAT-REASONING").performClick()
        waitForTag("YL-A-034-root")
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
        gateway.releaseStream()
        waitForTag("p03-tool-progress")
        composeRule.onNodeWithText("正在生成图片").assertExists()
        waitForTag("YL-A-024-C-P03_011-01")
        composeRule.onNodeWithTag("YL-A-024-C-P03_011-01").performClick()
        waitForCondition { "cancel" in gateway.calls }
        waitForCondition { "events" in gateway.calls }

        waitForTag("YL-A-025-C-P03_016-01")
        waitForTag("YL-A-023-C-P03_017-01")
        waitForTag("YL-A-026-C-P03_026-01")
        scrollChatTo("YL-A-026-C-P03_018-01")
        scrollChatTo("YL-A-027-C-P03_019-01")
        scrollChatTo("YL-A-028-C-P03_020-01")
        scrollChatTo("YL-A-029-C-P03_021-01")
        scrollChatTo("YL-A-030-C-P03_022-01")
        composeRule.onNodeWithTag("YL-A-030-C-P03_022-01").performClick()
        scrollChatTo("YL-A-030-C-P03_023-01")
        composeRule.onNodeWithTag("YL-A-030-C-P03_023-01").performScrollTo().performClick()
        waitForCall(gateway, "export-message")
        composeRule.waitForIdle()
        scrollChatTo("YL-A-030-C-P03_025-01")
        val feedbackNodes = composeRule.onAllNodesWithTag("YL-A-030-C-P03_025-01")
        check(feedbackNodes.fetchSemanticsNodes().size >= 2) { "Both feedback controls are not present" }
        feedbackNodes[0].performScrollTo().performClick()
        waitForCondition { "feedback-up" in gateway.calls || "feedback-down" in gateway.calls }
        composeRule.waitForIdle()
        composeRule.onAllNodesWithTag("YL-A-030-C-P03_025-01")[1].performScrollTo().performClick()
        waitForCall(gateway, "feedback-up")
        waitForCall(gateway, "feedback-down")
        composeRule.waitForIdle()
        // Feedback is at the end of a horizontal action row. Return it to the leading actions
        // before tapping regenerate so this is a visible device interaction.
        composeRule.onNodeWithTag("YL-A-030-C-P03_023-01").performTouchInput { swipeRight() }
        composeRule.waitForIdle()
        scrollChatTo("YL-A-030-C-P03_024-01")
        composeRule.onNodeWithTag("YL-A-030-C-P03_024-01").performScrollTo().performClick()
        waitForCall(gateway, "regenerate")
        composeRule.waitForIdle()
        scrollChatTo("YL-A-030-C-P03_030-01")
        composeRule.onNodeWithTag("YL-A-030-C-P03_030-01").performScrollTo().performClick()
        waitForCall(gateway, "speak")
        scrollChatTo("p03-copy-code")
        composeRule.onNodeWithTag("p03-copy-code").performClick()
        gateway.failNextSend = true
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").performTextInput("retry me")
        composeRule.onNodeWithTag("YL-A-023-C-P03_009-01").performClick()
        waitForCondition { gateway.sendAttempts >= 2 }
        scrollChatTo("YL-A-023-C-P03_027-01")
        waitForTag("YL-A-023-C-P03_027-01")
        waitForTag("CO-P03-001-CHAT-RETRY")
        composeRule.onNodeWithText("当前网络不可用").assertExists()
        composeRule.onNodeWithTag("CO-P03-001-CHAT-RETRY").performClick()
        waitForCondition { gateway.sendAttempts >= 3 }
        waitForCondition {
            composeRule.onAllNodesWithTag("YL-A-023-C-P03_009-01").fetchSemanticsNodes().isNotEmpty() &&
                composeRule.onAllNodesWithTag("YL-A-023-C-P03_027-01").fetchSemanticsNodes().isEmpty() &&
                composeRule.onAllNodesWithText("retry me").fetchSemanticsNodes().size == 1
        }
        hideKeyboardIfVisible()
        leaveChatForHome()

        composeRule.onNodeWithTag("p03-home-list")
            .performScrollToNode(hasTestTag("p03-conversation-alpha"))
        composeRule.onNodeWithTag("p03-conversation-alpha").performClick()
        waitForTag("YL-A-032-C-P03_032-01")
        composeRule.onNodeWithTag("YL-A-025-C-P03_016-01")
            .performScrollToNode(hasText("retry me"))
        waitForCondition { composeRule.onAllNodesWithText("retry me").fetchSemanticsNodes().size == 1 }
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").assertTextEquals("")

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

    @Test
    fun w07RequiredStatesUseProductionComposablesAndConsumerCopy() {
        activity = launchP03TargetActivity()
        composeRule.mainClock.autoAdvance = true
        val session = AuthSession("states@example.invalid", "session", "access", "refresh", "physical-device")
        val homeGate = CompletableDeferred<Unit>()
        val loadingGateway = StateGateway(homeGate = homeGate)

        composeRule.runOnUiThread {
            activity.setContent {
                YlvenTheme(darkTheme = false) {
                    key(loadingGateway) { P03HomePage(loadingGateway, session, {}, {}) }
                }
            }
        }
        waitForTag("yl-a-018-loading")
        homeGate.complete(Unit)
        waitForTag("p03-home-empty-state")
        composeRule.onNodeWithText("从一句问题开始").assertExists()

        val offlineGateway = StateGateway(homeFailure = IOException("offline"))
        composeRule.runOnUiThread {
            activity.setContent {
                YlvenTheme(darkTheme = false) {
                    key(offlineGateway) { P03HomePage(offlineGateway, session, {}, {}) }
                }
            }
        }
        waitForTag("p03-state-panel")
        composeRule.onNodeWithText("当前网络不可用").assertExists()

        val errorGateway = StateGateway(homeFailure = IllegalStateException("unexpected"))
        composeRule.runOnUiThread {
            activity.setContent {
                YlvenTheme(darkTheme = false) {
                    key(errorGateway) { P03HomePage(errorGateway, session, {}, {}) }
                }
            }
        }
        waitForTag("p03-state-panel")
        composeRule.onNodeWithText("首页加载失败，请重试").assertExists()

        val selectorGateway = StateGateway(conversations = listOf(Conversation("state", "状态覆盖", "active", "刚刚")))
        composeRule.runOnUiThread {
            activity.setContent {
                YlvenTheme(darkTheme = false) {
                    YlvenApp(gateway = selectorGateway, initialSession = session)
                }
            }
        }
        waitForTag("YL-A-018-C-P03_001-01")
        waitForTag("p03-conversation-state")
        composeRule.onNodeWithTag("CO-P03-001-HOME-SEND").performClick()
        waitForTag("p03-local-draft-conversation")
        hideKeyboardIfVisible()
        composeRule.onNodeWithTag("CO-P03-001-CHAT-MODEL").performClick()
        waitForTag("YL-A-033-root")
        waitForTag("p03-model-auto")
        composeRule.onNodeWithTag("p03-model-disabled-model").assertIsNotEnabled()
        composeRule.onNodeWithTag("p03-model-basic-model").performClick()
        waitForTagAbsent("YL-A-033-root")
        composeRule.onNodeWithTag("CO-P03-001-CHAT-MODEL").assertTextContains("基础模型")
        composeRule.onNodeWithTag("CO-P03-001-CHAT-REASONING").performClick()
        waitForTag("p03-response-mode-degraded")
        Espresso.pressBack()
        waitForTagAbsent("YL-A-034-root")
        composeRule.onNodeWithTag("CO-P03-001-CHAT-MODEL").performClick()
        waitForTag("YL-A-033-root")
        composeRule.onNodeWithTag("p03-model-advanced-model").performClick()
        waitForTagAbsent("YL-A-033-root")
        composeRule.onNodeWithTag("CO-P03-001-CHAT-MODEL").assertTextContains("高级模型")
        composeRule.onNodeWithTag("CO-P03-001-CHAT-REASONING").performClick()
        waitForTag("YL-A-034-root")
        waitForTag("p03-response-mode-auto")
        composeRule.onNodeWithTag("p03-response-mode-quick").assertIsNotEnabled()
        composeRule.onNodeWithTag("p03-response-mode-deep").performClick()
        composeRule.onNodeWithTag("CO-P03-001-CHAT-REASONING").assertTextContains("深度")

        assertTrue(consumerToolProgress("tool_file_parse") == "正在阅读文件")
        assertTrue(consumerToolProgress("tool_image") == "正在生成图片")
        assertTrue(consumerToolProgress("tool_presentation") == "正在制作演示文稿")
        assertTrue(consumerToolProgress("provider_debug_payload") == null)
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

    private fun leaveChatForHome() {
        composeRule.onNodeWithTag("p03-page-back").performClick()
        waitForTag("YL-A-018-C-P03_001-01")
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
        // Compose semantics can settle one frame before SurfaceFlinger replaces
        // the launch/sheet frame returned by UiAutomation.takeScreenshot().
        SystemClock.sleep(350)
        instrumentation.waitForIdleSync()
        if (scrollAnchorTag != null) {
            composeRule.onNodeWithTag(scrollAnchorTag).performScrollToIndex(0)
            composeRule.waitForIdle()
            instrumentation.waitForIdleSync()
            SystemClock.sleep(350)
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
            check(!containsVendorOverlay(bitmap)) {
                "P03 production screenshot contains an orange vendor overlay: $name"
            }
        } finally {
            bitmap.recycle()
        }
    }

    private fun containsVendorOverlay(bitmap: Bitmap): Boolean {
        var orangePixels = 0
        val startX = bitmap.width * 2 / 3
        val endY = bitmap.height * 3 / 20
        for (y in 0 until endY step 2) {
            for (x in startX until bitmap.width step 2) {
                val pixel = bitmap.getPixel(x, y)
                val red = Color.red(pixel)
                val green = Color.green(pixel)
                val blue = Color.blue(pixel)
                if (red > 220 && green in 100..205 && blue < 90) {
                    orangePixels++
                    if (orangePixels > 200) return true
                }
            }
        }
        return false
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

private class FlowGateway(private val enableAllTools: Boolean = false) : IdentityGateway {
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
    private val alpha = Conversation("alpha", "多服务器 AI 架构设计", "active", "刚刚")
    private val beta = Conversation("beta", "对话上下文与长期记忆方案", "active", "昨天")
    private val gamma = Conversation("gamma", "首页与聊天体验重构", "active", "21:16")
    private val delta = Conversation("delta", "YLVEN 发布流程优化", "active", "昨天")
    private val epsilon = Conversation("epsilon", "模型能力评估", "active", "8 月 8 日")

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
        return HomeSnapshot(listOf(alpha, beta), modelOptions(), composerTools())
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
            Pair(listOf(alpha, beta, gamma), "next")
        } else {
            calls += "list-page-2"
            Pair(listOf(delta, epsilon), null)
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

    override suspend fun conversationMessages(bearer: String, conversationId: String): List<MessageRecord> {
        calls += "messages"
        return messages()
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
        return Pair(run(runId), listOf(RunEvent(after + 1, "tool_image", ""), RunEvent(after + 2, "provider_debug_payload", "hidden")))
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

    override suspend fun rerunMessage(
        bearer: String,
        messageId: String,
        modelId: String,
        reasoningProfile: String,
    ): MessageRun = regenerate(bearer, messageId)

    override suspend fun speak(bearer: String, messageId: String): Boolean {
        calls += "speak"
        return true
    }

    override suspend fun messageCitations(bearer: String, messageId: String): List<MessageCitation> =
        listOf(MessageCitation("https://example.invalid/source", "Verified source"))

    private fun run(id: String) = MessageRun(id, alpha.id, currentRunStatus, 1, "assistant")

    private fun messages(): List<MessageRecord> {
        val initial = listOf(
            MessageRecord(
                "user",
                alpha.id,
                "user",
                "如何让多模型对话具备上下文，并为多服务器部署做好准备？",
                "2026-08-08T00:00:01Z",
            ),
            MessageRecord(
                "assistant",
                alpha.id,
                "assistant",
                "可以。建议把当前系统设计为无状态 AI Runtime，并由 PostgreSQL 保存完整会话事实。\n\n" +
                    "每次发送消息时，后端根据 conversation_id 重新组装最近对话、结构化摘要和当前附件。\n\n" +
                    "这样后续切换 GPT、Claude 或 Grok 时，仍然能够保持上下文连续。\n\n" +
                    "```kotlin\nprintln(\"ok\")\n```\n\n" +
                    "A | B\n--- | ---\n1 | 2",
                "2026-08-08T00:00:02Z",
            ),
        )
        return if (sendAttempts >= 3 && lastSentBody == "retry me") {
            initial + MessageRecord(
                "retry-user",
                alpha.id,
                "user",
                lastSentBody,
                "2026-08-08T00:00:03Z",
            )
        } else {
            initial
        }
    }

    private fun composerTools() = listOf(
        ComposerToolOption("camera", "拍照", true, "请分析我接下来拍摄的内容："),
        ComposerToolOption("image", "选择图片", enableAllTools, "请分析我接下来选择的图片："),
        ComposerToolOption("file", "上传文件", enableAllTools, "请分析我接下来上传的文件："),
        ComposerToolOption("image-generation", "生成图片", enableAllTools, "请帮我生成图片："),
        ComposerToolOption("presentation", "制作演示", enableAllTools, "请帮我制作演示文稿："),
        ComposerToolOption("deep-research", "深度研究", enableAllTools, "请帮我深入研究："),
    )

    private fun modelOptions() = listOf(
        ModelOption(
            "gpt-5.6-sol",
            "GPT-5.6 Sol",
            description = "复杂分析、编码与长任务",
            reasoningProfiles = listOf("auto", "quick", "standard", "deep"),
        ),
        ModelOption("claude-opus", "Claude Opus", description = "长文理解、写作与审查"),
        ModelOption("grok", "Grok", description = "实时信息与多模态理解"),
    )

    private fun session() = AuthSession("owner@example.invalid", "session", "access", "refresh")
}

private class StateGateway(
    private val homeGate: CompletableDeferred<Unit>? = null,
    private val homeFailure: Throwable? = null,
    private val conversations: List<Conversation> = emptyList(),
) : IdentityGateway {
    override suspend fun startRegistration(email: String) = OtpChallenge("register", email, "123456")
    override suspend fun finishRegistration(challenge: OtpChallenge, code: String, password: String) = Unit
    override suspend fun startLogin(email: String) = OtpChallenge("login", email, "123456")
    override suspend fun finishLogin(challenge: OtpChallenge, code: String) = AuthSession("states@example.invalid", "session", "access", "refresh")
    override suspend fun refresh(session: AuthSession) = session
    override suspend fun devices(bearer: String): List<DeviceSession> = emptyList()
    override suspend fun revokeDevice(bearer: String, sessionId: String) = Unit
    override suspend fun logout(bearer: String, allDevices: Boolean) = Unit

    override suspend fun home(bearer: String): HomeSnapshot {
        homeGate?.await()
        homeFailure?.let { throw it }
        return HomeSnapshot(conversations, stateModels(), defaultComposerToolOptions())
    }

    override suspend fun models(bearer: String): List<ModelOption> = stateModels()

    private fun stateModels() = listOf(
        ModelOption("basic-model", "基础模型", reasoningProfiles = listOf("auto")),
        ModelOption("advanced-model", "高级模型", reasoningProfiles = listOf("auto", "standard", "deep")),
        ModelOption("disabled-model", "不可用模型", enabled = false, reasoningProfiles = listOf("auto")),
    )
}
