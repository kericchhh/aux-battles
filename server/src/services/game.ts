import { db } from "../db/index.js";
import type { Transaction } from "../db/types.js";
import * as battles from "../db/queries/battles.js";
import * as rounds from "../db/queries/rounds.js";
import * as songs from "../db/queries/songs.js";
import { AppError } from "../utils/AppError.js";
import { isCloseMatch } from "../utils/fuzzyMatch.js";
import { createBattleInput, joinBattleInput, pickSongInput, guessInput } from "../validation/game.js";

type Battle = NonNullable<Awaited<ReturnType<typeof battles.getBattleById>>>;
type Round = NonNullable<Awaited<ReturnType<typeof rounds.getRoundById>>>;
const STAGES = ["DRUM", "BASS", "MELODY", "FULL"] as const;
const POINTS = {
    DRUM: 100,
    BASS: 75,
    MELODY: 50,
    FULL: 20
} as const;

function required<T>(value: T | null | undefined, message: string, status = 409): T {
    if (value == null) throw new AppError(message,status);
    return value;
}

function assertParticipant(battle: Battle, userId: string) {
  if (battle.hostId !== userId && battle.guestId !== userId) {
    throw new AppError("You are not a participant of this battle", 403);
  }
}

async function lockBattle(battleId: string, userId: string, tx: Transaction) {
    const battle = required(await battles.getBattleForUpdate(battleId, tx), "Battle not found", 404);
    assertParticipant(battle, userId);
    return battle;
}

async function lockActiveRound(battle: Battle, roundId: string, tx: Transaction) {
    if (battle.status !== "ONGOING") throw new AppError("Battle is not active", 409);
    const round = required(await rounds.getCurrentRoundForUpdate(battle.id,battle.currentRound, tx), "Current round not found");
    if (round.id !== roundId) throw new AppError("The current round has changed, refresh the battle", 409);
    return round;
}

export async function createBattle(userId: string, roundCount = 5) {
    const input = createBattleInput.parse({ rounds: roundCount});
    for (let attempt = 0; attempt < 5; attempt++) {
        const battle = await battles.createBattle({ hostId: userId, rounds: input.rounds });
        if (battle) return {id: battle.id}
    }
    throw new AppError("Could not allocate an invite code; retry", 503)
}

export async function joinBattle(code: string, userId: string) {
    const { inviteCode } = joinBattleInput.parse({ inviteCode: code });
    return db.transaction(async (tx) => {
        const existing = required(await battles.getBattleByInvite(inviteCode,tx), "Battle not found", 404)
        const battle = required(await battles.joinBattle(existing.id, userId, tx), "Lobby is full, already started or belongs to you")
        required(await rounds.createRound({battleId: battle.id, roundNumber: 1, status: "SONG_PICKS"}, tx), "Could not create round")
        return {id: battle.id};
    })
}

export async function pickSong(battleId: string, userId: string, roundId: string, songId: string) {
  pickSongInput.parse({ roundId, songId });
  return db.transaction(async (tx) => {
    const battle = await lockBattle(battleId, userId, tx);
    const round = await lockActiveRound(battle, roundId, tx);
    if (round.status !== "SONG_PICKS") throw new AppError("Song selection has ended", 409);
    const isHost = battle.hostId === userId;
    if (isHost ? round.hostSongId : round.guestSongId) throw new AppError("You already picked a song", 409);
    const song = required(await songs.getSongForShare(songId, tx), "Song not found", 404);
    if (song.status !== "READY") throw new AppError("Song is not ready", 409);
    if (!song.fullSongPath || !song.drumsPath || !song.bassPath || !song.melodyPath) {
      throw new AppError("Song audio is incomplete", 409);
    }
    const hostSongId = isHost ? songId : round.hostSongId;
    const guestSongId = isHost ? round.guestSongId : songId;
    const started = Boolean(hostSongId && guestSongId);
    required(await rounds.updateRoundSelection(round.id, {
      hostSongId, guestSongId, status: started ? "GUESSING" : "SONG_PICKS",
    }, tx), "Song selection changed");
    return { battleId, roundId, status: started ? "ROUND_STARTED" as const : "WAITING_ON_OPPONENT" as const };
  });
}

