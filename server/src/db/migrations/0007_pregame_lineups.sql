ALTER TYPE "public"."battle_status" ADD VALUE 'SELECTING' BEFORE 'ONGOING';--> statement-breakpoint
ALTER TABLE "rounds" DROP CONSTRAINT "rounds_songs_required_after_selection";--> statement-breakpoint
ALTER TABLE "rounds" DROP CONSTRAINT "rounds_finished_players";--> statement-breakpoint
ALTER TABLE "battles" DROP CONSTRAINT "battles_host_song_id_songs_id_fk";
--> statement-breakpoint
ALTER TABLE "battles" DROP CONSTRAINT "battles_guest_song_id_songs_id_fk";
--> statement-breakpoint
ALTER TABLE "rounds" ALTER COLUMN "status" DROP DEFAULT;--> statement-breakpoint
ALTER TABLE "rounds" ALTER COLUMN "status" SET DATA TYPE text USING "status"::text;--> statement-breakpoint
UPDATE "rounds" SET "status" = 'WAITING' WHERE "status" = 'SONG_PICKS';--> statement-breakpoint
DROP TYPE "public"."round_status";--> statement-breakpoint
CREATE TYPE "public"."round_status" AS ENUM('WAITING', 'GUESSING', 'FINISHED');--> statement-breakpoint
ALTER TABLE "rounds" ALTER COLUMN "status" SET DATA TYPE "public"."round_status" USING "status"::"public"."round_status";--> statement-breakpoint
ALTER TABLE "rounds" ALTER COLUMN "status" SET DEFAULT 'WAITING'::"public"."round_status";--> statement-breakpoint
ALTER TABLE "battles" DROP COLUMN "host_song_id";--> statement-breakpoint
ALTER TABLE "battles" DROP COLUMN "guest_song_id";--> statement-breakpoint
ALTER TABLE "rounds" ADD CONSTRAINT "rounds_songs_required_after_selection" CHECK ("rounds"."status" = 'WAITING' or ("rounds"."host_song_id" is not null and "rounds"."guest_song_id" is not null));
--> statement-breakpoint
ALTER TABLE "rounds" ADD CONSTRAINT "rounds_finished_players" CHECK ("rounds"."status" <> 'FINISHED' or ("rounds"."host_finished" and "rounds"."guest_finished"));
