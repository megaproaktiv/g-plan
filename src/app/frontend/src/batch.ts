// Batch mode of the command bar: several commands, one per line, run one after
// another by the backend (ChatBatch). The batch stops at the first error.
import type { BatchResult } from "./types";

/** batchSummary is the one-line result shown in the log next to the command bar. */
export function batchSummary(r: BatchResult): string {
  if (r.total === 0) return "Nothing to run: no commands.";
  if (r.failed > 0) return `Batch stopped at line ${r.failed} (${r.done} of ${r.total} commands done): ${r.error}`;
  if (r.stopped > 0) return `Batch stopped by the quit command at line ${r.stopped} (${r.done} of ${r.total} commands done).`;
  return `Batch done: ${r.done} of ${r.total} commands.`;
}

/** batchCommandCount counts the commands of a script the way the backend does. */
export function batchCommandCount(script: string): number {
  return script.split(/\r?\n/).map((l) => l.trim()).filter((l) => l && !l.startsWith("#") && !l.startsWith("//")).length;
}
