package cc.orbexa.ylven

import androidx.compose.ui.semantics.SemanticsConfiguration
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.hasTestTag
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.compose.ui.test.onAllNodesWithTag
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollTo
import androidx.compose.ui.test.performScrollToNode
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.test.performTextInput
import androidx.compose.ui.test.swipeLeft
import androidx.test.espresso.Espresso
import androidx.test.platform.app.InstrumentationRegistry
import cc.orbexa.ylven.identity.AuthSession
import cc.orbexa.ylven.identity.ApiException
import cc.orbexa.ylven.identity.Conversation
import cc.orbexa.ylven.identity.HttpIdentityGateway
import cc.orbexa.ylven.identity.SessionStore
import kotlinx.coroutines.delay
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

/** Exercises only W02 production paths against the installed APK and staging API. */
class P04W02LiveStagingFlowTest {
    @get:Rule
    val composeRule = createEmptyComposeRule()

    @Test
    fun authenticatedUserAppliesDefaultsUsesExplicitRunAndCreatesAnswerBranch() = runBlocking {
        val instrumentation = InstrumentationRegistry.getInstrumentation()
        val context = instrumentation.targetContext
        val gateway = HttpIdentityGateway(deviceId = "p04-w02-physical-${System.currentTimeMillis()}")
        val session = ensureSession(gateway, context)
        val models = gateway.models(session.bearer)
        val model = requireNotNull(models.firstOrNull {
            it.id == "ylven-default" && it.enabled && "auto" in it.reasoningProfiles
        }) { "Staging has no enabled default model with the automatic profile" }
        val alternateModel = requireNotNull(models.firstOrNull {
            it.id != model.id && it.enabled && "auto" in it.reasoningProfiles
        }) { "Staging exposes no second enabled model for the per-message selector" }
        val conversationsBefore = gateway.listConversations(session.bearer).first.map { it.id }.toSet()

        P03ScreenshotStorage.resetDirectory(
            context = context,
            legacyDirectory = "p04-live-staging",
            scopedDownloadDirectory = "ylven-p04-live-staging",
        )
        launchP03TargetActivity()
        waitForTag("YL-A-018-C-P03_001-01", 30_000)

        // P04-007: select and save the global default through the installed UI, then
        // read it back through the real service before continuing.
        composeRule.onNodeWithTag("p03-open-account").performClick()
        waitForTag("p04-open-workbench", 20_000)
        composeRule.onNodeWithTag("p04-open-workbench").performClick()
        waitForTag("p04-open-ai-settings", 20_000)
        composeRule.onNodeWithTag("p04-open-ai-settings").performClick()
        waitForTag("p04-ai-model-row", 20_000)
        composeRule.onNodeWithTag("p04-ai-model-row").performClick()
        waitForTag("p04-ai-model-${model.id}", 20_000)
        composeRule.onNodeWithTag("p04-ai-model-${model.id}").performScrollTo().performClick()
        capture("P04-W02-GLOBAL-DEFAULT")
        composeRule.onNodeWithTag("p04-save-ai-preference").performScrollTo().performClick()
        await("global preference readback") {
            gateway.aiPreference(session.bearer).let { it.modelId == model.id && it.reasoningProfile == "auto" }
        }

        // P04-009: switch the per-message selection to a distinct catalog model,
        // capture the actual composer state, then restore the staged runnable model
        // before sending. The alternate runtime is intentionally not treated as a
        // successful response path until staging configures its authorized channel.
        Espresso.pressBack()
        waitForTag("p04-open-ai-settings", 20_000)
        Espresso.pressBack()
        waitForTag("p04-open-workbench", 20_000)
        Espresso.pressBack()
        waitForTag("CO-P03-001-HOME-SEND", 20_000)
        composeRule.onNodeWithTag("CO-P03-001-HOME-SEND").performClick()
        waitForTag("YL-A-023-C-P03_013-01", 20_000)
        composeRule.onNodeWithTag("CO-P03-001-CHAT-MODEL").performClick()
        waitForTag("p03-model-${alternateModel.id}", 20_000)
        composeRule.onNodeWithTag("p03-model-${alternateModel.id}").performScrollTo().performClick()
        capture("P04-W02-PER-MESSAGE-SELECTOR")
        composeRule.onNodeWithTag("CO-P03-001-CHAT-MODEL").performClick()
        waitForTag("p03-model-${model.id}", 20_000)
        composeRule.onNodeWithTag("p03-model-${model.id}").performScrollTo().performClick()
        composeRule.onNodeWithTag("CO-P03-001-CHAT-REASONING").performClick()
        waitForTag("p03-response-mode-auto", 20_000)
        composeRule.onNodeWithTag("p03-response-mode-auto").performClick()
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01")
            .performTextInput("请说明本次回答的模型和推理档位。")
        // The Compose send action remains available above the IME. Avoid an
        // Espresso IME-idle wait, which can time out on vendor keyboards.
        composeRule.onNodeWithTag("YL-A-023-C-P03_009-01").performClick()
        waitForTagGone("YL-A-023-C-P03_009-01", 10_000)
        waitForTag("YL-A-023-C-P03_009-01", 110_000)
        waitForAnswerProvenance(20_000)
        val messageConversation = awaitConversation(gateway, session, "new W02 message conversation") {
            it.id !in conversationsBefore
        }
        val assistantMessage = awaitAssistantMessage(gateway, session, messageConversation.id)
        val source = requireNotNull(gateway.messageRunMetadata(session.bearer, assistantMessage.id)) {
            "Assistant answer has no source metadata"
        }
        assertEquals(model.id, source.modelId)
        assertEquals("auto", source.reasoningProfile)
        capture("P04-W02-PER-MESSAGE-PROVENANCE")
        composeRule.onNodeWithTag("YL-A-025-C-P03_016-01")
            .performScrollToNode(hasTestTag("p03-message-change-model"))
        composeRule.onNodeWithTag("p03-message-action-row")
            .performScrollTo()
            .performTouchInput { swipeLeft() }
        capture("P04-W02-MESSAGE-ACTIONS")

        // P04-010: send a real message and verify both API metadata and the
        // rendered source label. P04-008 and P04-012 then update the generated
        // conversation through its UI and create an answer branch.
        // settings, then create and display a branch from its answer history.
        Espresso.pressBack()
        waitForTag("CO-P03-001-HOME-SEND", 20_000)
        composeRule.onNodeWithTag("p03-open-account").performClick()
        waitForTag("p04-open-workbench", 20_000)
        composeRule.onNodeWithTag("p04-open-workbench").performClick()
        waitForTag("p04-open-conversation-settings-${messageConversation.id}", 30_000)
        composeRule.onNodeWithTag("p04-open-conversation-settings-${messageConversation.id}").performScrollTo().performClick()
        waitForTag("p04-conversation-model-row", 20_000)
        composeRule.onNodeWithTag("p04-conversation-model-row").performClick()
        waitForTag("p04-conversation-model-${model.id}", 20_000)
        composeRule.onNodeWithTag("p04-conversation-model-${model.id}").performScrollTo().performClick()
        capture("P04-W02-CONVERSATION-DEFAULT")
        composeRule.onNodeWithTag("p04-save-conversation-settings").performScrollTo().performClick()
        await("conversation preference readback") {
            gateway.listConversations(session.bearer).first.firstOrNull { it.id == messageConversation.id }
                ?.let { it.defaultModelId == model.id && it.defaultReasoningProfile == "auto" } == true
        }
        waitForTag("p04-open-branches-${messageConversation.id}", 30_000)
        val branchesBefore = gateway.conversationBranches(session.bearer, messageConversation.id).map { it.id }.toSet()
        composeRule.onNodeWithTag("p04-open-branches-${messageConversation.id}").performScrollTo().performClick()
        waitForTag("p04-create-branch", 20_000)
        composeRule.onNodeWithTag("p04-create-branch").performScrollTo().performClick()
        val createdBranch = awaitBranch(gateway, session, messageConversation.id, branchesBefore) {
            branchErrorText()
        }
        assertEquals(assistantMessage.id, createdBranch.forkedFromMessageId)
        waitForTag("p04-branch-${createdBranch.id}", 20_000)
        assertTrue(
            "Branch must be created from the answer conversation",
            gateway.conversationBranches(session.bearer, messageConversation.id).any { it.id == createdBranch.id },
        )
        assertTrue(
            "Branch list cannot be empty after branch creation",
            composeRule.onAllNodes(branchMatcher).fetchSemanticsNodes().isNotEmpty(),
        )
        /* The branch card is now the live response to the button action. */
        await("answer branch list refresh") {
            composeRule.onAllNodesWithTag("p04-branch-${createdBranch.id}").fetchSemanticsNodes().isNotEmpty()
        }
        capture("P04-W02-ANSWER-BRANCH")

        assertTrue(
            "W02 flow displayed an application error",
            composeRule.onAllNodesWithTag("p04-inline-error").fetchSemanticsNodes().isEmpty(),
        )
    }

