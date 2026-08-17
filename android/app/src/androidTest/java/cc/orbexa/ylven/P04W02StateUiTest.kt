package cc.orbexa.ylven

import androidx.activity.compose.setContent
import android.os.SystemClock
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.test.platform.app.InstrumentationRegistry
import cc.orbexa.ylven.ui.YlvenAcceptanceState
import cc.orbexa.ylven.ui.theme.YlvenTheme
import org.junit.Rule
import org.junit.Test

/** Captures exactly the Android contract states owned by P04-W02. */
class P04W02StateUiTest {
    @get:Rule
    val composeRule = createEmptyComposeRule()

    @Test
    fun everyP04W02AndroidStateIsRuntimeCaptured() {
        check(P04_W02_STATE_IDS.size == 25) { "P04-W02 state catalog changed; update this acceptance test." }
        val instrumentation = InstrumentationRegistry.getInstrumentation()
        val context = instrumentation.targetContext
        P03ScreenshotStorage.resetDirectory(
            context = context,
            legacyDirectory = "screenshots-p04",
            scopedDownloadDirectory = "ylven-p04",
        )
        val activeState = mutableStateOf(P04_W02_STATE_IDS.first())
        val activity = launchP03TargetActivity()
        composeRule.runOnUiThread {
            activity.setContent {
                YlvenTheme(darkTheme = false) {
                    key(activeState.value) { YlvenAcceptanceState(activeState.value) }
                }
            }
        }

        P04_W02_STATE_IDS.forEach { stateId ->
            composeRule.runOnIdle { activeState.value = stateId }
            composeRule.waitForIdle()
            composeRule.onNodeWithTag("acceptance-state-$stateId").assertExists()
            // The target activity's splash/window transition is a separate
            // layer from Compose; wait for the settled physical display
            // before taking the screenshot, as the P03 state runner does.
            SystemClock.sleep(350)
            instrumentation.waitForIdleSync()
            val bitmap = requireNotNull(instrumentation.uiAutomation.takeScreenshot()) {
                "Could not capture P04-W02 state $stateId"
            }
            try {
                check(bitmap.width > 0 && bitmap.height > 0) { "P04-W02 screenshot $stateId is empty" }
                P03ScreenshotStorage.writePng(
                    context = context,
                    bitmap = bitmap,
                    legacyDirectory = "screenshots-p04",
                    scopedDownloadDirectory = "ylven-p04",
                    fileName = "$stateId.png",
                    subject = "P04-W02 acceptance screenshot",
                )
            } finally {
                bitmap.recycle()
            }
        }
    }
}

private val P04_W02_STATE_IDS = buildList {
    addAll(listOf("YL-A-030-S01_DEFAULT", "YL-A-030-S02_DISABLED", "YL-A-030-S03_SUCCESS"))
    addAll(listOf("LOADING", "POPULATED", "EDIT_MODE", "DIRTY", "SUBMITTING", "SAVE_SUCCESS", "SAVE_ERROR", "OFFLINE").mapIndexed { index, code ->
        "YL-A-035-S${(index + 1).toString().padStart(2, '0')}_$code"
    })
    addAll(listOf("LOADING", "POPULATED", "EDIT_MODE", "DIRTY", "SUBMITTING", "SAVE_SUCCESS", "SAVE_ERROR", "OFFLINE").mapIndexed { index, code ->
        "YL-A-036-S${(index + 1).toString().padStart(2, '0')}_$code"
    })
    addAll(listOf("DEFAULT", "SERVICE_DEGRADED", "DISABLED").mapIndexed { index, code ->
        "YL-A-037-S${(index + 1).toString().padStart(2, '0')}_$code"
    })
    addAll(listOf("DEFAULT", "SERVICE_DEGRADED", "DISABLED").mapIndexed { index, code ->
        "YL-A-038-S${(index + 1).toString().padStart(2, '0')}_$code"
    })
}