async function advanceOrFinishBattle(battle: Battle, tx: Transaction) {
  if (battle.currentRound < battle.rounds) {
    const updated = required(await battles.advanceBattle(battle.id, battle.currentRound, tx),
      "Battle was already advanced");
    required(await rounds.createRound({
      battleId: battle.id, roundNumber: updated.currentRound, status: "SONG_PICKS",
    }, tx), "Could not create next round", 500);
    return updated;
  }
  const allRounds = await rounds.getRoundByBattleId(battle.id, tx);
  if (allRounds.length !== battle.rounds || allRounds.some((round) => round.status !== "FINISHED")) {
    throw new AppError("Battle history is inconsistent", 409);
  }
  const hostTotal = allRounds.reduce((sum, round) => sum + round.hostPoints, 0);
  const guestTotal = allRounds.reduce((sum, round) => sum + round.guestPoints, 0);
  const winnerId = hostTotal === guestTotal ? null : hostTotal > guestTotal ? battle.hostId : battle.guestId;
  return required(await battles.finishBattle(battle.id, winnerId, tx), "Battle was already finished");
}


export async function submitGuess(roundId: string, userId: string, guess: string, expectedAttempt: number) {
  const input = guessInput.parse({ guess, expectedAttempt });
  return db.transaction(async (tx) => {
    const reference = required(await rounds.getRoundById(roundId, tx), "Round not found", 404);
    const battle = await lockBattle(reference.battleId, userId, tx);
    const round = await lockActiveRound(battle, roundId, tx);
    if (round.status !== "GUESSING") throw new AppError("Round is not accepting guesses", 409);
    const isHost = battle.hostId === userId;
    const finished = isHost ? round.hostFinished : round.guestFinished;
    if (finished || await rounds.hasCorrectGuess(roundId, userId, tx)) {
      throw new AppError("You already finished this round", 409);
    }
    const attempt = await rounds.getGuessCount(roundId, userId, tx) + 1;
    if (attempt > STAGES.length || attempt !== input.expectedAttempt) {
      throw new AppError("Attempt already submitted or stale; refresh the battle", 409);
    }
    const stage = isHost ? round.hostStage : round.guestStage;
    if (stage !== STAGES[attempt - 1]) throw new AppError("Round progress is inconsistent", 409);
    const targetId = required(isHost ? round.guestSongId : round.hostSongId, "Target song missing");
    const song = required(await songs.getSongForShare(targetId, tx), "Song unavailable", 409);
    const correct = isCloseMatch(input.guess, song.title);
    const points = correct ? POINTS[stage] : 0;
    const playerFinished = correct || attempt === STAGES.length;
    const nextStage = playerFinished ? stage : required(STAGES[attempt], "Invalid stage", 500);
    required(await rounds.makeGuess({ roundId, userId, guess: input.guess, correct, attempt }, tx),
      "Could not save guess", 500);
    const updated = required(await rounds.updatePlayerProgress(roundId, isHost ? "host" : "guest", {
      stage: nextStage, points, finished: playerFinished,
    }, tx), "Round progress changed");
    const roundFinished = updated.hostFinished && updated.guestFinished;
    let battleStatus = battle.status;
    if (roundFinished) {
      required(await rounds.finishRound(roundId, tx), "Round was already finished");
      battleStatus = (await advanceOrFinishBattle(battle, tx)).status;
    }
    return {
      battleId: battle.id, roundId, attempt, correct,
      result: correct ? "CORRECT" as const : "INCORRECT" as const,
      pointsAwarded: points, stage: nextStage, playerFinished, roundFinished, battleStatus,
    };
  });
}

