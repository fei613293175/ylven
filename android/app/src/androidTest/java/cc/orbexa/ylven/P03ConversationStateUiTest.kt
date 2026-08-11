package cc.orbexa.ylven

import android.graphics.Bitmap
import androidx.activity.compose.setContent
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.compose.ui.test.captureToImage
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.test.platform.app.InstrumentationRegistry
import cc.orbexa.ylven.ui.YlvenAcceptanceState
import cc.orbexa.ylven.ui.theme.YlvenTheme
import org.junit.Rule
import org.junit.Test

/** Captures every P03 Android contract state from the running Compose surface. */
class P03ConversationStateUiTest {
    @get:Rule
    val composeRule = createEmptyComposeRule()

    @Test
    fun everyP03AndroidStateIsRuntimeCaptured() {
        check(P03_STATE_IDS.size == 89) { "P03 state catalog changed; update this acceptance test." }
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        P03ScreenshotStorage.resetDirectory(
            context = context,
            legacyDirectory = "screenshots",
            scopedDownloadDirectory = "ylven-p03",
        )
        val activeState = mutableStateOf(P03_STATE_IDS.first())
        val activity = launchP03TargetActivity()
        composeRule.runOnUiThread {
            activity.setContent {
                YlvenTheme(darkTheme = false) {
                    key(activeState.value) { YlvenAcceptanceState(activeState.value) }
                }
            }
        }
        P03_STATE_IDS.forEach { stateId ->
            composeRule.runOnIdle { activeState.value = stateId }
            composeRule.waitForIdle()
            composeRule.onNodeWithTag("acceptance-state-$stateId").assertExists()
            capture(stateId)
        }
    }

    private fun capture(name: String) {
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        composeRule.waitForIdle()
        val bitmap = composeRule.onNodeWithTag("acceptance-state-$name")
            .captureToImage()
            .asAndroidBitmap()
        check(bitmap.width > 0 && bitmap.height > 0) {
            "P03 runtime screenshot is empty for $name"
        }
        try {
            P03ScreenshotStorage.writePng(
                context = context,
                bitmap = bitmap,
                legacyDirectory = "screenshots",
                scopedDownloadDirectory = "ylven-p03",
                fileName = "$name.png",
                subject = "P03 acceptance screenshot",
            )
        } finally {
            bitmap.recycle()
        }
    }
}

private val P03_STATE_IDS = listOf(
    "YL-A-018-S01_LOADING", "YL-A-018-S02_POPULATED", "YL-A-018-S03_EMPTY", "YL-A-018-S04_REFRESHING", "YL-A-018-S05_OFFLINE_CACHE", "YL-A-018-S06_NETWORK_ERROR", "YL-A-018-S07_SERVICE_DEGRADED",
    "YL-A-020-S01_LOADING", "YL-A-020-S02_POPULATED", "YL-A-020-S03_EMPTY", "YL-A-020-S04_REFRESHING", "YL-A-020-S05_FILTER_ACTIVE", "YL-A-020-S06_OFFLINE_CACHE", "YL-A-020-S07_NETWORK_ERROR", "YL-A-020-S08_SERVER_ERROR", "YL-A-020-S09_PERMISSION_DENIED",
    "YL-A-021-S01_LOADING", "YL-A-021-S02_POPULATED", "YL-A-021-S03_EMPTY", "YL-A-021-S04_REFRESHING", "YL-A-021-S05_FILTER_ACTIVE", "YL-A-021-S06_OFFLINE_CACHE", "YL-A-021-S07_NETWORK_ERROR", "YL-A-021-S08_SERVER_ERROR", "YL-A-021-S09_PERMISSION_DENIED",
    "YL-A-022-S01_DEFAULT", "YL-A-022-S02_SUBMITTING", "YL-A-022-S03_SUCCESS", "YL-A-022-S04_SAVE_ERROR", "YL-A-022-S05_DISABLED",
    "YL-A-023-S01_DEFAULT", "YL-A-023-S02_INPUT_FOCUSED", "YL-A-023-S03_UPLOADING", "YL-A-023-S04_CONNECTING", "YL-A-023-S05_STREAMING", "YL-A-023-S06_TOOL_RUNNING", "YL-A-023-S07_COMPLETED", "YL-A-023-S08_STOPPED", "YL-A-023-S09_RECONNECTING", "YL-A-023-S10_OFFLINE", "YL-A-023-S11_RATE_LIMITED", "YL-A-023-S12_PROVIDER_ERROR", "YL-A-023-S13_CONTENT_BLOCKED",
    "YL-A-024-S01_DEFAULT", "YL-A-024-S02_INPUT_FOCUSED", "YL-A-024-S03_UPLOADING", "YL-A-024-S04_DISABLED", "YL-A-024-S05_OFFLINE", "YL-A-024-S06_TOOL_TRAY_OPEN",
    "YL-A-025-S01_OFFLINE_CACHE", "YL-A-025-S02_REFRESHING", "YL-A-025-S03_NETWORK_ERROR",
    "YL-A-026-S01_CONNECTING", "YL-A-026-S02_STREAMING", "YL-A-026-S03_TOOL_RUNNING", "YL-A-026-S04_COMPLETED", "YL-A-026-S05_STOPPED", "YL-A-026-S06_PROVIDER_ERROR", "YL-A-026-S07_CONTENT_BLOCKED",
    "YL-A-027-S01_DEFAULT", "YL-A-027-S02_COMPLETED", "YL-A-027-S03_CONTENT_BLOCKED",
    "YL-A-028-S01_DEFAULT", "YL-A-028-S02_COMPLETED", "YL-A-028-S03_OFFLINE",
    "YL-A-029-S01_DEFAULT", "YL-A-029-S02_COMPLETED", "YL-A-029-S03_NOT_FOUND", "YL-A-029-S04_OFFLINE",
    "YL-A-030-S01_DEFAULT", "YL-A-030-S02_DISABLED", "YL-A-030-S03_SUCCESS",
    "YL-A-032-S01_DEFAULT", "YL-A-032-S02_INPUT_FOCUSED", "YL-A-032-S03_UPLOADING", "YL-A-032-S04_DISABLED", "YL-A-032-S05_OFFLINE",
    "YL-A-033-S01_POPULATED", "YL-A-033-S02_FILTER_ACTIVE", "YL-A-033-S03_EMPTY", "YL-A-033-S04_DISABLED", "YL-A-033-S05_SERVICE_DEGRADED", "YL-A-033-S06_SERVER_ERROR",
    "YL-A-034-S01_POPULATED", "YL-A-034-S02_FILTER_ACTIVE", "YL-A-034-S03_EMPTY", "YL-A-034-S04_DISABLED", "YL-A-034-S05_SERVICE_DEGRADED", "YL-A-034-S06_SERVER_ERROR",
)
