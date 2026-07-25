import { useEffect, useRef } from "react";
import { io, type Socket } from "socket.io-client";

export function useSocket(battleId: string | undefined) {
  const socketRef = useRef<Socket | null>(null);

  useEffect(() => {
    if (!battleId) return;

    const socket = io("http://localhost:5000");
    socketRef.current = socket;

    socket.on("connect", () => {
      socket.emit("battle:join", battleId);
    });

    return () => {
      socket.disconnect();
    };
  }, [battleId]);

  return socketRef;
}
