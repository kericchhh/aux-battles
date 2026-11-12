import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

HTMLMediaElement.prototype.play = () => Promise.resolve()
HTMLMediaElement.prototype.pause = () => {}
HTMLMediaElement.prototype.load = () => {}

afterEach(cleanup)
