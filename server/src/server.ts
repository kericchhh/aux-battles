import express from "express";
import songRoutes from "./routes/song.routes.js"
import userRoutes from "./routes/user.routes.js"
import battleRoutes from "./routes/battles.routes.js"

import { errorHandler } from "./middleware/errorMiddleware.js";

const app = express();
const PORT = process.env.PORT || 5000;
app.use(express.json())
app.use("/songs", songRoutes)
app.use("/users", userRoutes)
app.use("/battles", battleRoutes)
app.use(errorHandler)
app.listen(PORT, () => {
    console.log("Server running on port 5000");
});
