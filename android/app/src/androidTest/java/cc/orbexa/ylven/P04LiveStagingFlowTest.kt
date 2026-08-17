package cc.orbexa.ylven

import android.graphics.Bitmap
import android.os.SystemClock
import androidx.compose.ui.semantics.SemanticsConfiguration
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.hasTestTag
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.compose.ui.test.onAllNodesWithTag
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollTo
import androidx.compose.ui.test.performScrollToNode
import androidx.compose.ui.test.performTextClearance
import androidx.compose.ui.test.performTextInput
import androidx.test.espresso.Espresso
import androidx.test.platform.app.InstrumentationRegistry
import cc.orbexa.ylven.identity.AuthSession
import cc.orbexa.ylven.identity.HttpIdentityGateway
import cc.orbexa.ylven.identity.SessionStore
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

/**
 * Exercises the P04 surface through the installed production APK and its real
 * HTTP gateway. It deliberately avoids the deterministic UI gateway used by
 * P03 visual-state tests, while retaining screenshots for direct review.
 */
class P04LiveStagingFlowTest {
    @get:Rule
    val composeRule = createEmptyComposeRule()

    @Test
    fun authenticatedUserCompletesP04ModelAndComparisonJourney() = runBlocking {
        val instrumentation = InstrumentationRegistry.getInstrumentation()
        val context = instrumentation.targetContext
        val gateway = HttpIdentityGateway(deviceId = "p04-physical-${System.currentTimeMillis()}")
        val session = ensureSession(gateway, context)
        val comparisonConversation = gateway.createConversation(session.bearer, "P04 真机验收 ${System.currentTimeMillis()}")

        P03ScreenshotStorage.resetDirectory(
            context = context,
            legacyDirectory = "p04-live-staging",
            scopedDownloadDirectory = "ylven-p04-live-staging",
        )
        launchP03TargetActivity()
        waitForTag("YL-A-018-C-P03_001-01", 30_000)

        // P04-W02: select a per-message model, verify the completed answer provenance,
        // then rerun with another model while retaining the original answer.
        composeRule.onNodeWithTag("CO-P03-001-HOME-SEND").performClick()
        waitForTag("YL-A-023-C-P03_013-01", 20_000)
        composeRule.onNodeWithTag("CO-P03-001-CHAT-MODEL").performClick()
        waitForTag("p03-model-ylven-default", 20_000)
        capture("P04-MODEL-SELECTOR")
        composeRule.onNodeWithTag("p03-model-ylven-default").performScrollTo().performClick()
        composeRule.onNodeWithTag("CO-P03-001-CHAT-REASONING").performClick()
        waitForTag("p03-response-mode-auto", 20_000)
        capture("P04-REASONING-PROFILE")
        composeRule.onNodeWithTag("p03-response-mode-auto").performClick()
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").performTextInput("请简要说明 P04 模型路由的验收重点。")
        Espresso.closeSoftKeyboard()
        composeRule.onNodeWithTag("YL-A-023-C-P03_009-01").performClick()
        waitForTagGone("YL-A-023-C-P03_009-01", 10_000)
        waitForTag("YL-A-023-C-P03_009-01", 110_000)
        waitForAnswerProvenance(1, 20_000)
        capture("P04-ANSWER-PROVENANCE")

        composeRule.onNodeWithTag("CO-P03-001-CHAT-MODEL").performClick()
        waitForTag("p03-model-p04-reasoner", 20_000)
        composeRule.onNodeWithTag("p03-model-p04-reasoner").performScrollTo().performClick()
        composeRule.onNodeWithTag("YL-A-025-C-P03_016-01")
            .performScrollToNode(hasTestTag("YL-A-030-C-P03_024-01"))
        composeRule.onNodeWithTag("YL-A-030-C-P03_024-01").performClick()
        waitForTagGone("YL-A-023-C-P03_009-01", 10_000)
        waitForTag("YL-A-023-C-P03_009-01", 110_000)
        waitForAnswerProvenance(2, 30_000)
        capture("P04-RERUN-PRESERVES-ANSWER")
        Espresso.pressBack()
        waitForTag("YL-A-018-C-P03_001-01", 20_000)

        composeRule.onNodeWithTag("p03-open-account").performClick()
        waitForTag("p04-open-workbench", 20_000)
        composeRule.onNodeWithTag("p04-open-workbench").performScrollTo().performClick()
        waitForTag("p04-open-ai-settings", 30_000)
        capture("P04-HUB")

        composeRule.onNodeWithTag("p04-open-ai-settings").performClick()
        waitForTag("p04-ai-model-row", 30_000)
        composeRule.onNodeWithTag("p04-ai-model-row").performClick()
        waitForTag("p04-ai-model-ylven-default", 20_000)
        composeRule.onNodeWithTag("p04-ai-model-ylven-default").performClick()
        waitForTag("p04-save-ai-preference", 20_000)
        capture("P04-AI-PREFERENCES")
        composeRule.onNodeWithTag("p04-save-ai-preference").performClick()
        waitForTag("p04-open-ai-settings", 20_000)
        assertNoP04Error()

        composeRule.onNodeWithTag("p04-open-service-status").performClick()
        waitForText("服务状态", 20_000)
        capture("P04-SERVICE-STATUS")
        Espresso.pressBack()
        waitForTag("p04-open-ai-settings", 20_000)

        val conversationTag = "p04-hub-conversation-${comparisonConversation.id}"
        waitForTag(conversationTag, 30_000)
        val conversationID = comparisonConversation.id
        composeRule.onNodeWithTag(conversationTag).performScrollTo().performClick()
        waitForTag("p04-conversation-model-row", 20_000)
        composeRule.onNodeWithTag("p04-conversation-model-row").performClick()
        waitForTag("p04-conversation-model-ylven-default", 20_000)
        composeRule.onNodeWithTag("p04-conversation-model-ylven-default").performClick()
        waitForTag("p04-save-conversation-settings", 20_000)
        capture("P04-CONVERSATION-SETTINGS")
        composeRule.onNodeWithTag("p04-save-conversation-settings").performClick()
        waitForTag("p04-open-ai-settings", 20_000)
        assertNoP04Error()

        composeRule.onNodeWithTag("p04-open-branches-$conversationID").performClick()
        waitForTag("p04-create-branch", 20_000)
        composeRule.onNodeWithTag("p04-create-branch").performClick()
        waitForBranchCard(20_000)
        capture("P04-BRANCHES")
        Espresso.pressBack()
        waitForTag("p04-open-ai-settings", 20_000)
        assertNoP04Error()

        composeRule.onNodeWithTag("p04-open-comparison-$conversationID").performClick()
        waitForTag("p04-comparison-prompt", 20_000)
        composeRule.onNodeWithTag("p04-comparison-prompt").performTextClearance()
        composeRule.onNodeWithTag("p04-comparison-prompt").performTextInput("请用两个模型概括 P04 真机回归的重点。")
        Espresso.closeSoftKeyboard()
        composeRule.onNodeWithTag("p04-comparison-model-ylven-default").performClick()
        composeRule.onNodeWithTag("p04-comparison-model-p04-reasoner").performClick()
        composeRule.onNodeWithTag("p04-create-comparison").assertIsEnabled().performClick()
        waitForTag("p04-comparison-status", 30_000)
        waitForComparison("completed", 150_000)
        val tabs = actionTags(comparisonTabMatcher)
        assertTrue("P04 comparison did not render multiple model tabs", tabs.size >= 2)
        composeRule.onNodeWithTag(tabs.last()).performClick()
        capture("P04-COMPARISON-RESULTS")
        composeRule.onNodeWithTag(firstActionTag(adoptMatcher)).performClick()
        waitForComparison("adopted", 30_000)
        composeRule.onNodeWithTag(firstActionTag(synthesisMatcher)).performClick()
        waitForComparison("synthesized", 150_000)
        capture("P04-COMPARISON-SYNTHESIS")
        assertNoP04Error()
    }

