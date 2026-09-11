import { useState } from "react";
import {
  Link,
  useLocation,
  useNavigate,
} from "react-router-dom";
import { errorMessage } from "../api/client";
import AuthFormLayout from "../components/AuthFormLayout";
import Button from "../components/Button";
import ErrorBanner from "../components/ErrorBanner";
import Input from "../components/Inputs";
import { useAuth } from "../context/auth-context";

interface LoginLocationState {
  from?: string;
  registered?: boolean;
  loggedOut?: boolean;
}

export default function Login() {
  const [identifier, setIdentifier] =
    useState("");

  const [password, setPassword] =
    useState("");

  const [error, setError] =
    useState<string | null>(null);

  const [loading, setLoading] =
    useState(false);

  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const locationState =
    location.state as LoginLocationState | null;

  async function handleSubmit(
    event: React.SyntheticEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    if (loading) return;

    setError(null);
    setLoading(true);

    try {
      await login(
        identifier.trim(),
        password,
      );

      navigate(
        locationState?.from ?? "/",
        {
          replace: true,
        },
      );
    } catch (caughtError) {
      setError(
        errorMessage(caughtError),
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthFormLayout
      title="Log in"
      onSubmit={handleSubmit}
    >
      {locationState?.registered && (
        <p
          role="status"
          className="text-sm text-green-300"
        >
          Account created. Log in to continue.
        </p>
      )}

      {locationState?.loggedOut && (
        <p
          role="status"
          className="text-sm text-muted"
        >
          You have been logged out.
        </p>
      )}

      {error && (
        <ErrorBanner message={error} />
      )}

      <Input
        type="text"
        id="identifier"
        name="identifier"
        autoComplete="username"
        aria-label="Username or email"
        placeholder="Username or email"
        value={identifier}
        onChange={(event) =>
          setIdentifier(event.target.value)
        }
        required
      />

      <Input
        type="password"
        id="password"
        name="password"
        autoComplete="current-password"
        aria-label="Password"
        placeholder="Password"
        value={password}
        onChange={(event) =>
          setPassword(event.target.value)
        }
        required
      />

      <Button
        type="submit"
        loading={loading}
        loadingText="Logging in…"
      >
        Log in
      </Button>

      <p className="mt-2 text-center text-sm text-muted">
        Don't have an account?{" "}
        <Link
          to="/register"
          className="text-primary hover:text-primary-hover hover:underline"
        >
          Register
        </Link>
      </p>
    </AuthFormLayout>
  );
}
