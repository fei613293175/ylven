package cc.orbexa.ylven

import android.graphics.Bitmap
import android.content.ContentValues
import android.os.Build
import android.os.Environment
import android.provider.MediaStore
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.compose.ui.test.captureToImage
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.onRoot
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.test.platform.app.InstrumentationRegistry
import cc.orbexa.ylven.identity.AuthSession
import cc.orbexa.ylven.identity.DeviceSession
import cc.orbexa.ylven.identity.IdentityGateway
import cc.orbexa.ylven.identity.OtpChallenge
import cc.orbexa.ylven.identity.SecurityChallenge
import cc.orbexa.ylven.ui.YlvenApp
import cc.orbexa.ylven.ui.theme.YlvenTheme
import java.io.File
import java.io.FileOutputStream
import org.junit.Rule
import org.junit.Test

class P01IdentityUiTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun registrationLoginAndAccountScreensAreOperableAndCaptured() {
        val gateway = FakeIdentityGateway()
        composeRule.setContent { YlvenTheme(darkTheme = false) { YlvenApp(gateway = gateway) } }
        composeRule.mainClock.advanceTimeBy(3_000)
        composeRule.waitForIdle()

        composeRule.onNodeWithText("登录 YLVEN").assertExists()
        capture("P01-AUTH-LOGIN")

        composeRule.onNodeWithTag("p01-go-register").performClick()
        composeRule.onNodeWithTag("p01-register-email").performTextInput("owner@example.com")
        composeRule.onNodeWithTag("p01-register-password").performTextInput("Testpass123")
        composeRule.onNodeWithTag("p01-register-confirm").performTextInput("Testpass123")
        capture("P01-AUTH-REGISTER")

        composeRule.onNodeWithTag("p01-register-submit").performClick()
        composeRule.waitForSecurityConfirmation()
        composeRule.onNodeWithTag("p02-security-answer").performTextInput("3")
        composeRule.onNodeWithTag("p01-security-confirm").assertIsEnabled()
        composeRule.onNodeWithTag("p01-security-confirm").performClick()
        composeRule.waitForIdle()
        composeRule.onNodeWithText("验证注册邮箱").assertExists()
        composeRule.onNodeWithTag("p01-debug-otp").assertExists()
        capture("P01-AUTH-REGISTER-OTP")

        composeRule.onNodeWithTag("p01-otp-submit").performClick()
        composeRule.waitForIdle()
        composeRule.onNodeWithText("账户已创建").assertExists()
        capture("P01-AUTH-REGISTERED")

        composeRule.onNodeWithTag("p01-registered-login").performClick()
        composeRule.onNodeWithTag("p01-login-email").performTextInput("owner@example.com")
        composeRule.onNodeWithTag("p01-login-send").performClick()
        composeRule.waitForSecurityConfirmation()
        composeRule.onNodeWithTag("p02-security-answer").performTextInput("3")
        composeRule.onNodeWithTag("p01-security-confirm").assertIsEnabled()
        composeRule.onNodeWithTag("p01-security-confirm").performClick()
        composeRule.waitForIdle()
        composeRule.onNodeWithTag("p01-otp-submit").performClick()
        composeRule.waitForIdle()
        composeRule.onNodeWithText("账户与设备").assertExists()
        composeRule.onNodeWithText("当前设备").assertExists()
        capture("P01-AUTH-ACCOUNT")
    }

    private fun capture(name: String) {
        composeRule.waitForIdle()
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        val bitmap = composeRule.onRoot().captureToImage().asAndroidBitmap()
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            val resolver = context.contentResolver
            val relativePath = "${Environment.DIRECTORY_DOWNLOADS}/ylven-p01/"
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
                "Could not create acceptance screenshot media entry"
            }
            try {
                val stream = requireNotNull(resolver.openOutputStream(uri)) {
                    "Could not open acceptance screenshot output"
                }
                stream.use {
                    check(bitmap.compress(Bitmap.CompressFormat.PNG, 100, it))
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

private class FakeIdentityGateway : IdentityGateway {
    private val session = AuthSession("owner@example.com", "session-current-1234", "access", "refresh", "device-current")
    override suspend fun startRegistration(email: String) = OtpChallenge("register-challenge", email, "123456")
    override suspend fun finishRegistration(challenge: OtpChallenge, code: String, password: String) = Unit
    override suspend fun startLogin(email: String) = OtpChallenge("login-challenge", email, "654321")
    override suspend fun createRegistrationChallenge(email: String) = SecurityChallenge("register-challenge", email, "register", "1 + 2 = ?")
    override suspend fun createLoginChallenge(email: String) = SecurityChallenge("login-challenge", email, "login", "1 + 2 = ?")
    override suspend fun verifyAnswer(challenge: SecurityChallenge, answer: String) {
        check(answer == "3") { "unexpected security answer" }
    }
    override suspend fun requestRegistrationOtp(challenge: SecurityChallenge) = OtpChallenge(challenge.id, challenge.email, "123456")
    override suspend fun requestLoginOtp(challenge: SecurityChallenge) = OtpChallenge(challenge.id, challenge.email, "654321")
    override suspend fun finishLogin(challenge: OtpChallenge, code: String) = session
    override suspend fun refresh(session: AuthSession) = session.copy(bearer = "access-rotated", renewal = "refresh-rotated")
    override suspend fun devices(bearer: String) = listOf(DeviceSession(session.deviceId, "2026-08-05T00:00:00Z", "2026-09-04T00:00:00Z", false))
    override suspend fun revokeDevice(bearer: String, sessionId: String) = Unit
    override suspend fun logout(bearer: String, allDevices: Boolean) = Unit
}

private fun androidx.compose.ui.test.junit4.ComposeTestRule.waitForSecurityConfirmation() {
    waitUntil(5_000) {
        try {
            onNodeWithTag("p01-security-confirm").assertIsEnabled()
            true
        } catch (_: AssertionError) {
            false
        }
    }
}
