import type { Issue } from "@/lib/types/api";

export const API_URL = (import.meta.env.VITE_API_URL || "http://localhost:5000").replace(/\/$/, "")
export class ApiError extends Error {
    status: number;
    issues: Issue[];
    constructor(message: string, status: number, issues: Issue[] = []) {
        super(message); 
        this.status = status; 
        this.issues = issues
    }
}
export async function apiFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
    const headers = new Headers(options.headers)
    if (options.body && !(options.body instanceof FormData)) headers.set("Content-Type", "application/json");
    const response = await fetch(`${API_URL}${path}`, {...options, headers, credentials: "include"});
    const text = await response.text();
    let data;
    try {
        data = text ? JSON.parse(text) : undefined
    } catch {
        throw new ApiError("The server returned an unexpected result", response.status)
        
    }
    if (!response.ok) {
        if (response.status === 401 && path !== "/users/login" && path !== "/users/register" && path !== "/users/me"){
            window.dispatchEvent(new Event("session:expired"))
        }
        throw new ApiError(data?.message || "Request failed", response.status, Array.isArray(data?.issues) ? data.issues : [])
    }
    return data as T
}

export function errorMessage(error: unknown) {
    if (error instanceof ApiError && error.issues.length){
        return error.issues.map(i => `${i.path.join(".") || "Input"}: ${i.message}`).join("; ")
    }
    return error instanceof Error ? error.message : "Something went wrong"
    
}
