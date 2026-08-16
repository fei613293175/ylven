package cc.orbexa.ylven

import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.compose.ui.test.onAllNodesWithTag
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollTo
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

/** Exercises P04-W01 through the installed APK and the real staging gateway. */
class P04W01LiveStagingFlowTest {
    @get:Rule
    val composeRule = createEmptyComposeRule()

    @Test
    fun authenticatedUserLoadsSelectsAndUsesARealModelProfile() = runBlocking {
        val instrumentation = InstrumentationRegistry.getInstrumentation()
        val context = instrumentation.targetContext
        val gateway = HttpIdentityGateway(deviceId = "p04-w01-physical-" + System.currentTimeMillis())
        ensureSession(gateway, context)

        P03ScreenshotStorage.resetDirectory(
            context = context,
            legacyDirectory = "p04-live-staging",
            scopedDownloadDirectory = "ylven-p04-live-staging",
        )
        launchP03TargetActivity()
        waitForTag("YL-A-018-C-P03_001-01", 30_000)

        composeRule.onNodeWithTag("CO-P03-001-HOME-SEND").performClick()
        waitForTag("YL-A-023-C-P03_013-01", 20_000)
        composeRule.onNodeWithTag("CO-P03-001-CHAT-MODEL").performClick()
        waitForTag("p03-model-ylven-default", 20_000)
        capture("P04-MODEL-SELECTOR")
        composeRule.onNodeWithTag("p03-model-ylven-default").performScrollTo().performClick()

        composeRule.onNodeWithTag("CO-P03-001-CHAT-REASONING").performClick()
        when (waitForResponseModeTerminalState(20_000)) {
            "p03-response-mode-auto" -> {
                capture("P04-REASONING-PROFILE")
                composeRule.onNodeWithTag("p03-response-mode-auto").performClick()
            }
            "p03-response-mode-degraded" -> {
                // The catalog currently exposes only automatic mode for this model.
                // This is a valid server-backed state, not a selector load failure.
                capture("P04-REASONING-PROFILE")
                Espresso.pressBack()
                waitForTagGone("YL-A-034-root", 10_000)
            }
            "p03-response-mode-error" -> error(
                "Staging could not load the selected model's reasoning profiles",
            )
            else -> error("Unexpected response mode selector state")
        }

        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01")
            .performTextInput("请说明模型目录和推理档位应如何协同工作。")
        // Keep IME focus while sending. The production control remains available above the IME,
        // and physical P03 flows exercise this same interaction without a system-animation wait.
        waitForTag("YL-A-023-C-P03_009-01", 10_000)
        composeRule.onNodeWithTag("YL-A-023-C-P03_009-01").performClick()
        waitForTagGone("YL-A-023-C-P03_009-01", 10_000)
        waitForTag("YL-A-023-C-P03_009-01", 110_000)
        waitForAnswerProvenance(20_000)
        assertTrue(
            "The W01 flow displayed an application error",
            composeRule.onAllNodesWithTag("p04-inline-error").fetchSemanticsNodes().isEmpty(),
        )
        Espresso.pressBack()
        waitForTag("YL-A-018-C-P03_001-01", 20_000)
    }

    private suspend fun ensureSession(gateway: HttpIdentityGateway, context: android.content.Context): AuthSession {
        SessionStore(context).load()?.let { return it }
        val email = "p04.w01." + System.currentTimeMillis() + "@example.com"
        val registration = gateway.startRegistration(email)
        val session = gateway.finishRegistrationSession(
            registration,
            requireNotNull(registration.debugCode),
            "P04W01Physical" + System.nanoTime() + "a1",
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

    private fun waitForTagGone(tag: String, timeout: Long) {
        composeRule.waitUntil(timeoutMillis = timeout) {
            composeRule.onAllNodesWithTag(tag).fetchSemanticsNodes().isEmpty()
        }
    }

    private fun waitForResponseModeTerminalState(timeout: Long): String {
        val tags = listOf(
            "p03-response-mode-auto",
            "p03-response-mode-degraded",
            "p03-response-mode-error",
        )
        composeRule.waitUntil(timeoutMillis = timeout) {
            tags.any { tag -> composeRule.onAllNodesWithTag(tag).fetchSemanticsNodes().isNotEmpty() }
        }
        return tags.first { tag ->
            composeRule.onAllNodesWithTag(tag).fetchSemanticsNodes().isNotEmpty()
        }
    }

    private fun waitForAnswerProvenance(timeout: Long) {
        composeRule.waitUntil(timeoutMillis = timeout) {
            composeRule.onAllNodes(answerProvenanceMatcher).fetchSemanticsNodes().any { node ->
                node.config.contains(SemanticsProperties.Text) && node.config[SemanticsProperties.Text]
                    .any { it.text.startsWith("回答来源：") && it.text.contains(" · ") }
            }
        }
    }

    private fun capture(name: String) {
        val instrumentation = InstrumentationRegistry.getInstrumentation()
        composeRule.waitForIdle()
        instrumentation.waitForIdleSync()
        val bitmap = requireNotNull(instrumentation.uiAutomation.takeScreenshot()) {
            "Could not capture " + name
        }
        try {
            P03ScreenshotStorage.writePng(
                context = instrumentation.targetContext,
                bitmap = bitmap,
                legacyDirectory = "p04-live-staging",
                scopedDownloadDirectory = "ylven-p04-live-staging",
                fileName = name + ".png",
                subject = "P04-W01 live staging screenshot",
            )
        } finally {
            bitmap.recycle()
        }
    }

    private companion object {
        val answerProvenanceMatcher = SemanticsMatcher("P04 answer provenance") { node ->
            node.config.contains(SemanticsProperties.TestTag) &&
                node.config[SemanticsProperties.TestTag].startsWith("p04-answer-source-")
        }
    }
}
