# Deploy

Releases deploy themselves to a local production stack, Compose project
`aux-prod`, in a separate clone at `~/aux-deploy`. GitHub Actions builds and
publishes the images for a version tag and creates the GitHub release; a
systemd user timer on this machine notices the new release and deploys it. GitHub
never connects to this machine, so no ports, SSH keys or deploy credentials are
needed.

Every 5 minutes while you are logged in, the timer runs
[`scripts/deploy.sh`](../../scripts/deploy.sh) with `--if-running`. Each run:

1. Exits if another run holds the lock, if `aux-prod` is stopped, or if the dev
   stack (project `aux-battles`) is running, because both use the same ports.
2. Reads the latest release from the GitHub API and exits if it is already
   deployed or failed before.
3. Fetches the release tag and pulls its API, web and worker images. If any of
   these fails, it exits and production keeps running. It tries again on the next run,
   which also covers a release whose images are still being published.
4. Stops web, api and worker and takes a backup with the `backup` job from the
   [backup runbook](backup-restore.md). If the backup fails, it starts them
   again, marks the release failed and sends a notification.
5. Checks out the tag and runs `docker compose up --wait`, which applies
   migrations, then checks `http://localhost:8000/api/healthz`.
6. On success, records the release and sends a notification. On failure, checks
   out and starts the previous release, marks the new one failed and sends a
   notification.

Notifications go to the Telegram chat configured for Grafana alerts. Production
is down from the start of the backup until the new release is healthy, usually
under a minute.

State lives in `~/.local/state/aux-deploy/`:

| File | Contents |
| --- | --- |
| `current` | The release production runs, for example `v0.2.0` |
| `failed` | Releases that failed, one per line; they are not tried again |
| `lock` | Prevents two runs at once |

## Release rules

