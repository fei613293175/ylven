package cc.orbexa.ylven

import android.content.ContentValues
import android.graphics.Bitmap
import android.os.Build
import android.os.Environment
import android.provider.MediaStore
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.compose.ui.test.captureToImage
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onRoot
import androidx.test.platform.app.InstrumentationRegistry
import androidx.compose.runtime.mutableStateOf
import cc.orbexa.ylven.ui.YlvenAcceptanceState
import cc.orbexa.ylven.ui.theme.YlvenTheme
import java.io.File
import java.io.FileOutputStream
import org.junit.Rule
import org.junit.Test

/** Captures every P02 Android state from the running Compose surface. */
class P02IdentityStateUiTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun everyP02AndroidStateIsRuntimeCaptured() {
        val activeState = mutableStateOf(P02_STATE_IDS.first())
        composeRule.setContent {
            YlvenTheme(darkTheme = false) { YlvenAcceptanceState(activeState.value) }
        }
        P02_STATE_IDS.forEach { stateId ->
            composeRule.runOnIdle { activeState.value = stateId }
            composeRule.waitForIdle()
            capture(stateId)
        }
    }

    private fun capture(name: String) {
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        val bitmap = composeRule.onRoot().captureToImage().asAndroidBitmap()
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            val resolver = context.contentResolver
            val relativePath = "${Environment.DIRECTORY_DOWNLOADS}/ylven-p02/"
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
                "Could not create P02 acceptance screenshot media entry"
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
            }
        } else {
            val output = File(context.getExternalFilesDir(null), "screenshots/$name.png")
            output.parentFile?.mkdirs()
            FileOutputStream(output).use { stream ->
                check(bitmap.compress(Bitmap.CompressFormat.PNG, 100, stream))
            }
        }
    }
}

private val P02_STATE_IDS = listOf(
    "YL-A-004-S01_LAUNCH", "YL-A-004-S02_RESTORING", "YL-A-004-S03_FIRST_RUN", "YL-A-004-S04_UPDATE_REQUIRED", "YL-A-004-S05_OFFLINE", "YL-A-004-S06_SERVER_ERROR",
    "YL-A-005-S01_DEFAULT", "YL-A-005-S02_INPUT_FOCUSED", "YL-A-005-S03_VALIDATION_ERROR", "YL-A-005-S04_SUBMITTING", "YL-A-005-S05_RATE_LIMITED", "YL-A-005-S06_OFFLINE", "YL-A-005-S07_SERVER_ERROR",
    "YL-A-006-S01_DEFAULT", "YL-A-006-S02_SECURITY_CHALLENGE", "YL-A-006-S03_SUBMITTING", "YL-A-006-S04_SUCCESS", "YL-A-006-S05_SAVE_ERROR", "YL-A-006-S06_RATE_LIMITED", "YL-A-006-S07_OFFLINE", "YL-A-006-S08_SERVER_ERROR",
    "YL-A-007-S01_LOADING", "YL-A-007-S02_SECURITY_CHALLENGE", "YL-A-007-S03_SUCCESS", "YL-A-007-S04_VALIDATION_ERROR", "YL-A-007-S05_CODE_EXPIRED", "YL-A-007-S06_OFFLINE", "YL-A-007-S07_SERVER_ERROR",
    "YL-A-008-S01_DEFAULT", "YL-A-008-S02_INPUT_FOCUSED", "YL-A-008-S03_CODE_SENT", "YL-A-008-S04_SUBMITTING", "YL-A-008-S05_SUCCESS", "YL-A-008-S06_INVALID_CODE", "YL-A-008-S07_CODE_EXPIRED", "YL-A-008-S08_RATE_LIMITED", "YL-A-008-S09_LOCKED", "YL-A-008-S10_OFFLINE", "YL-A-008-S11_SERVER_ERROR",
    "YL-A-009-S01_SUCCESS", "YL-A-009-S02_SAVE_ERROR", "YL-A-009-S03_OFFLINE",
    "YL-A-010-S01_DEFAULT", "YL-A-010-S02_INPUT_FOCUSED", "YL-A-010-S03_VALIDATION_ERROR", "YL-A-010-S04_SUBMITTING", "YL-A-010-S05_DISABLED", "YL-A-010-S06_OFFLINE", "YL-A-010-S07_SERVER_ERROR",
    "YL-A-011-S01_DEFAULT", "YL-A-011-S02_SECURITY_CHALLENGE", "YL-A-011-S03_SUBMITTING", "YL-A-011-S04_SUCCESS", "YL-A-011-S05_SAVE_ERROR", "YL-A-011-S06_RATE_LIMITED", "YL-A-011-S07_OFFLINE", "YL-A-011-S08_SERVER_ERROR",
    "YL-A-012-S01_DEFAULT", "YL-A-012-S02_INPUT_FOCUSED", "YL-A-012-S03_CODE_SENT", "YL-A-012-S04_SUBMITTING", "YL-A-012-S05_SUCCESS", "YL-A-012-S06_INVALID_CODE", "YL-A-012-S07_CODE_EXPIRED", "YL-A-012-S08_RATE_LIMITED", "YL-A-012-S09_LOCKED", "YL-A-012-S10_OFFLINE", "YL-A-012-S11_SERVER_ERROR",
    "YL-A-013-S01_SUCCESS", "YL-A-013-S02_SAVE_ERROR", "YL-A-013-S03_OFFLINE",
    "YL-A-014-S01_RESTORING", "YL-A-014-S02_SUCCESS", "YL-A-014-S03_UNAUTHORIZED", "YL-A-014-S04_OFFLINE", "YL-A-014-S05_SERVER_ERROR",
    "YL-A-015-S01_DEFAULT", "YL-A-015-S02_SUBMITTING", "YL-A-015-S03_SUCCESS", "YL-A-015-S04_SAVE_ERROR",
    "YL-A-016-S01_LOADING", "YL-A-016-S02_POPULATED", "YL-A-016-S03_EMPTY", "YL-A-016-S04_REFRESHING", "YL-A-016-S05_UNAUTHORIZED", "YL-A-016-S06_OFFLINE", "YL-A-016-S07_SERVER_ERROR",
    "YL-A-017-S01_DEFAULT", "YL-A-017-S02_OFFLINE", "YL-A-017-S03_NETWORK_ERROR", "YL-A-017-S04_TIMEOUT", "YL-A-017-S05_SERVICE_DEGRADED",
)
