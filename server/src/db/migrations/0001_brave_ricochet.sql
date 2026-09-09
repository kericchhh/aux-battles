CREATE TABLE "sessions" (
	"token_hash" varchar(64) PRIMARY KEY NOT NULL,
	"user_id" uuid NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"expires_at" timestamp with time zone NOT NULL
);
--> statement-breakpoint
ALTER TABLE "rounds" ALTER COLUMN "host_song_id" DROP NOT NULL;--> statement-breakpoint
ALTER TABLE "rounds" ALTER COLUMN "guest_song_id" DROP NOT NULL;--> statement-breakpoint
ALTER TABLE "rounds" ADD COLUMN "host_finished" boolean DEFAULT false NOT NULL;--> statement-breakpoint
ALTER TABLE "rounds" ADD COLUMN "guest_finished" boolean DEFAULT false NOT NULL;--> statement-breakpoint
ALTER TABLE "sessions" ADD CONSTRAINT "sessions_user_id_users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
CREATE INDEX "sessions_user_idx" ON "sessions" USING btree ("user_id");--> statement-breakpoint
CREATE INDEX "sessions_expiry_idx" ON "sessions" USING btree ("expires_at");--> statement-breakpoint
ALTER TABLE "guesses" ADD CONSTRAINT "guesses_round_user_attempt_unique" UNIQUE("round_id","user_id","attempt");--> statement-breakpoint
ALTER TABLE "rounds" ADD CONSTRAINT "rounds_battle_number_unique" UNIQUE("battle_id","round_number");--> statement-breakpoint
ALTER TABLE "battles" ADD CONSTRAINT "battles_distinct_players" CHECK ("battles"."guest_id" is null or "battles"."host_id" <> "battles"."guest_id");--> statement-breakpoint
ALTER TABLE "battles" ADD CONSTRAINT "battles_round_count_valid" CHECK ("battles"."rounds" between 1 and 10);--> statement-breakpoint
ALTER TABLE "battles" ADD CONSTRAINT "battles_current_round_valid" CHECK ("battles"."current_round" between 1 and "battles"."rounds");--> statement-breakpoint
ALTER TABLE "battles" ADD CONSTRAINT "battles_winner_valid" CHECK (
      "battles"."winner_id" is null
      or "battles"."winner_id" = "battles"."host_id"
      or (
        "battles"."guest_id" is not null
        and "battles"."winner_id" = "battles"."guest_id"
      )
    );--> statement-breakpoint
ALTER TABLE "guesses" ADD CONSTRAINT "guesses_attempt_valid" CHECK ("guesses"."attempt" between 1 and 4);--> statement-breakpoint
ALTER TABLE "rounds" ADD CONSTRAINT "rounds_number_positive" CHECK ("rounds"."round_number" > 0);--> statement-breakpoint
ALTER TABLE "rounds" ADD CONSTRAINT "rounds_point_valid" CHECK ("rounds"."host_points" between 0 and 100 and "rounds"."guest_points" between 0 and 100);--> statement-breakpoint
ALTER TABLE "rounds" ADD CONSTRAINT "rounds_songs_required_after_selection" CHECK ( "rounds"."status" = 'SONG_PICKS' or ("rounds"."host_song_id" is not null and "rounds"."guest_song_id" is not null));--> statement-breakpoint
ALTER TABLE "rounds" ADD CONSTRAINT "rounds_finished_players" CHECK ("rounds"."status" <> 'FINISHED' or ("rounds"."host_finished" and "rounds"."guest_finished"));
