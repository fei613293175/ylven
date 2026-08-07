package cc.orbexa.ylven

import android.content.ContentValues
import android.graphics.Bitmap
import android.os.Environment
import android.provider.MediaStore
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.compose.ui.test.assertExists
import androidx.compose.ui.test.assertTextContains
import androidx.compose.ui.test.captureToImage
import androidx.compose.ui.test.fetchSemanticsNodes
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onAllNodesWithTag
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.onRoot
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollTo
import androidx.compose.ui.test.performTextClearance
import androidx.compose.ui.test.performTextInput
import androidx.test.espresso.Espresso
import androidx.test.platform.app.InstrumentationRegistry
import cc.orbexa.ylven.identity.AuthSession
import cc.orbexa.ylven.identity.Conversation
import cc.orbexa.ylven.identity.DeviceSession
import cc.orbexa.ylven.identity.HomeSnapshot
import cc.orbexa.ylven.identity.IdentityGateway
import cc.orbexa.ylven.identity.MessageCitation
import cc.orbexa.ylven.identity.MessageRecord
import cc.orbexa.ylven.identity.MessageRun
import cc.orbexa.ylven.identity.OtpChallenge
import cc.orbexa.ylven.identity.RunEvent
import cc.orbexa.ylven.ui.YlvenApp
import cc.orbexa.ylven.ui.theme.YlvenTheme
import java.io.IOException
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
    val composeRule = createComposeRule()

    @Test
    fun conversationJourneyUsesEveryP03ControlOnDevice() {
        val gateway = FlowGateway()
        val session = AuthSession("owner@example.invalid", "session", "access", "refresh", "physical-device")
        composeRule.mainClock.autoAdvance = true
        composeRule.setContent {
            YlvenTheme(darkTheme = false) {
                YlvenApp(gateway = gateway, initialSession = session)
            }
        }

        waitForTag("YL-A-018-C-P03_001-01")
        captureProductionPage("YL-A-018-PRODUCTION")
        composeRule.onNodeWithTag("p03-refresh-home").performClick()
        composeRule.onNodeWithTag("YL-A-019-C-P03_002-01").performClick()
        waitForTag("p03-conversation-created")
        captureProductionPage("YL-A-019-PRODUCTION")
        composeRule.onNodeWithTag("YL-A-031-C-P03_028-01").performClick()
        waitForTag("p03-conversation-temporary")
        captureProductionPage("YL-A-031-PRODUCTION")

        composeRule.onNodeWithTag("p03-open-conversation-drawer").performClick()
        waitForTag("p03-conversation-drawer")
        captureProductionPage("YL-A-020-PRODUCTION")
        waitForTag("YL-A-020-C-P03_003-01")
        composeRule.onNodeWithTag("YL-A-020-C-P03_003-01").performClick()
        composeRule.onNodeWithTag("YL-A-021-C-P03_004-01").performTextInput("Alpha")
        waitForTag("p03-drawer-rename-alpha")
        captureProductionPage("YL-A-021-PRODUCTION")
        composeRule.onNodeWithTag("p03-drawer-rename-alpha").performClick()
        waitForTag("p03-rename-input")
        captureProductionPage("YL-A-022-PRODUCTION")
        composeRule.onNodeWithTag("p03-rename-input").performTextClearance()
        composeRule.onNodeWithTag("p03-rename-input").performTextInput("Alpha renamed")
        composeRule.onNodeWithTag("YL-A-022-C-P03_005-01").performClick()
        waitForCondition { "rename" in gateway.calls }
        composeRule.onAllNodesWithTag("YL-A-022-C-P03_006-01")[0].performClick()
        waitForCondition { "archive" in gateway.calls }
        composeRule.onAllNodesWithTag("YL-A-022-C-P03_007-01")[0].performClick()
        waitForCondition { "delete" in gateway.calls }
        composeRule.onNodeWithText("关闭").performClick()

        composeRule.onNodeWithTag("p03-conversation-alpha").performScrollTo().performClick()
        waitForTag("YL-A-023-C-P03_013-01")
        waitForTag("YL-A-032-C-P03_032-01")
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").assertTextContains("saved draft")
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").performTextClearance()
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").performTextInput("physical flow message")
        composeRule.onNodeWithTag("YL-A-023-C-P03_009-01").performClick()
        waitForTag("YL-A-023-C-P03_010-01")
        waitForTag("YL-A-023-C-P03_012-01")
        waitForTag("YL-A-024-C-P03_011-01")
        composeRule.onNodeWithTag("YL-A-024-C-P03_011-01").performClick()

        waitForTag("YL-A-026-C-P03_026-01")
        waitForTag("YL-A-025-C-P03_016-01")
        waitForTag("YL-A-023-C-P03_017-01")
        waitForTag("YL-A-026-C-P03_018-01")
        waitForTag("YL-A-027-C-P03_019-01")
        waitForTag("YL-A-028-C-P03_020-01")
        waitForTag("YL-A-029-C-P03_021-01")
        captureProductionPage("YL-A-023-PRODUCTION")
        composeRule.onNodeWithTag("YL-A-030-C-P03_022-01").performScrollTo().performClick()
        composeRule.onNodeWithTag("YL-A-030-C-P03_023-01").performScrollTo().performClick()
        composeRule.onAllNodesWithTag("YL-A-030-C-P03_025-01")[0].performScrollTo().performClick()
        composeRule.onAllNodesWithTag("YL-A-030-C-P03_025-01")[1].performScrollTo().performClick()
        composeRule.onNodeWithTag("YL-A-030-C-P03_024-01").performScrollTo().performClick()
        composeRule.onNodeWithTag("YL-A-030-C-P03_030-01").performScrollTo().performClick()
        composeRule.onNodeWithTag("p03-copy-code").performScrollTo().performClick()
        composeRule.onNodeWithTag("YL-A-022-C-P03_029-01").performClick()

        gateway.failNextSend = true
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").performTextInput("retry me")
        composeRule.onNodeWithTag("YL-A-023-C-P03_009-01").performClick()
        waitForTag("YL-A-023-C-P03_027-01")
        composeRule.onNodeWithTag("YL-A-023-C-P03_027-01").performScrollTo().performClick()
        Espresso.pressBack()
        waitForTag("YL-A-018-C-P03_001-01")

        composeRule.onNodeWithTag("p03-conversation-alpha").performScrollTo().performClick()
        waitForTag("YL-A-032-C-P03_032-01")
        composeRule.onNodeWithTag("YL-A-032-C-P03_032-01").assertTextContains("retry me")

        composeRule.onNodeWithTag("YL-A-032-C-P03_031-01").performClick()
        Espresso.pressBack()
        composeRule.waitForIdle()
        if (composeRule.onAllNodesWithTag("YL-A-018-C-P03_001-01").fetchSemanticsNodes().isEmpty()) {
            Espresso.pressBack()
        }
        waitForTag("YL-A-018-C-P03_001-01")

        val requiredCalls = setOf(
            "home", "create", "temporary", "list-page-1", "list-page-2", "search",
            "rename", "archive", "delete", "send", "events-retry", "cancel",
            "export-message", "export-conversation", "feedback-up", "feedback-down",
            "regenerate", "speak", "load-draft", "save-draft",
        )
        assertTrue("Missing gateway calls: ${requiredCalls - gateway.calls}", gateway.calls.containsAll(requiredCalls))
    }

    private fun waitForTag(tag: String) = waitForCondition {
        composeRule.onAllNodesWithTag(tag).fetchSemanticsNodes().isNotEmpty()
    }

    private fun waitForCondition(predicate: () -> Boolean) {
        composeRule.waitUntil(timeoutMillis = 8_000, condition = predicate)
    }

    private fun captureProductionPage(name: String) {
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        composeRule.waitForIdle()
        val bitmap = composeRule.onRoot().captureToImage().asAndroidBitmap()
        check(bitmap.width == 1080 && bitmap.height == 2400) {
            "P03 production screenshot has unexpected dimensions for $name: ${bitmap.width}x${bitmap.height}"
        }
        val resolver = context.contentResolver
        val relativePath = "${Environment.DIRECTORY_DOWNLOADS}/ylven-p03-production/"
        resolver.delete(
            MediaStore.Downloads.EXTERNAL_CONTENT_URI,
            "${MediaStore.MediaColumns.DISPLAY_NAME} = ? AND ${MediaStore.MediaColumns.RELATIVE_PATH} = ?",
            arrayOf("$name.png", relativePath),
        )
        val values = ContentValues().apply {
            put(MediaStore.MediaColumns.DISPLAY_NAME, "$name.png")
            put(MediaStore.MediaColumns.MIME_TYPE, "image/png")
            put(MediaStore.MediaColumns.RELATIVE_PATH, relativePath)
            put(MediaStore.MediaColumns.IS_PENDING, 1)
        }
        val uri = requireNotNull(resolver.insert(MediaStore.Downloads.EXTERNAL_CONTENT_URI, values)) {
            "Could not create P03 production screenshot media entry"
        }
        try {
            requireNotNull(resolver.openOutputStream(uri)).use { stream ->
                check(bitmap.compress(Bitmap.CompressFormat.PNG, 100, stream))
            }
            resolver.update(uri, ContentValues().apply {
                put(MediaStore.MediaColumns.IS_PENDING, 0)
            }, null, null)
        } catch (failure: Throwable) {
            resolver.delete(uri, null, null)
            throw failure
        } finally {
            bitmap.recycle()
        }
    }
}

