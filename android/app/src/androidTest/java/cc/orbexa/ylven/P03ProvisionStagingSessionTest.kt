package cc.orbexa.ylven

import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.test.platform.app.InstrumentationRegistry
import cc.orbexa.ylven.identity.HttpIdentityGateway
import cc.orbexa.ylven.identity.SessionStore
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Rule
import org.junit.Test

/**
 * Establishes the real staging identity used by the P03-001 through P03-029
 * physical-device journey after the owner-approved signing migration.
 */
class P03ProvisionStagingSessionTest {
    @get:Rule
    val composeRule = createAndroidComposeRule<MainActivity>()

    @Test
    fun createsAndPersistsRealStagingSession() = runBlocking {
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        val gateway = HttpIdentityGateway(deviceId = "p03-physical-${System.currentTimeMillis()}")
        val email = "p03.physical.${System.currentTimeMillis()}@example.com"
        val password = "P03Physical${System.nanoTime()}a1"

        val registration = gateway.startRegistration(email)
        assertFalse("Staging did not return its configured debug OTP", registration.debugCode.isNullOrBlank())
        val session = gateway.finishRegistrationSession(
            registration,
            requireNotNull(registration.debugCode),
            password,
        )
        assertNotNull("Registration did not issue an authenticated staging session", session)
        val authenticated = requireNotNull(session)
        gateway.initializeWorkspace(authenticated)
        SessionStore(context).save(authenticated)

        val restored = SessionStore(context).load()
        assertNotNull("The real staging session was not persisted through Android Keystore", restored)
        assertEquals(email, restored?.email)
        assertEquals(email, gateway.restore(requireNotNull(restored)).email)
    }
}
