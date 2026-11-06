// The status line text shown while an action (or milestone) is dragged on the
// canvas, so the user can read the state they are moving to: the START first,
// then the end, the duration and how far it moved.
import { workdayCount } from "./workdays";

const DAY = 86400000;

export type DragMode = "move" | "l" | "r";

/** isoDay formats a UTC timestamp as YYYY-MM-DD. */
export function isoDay(ms: number): string {
  return new Date(ms).toISOString().slice(0, 10);
}

const days = (n: number) => `${n} ${Math.abs(n) === 1 ? "day" : "days"}`;

/** change describes the shift against the original: "+3 days", "-1 day" or "unchanged". */
export function changeText(deltaMs: number): string {
  const d = Math.round(deltaMs / DAY);
  return d === 0 ? "unchanged" : `${d > 0 ? "+" : "-"}${days(Math.abs(d))}`;
}

export interface ActionDrag {
  name: string;
  mode: DragMode;
  origStart: number; origEnd: number; // ms, UTC
  start: number; end: number;         // ms, UTC: where the action would be now
  /** the section the action would be dropped into, when that is a different one */
  intoSection?: string;
}

/**
 * dragStatusText describes the action being dragged, e.g.
 *   Moving "AWS Landing Zone": start 2026-12-05 (+3 days) · end 2027-01-10 (+3 days) · 37 days (27 working days)
 */
export function dragStatusText(d: ActionDrag): string {
  const verb = d.mode === "move" ? "Moving" : d.mode === "l" ? "Changing the start of" : "Changing the end of";
  const total = Math.round((d.end - d.start) / DAY) + 1;
  const wd = workdayCount(isoDay(d.start), isoDay(d.end));
  const into = d.intoSection !== undefined ? ` · into section "${d.intoSection || "(none)"}"` : "";
  return `${verb} "${d.name}": start ${isoDay(d.start)} (${changeText(d.start - d.origStart)})`
    + ` · end ${isoDay(d.end)} (${changeText(d.end - d.origEnd)})`
    + ` · ${days(total)} (${wd} working ${wd === 1 ? "day" : "days"})${into}`;
}

/** milestoneDragStatusText describes a milestone being dragged to a date. */
export function milestoneDragStatusText(name: string, origDate: number, date: number): string {
  return `Moving milestone "${name}": date ${isoDay(date)} (${changeText(date - origDate)})`;
}
