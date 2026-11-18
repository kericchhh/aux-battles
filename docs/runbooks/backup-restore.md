# Back up and restore

A backup holds the PostgreSQL database and the song files from the `media`
volume. Always restore both together. In `~/aux-deploy`, export `AUX_VERSION`
first, as in the [deploy runbook](deploy.md#start-and-stop-production).

The examples use bash. In fish, set the drill variable with
`set DRILL docker compose -p aux-restore -f compose.yaml -f compose.restore-drill.yaml`.

## Take a backup

Stop the services that write data, so the database and files match:

```sh
mkdir -p backups
docker compose stop web api worker
docker compose --profile backup run --rm backup
docker compose start web api worker
```

This writes `backups/<UTC timestamp>/` with `db.dump`, a `pg_dump` of the
whole database, and `media.tar.gz`, with `songs/` and `incoming/`. A directory
still named `.<timestamp>.partial` is an incomplete backup; delete it. The
files are readable only by you, because the dump contains password hashes.
They belong to `BACKUP_UID:BACKUP_GID` from `.env`, default `1000:1000`; set
them to your `id -u` and `id -g`, or `0` with rootless Docker. `backups/` is
ignored by Git; keep a copy off this machine.

## Restore drill

Prove a backup works by restoring it into a separate project, `aux-restore`,
next to the running stack:

```sh
BACKUP=20261114T183129Z
DRILL="docker compose -p aux-restore -f compose.yaml -f compose.restore-drill.yaml"
$DRILL up -d --wait db
$DRILL run --rm --no-deps --entrypoint sh backup -c \
  'pg_restore --single-transaction --exit-on-error --no-owner -d "$PGDATABASE" "/backups/'"$BACKUP"'/db.dump"'
$DRILL run --rm --no-deps -v media:/restore --entrypoint sh backup -c \
  'tar -xzf "/backups/'"$BACKUP"'/media.tar.gz" -C /restore && chown 65532:65532 /restore'
$DRILL up -d --wait api
```

Compare row counts with the normal stack; they match if nothing changed since
the backup. The last value is the backup's migration version:

```sh
COUNTS="SELECT (SELECT count(*) FROM users), (SELECT count(*) FROM songs), (SELECT count(*) FROM battles), (SELECT max(version_id) FROM goose_db_version)"
docker compose exec -T db sh -c "psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -Atc \"$COUNTS\""
$DRILL exec -T db sh -c "psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -Atc \"$COUNTS\""
$DRILL down -v
```

## Restore the normal stack

This replaces the current data. Use the code version that took the backup, or
a newer one: migrations can move an old database forward, but old code cannot
run on a newer schema. The drill shows the backup's migration version.

1. Stop the app and back up the current state, in case you need to go back:

   ```sh
   docker compose stop web api worker
   docker compose --profile backup run --rm backup
   ```

2. Recreate the database and restore it:

   ```sh
   BACKUP=20261114T183129Z
   docker compose run --rm --no-deps --entrypoint sh backup -c \
     'dropdb --if-exists "$PGDATABASE" && createdb "$PGDATABASE" && pg_restore --single-transaction --exit-on-error --no-owner -d "$PGDATABASE" "/backups/'"$BACKUP"'/db.dump"'
   ```

3. Replace the song files:

   ```sh
   docker compose run --rm --no-deps -v media:/restore --entrypoint sh backup -c \
     'rm -rf /restore/songs /restore/incoming && tar -xzf "/backups/'"$BACKUP"'/media.tar.gz" -C /restore && chown 65532:65532 /restore'
   ```

4. Start the stack, then sign in and play a song:

   ```sh
   docker compose up -d --no-build
   curl -fsS http://localhost:8080/healthz
   ```

If step 2 fails after `dropdb`, the database is empty: fix the cause and run
step 2 again.
