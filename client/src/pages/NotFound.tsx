import ErrorPage from "../components/ErrorPage";

export default function NotFound() {
  return <ErrorPage code="404 · Track not found" title="This page skipped the queue" description="The link may be outdated, or the page may have moved. Head back and start a new battle." actionHref="/" actionLabel="Back to lobby" />;
}
