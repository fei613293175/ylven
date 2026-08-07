package cc.orbexa.ylven

import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.fetchSemanticsNodes
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onAllNodes
import androidx.compose.ui.test.onAllNodesWithTag
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollTo
import androidx.compose.ui.test.performTextClearance
import androidx.compose.ui.test.performTextInput
import androidx.test.espresso.Espresso
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
    val composeRule = createAndroidComposeRule<MainActivity>()

    @Test
    fun authenticatedStagingSessionCompletesARealConversationJourney() {
        waitForTag("YL-A-018-C-P03_001-01", 30_000)
        val marker = "device-${Instant.now().epochSecond}"

        composeRule.onNodeWithTag("p03-refresh-home").performClick()
        val existingCards = conversationCardTags()
        composeRule.onNodeWithTag("YL-A-019-C-P03_002-01").performClick()
        val createdCard = waitForNewConversationCard(existingCards)
        val conversationId = createdCard.removePrefix("p03-conversation-")
        composeRule.onNodeWithTag(createdCard).performScrollTo().performClick()
        waitForTag("YL-A-023-C-P03_013-01", 20_000)

        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").performTextClearance()
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").performTextInput("P03 physical staging $marker")
        Espresso.closeSoftKeyboard()
        composeRule.onNodeWithTag("YL-A-023-C-P03_009-01").performClick()
        waitForTagText("YL-A-026-C-P03_026-01", "completed", 45_000)
        assertNoInlineError()
        composeRule.onNodeWithTag("YL-A-022-C-P03_029-01").performClick()

        Espresso.pressBack()
        waitForTag("YL-A-018-C-P03_001-01", 20_000)
        composeRule.onNodeWithTag("p03-open-conversation-drawer").performClick()
        waitForTag("p03-conversation-drawer", 20_000)
        composeRule.onNodeWithTag("YL-A-021-C-P03_004-01").performTextInput(marker)
        waitForTag("p03-drawer-rename-$conversationId", 20_000)
        composeRule.onNodeWithTag("p03-drawer-rename-$conversationId").performClick()
        composeRule.onNodeWithTag("p03-rename-input").performTextClearance()
        composeRule.onNodeWithTag("p03-rename-input").performTextInput("P03 $marker")
        Espresso.closeSoftKeyboard()
        composeRule.onNodeWithTag("YL-A-022-C-P03_005-01").performClick()
        waitForTag("p03-drawer-rename-$conversationId", 20_000)
        assertNoInlineError()
        composeRule.onAllNodesWithTag("YL-A-022-C-P03_006-01")[0].performClick()
        waitForTagGone("p03-drawer-rename-$conversationId", 20_000)
        assertNoInlineError()

        composeRule.onNodeWithTag("YL-A-021-C-P03_004-01").performTextClearance()
        composeRule.onNodeWithTag("YL-A-021-C-P03_004-01").performTextInput("P03 $marker")
        waitForTag("p03-drawer-rename-$conversationId", 20_000)
        composeRule.onAllNodesWithTag("YL-A-022-C-P03_007-01")[0].performClick()
        waitForTagGone("p03-drawer-rename-$conversationId", 20_000)
        assertNoInlineError()
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

    private fun waitForNewConversationCard(existing: Set<String>): String {
        composeRule.waitUntil(timeoutMillis = 20_000) {
            (conversationCardTags() - existing).isNotEmpty()
        }
        return (conversationCardTags() - existing).single()
    }

    private fun assertNoInlineError() {
        composeRule.waitForIdle()
        assertTrue(
            "The live staging flow displayed an application error",
            composeRule.onAllNodesWithTag("p01-inline-error").fetchSemanticsNodes().isEmpty(),
        )
    }

    private companion object {
        val conversationCardMatcher = SemanticsMatcher("P03 conversation card") { node ->
            node.config.getOrNull(SemanticsProperties.TestTag)?.startsWith("p03-conversation-") == true
        }
    }
}