    private suspend fun ensureSession(gateway: HttpIdentityGateway, context: android.content.Context): AuthSession {
        SessionStore(context).load()?.let { return it }
        val email = "p04.physical.${System.currentTimeMillis()}@example.com"
        val registration = gateway.startRegistration(email)
        val session = gateway.finishRegistrationSession(
            registration,
            requireNotNull(registration.debugCode),
            "P04Physical${System.nanoTime()}a1",
        )
        gateway.initializeWorkspace(session)
        SessionStore(context).save(session)
        return session
    }

    private fun waitForTag(tag: String, timeout: Long) {
        composeRule.waitUntil(timeoutMillis = timeout) {
            composeRule.onAllNodesWithTag(tag).fetchSemanticsNodes().isNotEmpty()
        }
    }

    private fun waitForText(text: String, timeout: Long) {
        composeRule.waitUntil(timeoutMillis = timeout) {
            composeRule.onAllNodes(SemanticsMatcher("text $text") { node ->
                node.config.getOrNull(SemanticsProperties.Text)?.any { it.text == text } == true
            }).fetchSemanticsNodes().isNotEmpty()
        }
    }

    private fun waitForTagGone(tag: String, timeout: Long) {
        composeRule.waitUntil(timeoutMillis = timeout) {
            composeRule.onAllNodesWithTag(tag).fetchSemanticsNodes().isEmpty()
        }
    }

    private fun waitForHubConversation(): String {
        composeRule.waitUntil(timeoutMillis = 30_000) {
            hubConversationTags().isNotEmpty()
        }
        return hubConversationTags().first()
    }

