# Deploy

This is the single-host deploy: a Compose stack, project `aux-prod`, in a
separate clone at `~/aux-deploy`. The Kubernetes deploy is in the
[Kubernetes guide](../kubernetes.md).

Every 5 minutes, a systemd user timer runs
[`scripts/deploy.sh`](../../scripts/deploy.sh). GitHub never connects to the
machine. Each run:

1. Exits if production is stopped, the dev stack is running, or the latest
   GitHub release is already deployed or has failed before.
2. Pulls the release's images, then stops the app and takes a
   [backup](backup-restore.md#take-a-backup).
3. Starts the release, which applies migrations, and checks
   `http://localhost:8000/api/healthz`.
4. On failure, starts the previous release again and marks the new one failed.
5. Sends the result to the Telegram chat from `.env`.

State lives in `~/.local/state/aux-deploy/`: `current` holds the running
release and `failed` lists releases that are not tried again.

## Release

Tag a commit on `main`. CI publishes the images and the GitHub release, and the
timer deploys it:

```sh
git tag v0.4.0
git push origin v0.4.0
```

- **Migrations only add.** A rollback does not undo migrations, so the
  previous release must still work on the new schema. Drop or rename columns in
  a later release.
- **Only stable tags deploy.** A tag such as `v0.4.0-rc.1` publishes images
  without deploying them.

## Set up

```sh
git clone https://github.com/kericchhh/aux-battles.git ~/aux-deploy
cp ~/aux-deploy/.env.example ~/aux-deploy/.env
chmod 600 ~/aux-deploy/.env
```

In that `.env`, set `COMPOSE_PROJECT_NAME=aux-prod`, production passwords and
the Telegram values. Use only letters and digits in `POSTGRES_PASSWORD`,
because it goes into a connection URL, and choose it before the first start:
PostgreSQL applies it only when its volume is created. The images on GHCR must
be public, or run `docker login ghcr.io` once.

Then install the script and timer from this repository and run the first
deploy by hand:

```sh
install -Dm755 scripts/deploy.sh ~/.local/bin/aux-deploy
install -Dm644 -t ~/.config/systemd/user deploy/aux-deploy.service deploy/aux-deploy.timer
systemctl --user daemon-reload
systemctl --user enable --now aux-deploy.timer
~/.local/bin/aux-deploy
```

Run the first `install` again after changing `scripts/deploy.sh`.

## Check status

```sh
systemctl --user list-timers aux-deploy.timer
journalctl --user -u aux-deploy.service -n 50
cat ~/.local/state/aux-deploy/current
```

To deploy now instead of waiting, run `systemctl --user start aux-deploy.service`.

## Start and stop production

The timer never starts a stopped stack. Every `docker compose` command in
`~/aux-deploy` needs `AUX_VERSION`, or it uses the `dev` images:

```sh
cd ~/aux-deploy
export AUX_VERSION=$(sed 's/^v//' ~/.local/state/aux-deploy/current)
docker compose --profile observability up -d --wait
```

Stop it with `docker compose --profile observability stop` in the same shell.
To pause automatic deploys, run `systemctl --user disable --now aux-deploy.timer`.

## Retry a failed release

```sh
sed -i '/^v0.4.0$/d' ~/.local/state/aux-deploy/failed
systemctl --user start aux-deploy.service
```

## Roll back

The script only moves forward. To go back by hand, back up the current
release, then start the older one and record it:

```sh
cd ~/aux-deploy
export AUX_VERSION=$(sed 's/^v//' ~/.local/state/aux-deploy/current)
docker compose stop web api worker
docker compose --profile backup run --rm backup
git checkout --detach refs/tags/v0.3.0
AUX_VERSION=0.3.0 docker compose --profile observability up -d --wait &&
  echo v0.3.0 > ~/.local/state/aux-deploy/current &&
  echo v0.4.0 >> ~/.local/state/aux-deploy/failed
```

The `failed` line stops the timer from moving straight back to `v0.4.0`.

If a migration damaged data, use the first backup the bad release took, from
the first `backup written to` line in the deploy log. Check out the older tag,
run `export AUX_VERSION=0.3.0`, follow
[Restore the normal stack](backup-restore.md#restore-the-normal-stack) in the
same shell, then update `current` and `failed` as above.

## Housekeeping

Nothing is deleted automatically. Every deploy adds a backup to
`~/aux-deploy/backups/` and three images; remove old ones by hand, and keep a
recent backup off this machine.
