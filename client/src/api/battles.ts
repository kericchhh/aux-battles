import { apiFetch } from "./client";

export interface Battle {
    id: string,
    hostId: string,
    guestId: string,
    status: "PENDING" | "ONGOING" | "FINISHED",
    winnerId: string,
    rounds: number,
    currentRound: number,
    inviteCode: string,
    hostSongId: string | null,
    guestSongId: string | null,
    createdAt: string,
    updatedAt: string
}

type RoundStage = "DRUM" | "BASS" | "MELODY" | "FULL"
type RoundStatus = "SONG_PICKS" | "GUESSING" | "FINISHED"

export interface Round {
    id: string,
    battleId: string,
    hostSongId: string ,
    guestSongId: string ,
    roundNumber: number,
    hostStage: RoundStage,
    guestStage: RoundStage,
    hostPoints: number,
    guestPoints: number,
    status: RoundStatus
}

export interface BattleState {
    battle: Battle,
    round: Round | null
}

export function createBattle(rounds: number) {
    return apiFetch<Battle>("/battles", {
        method: "POST",
        body: JSON.stringify({rounds})
    })
}

export function joinBattle(inviteCode: string){
    return apiFetch<Battle>("/battles/join", {
        method: "POST",
        body: JSON.stringify({inviteCode})
    })
}

export function getBattle(id: string) {
   return apiFetch<Battle>(`/battles/${id}`, {
       method: "GET"
   }) 
}
