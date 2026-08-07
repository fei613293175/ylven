package cc.orbexa.ylven

import cc.orbexa.ylven.identity.AuthSession
import cc.orbexa.ylven.identity.OtpChallenge
import cc.orbexa.ylven.identity.userFacingMessage
import cc.orbexa.ylven.ui.theme.YlvenDimensions
import org.junit.Assert.assertEquals
import org.junit.Test

class AppStateTest {
    @Test
    fun identityModelsKeepTheP01SessionContract() {
        val challenge = OtpChallenge("challenge-1", "owner@example.com", "123456")
        assertEquals("challenge-1", challenge.id)
        assertEquals("owner@example.com", challenge.email)
        assertEquals("123456", challenge.debugCode)

        val session = AuthSession("owner@example.com", "session-1", "access-1", "refresh-1")
        assertEquals("session-1", session.sessionId)
        assertEquals("access-1", session.bearer)
        assertEquals("refresh-1", session.renewal)
    }

    @Test
    fun shellDimensionsMatchYlDs120() {
        assertEquals(56f, YlvenDimensions.TopAppBarHeight.value)
        assertEquals(64f, YlvenDimensions.BottomNavigationHeight.value)
        assertEquals(16f, YlvenDimensions.PageHorizontal.value)
        assertEquals(24f, YlvenDimensions.SectionGap.value)
        assertEquals(24f, YlvenDimensions.Icon.value)
        assertEquals(48f, YlvenDimensions.MinimumTouchTarget.value)
    }

    @Test
    fun internalErrorCodesAreNeverShownToUsers() {
        assertEquals("这个邮箱已经注册过了，可以直接登录", userFacingMessage("email_already_registered", "email_already_registered", 409))
        assertEquals("暂时无法完成，请稍后再试 (500)", userFacingMessage("unknown_failure", "Unknown failure", 500))
    }
}