    private fun waitForBranchCard(timeout: Long) {
        composeRule.waitUntil(timeoutMillis = timeout) {
            composeRule.onAllNodes(branchMatcher).fetchSemanticsNodes().isNotEmpty()
        }
    }

    private fun waitForAnswerProvenance(minimumCount: Int, timeout: Long) {
        composeRule.waitUntil(timeoutMillis = timeout) {
            composeRule.onAllNodes(answerProvenanceMatcher).fetchSemanticsNodes().count { node ->
                node.config.getOrNull(SemanticsProperties.Text)
                    ?.any { it.text.startsWith("回答来源：") && it.text.contains(" · ") } == true
            } >= minimumCount
        }
    }

    private fun waitForComparison(expected: String, timeout: Long) {
        val deadline = SystemClock.uptimeMillis() + timeout
        while (SystemClock.uptimeMillis() < deadline) {
            val complete = composeRule.onAllNodes(SemanticsMatcher("comparison status $expected") { node ->
                node.config.getOrNull(SemanticsProperties.Text)?.any { it.text == "状态：${comparisonStatusLabel(expected)}" } == true
            }).fetchSemanticsNodes().isNotEmpty()
            if (complete) return
            composeRule.onNodeWithTag("p04-refresh-comparison").performClick()
            SystemClock.sleep(2_000)
        }
        error("P04 comparison did not reach $expected within $timeout ms")
    }

    private fun hubConversationTags(): Set<String> =
        composeRule.onAllNodes(hubConversationMatcher).fetchSemanticsNodes()
            .mapNotNull { it.config.getOrNull(SemanticsProperties.TestTag) }
            .toSet()

    private fun firstActionTag(matcher: SemanticsMatcher): String {
        return actionTags(matcher).firstOrNull() ?: error("P04 comparison action is missing a test tag")
    }

    private fun actionTags(matcher: SemanticsMatcher): List<String> {
        composeRule.waitUntil(timeoutMillis = 30_000) {
            composeRule.onAllNodes(matcher).fetchSemanticsNodes().isNotEmpty()
        }
        return composeRule.onAllNodes(matcher).fetchSemanticsNodes().mapNotNull {
            it.config.getOrNull(SemanticsProperties.TestTag)
        }
    }

    private fun capture(name: String) {
        val instrumentation = InstrumentationRegistry.getInstrumentation()
        composeRule.waitForIdle()
        instrumentation.waitForIdleSync()
        val bitmap = requireNotNull(instrumentation.uiAutomation.takeScreenshot()) {
            "Could not capture $name"
        }
        try {
            P03ScreenshotStorage.writePng(
                context = instrumentation.targetContext,
                bitmap = bitmap,
                legacyDirectory = "p04-live-staging",
                scopedDownloadDirectory = "ylven-p04-live-staging",
                fileName = "$name.png",
                subject = "P04 live staging screenshot",
            )
        } finally {
            bitmap.recycle()
        }
    }

    private fun assertNoP04Error() {
        assertTrue(
            "The P04 live flow displayed an application error",
            composeRule.onAllNodesWithTag("p04-inline-error").fetchSemanticsNodes().isEmpty(),
        )
    }

    private companion object {
        fun comparisonStatusLabel(status: String): String = when (status) {
            "completed" -> "已完成"
            "adopted" -> "已采纳"
            "synthesized" -> "已生成综合回答"
            else -> status
        }

        val hubConversationMatcher = SemanticsMatcher("P04 hub conversation") { node ->
            node.config.getOrNull(SemanticsProperties.TestTag)?.startsWith("p04-hub-conversation-") == true
        }
        val branchMatcher = SemanticsMatcher("P04 branch") { node ->
            node.config.getOrNull(SemanticsProperties.TestTag)?.startsWith("p04-branch-") == true
        }
        val adoptMatcher = SemanticsMatcher("P04 comparison adopt") { node ->
            node.config.getOrNull(SemanticsProperties.TestTag)?.startsWith("p04-adopt-") == true
        }
        val synthesisMatcher = SemanticsMatcher("P04 comparison synthesis") { node ->
            node.config.getOrNull(SemanticsProperties.TestTag)?.startsWith("p04-synthesize-") == true
        }
        val comparisonTabMatcher = SemanticsMatcher("P04 comparison tab") { node ->
            node.config.getOrNull(SemanticsProperties.TestTag)?.startsWith("p04-comparison-tab-") == true
        }
        val answerProvenanceMatcher = SemanticsMatcher("P04 answer provenance") { node ->
            node.config.getOrNull(SemanticsProperties.TestTag)?.startsWith("p04-answer-source-") == true
        }
    }
}

private fun <T> SemanticsConfiguration.getOrNull(key: androidx.compose.ui.semantics.SemanticsPropertyKey<T>): T? =
    if (contains(key)) get(key) else null
