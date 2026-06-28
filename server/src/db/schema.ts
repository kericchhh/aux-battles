
import { uuid, pgTable, varchar, timestamp, integer } from "drizzle-orm/pg-core";

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
    userId: uuid("user_id").references(() => usersTable.id, {onDelete: "cascade"}).notNull(),
    expiresAt: timestamp("expires_at").notNull(),
    revokedAt: timestamp("revoked_at")
})

export const songsTable = pgTable("songs", {
    id: uuid("id").primaryKey().defaultRandom(),
    title: varchar("title",{length:255}).notNull(),
    artist: varchar("artist",{length: 255}).notNull(),
    genre: varchar("genre",{length: 50}).notNull(),
    coverImage: varchar("cover_image"),
    album: varchar("album",{length: 255}),
    duration: integer("duration").notNull(),
    status: varchar("status", {length: 20})
        .$type<"PROCESSING" | "READY" | "FAILED">()
        .default("PROCESSING")
        .notNull()
})

