import express from "express";
import http from "node:http"
import cors from "cors";
import songRoutes from "./routes/song.routes.js"
import userRoutes from "./routes/user.routes.js"
import battleRoutes from "./routes/battles.routes.js"
import roundsRoutes from "./routes/rounds.routes.js"
import { errorHandler } from "./middleware/errorMiddleware.js";
import { initSocket } from "./socket.js";

const app = express();
const PORT = process.env.PORT || 5000;

app.use(cors({
    origin: "http://localhost:5173",
    credentials: true
}))
app.use(express.json())
app.use("/songs", songRoutes)
app.use("/users", userRoutes)
app.use("/battles", battleRoutes)
app.use("/rounds", roundsRoutes)
app.use(errorHandler)

const httpServer = http.createServer(app)
initSocket(httpServer)

httpServer.listen(PORT, () => {
    console.log(`Server running on port ${PORT}`);
});