    private suspend fun ensureSession(gateway: HttpIdentityGateway, context: android.content.Context): AuthSession {
        val store = SessionStore(context)
        store.load()?.let { existing ->
            runCatching { gateway.models(existing.bearer) }
                .onSuccess { return existing }
                .onFailure { error ->
                    val invalid = (error as? ApiException)?.let { it.status == 401 || it.code.contains("session", ignoreCase = true) } == true
                    if (!invalid) throw error
                    store.save(null)
                }
        }
        val email = "p04.w02.${System.currentTimeMillis()}@example.com"
        val registration = gateway.startRegistration(email)
        val session = gateway.finishRegistrationSession(
            registration,
            requireNotNull(registration.debugCode),
            "P04W02Physical${System.nanoTime()}a1",
        )
        gateway.initializeWorkspace(session)
        store.save(session)
        return session
    }

    private suspend fun awaitConversation(
        gateway: HttpIdentityGateway,
        session: AuthSession,
        description: String,
        predicate: (Conversation) -> Boolean,
    ): Conversation {
        repeat(75) {
            gateway.listConversations(session.bearer).first.firstOrNull(predicate)?.let { return it }
            delay(400)
        }
        error("Timed out waiting for $description")
    }

    private suspend fun awaitBranch(
        gateway: HttpIdentityGateway,
        session: AuthSession,
        conversationId: String,
        previousIds: Set<String>,
        errorText: () -> String?,
    ): cc.orbexa.ylven.identity.ConversationBranch {
        repeat(75) {
            gateway.conversationBranches(session.bearer, conversationId)
                .firstOrNull { it.id !in previousIds }
                ?.let { return it }
            errorText()?.let { error("Answer branch creation failed in the installed UI: $it") }
            delay(400)
        }
        error("Timed out waiting for answer branch creation")
    }

