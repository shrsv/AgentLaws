import { useState, useEffect, useCallback, useRef } from "preact/hooks";
import { History, X, ChevronRight, ChevronDown, GitCommit, User, Mail, Plus, Minus, Pencil, File, ShieldCheck } from "lucide-preact";
import { api, type LogEntry, type CommitDetail, type Manifest, type LawbookDiff, type Section } from "../api";

interface Props {
  path: string;
  show: boolean;
  onClose: () => void;
  filterEntity?: { type: "section" | "law"; id: string; sectionId?: string } | null;
  sections: Section[];
}

export function HistorySidebar({ path, show, onClose, filterEntity, sections }: Props) {
  const [commits, setCommits] = useState<LogEntry[]>([]);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [detail, setDetail] = useState<CommitDetail | null>(null);
  const [loadingCommit, setLoadingCommit] = useState<string | null>(null);
  const [manifest, setManifest] = useState<Manifest | null>(null);
  const [verifyResult, setVerifyResult] = useState<string | null>(null);
  const [diffMode, setDiffMode] = useState<"inline" | "side">("inline");
  const fetchSeq = useRef(0);

  const loadCommits = useCallback(async () => {
    try {
      const log = await api.log(path);
      setCommits(log ?? []);
    } catch {
      setCommits([]);
    }
  }, [path]);

  const loadManifest = useCallback(async () => {
    try {
      const m = await api.manifest(path);
      setManifest(m);
    } catch {
      setManifest(null);
    }
  }, [path]);

  useEffect(() => {
    if (show) {
      loadCommits();
      loadManifest();
    } else {
      setExpanded(null);
      setDetail(null);
      setLoadingCommit(null);
      setVerifyResult(null);
    }
  }, [show, loadCommits, loadManifest]);

  async function toggleCommit(commit: string) {
    if (expanded === commit) {
      setExpanded(null);
      setDetail(null);
      return;
    }
    // Increment sequence to invalidate any in-flight fetches
    const seq = ++fetchSeq.current;
    setExpanded(commit);
    setDetail(null);
    setLoadingCommit(commit);
    try {
      const d = await api.commitDetail(path, commit);
      // Only apply if this is still the latest request
      if (fetchSeq.current === seq) {
        setDetail(d);
      }
    } catch {
      if (fetchSeq.current === seq) {
        setDetail(null);
      }
    } finally {
      if (fetchSeq.current === seq) {
        setLoadingCommit(null);
      }
    }
  }

  async function doVerify() {
    if (!manifest) return;
    setVerifyResult(null);
    if (!manifest.signature) {
      setVerifyResult(
        "No signature found. This book has not been signed yet. " +
        "To verify, first sign it with: alaws sign " + path
      );
      return;
    }
    try {
      const r = await api.verify(path, manifest);
      setVerifyResult(r.ok ? "Verified OK" : `Failed: ${r.error}`);
    } catch (e) {
      setVerifyResult(`Error: ${e}`);
    }
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
        <button
          class="icon-button"
          title="Close"
          onMouseDown={(e) => {
            e.preventDefault();
            onClose();
          }}
        >
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
                {verifyResult}
              </span>
            )}
          </div>
        </div>
      )}

      {filterEntity && (
        <div class="history-filter-badge">
          {filterEntity.type === "law" ? `Law ${filterEntity.id}` : sectionTitle(filterEntity.sectionId ?? filterEntity.id)}
          <button class="icon-button" onClick={() => onClose()} title="Clear filter">×</button>
        </div>
      )}

      <div class="history-timeline">
        {commits.length === 0 && <p class="empty-state">No commits found.</p>}
        {commits.map((c) => (
          <div key={c.Commit} class={`history-commit ${expanded === c.Commit ? "expanded" : "collapsed"}`}>
            <div class="history-commit-row" onClick={() => toggleCommit(c.Commit)}>
              {expanded === c.Commit ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
              <code class="history-commit-hash">{c.Commit.slice(0, 8)}</code>
              <span class="history-commit-author">{authorName(c.Author)}</span>
              <span class="history-commit-time">{relativeTime(c.Date)}</span>
            </div>
            <div class="history-commit-summary">{c.Summary}</div>

            {expanded === c.Commit && (
              <div class="history-detail">
                {loadingCommit === c.Commit && <p class="empty-state">Loading…</p>}
                {detail && <CommitDetailView detail={detail} diffMode={diffMode} setDiffMode={setDiffMode} sectionTitle={sectionTitle} />}
              </div>
            )}
          </div>
        ))}
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
          {detail.files.map((f) => (
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
        <p class="empty-state">No detailed changes available.</p>
      )}
    </>
  );
}

