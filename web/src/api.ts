// Typed client for the /api/ routes in internal/server/api.go. Every
// function here maps 1:1 to one HTTP endpoint - the same parity principle
// the whole project follows (docs/PLAN1.md §52): this file adds no logic
// beyond request/response shaping.

export type ExportFormat = "html" | "pdf" | "md";

export interface BookInfo {
  Path: string;
  ConfigPath: string;
  Title: string;
}

export interface Node {
  Path: string;
  ID: string;
  Level: number;
  ParentID: string;
}

export interface SourceRef {
  Path: string;
  LineStart: number;
  LineEnd: number;
}

export interface Law {
  Number: string;
  Index: number;
  Text: string;
  Slug: string;
  SectionID: string;
  Source: SourceRef;
}

export interface Section {
  ID: string;
  Number: string;
  Title: string;
  Level: number;
  ParentID: string;
  Source: SourceRef;
  Commentary: string;
  Laws: Law[] | null;
}

export interface Lawbook {
  Metadata: { Title: string; Ordering: string[]; PromptTemplates: string[] };
  Sections: Section[];
  Prompts: PromptTemplate[] | null;
  PromptBacklinks: Record<string, string[]> | null;
}

export interface Diagnostic {
  Severity: string;
  Code: string;
  Message: string;
  Source: SourceRef | null;
}

export interface RenderedSection {
  CommentaryHTML: string;
  LawHTML: Record<string, string>; // law citation number -> HTML
}

export interface PromptSegment {
  Kind: number; // 0=Text, 1=LawRef, 2=SectionRef, 3=PromptRef
  Text: string;
  RefToken: string;
  RefAnchor: string;
  RefLabel: string;
  Expanded: string;
}

export interface PromptTemplate {
  ID: string;
  Title: string;
  Commentary: string;
  Segments: PromptSegment[] | null;
  Template: string;
  Vars: string[] | null;
  ReferencedAnchors: string[] | null;
  Source: SourceRef;
}

export interface RenderedPrompt {
  CommentaryHTML: string;
  TemplateHTML: string;
  CompactHTML: string;
}

export interface SearchMatch {
  sectionId: string;
  sectionTitle: string;
  sectionNumber: string;
  sourcePath: string;
  lawIndex: number;
  lawNumber: string;
  line: number;
  before: string;
  match: string;
  after: string;
}

export interface CompileResult {
  ok: boolean;
  error: string;
  lawbook: Lawbook;
  diagnostics: Diagnostic[];
  rendered: Record<string, RenderedSection>; // section ID -> rendered HTML
  renderedPrompts: Record<string, RenderedPrompt>; // prompt ID -> rendered HTML
}

export interface Provenance {
  Revision: string;
  Dirty: boolean;
  WorkingTreeHash: string;
  CompiledAt: string;
  CompilerName: string;
  CompilerEmail: string;
  HeadCommitAuthor: string;
  HeadCommitDate: string;
  AgentLawsVersion: string;
  AgentLawsBuildTime: string;
}

export interface Manifest {
  lawbook: string;
  content_hash: string;
  provenance: Provenance;
  signature: string;
}

export interface HistoryEntry {
  Commit: string;
  Author: string;
  Date: string;
  Summary: string;
}

export interface LawHistory {
  Citation: string;
  Introduced: string;
  Modifications: HistoryEntry[];
}

export interface LogEntry {
  Commit: string;
  Author: string;
  Date: string;
  Summary: string;
}

export interface FileChange {
  Path: string;
  Status: string;
  Added: number;
  Deleted: number;
}

export interface LawChange {
  SectionID: string;
  Index: number;
  OldNumber: string;
  NewNumber: string;
  OldText: string;
  NewText: string;
}

export interface LawbookDiff {
  AddedSections: string[];
  RemovedSections: string[];
  AddedLaws: Law[];
  RemovedLaws: Law[];
  ModifiedLaws: LawChange[];
}

export interface CommitDetail {
  commit: string;
  author: string;
  date: string;
  summary: string;
  files: FileChange[];
  diff?: LawbookDiff;
}

export interface Param {
  name: string;
  kind: string;
  required: boolean;
  description: string;
}

export interface Operation {
  id: string;
  group: string;
  summary: string;
  method: string;
  path: string;
  params: Param[];
  cliTemplate: string;
  goTemplate: string;
}

export class ApiError extends Error {
  code: string;
  status: number;
  constructor(message: string, code: string, status: number) {
    super(message);
    this.code = code;
    this.status = status;
  }
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init);
  const body = await res.json().catch(() => null);
  if (!res.ok) {
    throw new ApiError(body?.error ?? res.statusText, body?.code ?? "unknown", res.status);
  }
  return body as T;
}

function qs(params: Record<string, string | number | boolean | string[] | undefined>): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === "") continue;
    if (Array.isArray(v)) {
      for (const item of v) q.append(k, item);
    } else {
      q.set(k, String(v));
    }
  }
  const s = q.toString();
  return s ? `?${s}` : "";
}

