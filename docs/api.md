# API

The examples assume the API runs at `http://localhost:8080`. Setup is in the
[development guide](development.md).

## Catalog and uploads

All song routes require a session. `GET /songs` lists ready songs, newest
first, as `{"songs":[...],"nextCursor":...}`. Optional query parameters:

- `q` matches title or artist, ignoring case (at most 100 characters);
- `limit` sets the page size from 1 to 100 (default 50);
- `cursor` continues from the previous page's `nextCursor`, which is `null` on
  the last page.

Pages are ordered by upload time, so a song that finishes processing after a
client has paged past its upload time appears only when the list is reloaded.

Only administrators can upload with `POST /songs` and read processing progress
at `GET /songs/{id}/status`; other users receive `403`. Accounts register as
`USER`. To promote one, run:

```sh
docker compose exec db psql -U aux -d aux_battles \
  -c "UPDATE users SET role = 'ADMIN' WHERE lower(username) = lower('USERNAME')"
```

The role is read on every request, so the change applies to existing sessions.

Signed-in clients can fetch a ready song at `GET /songs/{id}/audio/{part}`.
`part` is `full`, `drums`, `bass`, `melody`, or `vocals`. The response is MP3
audio and supports HTTP byte ranges.

## Battle lobbies

Battle routes require a valid session cookie. Create an account at `POST /users`
and log in at `POST /users/login` to obtain one. Use separate cookie jars for
host and guest accounts. With those jars saved as `host.cookies` and
`guest.cookies`, create and join a lobby:

```sh
curl -b host.cookies -H 'Content-Type: application/json' \
  -d '{"rounds":7,"guessSeconds":30}' http://localhost:8080/battles
curl -b host.cookies http://localhost:8080/battles/BATTLE_ID
curl -b guest.cookies -H 'Content-Type: application/json' \
  -d '{"inviteCode":"ABC123"}' http://localhost:8080/battles/join
```

