ALTER TABLE "battles" ADD COLUMN "current_round" integer DEFAULT 1 NOT NULL;--> statement-breakpoint
ALTER TABLE "battles" ADD COLUMN "host_song_id" uuid;--> statement-breakpoint
ALTER TABLE "battles" ADD COLUMN "guest_song_id" uuid;--> statement-breakpoint
ALTER TABLE "battles" ADD CONSTRAINT "battles_host_song_id_songs_id_fk" FOREIGN KEY ("host_song_id") REFERENCES "public"."songs"("id") ON DELETE no action ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "battles" ADD CONSTRAINT "battles_guest_song_id_songs_id_fk" FOREIGN KEY ("guest_song_id") REFERENCES "public"."songs"("id") ON DELETE no action ON UPDATE no action;