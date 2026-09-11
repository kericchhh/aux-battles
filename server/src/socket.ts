import { Server } from "socket.io";
import type {
    Server as HttpServer,
} from "node:http";
import * as game from "./services/game.js";
import { findSession } from "./services/sessions.js";
import { battleParams } from "./validation/game.js";

const DISCONNECT_GRACE_MS = 30_000;

let io: Server;

const forfeitTimers = new Map<
    string,
    ReturnType<typeof setTimeout>
>();

function battleRoom(battleId: string) {
    return `battle:${battleId}`;
}

function playerRoom(
    battleId: string,
    userId: string,
) {
    return `battle:${battleId}:user:${userId}`;
}

function timerKey(
    battleId: string,
    userId: string,
) {
    return `${battleId}:${userId}`;
}

function cancelForfeit(
    battleId: string,
    userId: string,
) {
    const key = timerKey(
        battleId,
        userId,
    );

    const timer = forfeitTimers.get(key);

    if (timer) {
        clearTimeout(timer);
        forfeitTimers.delete(key);
    }
}

function emitBattleChanged(
    battleId: string,
) {
    const payload = { battleId };

    io.to([
        battleId,
        battleRoom(battleId),
    ]).emit(
        "battle:changed",
        payload,
    );

    io.to([
        battleId,
        battleRoom(battleId),
    ]).emit(
        "battle:update",
        payload,
    );

    io.to([
        battleId,
        battleRoom(battleId),
    ]).emit(
        "round:update",
        payload,
    );
}

function scheduleForfeit(
    battleId: string,
    disconnectedUserId: string,
) {
    cancelForfeit(
        battleId,
        disconnectedUserId,
    );

    const key = timerKey(
        battleId,
        disconnectedUserId,
    );

    const timer = setTimeout(
        async () => {
            forfeitTimers.delete(key);

            try {
                const reconnectedSockets =
                    await io
                        .in(
                            playerRoom(
                                battleId,
                                disconnectedUserId,
                            ),
                        )
                        .fetchSockets();

                if (reconnectedSockets.length > 0) {
                    return;
                }

                const opponentId =
                    await game.getOpponentId(
                        battleId,
                        disconnectedUserId,
                    );

                if (!opponentId) {
                    return;
                }
                const opponentSockets =
                    await io
                        .in(
                            playerRoom(
                                battleId,
                                opponentId,
                            ),
                        )
                        .fetchSockets();

                if (opponentSockets.length === 0) {
                    return;
                }

                const result =
                    await game.forfeitBattle(
                        battleId,
                        disconnectedUserId,
                    );

                if (result) {
                    emitBattleChanged(battleId);
                }
            } catch (error) {
                console.error(
                    "Could not process battle forfeit:",
                    error,
                );
            }
        },
        DISCONNECT_GRACE_MS,
    );

    forfeitTimers.set(key, timer);
}

export function initSocket(
    httpServer: HttpServer,
) {
    io = new Server(httpServer, {
        cors: {
            origin:
                process.env.CLIENT_ORIGIN ??
                "http://localhost:5173",

            credentials: true,
        },
    });

    io.use(
        async (socket, next) => {
            try {
                const session =
                    await findSession(
                        socket.handshake.headers.cookie,
                    );

                if (!session) {
                    next(
                        new Error("Unauthorized"),
                    );

                    return;
                }

                socket.data.userId =
                    session.userId;

                next();
            } catch (error) {
                console.error(
                    "Socket authentication failed:",
                    error,
                );

                next(
                    new Error("Unauthorized"),
                );
            }
        },
    );

    io.on(
        "connection",
        (socket) => {
            const userId =
                socket.data.userId as string;

            const joinedBattles =
                new Set<string>();

            socket.on(
                "battle:join",
                async (
                    rawBattleId: unknown,
                    acknowledge?: (
                        result: {
                            ok: boolean;
                            message?: string;
                        },
                    ) => void,
                ) => {
                    try {
                        const { battleId } =
                            battleParams.parse({
                                battleId:
                                    rawBattleId,
                            });
                        await game.getBattleView(
                            battleId,
                            userId,
                        );

                        await socket.join([
                            battleId,
                            battleRoom(battleId),
                            playerRoom(
                                battleId,
                                userId,
                            ),
                        ]);

                        joinedBattles.add(
                            battleId,
                        );

                        cancelForfeit(
                            battleId,
                            userId,
                        );

                        acknowledge?.({
                            ok: true,
                        });
                    } catch (error) {
                        console.error(
                            "Could not join battle socket:",
                            error,
                        );

                        acknowledge?.({
                            ok: false,
                            message:
                                "Could not join battle updates",
                        });
                    }
                },
            );

            socket.on(
                "disconnect",
                () => {
                    for (
                        const battleId
                        of joinedBattles
                    ) {
                        scheduleForfeit(
                            battleId,
                            userId,
                        );
                    }
                },
            );
        },
    );

    return io;
}

export function getIO() {
    if (!io) {
        throw new Error(
            "Socket.io not initialized",
        );
    }

    return io;
}
