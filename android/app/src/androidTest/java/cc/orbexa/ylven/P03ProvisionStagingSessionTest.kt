package cc.orbexa.ylven

import android.content.Intent
import android.util.Log
import androidx.test.platform.app.InstrumentationRegistry
import cc.orbexa.ylven.identity.HttpIdentityGateway
import cc.orbexa.ylven.identity.SessionStore
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Test

/**
 * Establishes the real staging identity used by the P03-001 through P03-029
 * physical-device journey after the owner-approved signing migration.
 */
class P03ProvisionStagingSessionTest {
    @Test
    fun createsAndPersistsRealStagingSession() = runBlocking {
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        context.startActivity(
            Intent(context, MainActivity::class.java).addFlags(
                Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP,
            ),
        )
        InstrumentationRegistry.getInstrumentation().waitForIdleSync()
        val gateway = HttpIdentityGateway(deviceId = "p03-physical-${System.currentTimeMillis()}")
        val email = "p03.physical.${System.currentTimeMillis()}@example.com"
        val password = "P03Physical${System.nanoTime()}a1"

        reportStage("registration_start")
        val registration = gateway.startRegistration(email)
        reportStage("registration_otp_received")
        assertFalse("Staging did not return its configured debug OTP", registration.debugCode.isNullOrBlank())
        val session = gateway.finishRegistrationSession(
            registration,
            requireNotNull(registration.debugCode),
            password,
        )
        reportStage("registration_session_issued")
        assertNotNull("Registration did not issue an authenticated staging session", session)
        val authenticated = requireNotNull(session)
        gateway.initializeWorkspace(authenticated)
        reportStage("workspace_initialized")
        SessionStore(context).save(authenticated)
        reportStage("session_persisted")

        val restored = SessionStore(context).load()
        assertNotNull("The real staging session was not persisted through Android Keystore", restored)
        assertEquals(email, restored?.email)
        assertEquals(email, gateway.restore(requireNotNull(restored)).email)
        reportStage("session_restored")
    }

    private fun reportStage(stage: String) {
        // Vivo Android 14 can block the instrumentation Binder status channel
        // after a network response. Keep stage evidence in logcat instead.
        Log.i("YLVEN_P03_STAGE", stage)
    }
}
