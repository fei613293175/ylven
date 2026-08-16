package cc.orbexa.ylven

import androidx.activity.compose.setContent
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.test.platform.app.InstrumentationRegistry
import cc.orbexa.ylven.ui.P04AcceptanceState
import cc.orbexa.ylven.ui.theme.YlvenTheme
import org.junit.Rule
import org.junit.Test

/** Captures every P04 Android contract state from a running APK surface. */
class P04StateUiTest {
    @get:Rule
    val composeRule = createEmptyComposeRule()

    @Test
    fun everyP04AndroidStateIsRuntimeCaptured() {
        check(P04_STATE_IDS.size == 77) { "P04 state catalog changed; update this acceptance test." }
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        P03ScreenshotStorage.resetDirectory(
            context = context,
            legacyDirectory = "screenshots-p04",
            scopedDownloadDirectory = "ylven-p04",
        )
        val activeState = mutableStateOf(P04_STATE_IDS.first())
        val activity = launchP03TargetActivity()
        composeRule.runOnUiThread {
            activity.setContent {
                YlvenTheme(darkTheme = false) {
                    key(activeState.value) { P04AcceptanceState(activeState.value) }
                }
            }
        }
        P04_STATE_IDS.forEach { stateId ->
            composeRule.runOnIdle { activeState.value = stateId }
            composeRule.waitForIdle()
            val rootTag = "p04-acceptance-state-$stateId"
            composeRule.onNodeWithTag(rootTag).assertExists()
            val bitmap = requireNotNull(InstrumentationRegistry.getInstrumentation().uiAutomation.takeScreenshot()) {
                "Could not capture $stateId"
            }
            try {
                check(bitmap.width > 0 && bitmap.height > 0) { "P04 screenshot $stateId is empty" }
                P03ScreenshotStorage.writePng(
                    context = context,
                    bitmap = bitmap,
                    legacyDirectory = "screenshots-p04",
                    scopedDownloadDirectory = "ylven-p04",
                    fileName = "$stateId.png",
                    subject = "P04 acceptance screenshot",
                )
            } finally {
                bitmap.recycle()
            }
        }
    }
}

private val P04_STATE_IDS = buildList {
    addAll(listOf("YL-A-030-S01_DEFAULT", "YL-A-030-S02_DISABLED", "YL-A-030-S03_SUCCESS"))
    addAll(listOf("POPULATED", "FILTER_ACTIVE", "EMPTY", "DISABLED", "SERVICE_DEGRADED", "SERVER_ERROR").mapIndexed { index, code -> "YL-A-033-S${(index + 1).toString().padStart(2, '0')}_$code" })
    addAll(listOf("POPULATED", "FILTER_ACTIVE", "EMPTY", "DISABLED", "SERVICE_DEGRADED", "SERVER_ERROR").mapIndexed { index, code -> "YL-A-034-S${(index + 1).toString().padStart(2, '0')}_$code" })
    addAll(listOf("LOADING", "POPULATED", "EDIT_MODE", "DIRTY", "SUBMITTING", "SAVE_SUCCESS", "SAVE_ERROR", "OFFLINE").mapIndexed { index, code -> "YL-A-035-S${(index + 1).toString().padStart(2, '0')}_$code" })
    addAll(listOf("LOADING", "POPULATED", "EDIT_MODE", "DIRTY", "SUBMITTING", "SAVE_SUCCESS", "SAVE_ERROR", "OFFLINE").mapIndexed { index, code -> "YL-A-036-S${(index + 1).toString().padStart(2, '0')}_$code" })
    addAll(listOf("DEFAULT", "SERVICE_DEGRADED", "DISABLED").mapIndexed { index, code -> "YL-A-037-S${(index + 1).toString().padStart(2, '0')}_$code" })
    addAll(listOf("DEFAULT", "SERVICE_DEGRADED", "DISABLED").mapIndexed { index, code -> "YL-A-038-S${(index + 1).toString().padStart(2, '0')}_$code" })
    addAll(listOf("LOADING", "POPULATED", "EMPTY", "REFRESHING", "FILTER_ACTIVE", "OFFLINE_CACHE", "NETWORK_ERROR", "SERVER_ERROR", "PERMISSION_DENIED").mapIndexed { index, code -> "YL-A-039-S${(index + 1).toString().padStart(2, '0')}_$code" })
    addAll(listOf("DEFAULT", "INPUT_FOCUSED", "VALIDATION_ERROR", "PREPARING", "RUNNING", "SUCCESS", "FAILED", "OFFLINE").mapIndexed { index, code -> "YL-A-040-S${(index + 1).toString().padStart(2, '0')}_$code" })
    addAll(listOf("LOADING", "POPULATED", "NOT_FOUND", "EDIT_MODE", "DIRTY", "SAVE_SUCCESS", "SAVE_ERROR", "OFFLINE", "PERMISSION_DENIED").mapIndexed { index, code -> "YL-A-041-S${(index + 1).toString().padStart(2, '0')}_$code" })
    addAll(listOf("DEFAULT", "SUBMITTING", "SUCCESS", "SAVE_ERROR", "DISABLED").mapIndexed { index, code -> "YL-A-042-S${(index + 1).toString().padStart(2, '0')}_$code" })
    addAll(listOf("LOADING", "POPULATED", "EMPTY", "REFRESHING", "FILTER_ACTIVE", "OFFLINE_CACHE", "NETWORK_ERROR", "SERVER_ERROR", "PERMISSION_DENIED").mapIndexed { index, code -> "YL-A-043-S${(index + 1).toString().padStart(2, '0')}_$code" })
}
