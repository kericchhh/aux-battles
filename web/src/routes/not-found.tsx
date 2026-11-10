import { Link } from 'react-router'

export function NotFound() {
  return (
    <div className="grid gap-4">
      <h1 className="text-2xl">Dead air</h1>
      <p className="text-dim">Nothing plays at this address (404).</p>
      <Link to="/" className="link w-fit">
        Back to battles
      </Link>
    </div>
  )
}
