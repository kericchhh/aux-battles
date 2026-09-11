export const queryKeys = {
  me: ["me"] as const,

  battle: (userId: string, battleId: string) =>
    ["battle", userId, battleId] as const,

  songs: (query: string, offset: number) =>
    ["songs", query, offset] as const,

  profile: (id: string) =>
    ["profile", id] as const,
};
