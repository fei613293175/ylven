package cc.orbexa.ylven

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import cc.orbexa.ylven.ui.YlvenApp
import cc.orbexa.ylven.identity.HttpIdentityGateway
import cc.orbexa.ylven.identity.SessionStore
import cc.orbexa.ylven.ui.theme.YlvenTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        val sessionStore = SessionStore(this)
        setContent {
            YlvenTheme {
                YlvenApp(
                    gateway = HttpIdentityGateway(),
                    initialSession = sessionStore.load(),
                    onSessionChange = sessionStore::save,
                )
            }
        }
    }
}