    private fun branchErrorText(): String? = composeRule
        .onAllNodesWithTag("p04-inline-error")
        .fetchSemanticsNodes()
        .flatMap { node -> node.config.getOrNull(SemanticsProperties.Text).orEmpty() }
        .joinToString(separator = "") { it.text }
        .trim()
        .takeIf(String::isNotBlank)

    private suspend fun awaitAssistantMessage(
        gateway: HttpIdentityGateway,
        session: AuthSession,
        conversationId: String,
    ): cc.orbexa.ylven.identity.MessageRecord {
        repeat(75) {
            gateway.conversationMessages(session.bearer, conversationId).lastOrNull { it.role == "assistant" }?.let { return it }
            delay(400)
        }
        error("Timed out waiting for the completed assistant message")
    }

    private suspend fun await(description: String, condition: suspend () -> Boolean) {
        repeat(75) {
            if (condition()) return
            delay(400)
        }
        error("Timed out waiting for $description")
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

    private fun waitForAnswerProvenance(timeout: Long) {
        composeRule.waitUntil(timeoutMillis = timeout) {
            composeRule.onAllNodes(answerProvenanceMatcher).fetchSemanticsNodes().any { node ->
                node.config.getOrNull(SemanticsProperties.Text)?.any {
                    it.text.startsWith("回答来源：") && it.text.contains(" · ")
                } == true
            }
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
                subject = "P04-W02 live staging screenshot",
            )
        } finally {
            bitmap.recycle()
        }
    }

    private companion object {
        val answerProvenanceMatcher = SemanticsMatcher("P04 answer provenance") { node ->
            node.config.getOrNull(SemanticsProperties.TestTag)?.startsWith("p04-answer-source-") == true
        }
        val branchMatcher = SemanticsMatcher("P04 branch") { node ->
            node.config.getOrNull(SemanticsProperties.TestTag)?.startsWith("p04-branch-") == true
        }
    }
}

private fun <T> SemanticsConfiguration.getOrNull(key: androidx.compose.ui.semantics.SemanticsPropertyKey<T>): T? =
    if (contains(key)) get(key) else null
