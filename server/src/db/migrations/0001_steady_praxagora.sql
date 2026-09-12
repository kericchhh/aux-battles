CREATE TABLE "songs" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"title" varchar(255) NOT NULL,
	"artist" varchar(255) NOT NULL,
	"genre" varchar(50) NOT NULL,
	"cover_image" varchar,
	"album" varchar(255),
	"duration" integer NOT NULL,
	"status" varchar(20) DEFAULT 'PROCESSING' NOT NULL
);