export const api = {
  root: () => req<{ root: string }>("/api/meta/root"),

  discover: (root?: string) => req<BookInfo[]>(`/api/books${qs({ root })}`),

  exportAllURL: (root: string | undefined, format: ExportFormat, title?: string) =>
    `/api/export${qs({ root, format, title })}`,

  createBook: (path: string, title: string) =>
    req<{ path: string }>("/api/books", { method: "POST", body: JSON.stringify({ path, title }) }),

  book: (path: string) => req<{ title: string; sections: Node[] }>(`/api/book${qs({ path })}`),

  compile: (path: string) => req<CompileResult>(`/api/book/compile${qs({ path })}`),

  exportURL: (path: string, format: ExportFormat) => `/api/book/export${qs({ path, format })}`,

  render: (
    path: string,
    opts: { section?: string; law?: string; all?: boolean; vars?: Record<string, string>; onMissing?: string },
  ) =>
    req<{ text: string }>(
      `/api/book/render${qs({
        path,
        section: opts.section,
        law: opts.law,
        all: opts.all,
        onMissing: opts.onMissing,
        var: opts.vars ? Object.entries(opts.vars).map(([k, v]) => `${k}:${v}`) : undefined,
      })}`,
    ),

  createChapter: (book: string, file: string, title: string, id: string, after?: string) =>
    req<{ id: string }>("/api/book/chapters", {
      method: "POST",
      body: JSON.stringify({ book, file, title, id, after }),
    }),

  createSection: (book: string, file: string, title: string, id: string, parent: string) =>
    req<{ id: string }>("/api/book/sections", {
      method: "POST",
      body: JSON.stringify({ book, file, title, id, parent }),
    }),

  move: (
    book: string,
    kind: "chapter" | "section",
    id: string,
    opts: { newParentId?: string; before?: string; after?: string },
  ) =>
    req<{ moved: string }>("/api/book/move", {
      method: "POST",
      body: JSON.stringify({ book, kind, id, newParentId: opts.newParentId, before: opts.before, after: opts.after }),
    }),

  remove: (book: string, kind: "chapter" | "section", id: string, force = false) =>
    req<{ removed: string }>(`/api/book/sections${qs({ book, kind, id, force })}`, { method: "DELETE" }),

  listLaws: (book: string, sectionId: string) => req<string[]>(`/api/book/laws${qs({ book, sectionId })}`),

  addLaw: (book: string, sectionId: string, text: string) =>
    req<{ sectionId: string }>("/api/book/laws", {
      method: "POST",
      body: JSON.stringify({ book, sectionId, text }),
    }),

  removeLaw: (book: string, sectionId: string, number: number, force = false) =>
    req<{ sectionId: string }>(`/api/book/laws${qs({ book, sectionId, number, force })}`, { method: "DELETE" }),

  operations: () => req<Operation[]>("/api/meta/operations"),

  watch: (path: string, onEvent: (ev: CompileResult & { clusterPath: string }) => void): (() => void) => {
    const es = new EventSource(`/api/book/watch${qs({ path })}`);
    es.onmessage = (m) => onEvent(JSON.parse(m.data));
    return () => es.close();
  },

  manifest: (path: string) => req<Manifest>(`/api/book/manifest${qs({ path })}`),

  history: (path: string, citation: string) =>
    req<LawHistory>(`/api/book/history${qs({ path, citation })}`),

  log: (path: string, limit?: number) =>
    req<LogEntry[]>(`/api/book/log${qs({ path, limit })}`),

  verify: (path: string, manifest: Manifest) =>
    req<{ ok: boolean; error?: string }>("/api/book/verify", {
      method: "POST",
      body: JSON.stringify({ path, manifest }),
    }),

  commitDetail: (path: string, commit: string) =>
    req<CommitDetail>(`/api/book/commit-detail${qs({ path, commit })}`),

  source: (filePath: string, lineStart?: number, lineEnd?: number) =>
    req<{ path: string; lineStart: number; lineEnd: number; content: string }>(
      `/api/book/source${qs({ path: filePath, lineStart, lineEnd })}`,
    ),

  search: (
    book: string,
    q: string,
    opts?: { caseSensitive?: boolean; wholeWord?: boolean; regex?: boolean; sectionIds?: string[] },
  ) =>
    req<SearchMatch[]>(
      `/api/book/search${qs({
        path: book,
        q,
        caseSensitive: opts?.caseSensitive,
        wholeWord: opts?.wholeWord,
        regex: opts?.regex,
        sectionId: opts?.sectionIds,
      })}`,
    ),

  promptRender: (path: string, id: string, opts?: { vars?: Record<string, string>; onMissing?: string }) =>
    req<{ text: string }>(
      `/api/book/prompt/render${qs({
        path,
        id,
        onMissing: opts?.onMissing,
        var: opts?.vars ? Object.entries(opts.vars).map(([k, v]) => `${k}:${v}`) : undefined,
      })}`,
    ),

  createPrompt: (book: string, file: string, title: string, id: string, after?: string) =>
    req<{ id: string }>("/api/book/prompts", {
      method: "POST",
      body: JSON.stringify({ book, file, title, id, after }),
    }),

  removePrompt: (book: string, file: string) =>
    req<{ removed: string }>(`/api/book/prompts${qs({ book, file })}`, { method: "DELETE" }),
};
