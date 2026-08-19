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

    useEffect()
}
