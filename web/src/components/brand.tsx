import { Link } from 'react-router'

export function Brand() {
  return (
    <Link to="/" className="brand text-lg">
      aux<span className="text-accent-text">/</span>battles
    </Link>
  )
}