function DiffView({ diff, mode, sectionTitle }: { diff: LawbookDiff; mode: "inline" | "side"; sectionTitle: (id: string) => string }) {
  const hasChanges = diff.AddedSections.length > 0 || diff.RemovedSections.length > 0 ||
    diff.AddedLaws.length > 0 || diff.RemovedLaws.length > 0 || diff.ModifiedLaws.length > 0;

  if (!hasChanges) return <p class="empty-state">No semantic changes.</p>;

  if (mode === "side") {
    return (
      <div class="diff-side">
        <div class="diff-side-col">
          <div class="diff-side-header">Before</div>
          {diff.RemovedSections.map((id) => <div key={id} class="diff-item diff-del"><Minus size={10} /> {sectionTitle(id)}</div>)}
          {diff.RemovedLaws.map((l) => <div key={`r-${l.SectionID}-${l.Index}`} class="diff-item diff-del"><Minus size={10} /> {l.Number}: {l.Text.slice(0, 80)}</div>)}
          {diff.ModifiedLaws.map((m) => <div key={`m-${m.SectionID}-${m.Index}`} class="diff-item diff-del"><Pencil size={10} /> {m.OldNumber}: {m.OldText.slice(0, 80)}</div>)}
          {diff.RemovedSections.length === 0 && diff.RemovedLaws.length === 0 && diff.ModifiedLaws.length === 0 && <div class="empty-state">No removals</div>}
        </div>
        <div class="diff-side-col">
          <div class="diff-side-header">After</div>
          {diff.AddedSections.map((id) => <div key={id} class="diff-item diff-add"><Plus size={10} /> {sectionTitle(id)}</div>)}
          {diff.AddedLaws.map((l) => <div key={`a-${l.SectionID}-${l.Index}`} class="diff-item diff-add"><Plus size={10} /> {l.Number}: {l.Text.slice(0, 80)}</div>)}
          {diff.ModifiedLaws.map((m) => <div key={`m2-${m.SectionID}-${m.Index}`} class="diff-item diff-add"><Pencil size={10} /> {m.NewNumber}: {m.NewText.slice(0, 80)}</div>)}
          {diff.AddedSections.length === 0 && diff.AddedLaws.length === 0 && diff.ModifiedLaws.length === 0 && <div class="empty-state">No additions</div>}
        </div>
      </div>
    );
  }

  return (
    <div class="diff-inline">
      {diff.AddedSections.map((id) => (
        <div key={id} class="diff-item diff-add"><Plus size={10} /> Section: {sectionTitle(id)}</div>
      ))}
      {diff.RemovedSections.map((id) => (
        <div key={id} class="diff-item diff-del"><Minus size={10} /> Section: {sectionTitle(id)}</div>
      ))}
      {diff.AddedLaws.map((l) => (
        <div key={`a-${l.SectionID}-${l.Index}`} class="diff-item diff-add"><Plus size={10} /> {l.Number}: {l.Text.slice(0, 120)}</div>
      ))}
      {diff.RemovedLaws.map((l) => (
        <div key={`r-${l.SectionID}-${l.Index}`} class="diff-item diff-del"><Minus size={10} /> {l.Number}: {l.Text.slice(0, 120)}</div>
      ))}
      {diff.ModifiedLaws.map((m) => (
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