- **Migrations only add.** A rollback starts the previous code but does not undo
  migrations, so the previous release must still work on the new schema. Add
  tables, columns and indexes; do not drop or rename them, or add a constraint
  that old code would break, in the same release that stops using them. Remove
  them in a later release instead. If a release breaks this rule, see
  [A release broke the data](#a-release-broke-the-data).
- **Only stable tags deploy.** The GitHub API's latest release skips
  pre-releases, and the script accepts only `vX.Y.Z`. Use a tag such as
  `v0.3.0-rc.1` to publish images without deploying them.
- **Releases from `v0.2.0` onward.** Earlier tags do not use `AUX_VERSION` in
  `compose.yaml`, so they cannot run published images.

## Set up

1. Clone the repository and create its `.env` from `.env.example`, with
   production passwords that differ from the dev stack:

   ```sh
   git clone https://github.com/kericchhh/aux-battles.git ~/aux-deploy
   cp ~/aux-deploy/.env.example ~/aux-deploy/.env
   chmod 600 ~/aux-deploy/.env
   ```

   Set `COMPOSE_PROJECT_NAME=aux-prod` in that `.env`, so manual `docker compose`
   commands in `~/aux-deploy` use the production project and volumes. Fill in
   `TELEGRAM_BOT_TOKEN` and `TELEGRAM_CHAT_ID` to receive notifications.

2. Install the script and the systemd units from this repository, and start the
   timer:

   ```sh
   install -Dm755 scripts/deploy.sh ~/.local/bin/aux-deploy
   install -Dm644 -t ~/.config/systemd/user deploy/aux-deploy.service deploy/aux-deploy.timer
   systemctl --user daemon-reload
   systemctl --user enable --now aux-deploy.timer
   ```

   The timer runs a copy of the script, because a branch switch in this
   repository could change the script while it runs. After changing
   `scripts/deploy.sh`, run the first `install` again.

3. Start production with the first deploy. With prod stopped the timer skips,
   so run the script once yourself; with no `current` file it deploys the
   latest release without a backup:

   ```sh
   ~/.local/bin/aux-deploy
   ```

The images on GitHub Container Registry must be public, or run
`docker login ghcr.io` once with a token that can read packages.

## Check status

```sh
systemctl --user list-timers aux-deploy.timer
journalctl --user -u aux-deploy.service -n 50
cat ~/.local/state/aux-deploy/current
cat ~/.local/state/aux-deploy/failed
```

`failed` does not exist until a release has failed. A run that finds nothing
to do logs only systemd's start and finish lines. To check right away instead
of waiting for the timer, run `systemctl --user start aux-deploy.service`.

## Start and stop production

The timer never starts a stopped production stack, so start it yourself, with
the version from `current`:

```sh
cd ~/aux-deploy
export AUX_VERSION=$(sed 's/^v//' ~/.local/state/aux-deploy/current)
docker compose --profile observability up -d --wait
```

Stop it with `docker compose --profile observability stop` in the same shell.
Every `docker compose` command in `~/aux-deploy`, including those in the
[backup runbook](backup-restore.md), needs `AUX_VERSION` set this way;
without it Compose uses the `dev` images.

To stop automatic deploys, run `systemctl --user disable --now aux-deploy.timer`.

## Notifications

Every message starts with `aux-battles: `.

| Message | Meaning | What to do |
| --- | --- | --- |
| `deployed vX (was vY)` | The release is running and healthy; `vY` is `none` on the first deploy | Nothing |
| `vX failed health checks, rolled back to vY` | The previous release is running again | Check the logs below, fix the release and publish a new tag |
| `vX failed and prod could not be started, check aux-prod` | Neither release started | Check the logs below and [start production](#start-and-stop-production) |
| `backup failed, vX not deployed` | web, api and worker were started again on the previous release and `vX` is marked failed | Check `docker compose ps`, fix the backup, then [retry the release](#retry-a-failed-release) |

To see why a release failed, read the deploy run:

```sh
journalctl --user -u aux-deploy.service -n 100
```

After a rollback the containers belong to the previous release, so
`docker compose logs` no longer shows the failed one. Read the failed release's
API and worker logs in Grafana Explore with `{job="aux-battles"}`, narrowed to
the time of the deploy. Its migration output is not kept.

A failed or interrupted backup can leave a
`~/aux-deploy/backups/.<timestamp>.partial` directory; delete it.

## Retry a failed release

Remove the release from `failed`, then run the deploy:

```sh
sed -i '/^v0.3.0$/d' ~/.local/state/aux-deploy/failed
systemctl --user start aux-deploy.service
```

## Deploy an older release

The script only moves to the latest release. To go back to an older one by
hand, with prod running, take a backup as in the
[backup runbook](backup-restore.md#take-a-backup) with the current
`AUX_VERSION`, then check out the older release, start it and record it:

```sh
cd ~/aux-deploy
git checkout --detach refs/tags/v0.2.0
AUX_VERSION=0.2.0 docker compose --profile observability up -d --wait &&
  echo v0.2.0 > ~/.local/state/aux-deploy/current &&
  echo v0.3.0 >> ~/.local/state/aux-deploy/failed
```

The state files change only if the older release starts. The last line stops
the timer from moving straight back to `v0.3.0`. Older code cannot always run
on a newer schema; if the older release fails, restore the backup taken before
the newer one, as below.

## A release broke the data

If a migration damaged data, or a rollback left the previous release unable to
work with the new schema, restore the first backup taken for the bad
release: the first `backup written to` line in the deploy log after the last
successful deploy. Do not just take the newest backup: a retry or an
interrupted deploy can back up a database the bad release already migrated.

1. Make sure the timer cannot deploy the bad release again: it must be in
   `failed`, or add it.
2. Check out the release that was running when the backup was taken, and set
   its version:

   ```sh
   cd ~/aux-deploy
   git checkout --detach refs/tags/v0.2.0
   export AUX_VERSION=0.2.0
   ```

3. Follow [Restore the normal stack](backup-restore.md#restore-the-normal-stack)
   in the same shell.
4. Record the release with `echo v0.2.0 > ~/.local/state/aux-deploy/current`.

## Interrupted deploys

The timer only runs while you are logged in. Logging out, shutting down, or
the 20-minute unit timeout stops a deploy part way through. The containers
stay in whatever state they reached:

- **During the backup or the new release's start:** web, api and worker may be
  stopped or half started. `current` still holds the old release, so the next
  run after you log in backs up again and retries.
- **During a rollback:** the new release is not yet marked failed, so the next
  run tries it again and rolls back again.

If `current` is lost, the next run deploys the latest release without taking a
backup. Recreate it with the running version before then:
`echo v0.2.0 > ~/.local/state/aux-deploy/current`.

## Housekeeping

Nothing is deleted automatically.

- **Backups:** every deploy adds one to `~/aux-deploy/backups/`. Delete old
  ones, and keep a copy of recent ones somewhere private off this machine.
- **Images:** every release adds three images. List them with
  `docker images 'ghcr.io/kericchhh/aux-battles-*'` and remove versions other
  than the current and previous release with `docker image rm`. Keep the
  `dev` tag, which the dev stack uses.

## Limits

- A release deploys up to about 5 minutes after it is published, plus the time
  to pull its images.
- Nothing deploys while you are logged out or while the dev stack runs.
- The GitHub API allows 60 unauthenticated requests per hour; the timer makes
  12.
- There is one host and no zero-downtime deploy: production is down from the
  backup until the new release is healthy.
