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
