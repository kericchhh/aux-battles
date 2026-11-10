import { isRouteErrorResponse, Link, useRouteError } from 'react-router'

export function ErrorPage() {
  const error = useRouteError()
  const message = isRouteErrorResponse(error)
    ? `${error.status} ${error.statusText}`
    : error instanceof Error
      ? error.message
      : 'Unknown error'

  return (
    <main id="main" className="shell grid min-h-dvh content-center gap-4">
      <h1 className="text-2xl">Something broke</h1>
      <p role="alert" className="text-alarm">
        {message}
      </p>
      <Link to="/" reloadDocument className="link w-fit">
        Reload
      </Link>
    </main>
  )
}
