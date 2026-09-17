import { Component, type ErrorInfo, type ReactNode } from "react";
import ErrorPage from "./ErrorPage";

interface ErrorBoundaryState { failed: boolean }

export default class ErrorBoundary extends Component<{ children: ReactNode }, ErrorBoundaryState> {
  state: ErrorBoundaryState = { failed: false };
  static getDerivedStateFromError(): ErrorBoundaryState { return { failed: true }; }

  componentDidCatch(error: Error, info: ErrorInfo) {
    if (import.meta.env.DEV) console.error("Uncaught application error", error, info);
  }

  render() {
    if (this.state.failed) {
      return <ErrorPage code="Something went off beat" title="We couldn't load this screen" description="Your session is safe. Reload the page to try again, or return to the lobby." actionHref="/" actionLabel="Return to lobby" secondaryAction={{ label: "Reload page", onClick: () => window.location.reload() }} />;
    }
    return this.props.children;
  }
}
