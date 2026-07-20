CREATE TYPE "public"."battle_status" AS ENUM('PENDING', 'ONGOING', 'FINISHED');--> statement-breakpoint
CREATE TYPE "public"."round_stage" AS ENUM('DRUM', 'BASS', 'MELODY', 'FULL');--> statement-breakpoint
CREATE TYPE "public"."round_status" AS ENUM('SONG_PICKS', 'GUESSING', 'FINISHED');--> statement-breakpoint
CREATE TYPE "public"."song_status" AS ENUM('PROCESSING', 'READY', 'FAILED');--> statement-breakpoint
CREATE TABLE "battles" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"host_id" uuid NOT NULL,
	"guest_id" uuid,
	"status" "battle_status" DEFAULT 'PENDING' NOT NULL,
	"winner_id" uuid,
	"rounds" integer DEFAULT 5 NOT NULL,
	"current_round" integer DEFAULT 1 NOT NULL,
	"invite_code" varchar(6) NOT NULL,
	"host_song_id" uuid,
	"guest_song_id" uuid,
	"creted_at" timestamp DEFAULT now() NOT NULL,
	"updated_at" timestamp DEFAULT now() NOT NULL,
	CONSTRAINT "battles_invite_code_unique" UNIQUE("invite_code")
);
--> statement-breakpoint
CREATE TABLE "guesses" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"round_id" uuid NOT NULL,
	"user_id" uuid NOT NULL,
	"correct" boolean NOT NULL,
	"guess" varchar NOT NULL,
	"attempt" integer NOT NULL
);
--> statement-breakpoint
CREATE TABLE "refresh_tokens" (
	"token" varchar PRIMARY KEY NOT NULL,
	"created_at" timestamp DEFAULT now() NOT NULL,
	"updated_at" timestamp DEFAULT now() NOT NULL,
	"user_id" uuid NOT NULL,
	"expires_at" timestamp NOT NULL,
	"revoked_at" timestamp
);
--> statement-breakpoint
CREATE TABLE "rounds" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"battle_id" uuid NOT NULL,
	"host_song_id" uuid NOT NULL,
	"guest_song_id" uuid NOT NULL,
	"round_number" integer NOT NULL,
	"hostStage" "round_stage" DEFAULT 'DRUM' NOT NULL,
	"guestStage" "round_stage" DEFAULT 'DRUM' NOT NULL,
	"host_points" integer DEFAULT 0 NOT NULL,
	"guest_points" integer DEFAULT 0 NOT NULL,
	"status" "round_status" DEFAULT 'SONG_PICKS' NOT NULL
);
--> statement-breakpoint
CREATE TABLE "songs" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"title" varchar(255) NOT NULL,
	"artist" varchar(255) NOT NULL,
	"genre" varchar(50) NOT NULL,
	"cover_image" varchar,
	"album" varchar(255),
	"duration" integer NOT NULL,
	"status" "song_status" DEFAULT 'PROCESSING' NOT NULL,
	"full_song_path" varchar(255),
	"drums_path" varchar(255),
	"bass_path" varchar(255),
	"melody_path" varchar(255),
	"vocals_path" varchar(255)
);
--> statement-breakpoint
CREATE TABLE "users" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"username" varchar(30) NOT NULL,
	"email" varchar(255) NOT NULL,
	"password_hash" varchar(255) NOT NULL,
	"avatar_url" varchar(255),
	"role" varchar(20) DEFAULT 'USER' NOT NULL,
	"created_at" timestamp DEFAULT now() NOT NULL,
	"updated_at" timestamp DEFAULT now() NOT NULL,
	CONSTRAINT "users_username_unique" UNIQUE("username"),
	CONSTRAINT "users_email_unique" UNIQUE("email")
);
--> statement-breakpoint
ALTER TABLE "battles" ADD CONSTRAINT "battles_host_id_users_id_fk" FOREIGN KEY ("host_id") REFERENCES "public"."users"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "battles" ADD CONSTRAINT "battles_guest_id_users_id_fk" FOREIGN KEY ("guest_id") REFERENCES "public"."users"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "battles" ADD CONSTRAINT "battles_winner_id_users_id_fk" FOREIGN KEY ("winner_id") REFERENCES "public"."users"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "battles" ADD CONSTRAINT "battles_host_song_id_songs_id_fk" FOREIGN KEY ("host_song_id") REFERENCES "public"."songs"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "battles" ADD CONSTRAINT "battles_guest_song_id_songs_id_fk" FOREIGN KEY ("guest_song_id") REFERENCES "public"."songs"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "guesses" ADD CONSTRAINT "guesses_round_id_rounds_id_fk" FOREIGN KEY ("round_id") REFERENCES "public"."rounds"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "guesses" ADD CONSTRAINT "guesses_user_id_users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "refresh_tokens" ADD CONSTRAINT "refresh_tokens_user_id_users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "rounds" ADD CONSTRAINT "rounds_battle_id_battles_id_fk" FOREIGN KEY ("battle_id") REFERENCES "public"."battles"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "rounds" ADD CONSTRAINT "rounds_host_song_id_songs_id_fk" FOREIGN KEY ("host_song_id") REFERENCES "public"."songs"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "rounds" ADD CONSTRAINT "rounds_guest_song_id_songs_id_fk" FOREIGN KEY ("guest_song_id") REFERENCES "public"."songs"("id") ON DELETE no action ON UPDATE no action;
