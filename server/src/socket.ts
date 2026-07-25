import { Server } from "socket.io";
import type {Server as HttpServer} from "node:http";

let io: Server;

export function initSocket(httpServer: HttpServer) {
    io = new Server(httpServer, {
        cors: {origin: "*"}
    });
    io.on("connection", (socket) => {
        console.log("Client connected:", socket.id)

        socket.on("battle:join", (battleId: string) => {
            socket.join(battleId);
            console.log(`${socket.id} joined ${battleId}`)
        })

        socket.on("disconnect", () => {
            console.log("Client disconnected:",socket.id)
        })
    });
    return io
}

export function getIO() {
    if(!io) throw new Error("Socket.io not initialized")
    return io
}