Replace `ABC123` with the invite code from the host's battle view, and
`BATTLE_ID` with the ID returned when creating the battle. Omitting `rounds`
from the JSON object defaults to five; the allowed range is one to seven.
`guessSeconds` sets a time limit per stem of `15` or `30` seconds; `null` or
omitting it turns the timer off. See [Guess timer](#guess-timer).
Joining creates all rounds and changes the lobby from `PENDING` to `SELECTING`
in one transaction. Only one guest can join; self-joins and subsequent joins
return `409`. An unknown invite code returns `404`. Battle reads return `404`
for both missing battles and nonparticipants, and use `Cache-Control: no-store`.
The view follows the reference client's field names and hides song answers until
both players finish a round (see `completedRounds`).

`GET /battles` lists the caller's 20 most relevant battles: unfinished ones
first, then by most recent activity. Each entry has the battle's `id`,
`status`, `rounds`, `currentRound`, `opponent` (`{"username":...}` or `null`),
`myScore`, `opponentScore`, `outcome`, `endReason` and `lastActivityAt`.
`outcome` is `null` unless the battle is `FINISHED`, and `endReason` is `null`
until the battle ends. Only the host of a `PENDING` lobby receives its
`inviteCode`.

While a battle is `ONGOING`, scores in the list and the battle view count only
finished rounds, so they never reveal the opponent's points or stage in the
current round. `round.opponentFinished` still shows whether the opponent has
finished it. Once the battle ends, scores include every round, so a battle that
ends by forfeit or timeout mid-round counts that round's partial points even
though it does not appear in `completedRounds`.

Each player locks an ordered lineup using
`PUT /battles/{battleId}/lineup`. For a two-round battle, the body is:

```json
{"songIds":["FIRST_SONG_UUID","SECOND_SONG_UUID"]}
```

Replace the placeholders with IDs from the curated catalog. Only upload and
catalog tracks whose permissions cover storage, processing, and game playback.
`READY` indicates successful processing, not verification of those permissions.

Choose exactly one distinct song per round. Songs must be `READY` and all five
audio files must be readable. The order of IDs determines the round order.
Both players may choose from the same catalog, but neither can repeat a song
within their own lineup. Once submitted, a lineup cannot be changed.

The first successful submission returns `WAITING_ON_OPPONENT`. Once both lineups
are locked, the response is `BATTLE_STARTED`, the battle becomes `ONGOING`, and
round one becomes `GUESSING`. These state changes are committed together.
Subsequent submissions return `409`; nonparticipants receive `404`. Battle views
show each player's lineup readiness without exposing selected song IDs. They
also include `me` and `opponent` (`{"username":...}`; `opponent` is `null` until
someone joins). To follow progress without polling, see [Live
updates](#live-updates). Battles can also end early; see [Ending battles
early](#ending-battles-early).

## Playing rounds

Participants listen through `GET /rounds/{roundId}/audio` and submit guesses to
`POST /rounds/{roundId}/guess`. Take the active round ID from the battle view:

```sh
curl -b host.cookies http://localhost:8080/rounds/ROUND_ID/audio -o round.mp3
curl -b host.cookies -H 'Content-Type: application/json' \
  -d '{"guess":"Song title","expectedAttempt":1}' \
  http://localhost:8080/rounds/ROUND_ID/guess
```

The audio route chooses the opponent's selection and the caller's current stage;
clients cannot select a later stage through this route. It supports byte ranges
and sends `Cache-Control: private, no-store`. Nonparticipants receive `404`.
Future, previous, and finished-battle rounds return `409` to participants.
A player who finishes first can replay their last unlocked stage while waiting.

Each wrong guess unlocks the next stage. A correct answer, or four incorrect
answers, finishes the player's round. Both players must finish before the next
round starts. Completing the final round records the winner, or a draw when
scores are equal. A battle has at most seven rounds.

| Stage | Base points for a correct answer |
| --- | ---: |
| Drums | 100 |
| Bass | 75 |
| Melody | 50 |
| Full clip | 20 |

A correct answer adds 10 points per preceding consecutive round answered
correctly, capped at 20 bonus points. Three consecutive drums answers score
100, 110, and 120. Four wrong guesses score zero and reset that player's streak.
The bonus applies only to a correct answer; wrong guesses within a round do not
reset the streak unless the player exhausts all four attempts. Guess responses
separate `basePoints`, `streakBonus`, and their sum, `pointsAwarded`.

Send `expectedAttempt` from 1 to 4, using `myAttempts + 1` from the battle view.
Concurrent duplicates and stale attempt numbers return `409`; refresh the view
before submitting again. Guesses, scores, round completion, and battle progress
commit together. If a response is lost, refresh rather than assuming the request
failed: the database may already have committed it.

Once both players finish a round, the battle view's `completedRounds` lists it
with `number`, `myPoints`, `opponentPoints`, `mySong` (the caller's pick) and
`opponentSong` (the song the caller guessed), each `{"title","artist"}`. A round
that is still being played is never listed.

Guesses must contain a title of at most 255 characters. Matching ignores case,
accents, punctuation, bracketed annotations, and trailing `feat.`, `ft.` or
`featuring` credits; a bare `feat` or `ft` word is part of the title. A credit
is kept when removing it would leave fewer than five normalized characters.
Non-Latin letters are retained. Titles shorter than five normalized characters
require an exact match; longer titles allow an edit distance of at most 15% of
the normalized title length, rounded down, with a minimum of one. Empty
normalized guesses are rejected. Responses exclude the answer, opponent song
IDs, and storage paths.

The authenticated catalog audio route remains available for browsing curated
tracks. Round-level stage gating does not prevent someone from identifying a
song using the catalog or an external recognition tool; it is not an anti-cheat
system.

## Guess timer

In a battle created with `guessSeconds`, each stem has that many seconds. A
player's clock starts when the stem unlocks: both clocks start when a round
begins, and a player's clock restarts each time they reach the next stem. The
battle view shows `guessSeconds` and, while the caller is guessing,
`round.stageEndsAt`; both are `null` without a timer. Every view also includes
`serverTime`, so clients can count down using the server's clock rather than
the device's.

Running out of time counts as a wrong guess. The server records a timed-out
attempt worth 0 points and unlocks the next stem, whose clock starts at the
missed deadline. Running out on the full clip finishes the player's round with
0 points, like four wrong guesses. A player who is away for several deadlines
receives one timeout per missed stem. If a round ends without either player
guessing at all, the battle is cancelled with end reason `TIMEOUT` and does not
count in stats.

The API applies due timeouts every second, after a one-second grace period for
network delay. A guess for an attempt that has already timed out returns `409`
with "time ran out on this stem; refresh the battle", whether the API recorded
the timeout just before or while handling the guess. Timeouts count toward
`myAttempts` but are not moves.

## Live updates

`GET /battles/{battleId}/events` streams
[Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events)
to participants. It sends a `changed` event when the stream opens and whenever
the battle or its rounds change, including the opponent's moves, timeouts,
forfeits and inactivity endings. Events carry no data; refetch the battle view
after each one. Comment lines keep the connection alive every 25 seconds.

```sh
curl -N -b guest.cookies http://localhost:8080/battles/BATTLE_ID/events
```

Nonparticipants receive `404`. The server closes each stream after 30 minutes,
and when it shuts down; browsers using `EventSource` reconnect automatically,
which also checks the session again. If the API loses its database listener,
it reconnects and sends `changed` to every open stream, because changes made
while it was disconnected are not replayed.

## Player stats

`GET /users/me/stats` returns the caller's `played`, `wins`, `losses`, `draws`,
`bestScore` and `averageScore` (rounded). Only `FINISHED` battles count,
including those that ended by forfeit or inactivity; battles cancelled for any
reason, including an idle timed round, do not. A
battle's score is the player's total across all its rounds.

## Ending battles early

A participant can quit with `POST /battles/{battleId}/forfeit`:

```sh
curl -b guest.cookies -X POST http://localhost:8080/battles/BATTLE_ID/forfeit
```

Forfeiting an unjoined lobby cancels it. Forfeiting during lineup selection or
a round finishes the battle and awards it to the opponent. The response is the
updated battle view; ended battles return `409` and nonparticipants `404`.

Joining, submitting a lineup and guessing count as moves; guess timer
timeouts do not. Once a minute, the
API ends battles without a recent move: lobbies after 15 minutes, lineup
selection and rounds after 5. If the battle is waiting on only one player (an
unlocked lineup, or an unfinished round the opponent has finished), that player
loses. Otherwise the battle is cancelled with no winner.

Battle views include `endReason` (`COMPLETED`, `FORFEIT` or `TIMEOUT`) once a
battle ends. Cancelled battles have status `CANCELLED` and a `null` outcome.
