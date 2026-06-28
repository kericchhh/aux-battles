
import { uuid, pgTable, varchar, timestamp, integer, boolean, pgEnum } from "drizzle-orm/pg-core";

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

export const songStatus = pgEnum("song_status", ["PROCESSING", "ONGOING", "FINISHED"])
export const battleStatus = pgEnum("battle_status", ["PENDING", "ONGOING", "FINISHED"])
export const roundStatus = pgEnum("round_status", ["SONG_PICKS", "GUESSING", "FINISHED"]) 
export const roundStage = pgEnum("round_stage", ["DRUM", "BASS", "MELODY", "FULL"])

export const songsTable = pgTable("songs", {
    id: uuid("id").primaryKey().defaultRandom(),
    title: varchar("title", { length: 255 }).notNull(),
    artist: varchar("artist", { length: 255 }).notNull(),
    genre: varchar("genre", { length: 50 }).notNull(),
    coverImage: varchar("cover_image"),
    album: varchar("album", { length: 255 }),
    duration: integer("duration").notNull(),
    status: songStatus().default("PROCESSING").notNull() 
})

export const battlesTable = pgTable("battles", {
    id: uuid("id").primaryKey().defaultRandom(),
    hostId: uuid("host_id").references(() => usersTable.id).notNull(),
    guestId: uuid("guest_id").references(() => usersTable.id),
    status: battleStatus().default("PENDING").notNull(),
    winnerId: uuid("winner_id").references(() => usersTable.id),
    rounds: integer("rounds").default(5).notNull(),
    createdAt: timestamp("creted_at").defaultNow().notNull(),
    updatedAt: timestamp("updated_at").defaultNow().notNull().$onUpdate(() => new Date())
})

export const roundsTable = pgTable("rounds", {
    id: uuid("id").primaryKey().defaultRandom(),
    battleId: uuid("battle_id").references(() => battlesTable.id, {onDelete: "cascade"}).notNull(),
    hostSongId: uuid("host_song_id").references(() => songsTable.id).notNull(),
    guestSongId: uuid("guest_song_id").references(() => songsTable.id).notNull(),
    roundNumber: integer("round_number").notNull(),
    hostStage: roundStage() 
        .default("DRUM")
        .notNull(),
    guestStage: roundStage() 
        .default("DRUM")
        .notNull(),
    hostPoints: integer("host_points").default(0).notNull(),
    guestPoints: integer("guest_points").default(0).notNull(),
    status: roundStatus()
})

export const guessesTable = pgTable("guesses", {
    id: uuid("id").primaryKey().defaultRandom(),
    roundId: uuid("round_id").references(() => roundsTable.id, {onDelete: "cascade"}).notNull(),
    userId: uuid("user_id").references(() => usersTable.id, {onDelete: "cascade"}).notNull(),
    correct: boolean("correct").notNull(),
    guess: varchar("guess").notNull(),
    attempt: integer("attempt").notNull()
})
