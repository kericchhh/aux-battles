import { uuid, pgTable, varchar, timestamp, integer, boolean, pgEnum, check, index, unique } from "drizzle-orm/pg-core";
import { sql } from "drizzle-orm"

export const sessionsTable = pgTable("sessions", {
    tokenHash: varchar("token_hash", { length: 64 }).primaryKey(),

    userId: uuid("user_id").notNull().references(() => usersTable.id, { onDelete: "cascade" }),
    createdAt: timestamp("created_at", { withTimezone: true }).defaultNow().notNull(),
    expiresAt: timestamp("expires_at", { withTimezone: true }).notNull(),
},
    (table) => [
        index("sessions_user_idx").on(table.userId),
        index("sessions_expiry_idx").on(table.expiresAt)
    ]
)

export const usersTable = pgTable("users", {
    id: uuid("id").primaryKey().defaultRandom(),
    username: varchar("username", { length: 30 }).notNull().unique(),
    email: varchar("email", { length: 255 }).notNull().unique(),
    passwordHash: varchar("password_hash", { length: 255 }).notNull(),
    avatarUrl: varchar("avatar_url", { length: 255 }),
    role: varchar("role", { length: 20 })
        .$type<"USER" | "ADMIN">()
        .default("USER")
        .notNull(),

    createdAt: timestamp("created_at").defaultNow().notNull(),
    updatedAt: timestamp("updated_at").defaultNow().notNull().$onUpdate(() => new Date())
});

export const refreshTokens = pgTable("refresh_tokens", {
    token: varchar("token").primaryKey(),
    createdAt: timestamp("created_at").notNull().defaultNow(),
    updatedAt: timestamp("updated_at").notNull().defaultNow().$onUpdate(() => new Date()),
    userId: uuid("user_id").references(() => usersTable.id, { onDelete: "cascade" }).notNull(),
    expiresAt: timestamp("expires_at").notNull(),
    revokedAt: timestamp("revoked_at")
})

export const songStatus = pgEnum("song_status", ["PROCESSING", "READY", "FAILED"])
export const battleStatus = pgEnum("battle_status", ["PENDING", "SELECTING", "ONGOING", "FINISHED"])
export const roundStatus = pgEnum("round_status", ["WAITING", "GUESSING", "FINISHED"])
export const roundStage = pgEnum("round_stage", ["DRUM", "BASS", "MELODY", "FULL"])

export const songsTable = pgTable("songs", {
    id: uuid("id").primaryKey().defaultRandom(),
    title: varchar("title", { length: 255 }).notNull(),
    artist: varchar("artist", { length: 255 }).notNull(),
    genre: varchar("genre", { length: 50 }).notNull(),
    coverImage: varchar("cover_image"),
    album: varchar("album", { length: 255 }),
    duration: integer("duration").notNull(),
    status: songStatus().default("PROCESSING").notNull(),
    fullSongPath: varchar("full_song_path", { length: 255 }),
    drumsPath: varchar("drums_path", { length: 255 }),
    bassPath: varchar("bass_path", { length: 255 }),
    melodyPath: varchar("melody_path", { length: 255 }),
    vocalsPath: varchar("vocals_path", { length: 255 })
})

export const battlesTable = pgTable("battles", {
    id: uuid("id").primaryKey().defaultRandom(),
    hostId: uuid("host_id").references(() => usersTable.id).notNull(),
    guestId: uuid("guest_id").references(() => usersTable.id),
    status: battleStatus().default("PENDING").notNull(),
    winnerId: uuid("winner_id").references(() => usersTable.id),
    rounds: integer("rounds").default(5).notNull(),
    currentRound: integer("current_round").default(1).notNull(),
    inviteCode: varchar("invite_code", { length: 6 }).notNull().unique(),
    createdAt: timestamp("creted_at").defaultNow().notNull(),
    updatedAt: timestamp("updated_at").defaultNow().notNull().$onUpdate(() => new Date())
},
    (table) => [
        check(
            "battles_distinct_players",
            sql`${table.guestId} is null or ${table.hostId} <> ${table.guestId}`,
        ),

        check(
            "battles_round_count_valid",
            sql`${table.rounds} between 1 and 10`,
        ),

        check(
            "battles_current_round_valid",
            sql`${table.currentRound} between 1 and ${table.rounds}`,
        ),

        check(
            "battles_winner_valid",
            sql`
      ${table.winnerId} is null
      or ${table.winnerId} = ${table.hostId}
      or (
        ${table.guestId} is not null
        and ${table.winnerId} = ${table.guestId}
      )
    `,
        ),
    ]
)


export const roundsTable = pgTable("rounds", {
    id: uuid("id").primaryKey().defaultRandom(),
    battleId: uuid("battle_id").references(() => battlesTable.id, { onDelete: "cascade" }).notNull(),
    hostSongId: uuid("host_song_id").references(() => songsTable.id),
    guestSongId: uuid("guest_song_id").references(() => songsTable.id),
    roundNumber: integer("round_number").notNull(),
    hostStage: roundStage()
        .default("DRUM")
        .notNull(),
    guestStage: roundStage()
        .default("DRUM")
        .notNull(),
    hostPoints: integer("host_points").default(0).notNull(),
    guestPoints: integer("guest_points").default(0).notNull(),
    hostFinished: boolean("host_finished").default(false).notNull(),
    guestFinished: boolean("guest_finished").default(false).notNull(),
    status: roundStatus().default("WAITING").notNull()
},
    (table) => [
        unique("rounds_battle_number_unique").on(table.battleId, table.roundNumber),
        check("rounds_number_positive", sql`${table.roundNumber} > 0`),
        check("rounds_point_valid", sql`${table.hostPoints} between 0 and 100 and ${table.guestPoints} between 0 and 100`),
        check("rounds_songs_required_after_selection", sql`${table.status} = 'WAITING' or (${table.hostSongId} is not null and ${table.guestSongId} is not null)`),
        check("rounds_finished_players", sql`${table.status} <> 'FINISHED' or (${table.hostFinished} and ${table.guestFinished})`)
    ],

)

export const guessesTable = pgTable("guesses", {
    id: uuid("id").primaryKey().defaultRandom(),
    roundId: uuid("round_id").references(() => roundsTable.id, { onDelete: "cascade" }).notNull(),
    userId: uuid("user_id").references(() => usersTable.id, { onDelete: "cascade" }).notNull(),
    correct: boolean("correct").notNull(),
    guess: varchar("guess").notNull(),
    attempt: integer("attempt").notNull()
},
    (table) => [
        unique("guesses_round_user_attempt_unique").on(table.roundId, table.userId, table.attempt),
        check("guesses_attempt_valid", sql`${table.attempt} between 1 and 4`)
    ]
)
