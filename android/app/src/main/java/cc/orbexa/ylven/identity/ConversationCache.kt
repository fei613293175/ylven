package cc.orbexa.ylven.identity

import android.content.Context
import androidx.room.Dao
import androidx.room.Database
import androidx.room.Entity
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.PrimaryKey
import androidx.room.Query
import androidx.room.Room
import androidx.room.RoomDatabase

@Entity(tableName = "cached_messages")
data class CachedMessage(
    @PrimaryKey val id: String,
    val conversationId: String,
    val role: String,
    val body: String,
    val createdAt: String,
)

@Dao
interface ConversationCacheDao {
    @Query("SELECT * FROM cached_messages WHERE conversationId = :conversationId ORDER BY createdAt")
    suspend fun messages(conversationId: String): List<CachedMessage>

    @Query("DELETE FROM cached_messages WHERE conversationId = :conversationId")
    suspend fun clear(conversationId: String)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertAll(messages: List<CachedMessage>)
}

@Database(entities = [CachedMessage::class], version = 1, exportSchema = false)
abstract class ConversationCacheDatabase : RoomDatabase() {
    abstract fun dao(): ConversationCacheDao
}

/** Local Room snapshot cache. Server data remains authoritative; this cache is
 * used only to preserve previously synchronized conversations while offline. */
class ConversationCache(context: Context) {
    private val dao = Room.databaseBuilder(context.applicationContext, ConversationCacheDatabase::class.java, "ylven_conversations.db").build().dao()

    suspend fun save(conversationId: String, messages: List<MessageRecord>) {
        dao.clear(conversationId)
        dao.insertAll(messages.map { CachedMessage(it.id, conversationId, it.role, it.body, it.createdAt) })
    }

    suspend fun load(conversationId: String): List<MessageRecord> =
        dao.messages(conversationId).map { MessageRecord(it.id, it.conversationId, it.role, it.body, it.createdAt) }
}
