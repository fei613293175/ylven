package cc.orbexa.ylven

import android.os.Bundle
import android.os.SystemClock
import androidx.compose.ui.semantics.SemanticsConfiguration
import androidx.compose.ui.semantics.SemanticsPropertyKey
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.SemanticsMatcher
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
import java.time.Instant
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

/**
 * Uses MainActivity, an encrypted session established through the real public
 * staging registration flow, and the configured public API. No Fake gateway
 * is involved in this test.
 */
class P03LiveStagingFlowTest {
    @get:Rule
    val composeRule = createEmptyComposeRule()

    @Test
    fun authenticatedStagingSessionCompletesARealConversationJourney() {
        reportStage("shell_activity_launch")
        launchP03TargetActivity()
        reportStage("shell_activity_ready")
        composeRule.mainClock.advanceTimeBy(3_100)
        reportStage("home_wait")
        waitForTag("YL-A-018-C-P03_001-01", 30_000)
        reportStage("home_ready")
        val marker = "device-${Instant.now().epochSecond}"

        composeRule.onNodeWithTag("p03-open-new-conversation").performClick()
        waitForTag("YL-A-019-root", 20_000)
        reportStage("new_conversation_form_ready")
        composeRule.onNodeWithTag("YL-A-019-C-P03_002-01").performClick()
        waitForTag("YL-A-023-C-P03_013-01", 20_000)
        reportStage("conversation_created")

        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").performTextClearance()
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").performTextInput("P03 physical staging $marker")
        Espresso.closeSoftKeyboard()
        reportStage("message_send_start")
        composeRule.onNodeWithTag("YL-A-023-C-P03_009-01").performClick()
        reportStage("message_send_clicked")
        waitForTagGone("YL-A-023-C-P03_009-01", 10_000)
        // The real upstream request can legitimately run for up to the API's
        // bounded 95-second deadline. Keep the physical-flow assertion above
        // that budget so slow successful replies are not reported as failures.
        waitForTag("YL-A-023-C-P03_009-01", 110_000)
        composeRule.onNodeWithTag("YL-A-025-C-P03_016-01")
            .performScrollToNode(hasTestTag("YL-A-026-C-P03_026-01"))
        waitForTagText("YL-A-026-C-P03_026-01", "completed", 10_000)
        assertNoInlineError()
        reportStage("message_completed")

        composeRule.onNodeWithTag("p03-open-conversation-menu").performClick()
        waitForTag("YL-A-022-root", 20_000)
        performSheetActionAndWaitForDismiss("YL-A-022-C-P03_029-01", 20_000)
        reportStage("conversation_exported")

        composeRule.onNodeWithTag("p03-open-conversation-menu").performClick()
        composeRule.onNodeWithTag("p03-open-rename-current").performClick()
        waitForTag("p03-rename-input", 20_000)
        composeRule.onNodeWithTag("p03-rename-input").performTextClearance()
        composeRule.onNodeWithTag("p03-rename-input").performTextInput("P03 $marker")
        Espresso.closeSoftKeyboard()
        performSheetActionAndWaitForDismiss("YL-A-022-C-P03_005-01", 20_000)
        assertNoInlineError()
        reportStage("conversation_renamed")

        Espresso.pressBack()
        waitForTag("YL-A-018-C-P03_001-01", 20_000)
        composeRule.onNodeWithTag("p03-home-list")
            .performScrollToNode(hasTestTag("p03-open-conversation-drawer"))
        composeRule.onNodeWithTag("p03-open-conversation-drawer").performClick()
        waitForTag("p03-conversation-drawer", 20_000)
        composeRule.onNodeWithTag("p03-open-search").performClick()
        waitForTag("YL-A-021-root", 20_000)
        composeRule.onNodeWithTag("YL-A-021-C-P03_004-01").performTextInput(marker)
        val matchingConversation = waitForConversationCard()
        composeRule.onNodeWithTag(matchingConversation).performScrollTo().performClick()
        waitForTag("YL-A-023-C-P03_013-01", 20_000)
        composeRule.onNodeWithTag("p03-open-conversation-menu").performClick()
        performSheetActionAndWaitForDismiss("YL-A-022-C-P03_006-01", 20_000)
        waitForTag("YL-A-018-C-P03_001-01", 20_000)
        assertNoInlineError()
        reportStage("conversation_archived")

        composeRule.onNodeWithTag("p03-home-list")
            .performScrollToNode(hasTestTag("p03-open-temporary-conversation"))
        composeRule.onNodeWithTag("p03-open-temporary-conversation").performClick()
        waitForTag("YL-A-031-root", 20_000)
        composeRule.onNodeWithTag("p03-temporary-title").performTextInput("P03 temporary $marker")
        Espresso.closeSoftKeyboard()
        composeRule.onNodeWithTag("YL-A-031-C-P03_028-01").performScrollTo().performClick()
        waitForTag("YL-A-023-C-P03_013-01", 20_000)
        composeRule.onNodeWithTag("p03-open-conversation-menu").performClick()
        composeRule.onNodeWithTag("YL-A-022-C-P03_007-01").performClick()
        waitForTag("p03-confirm-delete", 20_000)
        performSheetActionAndWaitForDismiss("p03-confirm-delete", 20_000)
        waitForTag("YL-A-018-C-P03_001-01", 20_000)
        assertNoInlineError()
        reportStage("temporary_conversation_deleted")
    }

