import { apiFetch } from "./client";

export function register(data: { username: string; email: string; password: string }) {
  return apiFetch<{ token: string; refreshToken: string; id: string; username: string }>(
    "/register",
    { method: "POST", body: JSON.stringify(data) }
  );
}

export function login(data: { identifier: string; password: string }) {
  return apiFetch<{ token: string; refreshToken: string; id: string; username: string }>(
    "/login",
    { method: "POST", body: JSON.stringify(data) }
  );
}
