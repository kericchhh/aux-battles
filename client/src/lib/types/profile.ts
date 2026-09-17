export interface Profile {
  id: string;
  username: string;
  avatarUrl: string | null;
  createdAt: string;
  battlesPlayed: number;
  wins: number;
  draws: number;
}
