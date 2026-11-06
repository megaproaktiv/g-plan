// Working days: Monday to Friday, weekends excluded, no holiday calendar.
// Mirrors plan.WorkdaysBetween in Go (inclusive window).

const DAY = 86400000;

/** workdayCount returns the Mon-Fri days in the inclusive YYYY-MM-DD window; 0 if invalid or reversed. */
export function workdayCount(start: string, end: string): number {
  const s = Date.parse(start + "T00:00:00Z"), e = Date.parse(end + "T00:00:00Z");
  if (!isFinite(s) || !isFinite(e) || e < s) return 0;
  const total = Math.round((e - s) / DAY) + 1;
  let n = Math.floor(total / 7) * 5;           // whole weeks hold exactly 5 workdays
  for (let i = 0, d = s + Math.floor(total / 7) * 7 * DAY; i < total % 7; i++, d += DAY) {
    const wd = new Date(d).getUTCDay();         // 0 = Sunday, 6 = Saturday
    if (wd !== 0 && wd !== 6) n++;
  }
  return n;
}
