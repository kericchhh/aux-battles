import { apiFetch, ApiError } from "./client";
import type { User } from "@/lib/types/auth";

export const register = (data: {username: string; email: string; password: string}) =>
    apiFetch<User>("/users/register", {method: "POST", body: JSON.stringify(data)})

export const login = (data: {identifier: string; password: string}) =>
    apiFetch<User>("/users/login", {method: "POST", body: JSON.stringify(data)})

export const logout = () => apiFetch<void>("/users/logout", {method: "POST"})

export async function getMe(signal?: AbortSignal) {
    try {
        return await apiFetch<User>("/users/me", {signal})
    } catch (error) {
        if (error instanceof ApiError && error.status === 401){
            return null
        }
        throw error
    }
}
