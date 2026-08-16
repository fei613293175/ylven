package cc.orbexa.ylven

import androidx.activity.compose.setContent
import androidx.compose.ui.semantics.SemanticsConfiguration
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.compose.ui.test.onAllNodes
import androidx.compose.ui.test.onAllNodesWithTag
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.test.platform.app.InstrumentationRegistry
import cc.orbexa.ylven.identity.ApiException
import cc.orbexa.ylven.identity.AuthSession
import cc.orbexa.ylven.identity.Conversation
import cc.orbexa.ylven.identity.DeviceSession
import cc.orbexa.ylven.identity.IdentityGateway
import cc.orbexa.ylven.identity.MessageRecord
import cc.orbexa.ylven.identity.MessageRun
import cc.orbexa.ylven.identity.ModelAvailability
import cc.orbexa.ylven.identity.ModelOption
import cc.orbexa.ylven.identity.OtpChallenge
import cc.orbexa.ylven.identity.RunEvent
import cc.orbexa.ylven.ui.P03ChatPage
import cc.orbexa.ylven.ui.theme.YlvenTheme
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

class P04FallbackUiTest {
    @get:Rule
    val composeRule = createEmptyComposeRule()

    @Test
    fun unavailableModelLetsUserChooseFallbackAndContinuesTheOriginalPrompt() {
        val gateway = UnavailableModelGateway()
        val activity = launchP03TargetActivity()
        composeRule.runOnUiThread {
            activity.setContent {
                YlvenTheme(darkTheme = false) {
                    P03ChatPage(
                        gateway = gateway,
                        session = gateway.session,
                        conversation = Conversation(
                            id = "fallback-conversation",
                            title = "回退测试",
                            status = "active",
                            updatedAt = "",
                            defaultModelId = "primary",
                            defaultReasoningProfile = "auto",
                        ),
                        onBack = {},
                        onOpenConversation = {},
                    )
                }
            }
        }

        waitForTag("YL-A-023-C-P03_013-01")
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").performTextInput("请继续这条请求")
        composeRule.onNodeWithTag("YL-A-023-C-P03_009-01").performClick()
        waitForTag("p04-model-fallback-dialog")
        capture("P04-MODEL-FALLBACK")

        composeRule.onNodeWithTag("p04-fallback-model-secondary").performClick()
        waitForText("回答来源：替代模型 · 自动")

        assertEquals(listOf("primary", "secondary"), gateway.requestedModels)
        assertTrue(
            "Fallback completion must retain the original prompt",
            composeRule.onAllNodes(SemanticsMatcher("original fallback prompt") { node ->
                node.config.getOrNull(SemanticsProperties.Text)?.any { it.text == "请继续这条请求" } == true
            }).fetchSemanticsNodes().isNotEmpty(),
        )
    }

    private fun waitForTag(tag: String) {
        composeRule.waitUntil(timeoutMillis = 20_000) {
            composeRule.onAllNodesWithTag(tag).fetchSemanticsNodes().isNotEmpty()
        }
    }

    private fun waitForText(text: String) {
        composeRule.waitUntil(timeoutMillis = 20_000) {
            composeRule.onAllNodes(SemanticsMatcher("text $text") { node ->
                node.config.getOrNull(SemanticsProperties.Text)?.any { it.text == text } == true
            }).fetchSemanticsNodes().isNotEmpty()
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
                subject = "P04 fallback screenshot",
            )
        } finally {
            bitmap.recycle()
        }
    }
}

private class UnavailableModelGateway : IdentityGateway {
    val session = AuthSession("p04@example.com", "session", "bearer", "renewal")
    val requestedModels = mutableListOf<String>()
    private val primary = ModelOption("primary", "主模型", reasoningProfiles = listOf("auto"))
    private val secondary = ModelOption("secondary", "替代模型", reasoningProfiles = listOf("auto"))
    private val completedRun = MessageRun(
        id = "fallback-run",
        conversationId = "fallback-conversation",
        status = "completed",
        cursor = 1,
        assistantMessageId = "fallback-answer",
        modelId = "secondary",
        reasoningProfile = "auto",
    )

    override suspend fun startRegistration(email: String): OtpChallenge = error("unused")
    override suspend fun finishRegistration(challenge: OtpChallenge, code: String, password: String) = Unit
    override suspend fun startLogin(email: String): OtpChallenge = error("unused")
    override suspend fun finishLogin(challenge: OtpChallenge, code: String): AuthSession = session
    override suspend fun refresh(session: AuthSession): AuthSession = session
    override suspend fun devices(bearer: String): List<DeviceSession> = emptyList()
    override suspend fun revokeDevice(bearer: String, sessionId: String) = Unit
    override suspend fun logout(bearer: String, allDevices: Boolean) = Unit
    override suspend fun models(bearer: String): List<ModelOption> = listOf(primary, secondary)
    override suspend fun conversationMessages(bearer: String, conversationId: String): List<MessageRecord> = emptyList()

    override suspend fun sendMessageIdempotentWithProfile(
        bearer: String,
        conversationId: String,
        body: String,
        model: String,
        reasoningProfile: String,
        idempotencyKey: String,
    ): MessageRun {
        requestedModels += model
        if (model == "primary") throw ApiException(503, "model_unavailable", "unavailable")
        return completedRun
    }

    override suspend fun modelAvailability(bearer: String, modelId: String): ModelAvailability =
        ModelAvailability(modelId = modelId, status = "unavailable", fallbackModels = listOf(secondary))

    override suspend fun runEvents(bearer: String, runId: String, after: Long): Pair<MessageRun, List<RunEvent>> =
        Pair(completedRun, emptyList())

    override suspend fun runStatus(bearer: String, runId: String): Pair<MessageRun, List<MessageRecord>> =
        Pair(completedRun, listOf(
            MessageRecord("fallback-user", "fallback-conversation", "user", "请继续这条请求", ""),
            MessageRecord("fallback-answer", "fallback-conversation", "assistant", "已由替代模型完成", ""),
        ))

    override suspend fun messageRunMetadata(bearer: String, messageId: String): MessageRun? = completedRun
}

private fun <T> SemanticsConfiguration.getOrNull(key: androidx.compose.ui.semantics.SemanticsPropertyKey<T>): T? =
    if (contains(key)) get(key) else null
