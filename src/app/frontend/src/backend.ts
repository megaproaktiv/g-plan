// Thin typed wrapper over the Wails v3 generated service bindings. v3 generates
// a typed `App` service into ../bindings/<module path>; we delegate to it. No more
// window.go global — the generated functions call the Go service directly.
import { App } from "../bindings/github.com/megaproaktiv/g-plan/src/app";
import type { BatchResult, ConnTest, GlossaryTerm, HelpPage, MonitorSummary, Project, SystemConfig, SystemInfo } from "./types";

// In v3 the runtime is ready as soon as the module loads inside the app; keep a
// no-op whenReady for call-site compatibility.
export function whenReady(): Promise<void> {
  return Promise.resolve();
}

// Each call returns the updated ProjectDTO (or void). We cast through unknown to
// our local Project type (structurally identical to the generated model).
const P = <T,>(p: Promise<unknown>) => p as Promise<T>;

export const backend = {
  getProject: () => P<Project>(App.GetProject()),
  openProject: (base: string) => P<Project>(App.OpenProject(base)),
  openDialog: () => P<Project>(App.OpenDialog()),
  saveDialog: () => P<Project>(App.SaveDialog()),
  saveProject: (base: string) => App.SaveProject(base),
  setConfig: (unit: string, hoursPerDay: number) => P<Project>(App.SetConfig(unit, hoursPerDay)),
  setLanguage: (code: string) => P<Project>(App.SetLanguage(code)),
  languages: () => App.Languages() as Promise<{ code: string; iso: string; name: string }[]>,
  relationshipRules: (lang: string) =>
    App.RelationshipRules(lang) as Promise<{ kind: string; abbr: string; name: string; rule: string; example: string }[]>,
  guiStrings: (lang: string) => App.GUIStrings(lang) as Promise<Record<string, string>>,
  commandHelp: (lang: string) => App.CommandHelp(lang) as Promise<string>,
  setTitle: (name: string) => P<Project>(App.SetTitle(name)),
  setProjectRange: (start: string, end: string) => P<Project>(App.SetProjectRange(start, end)),
  setProjectDetails: (name: string, description: string, start: string, end: string) =>
    P<Project>(App.SetProjectDetails(name, description, start, end)),

  moveWorkPackage: (id: string, start: string, end: string) => P<Project>(App.MoveWorkPackage(id, start, end)),
  setWorkPackageMode: (id: string, mode: string) => P<Project>(App.SetWorkPackageMode(id, mode)),
  setWorkPackageSummary: (id: string, summary: boolean) => P<Project>(App.SetWorkPackageSummary(id, summary)),
  setActionDetailsHidden: (id: string, hidden: boolean) => P<Project>(App.SetActionDetailsHidden(id, hidden)),
  setDetailSummary: (wp: string, det: string, summary: boolean) => P<Project>(App.SetDetailSummary(wp, det, summary)),
  setWorkPackageSameLine: (id: string, sameLine: boolean) => P<Project>(App.SetWorkPackageSameLine(id, sameLine)),
  addWorkPackage: (name: string, start: string, end: string) => P<Project>(App.AddWorkPackage(name, start, end)),
  removeWorkPackage: (id: string) => P<Project>(App.RemoveWorkPackage(id)),

  addDetail: (wp: string, name: string, start: string, end: string, est: string, desc: string, role: string) =>
    P<Project>(App.AddDetail(wp, name, start, end, est, desc, role)),
  moveDetail: (wp: string, det: string, start: string, end: string) => P<Project>(App.MoveDetail(wp, det, start, end)),
  setDetailEstimation: (wp: string, det: string, display: string) => P<Project>(App.SetDetailEstimation(wp, det, display)),
  setDetailFields: (wp: string, det: string, name: string, desc: string, est: string, role: string) =>
    P<Project>(App.SetDetailFields(wp, det, name, desc, est, role)),
  removeDetail: (wp: string, det: string) => P<Project>(App.RemoveDetail(wp, det)),
  moveDetailOrder: (wp: string, det: string, dir: number) => P<Project>(App.MoveDetailOrder(wp, det, dir)),

  addDependency: (from: string, to: string, kind: string, fromAnchor: string, toAnchor: string) =>
    P<Project>(App.AddDependency(from, to, kind, fromAnchor, toAnchor)),
  removeDependency: (from: string, to: string) => P<Project>(App.RemoveDependency(from, to)),
  setActionFields: (id: string, name: string, desc: string, est: string, start: string, end: string) =>
    P<Project>(App.SetActionFields(id, name, desc, est, start, end)),
  setActionColor: (id: string, color: string) => P<Project>(App.SetActionColor(id, color)),
  setActionFontColor: (id: string, color: string) => P<Project>(App.SetActionFontColor(id, color)),
  setDetailFontColor: (wp: string, det: string, color: string) => P<Project>(App.SetDetailFontColor(wp, det, color)),
  setActionTextPos: (id: string, pos: string) => P<Project>(App.SetActionTextPos(id, pos)),
  setActionNote: (id: string, visible: boolean, pos: string) => P<Project>(App.SetActionNote(id, visible, pos)),
  setDetailTextPos: (wp: string, det: string, pos: string) => P<Project>(App.SetDetailTextPos(wp, det, pos)),
  setActionCluster: (id: string, cluster: string) => P<Project>(App.SetActionCluster(id, cluster)),
  setDependencyColor: (from: string, to: string, color: string) => P<Project>(App.SetDependencyColor(from, to, color)),
  setSectionColor: (section: string, color: string) => P<Project>(App.SetSectionColor(section, color)),
  setArrowStyle: (style: string) => P<Project>(App.SetArrowStyle(style)),
  setViewOptions: (showDetails: boolean, showRoleActions: boolean, showPhases: boolean, showProgress: boolean) =>
    P<Project>(App.SetViewOptions(showDetails, showRoleActions, showPhases, showProgress)),
  setRoleLabelOptions: (showPercent: boolean, showDays: boolean) =>
    P<Project>(App.SetRoleLabelOptions(showPercent, showDays)),
  setWorkPackageEstimation: (id: string, display: string) => P<Project>(App.SetWorkPackageEstimation(id, display)),
  setWorkPackageRealEffort: (id: string, display: string) => P<Project>(App.SetWorkPackageRealEffort(id, display)),
  monitoringHelp: (lang: string) => App.MonitoringHelp(lang) as unknown as Promise<HelpPage>,
  objectHelp: (lang: string) => App.ObjectHelp(lang) as unknown as Promise<HelpPage>,
  glossary: (lang: string) => App.Glossary(lang) as unknown as Promise<GlossaryTerm[]>,
  monitorSystems: () => App.MonitorSystems() as unknown as Promise<SystemInfo[]>,
  setMonitor: (enabled: boolean, system: string, projectId: string) => P<Project>(App.SetMonitor(enabled, system, projectId)),
  setMonitorDefaultAction: (id: string) => P<Project>(App.SetMonitorDefaultAction(id)),
  setMonitorRange: (before: boolean, after: boolean) => P<Project>(App.SetMonitorRange(before, after)),
  setActionExternalId: (id: string, value: string) => P<Project>(App.SetActionExternalID(id, value)),
  getSystemConfig: (system: string) => P<SystemConfig>(App.GetSystemConfig(system)),
  saveSystemConfig: (system: string, baseUrl: string, username: string, password: string, clientId: string, clientSecret: string) =>
    App.SaveSystemConfig(system, baseUrl, username, password, clientId, clientSecret) as Promise<void>,
  testSystemConnection: (system: string, projectId: string) => P<ConnTest>(App.TestSystemConnection(system, projectId)),
  syncMonitor: () => P<MonitorSummary>(App.SyncMonitor()),
  cancelSync: () => App.CancelSync() as Promise<void>,
  exportMonitorMarkdown: () => App.ExportMonitorMarkdown() as Promise<string[]>,
  getMonitorSummary: () => P<MonitorSummary | null>(App.GetMonitorSummary()),
  setWorkPackageContour: (id: string, kind: string) => P<Project>(App.SetWorkPackageContour(id, kind)),
  moveSection: (cluster: string, dir: number) => P<Project>(App.MoveSection(cluster, dir)),
  moveAction: (id: string, dir: number) => P<Project>(App.MoveAction(id, dir)),
  removeSection: (name: string) => P<Project>(App.RemoveSection(name)),
  setZoom: (zoom: string) => P<Project>(App.SetZoom(zoom)),
  newProject: (autosave: boolean) => P<Project>(App.NewProject(autosave)),
  savePath: () => App.SavePath() as Promise<string>,
  isDirty: () => App.IsDirty() as Promise<boolean>,
  saveSmart: () => App.SaveSmart() as Promise<string>,
  autosaveTick: () => App.AutosaveTick() as Promise<string>,
  setAutosaveConfig: (enabled: boolean, minutes: number, onExit: boolean, saveVersions: boolean, versionCount: number) =>
    P<Project>(App.SetAutosaveConfig(enabled, minutes, onExit, saveVersions, versionCount)),
  select: (kind: string, ref: string) => P<Project>(App.Select(kind, ref)),
  deselect: () => P<Project>(App.Deselect()),
  selection: () => App.Selection() as Promise<{ kind: string; id: string }>,
  addMilestone: (name: string, date: string, cluster: string) => P<Project>(App.AddMilestone(name, date, cluster)),
  setMilestoneFields: (id: string, name: string, date: string, color: string) => P<Project>(App.SetMilestoneFields(id, name, date, color)),
  moveMilestone: (id: string, date: string) => P<Project>(App.MoveMilestone(id, date)),
  removeMilestone: (id: string) => P<Project>(App.RemoveMilestone(id)),
  undo: () => P<Project>(App.Undo()),

  captureBaseline: (name: string) => App.CaptureBaseline(name) as Promise<any[]>,
  listBaselines: () => App.ListBaselines() as Promise<any[]>,
  deleteBaseline: (name: string) => App.DeleteBaseline(name) as Promise<any[]>,
  compareBaseline: (name: string) => App.CompareBaseline(name) as Promise<any[]>,

  addRole: (name: string, description: string) => P<Project>(App.AddRole(name, description)),
  setRoleFields: (oldName: string, newName: string, description: string) =>
    P<Project>(App.SetRoleFields(oldName, newName, description)),
  setRoleColor: (name: string, color: string) => P<Project>(App.SetRoleColor(name, color)),
  removeRole: (name: string) => P<Project>(App.RemoveRole(name)),
  addSection: (name: string) => P<Project>(App.AddSection(name)),
  renameSection: (oldName: string, newName: string) => P<Project>(App.RenameSection(oldName, newName)),
  setActionRoles: (id: string, roleNames: string[], percents: number[]) =>
    P<Project>(App.SetActionRoles(id, roleNames, percents)),
  assignRole: (role: string, wp: string, percent: number) => P<Project>(App.AssignRole(role, wp, percent)),
  roleLoad: (role: string) => App.RoleLoad(role) as Promise<Record<string, string>>,

  renderPNG: (outPath: string, scale: number) => App.RenderPNG(outPath, scale),
  exportPDFReport: (outPath: string) => App.ExportPDFReport(outPath),
  exportPDFDialog: () => App.ExportPDFDialog() as Promise<string>,
  setShowWeekNumbers: (on: boolean) => P<Project>(App.SetShowWeekNumbers(on)),
  setReportConfig: (format: string, todayLine: boolean, grid: string, showNotes: boolean) =>
    P<Project>(App.SetReportConfig(format, todayLine, grid, showNotes)),
  exportReport: (overviewBase64: string, detailsJSON: string) =>
    App.ExportReport(overviewBase64, detailsJSON) as Promise<string>,
  markdown: () => App.Markdown() as Promise<string>,
  warnings: () => App.Warnings() as Promise<string[]>,
  chat: (line: string) => App.Chat(line) as Promise<string>,
  chatBatch: (script: string) => App.ChatBatch(script) as unknown as Promise<BatchResult>,
  chatHelp: () => App.ChatHelp() as Promise<string>,
  mode: () => App.Mode() as Promise<string>,
  setMode: (m: string) => App.SetMode(m) as Promise<string>,
  // Tell the command bar which action's detail plane is open ("" = none), so
  // `add ...` typed in the chat adds a detail of that action.
  setDrill: (id: string) => App.SetDrill(id) as Promise<void>,
};
