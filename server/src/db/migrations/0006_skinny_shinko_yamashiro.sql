CREATE TABLE IF NOT EXISTS "sessions" (
	"token_hash" varchar(64) PRIMARY KEY NOT NULL,
	"user_id" uuid NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"expires_at" timestamp with time zone NOT NULL
);
--> statement-breakpoint
DO $$
DECLARE
	status_labels text[];
BEGIN
	SELECT array_agg(enumlabel ORDER BY enumsortorder)
	INTO status_labels
	FROM pg_enum
	WHERE enumtypid = 'public.song_status'::regtype;

	IF status_labels IS DISTINCT FROM ARRAY['PROCESSING', 'READY', 'FAILED']::text[] THEN
		ALTER TABLE "songs" ALTER COLUMN "status" DROP DEFAULT;
		ALTER TABLE "songs" ALTER COLUMN "status" SET DATA TYPE text USING "status"::text;
		UPDATE "songs"
		SET "status" = CASE "status"
			WHEN 'ONGOING' THEN 'PROCESSING'
			WHEN 'FINISHED' THEN 'READY'
			ELSE "status"
		END;
		DROP TYPE "public"."song_status";
		CREATE TYPE "public"."song_status" AS ENUM('PROCESSING', 'READY', 'FAILED');
		ALTER TABLE "songs" ALTER COLUMN "status" SET DATA TYPE "public"."song_status" USING "status"::"public"."song_status";
	END IF;
END $$;
--> statement-breakpoint
ALTER TABLE "songs" ALTER COLUMN "status" SET DEFAULT 'PROCESSING'::"public"."song_status";--> statement-breakpoint
ALTER TABLE "rounds" ALTER COLUMN "host_song_id" DROP NOT NULL;--> statement-breakpoint
ALTER TABLE "rounds" ALTER COLUMN "guest_song_id" DROP NOT NULL;--> statement-breakpoint
ALTER TABLE "rounds" ADD COLUMN IF NOT EXISTS "host_finished" boolean DEFAULT false NOT NULL;--> statement-breakpoint
ALTER TABLE "rounds" ADD COLUMN IF NOT EXISTS "guest_finished" boolean DEFAULT false NOT NULL;--> statement-breakpoint
ALTER TABLE "songs" ADD COLUMN IF NOT EXISTS "full_song_path" varchar(255);--> statement-breakpoint
ALTER TABLE "songs" ADD COLUMN IF NOT EXISTS "drums_path" varchar(255);--> statement-breakpoint
ALTER TABLE "songs" ADD COLUMN IF NOT EXISTS "bass_path" varchar(255);--> statement-breakpoint
ALTER TABLE "songs" ADD COLUMN IF NOT EXISTS "melody_path" varchar(255);--> statement-breakpoint
ALTER TABLE "songs" ADD COLUMN IF NOT EXISTS "vocals_path" varchar(255);--> statement-breakpoint
DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'public.sessions'::regclass AND conname = 'sessions_user_id_users_id_fk') THEN
		ALTER TABLE "sessions" ADD CONSTRAINT "sessions_user_id_users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE cascade ON UPDATE no action;
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'public.guesses'::regclass AND conname = 'guesses_round_user_attempt_unique') THEN
		ALTER TABLE "guesses" ADD CONSTRAINT "guesses_round_user_attempt_unique" UNIQUE("round_id", "user_id", "attempt");
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'public.rounds'::regclass AND conname = 'rounds_battle_number_unique') THEN
		ALTER TABLE "rounds" ADD CONSTRAINT "rounds_battle_number_unique" UNIQUE("battle_id", "round_number");
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'public.battles'::regclass AND conname = 'battles_distinct_players') THEN
		ALTER TABLE "battles" ADD CONSTRAINT "battles_distinct_players" CHECK ("guest_id" is null or "host_id" <> "guest_id");
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'public.battles'::regclass AND conname = 'battles_round_count_valid') THEN
		ALTER TABLE "battles" ADD CONSTRAINT "battles_round_count_valid" CHECK ("rounds" between 1 and 10);
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'public.battles'::regclass AND conname = 'battles_current_round_valid') THEN
		ALTER TABLE "battles" ADD CONSTRAINT "battles_current_round_valid" CHECK ("current_round" between 1 and "rounds");
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'public.battles'::regclass AND conname = 'battles_winner_valid') THEN
		ALTER TABLE "battles" ADD CONSTRAINT "battles_winner_valid" CHECK (
			"winner_id" is null
			or "winner_id" = "host_id"
			or ("guest_id" is not null and "winner_id" = "guest_id")
		);
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'public.guesses'::regclass AND conname = 'guesses_attempt_valid') THEN
		ALTER TABLE "guesses" ADD CONSTRAINT "guesses_attempt_valid" CHECK ("attempt" between 1 and 4);
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'public.rounds'::regclass AND conname = 'rounds_number_positive') THEN
		ALTER TABLE "rounds" ADD CONSTRAINT "rounds_number_positive" CHECK ("round_number" > 0);
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'public.rounds'::regclass AND conname = 'rounds_point_valid') THEN
		ALTER TABLE "rounds" ADD CONSTRAINT "rounds_point_valid" CHECK ("host_points" between 0 and 100 and "guest_points" between 0 and 100);
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'public.rounds'::regclass AND conname = 'rounds_songs_required_after_selection') THEN
		ALTER TABLE "rounds" ADD CONSTRAINT "rounds_songs_required_after_selection" CHECK ("status" = 'SONG_PICKS' or ("host_song_id" is not null and "guest_song_id" is not null));
	END IF;
END $$;
--> statement-breakpoint
CREATE INDEX IF NOT EXISTS "sessions_user_idx" ON "sessions" USING btree ("user_id");--> statement-breakpoint
CREATE INDEX IF NOT EXISTS "sessions_expiry_idx" ON "sessions" USING btree ("expires_at");--> statement-breakpoint
UPDATE "rounds" SET "host_finished" = true, "guest_finished" = true WHERE "status" = 'FINISHED' AND NOT ("host_finished" AND "guest_finished");--> statement-breakpoint
DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'public.rounds'::regclass AND conname = 'rounds_finished_players') THEN
		ALTER TABLE "rounds" ADD CONSTRAINT "rounds_finished_players" CHECK ("status" <> 'FINISHED' or ("host_finished" and "guest_finished"));
	END IF;
END $$;
