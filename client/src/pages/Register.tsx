import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { errorMessage } from "../api/client";
import { register } from "../api/auth";
import AuthFormLayout from "../components/AuthFormLayout";
import ErrorBanner from "../components/ErrorBanner";
import Input from "../components/Inputs";
import RegisterButton from "../components/RegisterButton";

export default function Register() {
    const [username, setUsername] = useState("");
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);

    const navigate = useNavigate()

    async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
        e.preventDefault()
        setError(null)
        setLoading(true)
        try {
            await register({ username: username.trim(), email: email.trim(), password });
            navigate("/login", {state: {registered: true}})
        } catch (err) {
            setError(errorMessage(err))
        } finally {
            setLoading(false)
        }
    }
    return (
        <AuthFormLayout title="Create account" onSubmit={handleSubmit}>
            {error && <ErrorBanner message={error} />}

            <Input
                type="text"
                id="username" name="username" autoComplete="username"
                aria-label="Username" placeholder="Username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
            />
            <Input
                type="email"
                id="email" name="email" autoComplete="email"
                aria-label="Email" placeholder="Email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
            />
            <Input
                type="password"
                id="password" name="password" autoComplete="new-password"
                aria-label="Password" placeholder="Password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
            />

            <RegisterButton type="submit" loading={loading}>
                {loading ? "Creating account..." : "Register"}
            </RegisterButton>

            <p className="text-neutral-400 text-sm text-center mt-2">
                Already have an account?{" "}
                <Link to="/login" className="text-[#7780b6] hover:underline">
                    Log in
                </Link>
            </p>
        </AuthFormLayout>
    );
}
