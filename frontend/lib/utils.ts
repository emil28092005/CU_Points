// Utility helpers for formatting numbers and dates throughout the UI.

/**
 * Formats a point balance for display: 1234 → "1 234 pts"
 * Uses the Russian locale so thousands are separated by a space.
 */
export function formatPoints(n: number): string {
  return `${new Intl.NumberFormat('ru-RU').format(n)} pts`;
}

/**
 * Formats a transaction amount with a sign prefix.
 * Positive amounts use a "+" prefix; negative amounts use the Unicode minus sign "−".
 * Example: 50 → "+50", -30 → "−30"
 */
export function formatTransactionAmount(amount: number): string {
  if (amount >= 0) return `+${amount}`;
  return `−${Math.abs(amount)}`; // U+2212 MINUS SIGN, visually distinct from hyphen
}

/**
 * Formats a UTC ISO timestamp to a compact Russian date+time string.
 * Example: "2024-04-28T14:32:00Z" → "28 апр., 14:32"
 */
export function formatDate(iso: string): string {
  return new Intl.DateTimeFormat('ru-RU', {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(iso));
}
