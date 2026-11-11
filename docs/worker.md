# Song processing

The worker turns uploaded songs into the clip and stems used in battles. To run
it, see the [development guide](development.md).

## Jobs and media

Both processes use `./storage` by default. Set the same `MEDIA_ROOT` for both
when using another directory. Uploads create queued jobs. The worker extracts
the selected 12-second clip, creates four stems, and changes the song status to
`READY` after all files are present. A failed job makes up to three attempts;
expired leases let another worker resume after a crash. Only `READY` songs
appear in the catalog. Each processing attempt is limited to 30 minutes by
default; set `WORKER_ATTEMPT_TIMEOUT` to a positive Go duration such as `45m`
to change it. The worker also checks the media volume at startup and hourly,
removing old abandoned attempts and inputs while preserving active attempts
and files referenced by songs.

## Process handling

The worker keeps only the last 2,000 bytes of FFmpeg and Demucs output for
failure diagnostics. On Linux, macOS, and FreeBSD, cancellation terminates the
command's process group, including descendants that remain in that group.
Other platforms use Go's direct-process cancellation. Waiting on inherited
output pipes is bounded by two seconds after cancellation or process exit.

While processing, the worker renews the job lease every minute. Each renewal
query has a five-second deadline. A renewal failure cancels processing, removes
that attempt's output, and retains the input so the job can be recovered after
lease expiry. If the outcome of the final database update is unknown, the
worker retains both the input and published output to avoid deleting files
that the database may already reference.

Internally, worker iterations distinguish idle polling, completed processing,
scheduled retries, permanent failures, polling errors, and unrecorded attempts.
A successfully recorded retry or permanent failure is an outcome rather than
an operational error; the processing cause remains in the logs. Attempt counters
retain their `succeeded` and `failed` labels.