    private fun waitForTag(tag: String, timeout: Long) {
        composeRule.waitUntil(timeoutMillis = timeout) {
            composeRule.onAllNodesWithTag(tag).fetchSemanticsNodes().isNotEmpty()
        }
    }

    private fun waitForTagGone(tag: String, timeout: Long) {
        composeRule.waitUntil(timeoutMillis = timeout) {
            composeRule.onAllNodesWithTag(tag).fetchSemanticsNodes().isEmpty()
        }
    }

    private fun performSheetActionAndWaitForDismiss(tag: String, timeout: Long) {
        waitForTag(tag, timeout)
        composeRule.mainClock.autoAdvance = false
        try {
            composeRule.onNodeWithTag(tag).performClick()
            val deadline = SystemClock.uptimeMillis() + timeout
            while (SystemClock.uptimeMillis() < deadline) {
                composeRule.runOnUiThread { composeRule.mainClock.advanceTimeByFrame() }
                if (composeRule.onAllNodesWithTag("YL-A-022-root").fetchSemanticsNodes().isEmpty()) {
                    return
                }
                Thread.sleep(10)
            }
            error("Conversation action sheet did not close within $timeout ms after $tag")
        } finally {
            composeRule.mainClock.autoAdvance = true
        }
    }

    private fun waitForTagText(tag: String, expected: String, timeout: Long) {
        composeRule.waitUntil(timeoutMillis = timeout) {
            composeRule.onAllNodesWithTag(tag).fetchSemanticsNodes().any { node ->
                node.config.getOrNull(SemanticsProperties.Text)
                    ?.any { text -> text.text.contains(expected, ignoreCase = true) } == true
            }
        }
    }

    private fun conversationCardTags(): Set<String> =
        composeRule.onAllNodes(conversationCardMatcher).fetchSemanticsNodes()
            .mapNotNull { it.config.getOrNull(SemanticsProperties.TestTag) }
            .toSet()

    private fun waitForConversationCard(): String {
        composeRule.waitUntil(timeoutMillis = 20_000) {
            conversationCardTags().isNotEmpty()
        }
        return conversationCardTags().first()
    }

    private fun assertNoInlineError() {
        composeRule.waitForIdle()
        assertTrue(
            "The live staging flow displayed an application error",
            composeRule.onAllNodesWithTag("p01-inline-error").fetchSemanticsNodes().isEmpty(),
        )
    }

    private fun reportStage(stage: String) {
        InstrumentationRegistry.getInstrumentation().sendStatus(
            2,
            Bundle().apply { putString("ylven_stage", stage) },
        )
    }

    private companion object {
        val conversationCardMatcher = SemanticsMatcher("P03 conversation card") { node ->
            node.config.getOrNull(SemanticsProperties.TestTag)?.startsWith("p03-conversation-") == true
        }
    }
}

private fun <T> SemanticsConfiguration.getOrNull(key: SemanticsPropertyKey<T>): T? =
    if (contains(key)) get(key) else null
