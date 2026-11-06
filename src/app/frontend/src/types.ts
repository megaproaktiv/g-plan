// DTO shapes mirroring ptplan/ptplan/dto.go.

export interface Milestone { id: string; displayId: string; name: string; date: string; cluster: string; color: string; }
export interface Phase { id: string; name: string; start: string; end: string; }
export interface Detail { id: string; displayId: string; name: string; desc: string; role: string; start: string; end: string; est: string; summary: boolean; fontColor: string; textPos: string; }
export interface WorkPack {
  id: string; num: number; displayId: string; name: string; desc: string; start: string; end: string;
  est: string; realEst: string; contour: string; externalId: string; workdays: number; estHours: string; estDays: string; realHours: string; realDays: string; synced: boolean; syncedAt: string;
  progressPermille: number; contourPermille: number; progressPercent: number; completionPos: number; mode: string; summary: boolean; detailsHidden: boolean; color: string; fontColor: string; textPos: string; noteVisible: boolean; notePos: string; cluster: string;
  sameLine: boolean; hasWarning: boolean; roleFit?: RoleFit; details: Detail[];
}
export interface RoleFit { effort: string; capacity: string; perDay: string; fits: boolean; fitsWindow: boolean; message: string; }
export interface Role { name: string; id: number; description: string; color: string; inUse: boolean; usedBy: string; totalEst: string; assignments: Assignment[]; }
export interface Dep { from: string; to: string; kind: string; fromAnchor: string; toAnchor: string; color: string; violated: boolean; }
export interface Assignment { wp: string; percent: number; consumed: string; }
export interface Project {
  title: string; description: string; unit: string; hoursPerDay: number; language: string;
  start: string; end: string; totalEst: string;
  milestones: Milestone[]; phases: Phase[]; packages: WorkPack[];
  deps: Dep[]; roles: Role[]; sectionColors: Record<string, string>; sectionNums: Record<string, number>;
  arrowStyle: string; showDetails: boolean; showRoleActions: boolean; showRolePercent: boolean; showRoleDays: boolean; showPhases: boolean; showProgress: boolean; monitor: Monitor;
  zoom: string; autosaveEnabled: boolean; autosaveMinutes: number; autosaveOnExit: boolean; saveVersions: boolean; versionCount: number;
  reportFormat: string; reportTodayLine: boolean; reportGrid: string; reportShowNotes: boolean; showWeekNumbers: boolean;
  savePath: string; warnings: string[];
}

export type Zoom = "days" | "weeks" | "months" | "year" | "window";

// --- monitoring (external system synchronization) ---
export interface Monitor { enabled: boolean; system: string; projectId: string; defaultAction: string;
  includeBefore: boolean; includeAfter: boolean; syncFrom: string; syncTo: string; }
export interface SystemInfo { id: string; label: string; available: boolean; }
export interface SystemConfig { baseUrl: string; username: string; clientId: string; passwordSet: boolean; clientSecretSet: boolean; }
export interface ConnTest { projectName: string; status: number; start: string; end: string; }
export interface SyncProgress { fetched: number; total: number; }
export interface MonitorAction {
  id: string; externalId: string; name: string; actual: string; entries: number;
  est: string; remaining: string; remainingPct: string;
  isDefault: boolean; defaultActual: string; defaultEntries: number;
  actualHours: string; actualDays: string; matchedHours: string; matchedDays: string; matchedEntries: number;
  defaultHours: string; defaultDays: string;
}
export interface MonitorSection {
  name: string; actions: number; tracked: number; est: string; estTracked: string;
  actual: string; entries: number; remaining: string; remainingPct: string; details: MonitorAction[];
}
export interface MonitorSummary {
  sections: MonitorSection[]; est: string; estTracked: string; actual: string; entries: number; remaining: string; remainingPct: string;
  unmatchedActual: string; unmatchedEntries: number; defaultAction: string; defaultActual: string; defaultEntries: number; fetched: number; from: string; to: string; at: string;
}

// --- glossary (help) ---
export interface GlossaryTerm { key: string; group: string; term: string; alsoCalled: string; definition: string; example: string; note: string; }

// --- structured help pages ---
export interface HelpBlock { kind: string; text: string; items: string[]; head: string[]; rows: string[][]; }
export interface HelpSection { key: string; heading: string; blocks: HelpBlock[]; }
export interface HelpPage { title: string; intro: string; sections: HelpSection[]; }

export interface BatchResult { total: number; done: number; failed: number; stopped: number; error: string; log: string[]; }
