package cc.orbexa.ylven

import cc.orbexa.ylven.identity.HttpIdentityGateway
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class P01LiveBackendTest {
    @Test
    fun publicBackendSupportsRegistrationLoginRefreshDevicesAndLogout() = runBlocking {
        val gateway = HttpIdentityGateway()
        val email = "p01.android.${System.currentTimeMillis()}@example.com"
        val password = "Testpass123"

        val registration = gateway.startRegistration(email)
        assertEquals(email, registration.email)
        assertFalse(registration.debugCode.isNullOrBlank())
        gateway.finishRegistration(registration, requireNotNull(registration.debugCode), password)

        val login = gateway.startLogin(email)
        assertFalse(login.debugCode.isNullOrBlank())
        val session = gateway.finishLogin(login, requireNotNull(login.debugCode))
        assertEquals(email, session.email)
        assertTrue(gateway.devices(session.bearer).any { it.id == session.deviceId })

        val rotated = gateway.refresh(session)
        assertNotEquals(session.renewal, rotated.renewal)
        val devices = gateway.devices(rotated.bearer)
        assertEquals(1, devices.size)
        assertTrue(devices.any { it.id == rotated.deviceId })
        gateway.logout(rotated.bearer, allDevices = true)
    }
}
