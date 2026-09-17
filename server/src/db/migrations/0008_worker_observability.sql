CREATE TABLE "worker_heartbeats" (
	"name" varchar(64) PRIMARY KEY NOT NULL,
	"instance_id" uuid NOT NULL,
	"last_seen_at" timestamp with time zone NOT NULL
);
--> statement-breakpoint
ALTER TABLE "songs" ADD COLUMN "processing_error" varchar(500);