private class FlowGateway : IdentityGateway {
    val calls = linkedSetOf<String>()
    var failNextSend = false
    private var failFirstEvent = true
    private var currentRunStatus = "streaming"
    private var draft = "saved draft"
    private val alpha = Conversation("alpha", "Alpha", "active", "2026-08-08T00:00:00Z")
    private val beta = Conversation("beta", "Beta", "active", "2026-08-08T00:00:00Z")

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
        return HomeSnapshot(listOf(alpha, beta), listOf("YLVEN Default"))
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
            Pair(listOf(Conversation("gamma", "Gamma", "active", "2026-08-08T00:00:00Z")), null)
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

    override suspend fun sendMessage(bearer: String, conversationId: String, body: String): MessageRun {
        calls += "send"
        if (failNextSend) {
            failNextSend = false
            throw IOException("controlled network failure")
        }
        currentRunStatus = "streaming"
        return run("run-1")
    }

    override suspend fun runEvents(bearer: String, runId: String, after: Long): Pair<MessageRun, List<RunEvent>> {
        if (failFirstEvent) {
            failFirstEvent = false
            calls += "events-retry"
            throw IOException("controlled SSE disconnect")
        }
        calls += "events"
        return Pair(run(runId), listOf(RunEvent(after + 1, "delta", "answer")))
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
        MessageRecord("user", alpha.id, "user", "physical flow message", "2026-08-08T00:00:01Z"),
        MessageRecord(
            "assistant",
            alpha.id,
            "assistant",
            "# Result\n```kotlin\nprintln(\"ok\")\n```\nA | B\n--- | ---\n1 | 2",
            "2026-08-08T00:00:02Z",
        ),
    )

    private fun session() = AuthSession("owner@example.invalid", "session", "access", "refresh")
}
