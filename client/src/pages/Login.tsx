import { useState } from "react";
import { useNavigate, useLocation, Link } from "react-router-dom";
import { useAuth } from "../context/auth-context";
import { errorMessage } from "../api/client";
import AuthFormLayout from "../components/AuthFormLayout";
import Input from "../components/Inputs";
import Button from "../components/RegisterButton";
import ErrorBanner from "../components/ErrorBanner";

export default function Login() {
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await login(identifier.trim(), password);
      navigate("/", {replace: true});
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthFormLayout title="Log In" onSubmit={handleSubmit}>
      {location.state?.registered && <p role="status" className="text-green-300">Account created. Log in to continue.</p>}
      {error && <ErrorBanner message={error} />}

      <Input
        type="text"
        id="identifier" name="identifier" autoComplete="username"
                aria-label="Username or email" placeholder="Username or email"
        value={identifier}
        onChange={(e) => setIdentifier(e.target.value)}
        required
      />
      <Input
        type="password"
        id="password" name="password" autoComplete="current-password"
                aria-label="Password" placeholder="Password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        required
      />

      <Button type="submit" loading={loading}>
        {loading ? "Logging in..." : "Log In"}
      </Button>

      <p className="text-neutral-400 text-sm text-center mt-2">
        Don't have an account?{" "}
        <Link to="/register" className="text-[#7780b6] hover:underline">
          Register
        </Link>
      </p>
    </AuthFormLayout>
  );
}
