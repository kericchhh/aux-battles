import { db } from "./index.js";
import { songsTable } from "./schema.js";

await db.insert(songsTable).values([
    {
        title: "Bohemian Rhapsody",
        artist: "Queen",
        genre: "ROCK",
        duration: 355
    },
    {
        title: "Numb",
        artist: "Linkin Park",
        genre: "ROCK",
        duration: 186
    },
    {
        title: "Lose Yourself",
        artist: "Eminem",
        genre: "HIP-HOP",
        duration: 320
    },
    {
        title: "Thriller",
        artist: "Michael Jackson",
        genre: "POP",
        duration: 358
    },
]);
