import { useCallback, useEffect, useState } from "react";
import { Link, useLocation, useParams } from "react-router-dom";
import { getBattle, type Battle as BattleData, type BattleState} from "../api/battles"
import ErrorBanner from "../components/ErrorBanner";
import { useAuth } from "../context/AuthContext";
import { useSocket } from "../hooks/useSocket";

interface BattleLocationState {
    battle?: BattleData
}
export default function Battle() {
    const {battleId} = useParams<{battleId: string}>()
    const location = useLocation()
    const {userId} = useAuth()
    const initialBattle = (location.state as BattleLocationState | null)?.battle
    const [battleState, setBattleState] = useState<BattleState | null>(() => 
        initialBattle ? {battle: initialBattle, round: null} : null
    )
    const [isLoading, setIsLoading] = useState(!initialBattle)
    const [error, setError] = useState("")
    const [copied, setCopied] = useState(false)
    const socket = useSocket(battleId)
    
    const loadBattle = useCallback(async (showLoading = false) => {
        if(!battleId){
            setError("Invalid battle URL")
            setIsLoading(false)
            return
        }

        if(showLoading) setIsLoading(true)

        try{
            const state = await getBattle(battleId) 
            setBattleState(state)
            setError("")
        }catch (err) {
            setError(err instanceof Error ? err.message : "Could not load the battle")
        }finally {
            setIsLoading(false)
        }
    }, [battleId])

    useEffect(() => {
        if (!socket) return 
        const refreshBattle = () => {
            void loadBattle()
        }
        socket.on("battle:update", refreshBattle)
        socket.on("round:update", refreshBattle)
        return () => {
            socket.off("battle:update", refreshBattle)
            socket.off("round:upate", refreshBattle)
        }
    }, [loadBattle, socket])

    const copyInviteCode = async () => {
        const code = battleState?.battle.inviteCode
        if (!code) return
        try {
            await navigator.clipboard.writeText(code)
            setCopied(true)
            window.setTimeout(() => setCopied(false), 1500)
        } catch {
            setError("Could not copy code")
        }
    }

    if (isLoading) {
        return (
            <main className="flex h-full items-center justify-center bg-[radial-gradient(circle_at_35%_35%,#29345f_0%,#171b32_35%,#0d1020_100%)] text-[#f4f0f7]">
                <p className="w-full max-w-md space-y-5 text-center">Loading battle...</p>
            </main>
        )
    }

    if (!battleState) {
        return (
            <main className="flex h-full items-center justify-center bg-[#0d1020] px-6 text-[#f4f0f7]">
                <div className="w-full max-w-md space-y-5 text-center">
                    <ErrorBanner message={error || "Battle not found"} />
                    <Link to="/" className="inline-block rounded-xl bg-[#5964a6] px-6 py-3 transition-colors hover:bg-[#959cc6] hover:text-black">Return to lobby</Link>
                </div>
            </main>
        )
    }

    const {battle, round} = battleState
    const isHost = battle.hostId === userId;
    const waitingForOpponent = battle.status === "PENDING" && !battle.guest.id

    return(
        <main>
            <div>
            </div>
        </main>
    )
}
