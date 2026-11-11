# Migration notes

Migrations live in `db/migrations` and are applied with goose, as shown in the
[development guide](development.md). These notes cover migrations that need
care on a database that already has data.

## 00003_battle_round_limit

Migration `00003_battle_round_limit.sql` enforces the seven-round limit for
existing databases. It fails without converting or deleting data if an existing
battle exceeds seven rounds. Inspect such battles before migrating:

```sql
SELECT id, status, round_count FROM battles WHERE round_count > 7;
```

Resolve those records deliberately before rerunning migrations. The migration's
rollback removes the new constraint and restores the previous maximum of ten.

## 00004_round_scoring

Migration `00004_round_scoring.sql` adds streak bonuses; apply it before running
an API that awards them. It preserves existing scores, raises the per-round
ceiling to 120, and permits the base-plus-bonus guess scores. Existing battle
views can still read these scores. Rollback restores the old checks only when
every stored score satisfies them. It refuses incompatible scores atomically
without changing or deleting data; inspect and decide how to handle affected
matches before retrying rollback:

```sql
SELECT id, host_points, guest_points FROM rounds
WHERE host_points > 100 OR guest_points > 100;
SELECT id, points_awarded FROM guesses
WHERE points_awarded NOT IN (0, 20, 50, 75, 100);
```

Do not roll back while an API that awards streak bonuses is still accepting
guesses.

## 00005_battle_endings

Migration `00005_battle_endings.sql` records activity times and end reasons and
marks existing finished battles `COMPLETED`. Battles still running when it is
applied count as active at that moment, so none can time out until five or
fifteen minutes after the migration. Its rollback refuses, without
changing data, while any battle is `CANCELLED`; it drops recorded end reasons.

## 00006_list_indexes

Migration `00006_list_indexes.sql` adds indexes for the battle list and the
catalog. Its rollback drops them.

## 00007_guess_timer_and_events

Migration `00007_guess_timer_and_events.sql` adds the optional per-battle guess
timer, per-player stem start times, a `timed_out` flag on guesses, and triggers
that send a `battle_events` notification when a battle or round changes.
Existing battles have no timer, and rounds in progress get no deadline. Rollback
drops the triggers and columns, including the timer settings and timeout flags;
timed-out attempts remain as wrong guesses with an empty guess.
