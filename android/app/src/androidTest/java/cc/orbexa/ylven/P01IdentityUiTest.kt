package cc.orbexa.ylven

import android.graphics.Bitmap
import androidx.compose.ui.graphics.asAndroidBitmap
import androidx.compose.ui.test.captureToImage
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.onRoot
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.test.platform.app.InstrumentationRegistry
import cc.orbexa.ylven.identity.AuthSession
import cc.orbexa.ylven.identity.DeviceSession
import cc.orbexa.ylven.identity.IdentityGateway
import cc.orbexa.ylven.identity.OtpChallenge
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

        composeRule.onNodeWithText("登录 YLVEN").assertExists()
        capture("P01-AUTH-LOGIN")

        composeRule.onNodeWithTag("p01-go-register").performClick()
        composeRule.onNodeWithTag("p01-register-email").performTextInput("owner@example.com")
        composeRule.onNodeWithTag("p01-register-password").performTextInput("Testpass123")
        composeRule.onNodeWithTag("p01-register-confirm").performTextInput("Testpass123")
        capture("P01-AUTH-REGISTER")

        composeRule.onNodeWithTag("p01-register-submit").performClick()
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
        // Keep acceptance screenshots in the debug package so API 35 scoped-storage rules
        // do not prevent the CI runner from exporting them with run-as.
        val output = File(context.filesDir, "screenshots/$name.png")
        output.parentFile?.mkdirs()
        FileOutputStream(output).use { stream ->
            check(composeRule.onRoot().captureToImage().asAndroidBitmap().compress(Bitmap.CompressFormat.PNG, 100, stream))
        }
    }
}

private class FakeIdentityGateway : IdentityGateway {
    private val session = AuthSession("owner@example.com", "session-current-1234", "access", "refresh")
    override suspend fun startRegistration(email: String) = OtpChallenge("register-challenge", email, "123456")
    override suspend fun finishRegistration(challenge: OtpChallenge, code: String, password: String) = Unit
    override suspend fun startLogin(email: String) = OtpChallenge("login-challenge", email, "654321")
    override suspend fun finishLogin(challenge: OtpChallenge, code: String) = session
    override suspend fun refresh(session: AuthSession) = session.copy(bearer = "access-rotated", renewal = "refresh-rotated")
    override suspend fun devices(bearer: String) = listOf(DeviceSession(session.sessionId, "2026-08-05T00:00:00Z", "2026-09-04T00:00:00Z", false))
    override suspend fun revokeDevice(bearer: String, sessionId: String) = Unit
    override suspend fun logout(bearer: String, allDevices: Boolean) = Unit
}
