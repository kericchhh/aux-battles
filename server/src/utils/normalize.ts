export function normalizeGuess(s: string): string {
  return s
    .toLowerCase()
    .normalize("NFD").replace(/[\u0300-\u036f]/g, "") // strip accents (café → cafe)
    .replace(/\(.*?\)/g, "")        // remove anything in parens: "(Remastered)"
    .replace(/\[.*?\]/g, "")        // remove anything in brackets
    .replace(/feat\.?.*$/i, "")     // strip "feat. X" and everything after
    .replace(/ft\.?.*$/i, "")       // strip "ft. X" variant
    .replace(/[^a-z0-9\s]/g, "")    // strip remaining punctuation
    .replace(/\s+/g, " ")           // collapse multiple spaces
    .trim()
}
