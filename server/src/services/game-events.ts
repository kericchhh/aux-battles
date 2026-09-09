import { getIO } from "../socket.js";

export function notifyBattleChanged(battleId: string) {
  try {
    const rooms = getIO().to([battleId, `battle:${battleId}`]);
    rooms.emit("battle:changed", { battleId });
    // Compatibility for the original useSocket/Battle page; remove after migration.
    rooms.emit("battle:update", { battleId });
    rooms.emit("round:update", { battleId });
  } catch (error) {
    console.error("Battle committed, but realtime notification failed", error);
  }
}
