# Aux Battles

Aux Battles is a real-time music guessing game. A host creates a battle, shares an invite code, and an opponent joins from a separate account. Players pick songs and take turns guessing the song title across several rounds.

## Table of contents

- [Project structure](#project-structure)
- [Requirements](#requirements)
- [Backend setup](#backend-setup)
- [Database setup](#database-setup)
- [Frontend setup](#frontend-setup)
- [Adding songs](#adding-songs)
- [Authentication](#authentication)
- [Testing two players](#testing-two-players)
- [API routes](#api-routes)
## Project structure

```text
aux-battles/
├── server/   Express 5 API, PostgreSQL, Drizzle ORM, Socket.IO
└── client/   React + Vite frontend
```
## Requirements
- Node.js 20 or newer
- PostgreSQL 14 or newer
- npm
- FFmpeg and ffprobe
- Python with Demucs installed in `worker/venv`

## Backend setup
```code
cd server
npm install
cp .env.example .env
```
#### Set DATABASE_URL in .env:
```code
DATABASE_URL=postgres://postgres:password@localhost:5432/aux_battles
PORT=5000
NODE_ENV=development
CLIENT_ORIGIN=http://localhost:5173
```
#### Start the API:
```code
npm run dev
```
#### Start the song-processing worker in another terminal:
```code
npm run worker
```
#### The API runs at http://localhost:5000.

## Database migrations

#### The schema is defined in server/src/db/schema.ts and migrations are stored in server/src/db/migrations.
#### Generate a migration after changing the schema:
```code
cd server
npx drizzle-kit generate --config=./drizzle.config.ts
```
#### Apply migrations:
```code
npx drizzle-kit migrate --config=./drizzle.config.ts
```
```text
For a local development database where migration history is out of sync, inspect the proposed changes carefully before using:
code
npx drizzle-kit push --config=./drizzle.config.ts
```
Never truncate rounds, guesses, or other tables just to add a constraint. Resolve duplicate data first.
#### Confirm the session table exists:
```code
psql "$DATABASE_URL" -c "SELECT to_regclass('public.sessions');"
```

## Frontend setup
```code
cd client
npm install
```
#### Create client/.env:
```code
VITE_API_URL=http://localhost:5000
```
#### Start Vite:
```code
npm run dev
```
#### Open http://localhost:5173.

#### The frontend uses cookie-based authentication. API requests must include credentials:
```code
fetch(`${API_URL}${path}`, {
  ...options,
  credentials: "include",
});
```
```text
Do not store the session token in local storage. The server stores only a hash and sends the raw token as an HttpOnly cookie.
```
## Adding songs

Signed-in players can upload songs from the song-selection menu during a battle. Uploads enter the shared song catalog after background processing succeeds. The request uses multipart form data and must include:

- `song`: the audio file
- `title`: the song title
- `artist`: the artist name
- `genre`: the genre
- `duration`: the source duration in seconds
- `clipStartSeconds`: where the 12-second game clip should begin

Example using cURL:

```bash
curl -X POST http://localhost:5000/songs \
  -H "Cookie: aux_session=YOUR_SESSION_COOKIE" \
  -F "title=Example Song" \
  -F "artist=Example Artist" \
  -F "genre=Rock" \
  -F "duration=180" \
  -F "clipStartSeconds=30" \
  -F "song=@/absolute/path/to/example.mp3"
```
Generating stems with Demucs
Aux Battles uses Demucs to separate an uploaded song into vocals, drums, bass, and melody/other stems.

1.Install Python and FFmpeg first:
```bash
sudo apt install ffmpeg
```
2. Create the worker's Python virtual environment from the repository root:
```bash
python3 -m venv worker/venv
source worker/venv/bin/activate
pip install demucs
```
3. Test Demucs manually:
```bash
python -m demucs \
  -n htdemucs \
  --mp3 \
  --out uploads/stems \
  /absolute/path/to/song.mp3
```
__Demucs creates files similar to:__
```text
uploads/stems/htdemucs/song/
├── bass.mp3
├── drums.mp3
├── other.mp3
└── vocals.mp3
```
#### The API queues processing after an upload. The separate worker extracts a 12-second clip, runs Demucs, and stores paths for:
```text
fullSongPath
drumsPath
bassPath
melodyPath
vocalsPath
```
Make sure the API and worker can both create and read the configured media directory. The default is `storage/` in the repository root and can be changed with `MEDIA_ROOT`.
## Authentication flow

1. POST /users/register creates an account.

2. POST /users/login verifies the identifier and password and sets the session cookie.

3. GET /users/me returns the current user.

4. POST /users/logout revokes the session and clears the cookie.

__The login and registration body fields are__:
```code
{
  "identifier": "username-or-email",
  "password": "your-password"
}
```
> Registration uses username, email, and password.

## Two-player local testing

1. Use two separate browser profiles (or a normal window and a private window) and create two different accounts.
2. Log in as player one and create a battle.
3. Copy the six-character invite code.
4. Log in as player two in the second profile.
5. Select Join with a code and enter the invite code.
6. Confirm that GET /users/me returns a different user ID in each profile.

## Main API routes

| Method | Route | Purpose |
|---|---|---|
| POST | `/users/register` | Create an account |
| POST | `/users/login` | Start a session |
| GET | `/users/me` | Read the current user |
| POST | `/users/logout` | End a session |
| POST | `/battles` | Create a battle |
| POST | `/battles/join` | Join with an invite code |
| GET | `/battles/:battleId` |Read battle state |
| POST | `/battles/:battleId/rounds/picks` |Pick a song |
| POST | `/rounds/:roundId/guess` | Submit a guess |
| GET | `/rounds/:roundId/audio` |Stream authorized round audio |
| POST | `/songs` | Queue an authenticated MP3 upload |
| GET | `/songs/:id/status` | Read authenticated processing status |

> All battle and round routes require an authenticated session.

### Game rules and consistency

- A battle has one host and at most one guest.
- A user cannot join their own battle.
- Each battle round has a unique (battle_id, round_number) pair.
- Each player has at most one guess per attempt.
- Guess submissions include the expected attempt and round state to prevent stale or duplicate requests.
- Song answers are checked on the server.
- Audio access is authorized per round and player.
