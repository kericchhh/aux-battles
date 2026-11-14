# Back up and restore

A backup holds the PostgreSQL database and the song files from the `media`
volume. Restoring only one of them leaves songs whose database rows point to
missing audio, or audio files that no song uses.

The examples use a POSIX shell. In fish, set variables with `set` and leave
`DRILL` unquoted so it stays a list of words:
`set DRILL docker compose -p aux-restore -f compose.yaml -f compose.restore-drill.yaml`.

## Take a backup

The `backup` service is a one-shot job in the `backup` Compose profile. It
reads the credentials in `.env` and starts `db` if it is not running. Stop the
services that write to the database and media first, so the database dump and
the media archive describe the same moment:

```sh
mkdir -p backups
docker compose stop web api worker
docker compose --profile backup run --rm backup
docker compose start web api worker
```

The worker deletes upload files when jobs finish and cleans up old attempts,
and the API writes new uploads. A backup taken while they run can contain a
queued song whose upload is missing from the archive, or fail because a file
changed while it was being read.

The job writes `backups/<UTC timestamp>/` with two files:

| File | Contents |
| --- | --- |
| `db.dump` | `pg_dump` custom-format dump of the whole database |
| `media.tar.gz` | `songs/` and `incoming/` from the `media` volume |

`work/` holds temporary processing files and `cache/` holds the Demucs model,
which the worker downloads again, so neither is included. A new install with no
processed songs yet produces an empty media archive.

How the job works:

- It uses the `postgres:18` image, so `pg_dump` matches the server version.
- It runs as root because the API and worker keep media readable only by their
  own user. The `media` volume is mounted read-only, so the job cannot change
  it.
- It writes into a hidden `.<timestamp>.partial` directory and renames it when
  both files are complete. A directory without `.partial` is a complete backup.
  If a run fails, delete the `.partial` directory it left.
- It creates files readable only by their owner, because the dump contains
  password hashes, email addresses and session token hashes.
- When it exits, including after a failure or `docker stop`, it gives the new
  backup to `BACKUP_UID:BACKUP_GID` (default `1000:1000`), so you can delete
  old backups without `sudo`. Set both in `.env` if `id -u` and `id -g` show
  other values. With rootless Docker, container root is already your user, so
  set them to `0`.

`backups/` is ignored by Git. Keep at least one copy somewhere private off this
machine.

To check that a dump is readable without restoring it:

```sh
BACKUP=20261114T183129Z
docker compose --profile backup run --rm --no-deps --entrypoint pg_restore backup --list "/backups/$BACKUP/db.dump"
```

## Restore drill

Run this after changing the backup job, and from time to time, to prove that a
backup can be restored. It restores into a separate Compose project,
`aux-restore`, with its own empty volumes. `compose.restore-drill.yaml` removes
the host ports so the drill can run next to the normal stack. It never touches
the normal database or media.

1. Choose a backup and start an empty database:

   ```sh
   BACKUP=20261114T183129Z
   DRILL="docker compose -p aux-restore -f compose.yaml -f compose.restore-drill.yaml"
   $DRILL up -d --wait db
   ```

2. Restore the database. `--single-transaction` makes the restore all or
   nothing:

   ```sh
   $DRILL run --rm --no-deps --entrypoint sh backup -c \
     'pg_restore --single-transaction --exit-on-error --no-owner -d "$PGDATABASE" "/backups/'"$BACKUP"'/db.dump"'
   ```

3. Restore the media. Extracting as root keeps the original file owners; the
   `chown` gives the volume's top directory to the app user so the worker can
   create its `work/` and `cache/` directories:

   ```sh
   $DRILL run --rm --no-deps -v media:/restore --entrypoint sh backup -c \
     'tar -xzf "/backups/'"$BACKUP"'/media.tar.gz" -C /restore && chown 65532:65532 /restore'
   ```

4. Start the API. `migrate` runs first and applies any migrations newer than
   the backup. Then compare row counts with the normal stack; the two lines
   should match if nothing has changed since the backup:

   ```sh
   $DRILL up -d --wait api
   COUNTS="SELECT (SELECT count(*) FROM users), (SELECT count(*) FROM songs), (SELECT count(*) FROM battles), (SELECT max(version_id) FROM goose_db_version)"
   docker compose exec -T db sh -c "psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -Atc \"$COUNTS\""
   $DRILL exec -T db sh -c "psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -Atc \"$COUNTS\""
   ```

5. Check that the API user can read every file of every ready song:

   ```sh
   $DRILL exec -T db sh -c "psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -Atc \"SELECT unnest(ARRAY[full_song_path, drums_path, bass_path, melody_path, vocals_path]) FROM songs WHERE status = 'READY'\"" |
     $DRILL exec -T api bash -c 'checked=0; missing=0; while read -r f; do checked=$((checked + 1)); [ -r "$f" ] || { echo "missing $f"; missing=$((missing + 1)); }; done; echo "checked $checked files, $missing missing"'
   ```

6. Remove the drill stack and its volumes:

   ```sh
   $DRILL down -v
   ```

The drill passes when `pg_restore` and `tar` exit without errors, the counts
match, and step 5 checks five files per ready song with none missing. Record
the date and the backup used.

## Restore the normal stack

This replaces the current database and song files with the backup. Restore
with the code version that was running when the backup was taken, or a newer
one: `migrate` can bring an older database forward, but older code cannot run
against a newer schema. Step 4 of the drill shows the backup's migration
version.

1. Stop everything that writes to the database or media, then back up the
   current state so you can go back if the restore is wrong:

   ```sh
   docker compose stop web api worker
   docker compose --profile backup run --rm backup
   ```

2. Recreate the database empty and restore it. Restoring into an empty
   database means nothing from the current schema is left behind:

   ```sh
   BACKUP=20261114T183129Z
   docker compose run --rm --no-deps --entrypoint sh backup -c \
     'dropdb --if-exists "$PGDATABASE" && createdb "$PGDATABASE" && pg_restore --single-transaction --exit-on-error --no-owner -d "$PGDATABASE" "/backups/'"$BACKUP"'/db.dump"'
   ```

3. Replace the song files. Files newer than the backup are removed, because a
   restored song that is processed again cannot publish over a newer attempt
   directory with the same name:

   ```sh
   docker compose run --rm --no-deps -v media:/restore --entrypoint sh backup -c \
     'rm -rf /restore/songs /restore/incoming && tar -xzf "/backups/'"$BACKUP"'/media.tar.gz" -C /restore && chown 65532:65532 /restore'
   ```

4. Start the stack and check it:

   ```sh
   docker compose up -d --no-build
   curl -fsS http://localhost:8080/healthz
   ```

   Sign in with an account that existed at backup time and play a song.

Sessions come back as they were at backup time. Sessions created since then no
longer exist, and sessions that were logged out since then work again until
they expire. To sign everyone out after a restore:

```sh
docker compose exec -T db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "DELETE FROM sessions"'
```

If step 2 fails after `dropdb`, the database is empty. Fix the cause and run
step 2 again, or restore the backup taken in step 1 the same way.
