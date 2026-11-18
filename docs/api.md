# API

The examples assume the API at `http://localhost:8080`. Through the web client
the same routes are under `/api`, for example `http://localhost:8000/api/healthz`.

## Routes

Every route except `/healthz`, `/metrics`, sign-up and login needs a session
cookie.

| Route | Purpose |
| --- | --- |
| `POST /users`, `POST /users/login`, `POST /users/logout` | Sign up, log in and log out |
| `GET /users/me`, `GET /users/me/stats` | Current user, and wins, losses and scores from finished battles |
| `GET /songs?q=&limit=&cursor=` | Ready songs, newest first, paged with `nextCursor` |
| `GET /songs/{id}`, `GET /songs/{id}/audio/{part}` | A song, and its audio; `part` is `full`, `drums`, `bass`, `melody` or `vocals` |
| `POST /songs`, `GET /songs/{id}/status` | Upload an MP3 and follow its processing (admins only) |
| `POST /battles`, `POST /battles/join`, `GET /battles`, `GET /battles/{battleId}` | Create a lobby, join with an invite code, list and view battles |
| `PUT /battles/{battleId}/lineup` | Lock your ordered song choices, one per round |
| `GET /rounds/{roundId}/audio`, `POST /rounds/{roundId}/guess` | Hear your current stem and guess the title |
| `GET /battles/{battleId}/events` | Server-sent events; refetch the battle on each `changed` event |
| `POST /battles/{battleId}/forfeit` | Quit; the opponent wins, or an unjoined lobby is cancelled |
| `GET /healthz`, `GET /metrics` | Health check and Prometheus metrics; the web client does not expose `/metrics` |

Non-participants get `404` for a battle, and conflicting or stale moves get
`409`; refetch the battle and try again.

## Example

Create an account and log in:

```sh
curl -H 'Content-Type: application/json' \
  -d '{"username":"host_player","email":"host@example.com","password":"at-least-8-chars"}' \
  http://localhost:8080/users
curl -c host.cookies -H 'Content-Type: application/json' \
  -d '{"identifier":"host_player","password":"at-least-8-chars"}' \
  http://localhost:8080/users/login
```

Accounts register as `USER`. To upload songs, promote one to admin. Only
upload tracks you have the rights to store, process and play:

```sh
echo "UPDATE users SET role = 'ADMIN' WHERE lower(username) = lower('host_player');" |
  docker compose exec -T db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
curl -b host.cookies -F song=@song.mp3 -F title='Title' -F artist='Artist' \
  -F genre='Genre' -F clipStartSeconds=30 http://localhost:8080/songs
```

Create a second account the same way, saving its cookie to `guest.cookies`.
Then create a battle and join it with the invite code from the battle view:

```sh
curl -b host.cookies -H 'Content-Type: application/json' \
  -d '{"rounds":5,"guessSeconds":30}' http://localhost:8080/battles
curl -b guest.cookies -H 'Content-Type: application/json' \
  -d '{"inviteCode":"ABC123"}' http://localhost:8080/battles/join
```

Then each player sends `{"songIds":[...]}` to the lineup route, and guesses
with `{"guess":"Song title","expectedAttempt":1}`, where `expectedAttempt` is
`myAttempts + 1` from the battle view.

## Game rules

- A battle has 1 to 7 rounds (default 5). Each player picks one ready song per
  round, and hears the opponent's.
- Each wrong guess unlocks the next stem. A correct guess, or four wrong ones,
  ends your round; the next round starts when both players finish.

  | Stem | Points |
  | --- | ---: |
  | Drums | 100 |
  | Bass | 75 |
  | Melody | 50 |
  | Full clip | 20 |

- Each consecutive correct round adds 10 bonus points, up to 20.
- Matching ignores case, accents, punctuation, bracketed text and trailing
  `feat.` credits, and allows small typos in titles of five or more
  characters.
- With `guessSeconds` (`15` or `30`), running out of time counts as a wrong
  guess. A round where neither player guesses cancels the battle.
- Inactive battles end after 15 minutes in the lobby, or 5 minutes in lineup
  selection or a round. A player the battle was waiting on loses; otherwise it
  is cancelled.
