import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { login as loginRequest } from "../api/auth";
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

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const data = await loginRequest({ identifier, password });
      login(data.token, data.id, data.username);
      navigate("/");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthFormLayout title="Log In" onSubmit={handleSubmit}>
      {error && <ErrorBanner message={error} />}

      <Input
        type="text"
        placeholder="Username or email"
        value={identifier}
        onChange={(e) => setIdentifier(e.target.value)}
        required
      />
      <Input
        type="password"
        placeholder="Password"
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