function playerRound(round: Round, isHost: boolean, attempts: number) {
  return {
    id: round.id, number: round.roundNumber, status: round.status,
    myStage: isHost ? round.hostStage : round.guestStage,
    myPoints: isHost ? round.hostPoints : round.guestPoints,
    myFinished: isHost ? round.hostFinished : round.guestFinished,
    opponentFinished: isHost ? round.guestFinished : round.hostFinished,
    myHasPicked: Boolean(isHost ? round.hostSongId : round.guestSongId),
    opponentHasPicked: Boolean(isHost ? round.guestSongId : round.hostSongId),
    myAttempts: attempts,
  };
}

export async function getBattleView(battleId: string, userId: string) {
  return db.transaction(async (tx) => {
    const battle = await lockBattle(battleId, userId, tx);
    const allRounds = await rounds.getRoundByBattleId(battle.id, tx);
    const current = allRounds.find((round) => round.roundNumber === battle.currentRound);
    const isHost = battle.hostId === userId;
    const attempts = current ? await rounds.getGuessCount(current.id, userId, tx) : 0;
    const previous = [...allRounds].reverse().find((round) => round.status === "FINISHED");
    return {
      id: battle.id, status: battle.status, inviteCode: battle.inviteCode,
      currentRound: battle.currentRound, rounds: battle.rounds,
      opponentJoined: battle.guestId !== null,
      outcome: battle.status !== "FINISHED" ? null : battle.winnerId === null ? "DRAW" as const
        : battle.winnerId === userId ? "WIN" as const : "LOSS" as const,
      myScore: allRounds.reduce((sum, round) => sum + (isHost ? round.hostPoints : round.guestPoints), 0),
      opponentScore: allRounds.reduce((sum, round) => sum + (isHost ? round.guestPoints : round.hostPoints), 0),
      round: current ? playerRound(current, isHost, attempts) : null,
      previousRound: previous ? {
        number: previous.roundNumber,
        myPoints: isHost ? previous.hostPoints : previous.guestPoints,
        opponentPoints: isHost ? previous.guestPoints : previous.hostPoints,
      } : null,
    };
  });
}

export async function getAudioPath(roundId: string, userId: string) {
  return db.transaction(async (tx) => {
    const reference = required(await rounds.getRoundById(roundId, tx), "Round not found", 404);
    const battle = await lockBattle(reference.battleId, userId, tx);
    const round = await lockActiveRound(battle, roundId, tx);
    if (round.status !== "GUESSING") throw new AppError("Round audio unavailable", 409);
    const isHost = battle.hostId === userId;
    const songId = required(isHost ? round.guestSongId : round.hostSongId, "Target song missing");
    const song = required(await songs.getSongForShare(songId, tx), "Song unavailable", 404);
    if (song.status !== "READY") throw new AppError("Song is not ready", 409);
    const stage = isHost ? round.hostStage : round.guestStage;
    const paths = { DRUM: song.drumsPath, BASS: song.bassPath, MELODY: song.melodyPath, FULL: song.fullSongPath };
    return required(paths[stage], "Audio file is unavailable", 404);
  });
}

export async function getOpponentId(
  battleId: string,
  userId: string,
) {
  const battle = required(
    await battles.getBattleById(battleId),
    "Battle not found",
    404,
  );

  assertParticipant(battle, userId);

  return battle.hostId === userId
    ? battle.guestId
    : battle.hostId;
}

export async function forfeitBattle(
  battleId: string,
  disconnectedUserId: string,
) {
  return db.transaction(async (tx) => {
    const battle = await lockBattle(
      battleId,
      disconnectedUserId,
      tx,
    );

    if (battle.status !== "ONGOING") {
      return null;
    }

    const winnerId =
      battle.hostId === disconnectedUserId
        ? battle.guestId
        : battle.hostId;

    if (!winnerId) {
      return null;
    }

    return battles.finishBattle(
      battle.id,
      winnerId,
      tx,
    );
  });
}
