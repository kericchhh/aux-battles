import { distance } from "fastest-levenshtein";
import { normalizeGuess } from "./normalize.js";

export function isCloseMatch(guess: string, target: string): boolean {
  const a = normalizeGuess(guess);
  const b = normalizeGuess(target);
  if (a === b) return true; 
  const editDistance = distance(a, b);
  const threshold = Math.max(1, Math.floor(b.length * 0.15));
  return editDistance <= threshold;
}
