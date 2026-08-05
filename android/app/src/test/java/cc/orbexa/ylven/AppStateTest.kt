package cc.orbexa.ylven

import cc.orbexa.ylven.identity.AuthSession
import cc.orbexa.ylven.identity.OtpChallenge
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
}
