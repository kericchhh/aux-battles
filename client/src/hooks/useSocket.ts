import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { io } from "socket.io-client";
import { API_URL } from "../api/client";
import { queryKeys } from "../lib/queryKeys";
export function useSocket(battleId: string, userId: string) {
  const cache = useQueryClient();
  const [connection, setConnection] = useState("Connecting…");
  useEffect(() => {
    const socket = io(API_URL, {withCredentials: true});
    const refresh = () => { void cache.invalidateQueries({queryKey: queryKeys.battle(userId, battleId)}); };
    socket.on("battle:changed", refresh);
    socket.on("connect", () => {
      socket.timeout(5000).emit("battle:join", battleId, (error: Error | null, result?: {ok: boolean}) => {
        if (error || !result?.ok) { setConnection("Live updates unavailable; refreshing periodically"); void cache.invalidateQueries({queryKey: queryKeys.me}); }
        else setConnection("Live updates connected");
        refresh();
      });
    });
    socket.on("disconnect", reason => {
      setConnection("Disconnected; refreshing periodically");
      if (reason === "io server disconnect") void cache.invalidateQueries({queryKey: queryKeys.me});
    });
    socket.on("connect_error", () => {setConnection("Reconnecting…"); void cache.invalidateQueries({queryKey: queryKeys.me});});
    return () => {socket.removeAllListeners(); socket.disconnect();};
  }, [battleId, userId, cache]);
  return connection;
}
