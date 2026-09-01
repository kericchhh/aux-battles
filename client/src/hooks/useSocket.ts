import { useEffect, useState } from "react";
import { io, type Socket } from "socket.io-client";

export function useSocket(battleId: string | undefined) {
  const [socket, setSocket] = useState<Socket | null>(null);

  useEffect(() => {
    if (!battleId) return;

    const s = io("http://localhost:5000");

    s.on("connect", () => {
      s.emit("battle:join", battleId);
    });

    setSocket(s);

    return () => {
      s.disconnect();
      setSocket(null);
    };
  }, [battleId]);

  return socket;
}
