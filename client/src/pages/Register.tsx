import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { register } from "@/api/auth";
import { errorMessage } from "@/api/client";
import AuthFormLayout from "@/components/AuthFormLayout";
import Button from "@/components/Button";
import ErrorBanner from "@/components/ErrorBanner";
import Input from "@/components/Inputs";

export default function Register() {
    const [username, setUsername] = useState("");
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);

    const navigate = useNavigate();

    async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
        e.preventDefault();
        setError(null);
        setLoading(true);
        try {
            await register({ username: username.trim(), email: email.trim(), password });
            navigate("/login", { state: { registered: true } });
        } catch (err) {
            setError(errorMessage(err));
        } finally {
            setLoading(false);
        }
    }
    return (
        <AuthFormLayout title="Create account" onSubmit={handleSubmit}>
            {error && <ErrorBanner message={error} />}

            <label htmlFor="username" className="-mb-3 text-sm font-medium">Username</label>
            <Input
                type="text"
                id="username" name="username" autoComplete="username"
                aria-label="Username" placeholder="Username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
            />
            <label htmlFor="email" className="-mb-3 text-sm font-medium">Email</label>
            <Input
                type="email"
                id="email" name="email" autoComplete="email"
                aria-label="Email" placeholder="Email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
            />
            <label htmlFor="password" className="-mb-3 text-sm font-medium">Password</label>
            <Input
                type="password"
                id="password" name="password" autoComplete="new-password"
                aria-label="Password" placeholder="Password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
            />

            <Button type="submit" loading={loading} loadingText="Creating account…">Register</Button>

            <p className="text-neutral-400 text-sm text-center mt-2">
                Already have an account?{" "}
                <Link to="/login" className="text-muted hover:underline">
                    Log in
                </Link>
            </p>
        </AuthFormLayout>
    );
}
