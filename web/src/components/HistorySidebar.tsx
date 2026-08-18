import { useState, useEffect, useRef } from "preact/hooks";
import { History, X, ChevronRight, ChevronDown, GitCommit, User, Mail, Plus, Minus, Pencil, File, ShieldCheck, Loader2, CircleCheck, CircleX, Inbox, CircleAlert } from "lucide-preact";
import { api, type LogEntry, type CommitDetail, type Manifest, type LawbookDiff, type Section } from "../api";

// Go serializes nil slices as JSON null — normalize to empty arrays so
// .map() calls never crash.
function normalizeDetail(d: CommitDetail): CommitDetail {
  if (d.diff) {
    d.diff.AddedSections ??= [];
    d.diff.RemovedSections ??= [];
    d.diff.AddedLaws ??= [];
    d.diff.RemovedLaws ??= [];
    d.diff.ModifiedLaws ??= [];
  }
  d.files ??= [];
  return d;
}

interface Props {
  path: string;
  show: boolean;
  onClose: () => void;
  filterEntity?: { type: "section" | "law"; id: string; sectionId?: string } | null;
  sections: Section[];
}

export function HistorySidebar({ path, show, onClose, filterEntity, sections }: Props) {
  const [commits, setCommits] = useState<LogEntry[]>([]);
  const [expandedCommit, setExpandedCommit] = useState<string | null>(null);
  const [details, setDetails] = useState<Record<string, CommitDetail | "loading" | "error">>({});
  const [manifest, setManifest] = useState<Manifest | null>(null);
  const [verifyResult, setVerifyResult] = useState<string | null>(null);
  const [diffMode, setDiffMode] = useState<"inline" | "side">("inline");
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    if (!show) return;
    let cancelled = false;
    api.log(path).then((log) => { if (!cancelled) setCommits(log ?? []); }).catch(() => { if (!cancelled) setCommits([]); });
    api.manifest(path).then((m) => { if (!cancelled) setManifest(m); }).catch(() => { if (!cancelled) setManifest(null); });
    return () => { cancelled = true; };
  }, [show, path]);

  // Reset state when sidebar closes
  useEffect(() => {
    if (!show) {
      setExpandedCommit(null);
      setDetails({});
      setVerifyResult(null);
    }
  }, [show]);

  function toggleCommit(commit: string) {
    if (expandedCommit === commit) {
      setExpandedCommit(null);
      return;
    }
    setExpandedCommit(commit);
    // Only fetch if not already loaded/loading
    if (details[commit]) return;
    setDetails((prev) => ({ ...prev, [commit]: "loading" }));
    // Abort any previous in-flight request
    if (abortRef.current) abortRef.current.abort();
    const ctrl = new AbortController();
    abortRef.current = ctrl;
    api.commitDetail(path, commit).then((d) => {
      if (!ctrl.signal.aborted) setDetails((prev) => ({ ...prev, [commit]: normalizeDetail(d) }));
    }).catch(() => {
      if (!ctrl.signal.aborted) setDetails((prev) => ({ ...prev, [commit]: "error" }));
    });
  }

  function doVerify() {
    if (!manifest) return;
    setVerifyResult(null);
    if (!manifest.signature) {
      setVerifyResult("No signature found. Sign this book first: alaws sign " + path);
      return;
    }
    api.verify(path, manifest).then((r) => {
      setVerifyResult(r.ok ? "Verified OK" : `Failed: ${r.error}`);
    }).catch((e) => {
      setVerifyResult(`Error: ${e}`);
    });
  }

  if (!show) return null;

  const sectionTitle = (id: string) => sections.find((s) => s.ID === id)?.Title ?? id;

  return (
    <div class="history-sidebar">
      <div class="history-sidebar-header">
        <span class="history-sidebar-title">
          <History size={14} />
          History
        </span>
        <button class="icon-button" title="Close" onClick={() => onClose()}>
          <X size={14} />
        </button>
      </div>

      {manifest && (
        <div class="history-summary">
          <div class="history-summary-row">
            <code class="history-summary-hash">{manifest.provenance.Revision?.slice(0, 12)}{manifest.provenance.Dirty ? " (dirty)" : ""}</code>
            <span class="history-summary-compiler">{manifest.provenance.CompilerName}</span>
          </div>
          <div class="history-summary-row">
            <button class="btn btn-sm" onClick={doVerify}>
              <ShieldCheck size={12} /> Verify
            </button>
            {verifyResult && (
              <span class={`verify-result ${verifyResult.startsWith("Verified") ? "verify-ok" : "verify-fail"}`}>
                {verifyResult.startsWith("Verified") ? <CircleCheck size={12} /> : <CircleX size={12} />}
                {" "}{verifyResult}
              </span>
            )}
          </div>
        </div>
      )}

      {filterEntity && (
        <div class="history-filter-badge">
          {filterEntity.type === "law" ? `Law ${filterEntity.id}` : sectionTitle(filterEntity.sectionId ?? filterEntity.id)}
          <button class="icon-button" onClick={() => onClose()} title="Clear filter"><X size={12} /></button>
        </div>
      )}

      <div class="history-timeline">
        {commits.length === 0 && <p class="empty-state"><Inbox size={14} /> No commits found.</p>}
        {commits.map((c) => {
          const isExpanded = expandedCommit === c.Commit;
          const state = details[c.Commit];
          return (
            <div key={c.Commit} class={`history-commit ${isExpanded ? "expanded" : "collapsed"}`}>
              <div class="history-commit-row" onClick={() => toggleCommit(c.Commit)}>
                {isExpanded ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
                <code class="history-commit-hash">{c.Commit.slice(0, 8)}</code>
                <span class="history-commit-author">{authorName(c.Author)}</span>
                <span class="history-commit-time">{relativeTime(c.Date)}</span>
              </div>
              <div class="history-commit-summary">{c.Summary}</div>

              {isExpanded && (
                <div class="history-detail">
                  {state === "loading" && (
                    <div class="history-loading">
                      <Loader2 size={14} class="spin" /> Loading diff…
                    </div>
                  )}
                  {state === "error" && <p class="empty-state"><CircleAlert size={12} /> Failed to load details.</p>}
                  {state && typeof state === "object" && (
                    <CommitDetailView detail={state} diffMode={diffMode} setDiffMode={setDiffMode} sectionTitle={sectionTitle} />
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

function CommitDetailView({ detail, diffMode, setDiffMode, sectionTitle }: {
  detail: CommitDetail;
  diffMode: "inline" | "side";
  setDiffMode: (m: "inline" | "side") => void;
  sectionTitle: (id: string) => string;
}) {
  const authorEmail = extractEmail(detail.author);

  return (
    <>
      <div class="history-detail-contact">
        <User size={12} />
        <span>{detail.author}</span>
        {authorEmail && (
          <a class="history-detail-email" href={`mailto:${authorEmail}?subject=Re: ${encodeURIComponent(detail.summary)}`}>
            <Mail size={12} />
          </a>
        )}
      </div>

      {detail.files && detail.files.length > 0 && (
        <div class="history-detail-section">
          <div class="history-detail-header">
            <File size={11} /> Files ({detail.files.length})
          </div>
          {(detail.files ?? []).map((f) => (
            <div key={f.Path} class="history-file-item">
              <span class={`history-file-status status-${f.Status.toLowerCase()}`}>{f.Status}</span>
              <span class="history-file-path">{f.Path}</span>
              <span class="history-file-stats">
                {f.Added > 0 && <span class="diff-add">+{f.Added}</span>}
                {f.Deleted > 0 && <span class="diff-del">-{f.Deleted}</span>}
              </span>
            </div>
          ))}
        </div>
      )}

      {detail.diff && (
        <div class="history-detail-section">
          <div class="history-detail-header">
            <GitCommit size={11} /> Changes
            <div class="diff-mode-toggle">
              <button class={`btn btn-sm ${diffMode === "inline" ? "active" : ""}`} onClick={() => setDiffMode("inline")}>Inline</button>
              <button class={`btn btn-sm ${diffMode === "side" ? "active" : ""}`} onClick={() => setDiffMode("side")}>Side</button>
            </div>
          </div>
          <DiffView diff={detail.diff} mode={diffMode} sectionTitle={sectionTitle} />
        </div>
      )}

      {!detail.diff && !detail.files?.length && (
        <p class="empty-state"><Inbox size={12} /> No detailed changes available.</p>
      )}
    </>
  );
}

function DiffView({ diff, mode, sectionTitle }: { diff: LawbookDiff; mode: "inline" | "side"; sectionTitle: (id: string) => string }) {
  const addedSec = diff.AddedSections ?? [];
  const removedSec = diff.RemovedSections ?? [];
  const addedLaws = diff.AddedLaws ?? [];
  const removedLaws = diff.RemovedLaws ?? [];
  const modifiedLaws = diff.ModifiedLaws ?? [];

  const hasChanges = addedSec.length > 0 || removedSec.length > 0 ||
    addedLaws.length > 0 || removedLaws.length > 0 || modifiedLaws.length > 0;

  if (!hasChanges) return <p class="empty-state"><Inbox size={12} /> No semantic changes.</p>;

  if (mode === "side") {
    return (
      <div class="diff-side">
        <div class="diff-side-col">
          <div class="diff-side-header">Before</div>
          {removedSec.map((id) => <div key={id} class="diff-item diff-del"><Minus size={10} /> {sectionTitle(id)}</div>)}
          {removedLaws.map((l) => <div key={`r-${l.SectionID}-${l.Index}`} class="diff-item diff-del"><Minus size={10} /> {l.Number}: {l.Text.slice(0, 80)}</div>)}
          {modifiedLaws.map((m) => <div key={`m-${m.SectionID}-${m.Index}`} class="diff-item diff-del"><Pencil size={10} /> {m.OldNumber}: {m.OldText.slice(0, 80)}</div>)}
          {removedSec.length === 0 && removedLaws.length === 0 && modifiedLaws.length === 0 && <div class="empty-state"><Inbox size={10} /> No removals</div>}
        </div>
        <div class="diff-side-col">
          <div class="diff-side-header">After</div>
          {addedSec.map((id) => <div key={id} class="diff-item diff-add"><Plus size={10} /> {sectionTitle(id)}</div>)}
          {addedLaws.map((l) => <div key={`a-${l.SectionID}-${l.Index}`} class="diff-item diff-add"><Plus size={10} /> {l.Number}: {l.Text.slice(0, 80)}</div>)}
          {modifiedLaws.map((m) => <div key={`m2-${m.SectionID}-${m.Index}`} class="diff-item diff-add"><Pencil size={10} /> {m.NewNumber}: {m.NewText.slice(0, 80)}</div>)}
          {addedSec.length === 0 && addedLaws.length === 0 && modifiedLaws.length === 0 && <div class="empty-state"><Inbox size={10} /> No additions</div>}
        </div>
      </div>
    );
  }

  return (
    <div class="diff-inline">
      {addedSec.map((id) => <div key={id} class="diff-item diff-add"><Plus size={10} /> Section: {sectionTitle(id)}</div>)}
      {removedSec.map((id) => <div key={id} class="diff-item diff-del"><Minus size={10} /> Section: {sectionTitle(id)}</div>)}
      {addedLaws.map((l) => <div key={`a-${l.SectionID}-${l.Index}`} class="diff-item diff-add"><Plus size={10} /> {l.Number}: {l.Text.slice(0, 120)}</div>)}
      {removedLaws.map((l) => <div key={`r-${l.SectionID}-${l.Index}`} class="diff-item diff-del"><Minus size={10} /> {l.Number}: {l.Text.slice(0, 120)}</div>)}
      {modifiedLaws.map((m) => (
        <div key={`m-${m.SectionID}-${m.Index}`} class="diff-item diff-mod">
          <Pencil size={10} />
          <span class="diff-mod-old">{m.OldText.slice(0, 60)}</span>
          <span class="diff-mod-arrow">→</span>
          <span class="diff-mod-new">{m.NewText.slice(0, 60)}</span>
        </div>
      ))}
    </div>
  );
}

function authorName(author: string): string {
  const m = author.match(/^(.+?)\s*<.*>$/);
  return m ? m[1].trim() : author;
}

function extractEmail(author: string): string | null {
  const m = author.match(/<(.+?)>/);
  return m ? m[1] : null;
}

function relativeTime(dateStr: string): string {
  if (!dateStr) return "";
  const d = new Date(dateStr);
  const now = Date.now();
  const diffMs = now - d.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  const diffHr = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHr / 24);

  if (diffMin < 1) return "just now";
  if (diffMin < 60) return `${diffMin}m ago`;
  if (diffHr < 24) return `${diffHr}h ago`;
  if (diffDay < 7) return `${diffDay}d ago`;
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}
