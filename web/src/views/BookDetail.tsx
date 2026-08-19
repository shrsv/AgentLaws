import { useEffect, useMemo, useState, useRef } from "preact/hooks";
import type { VNode } from "preact";
import { History, Plus, X, Clock, User, ArrowLeft, FileText, Download, FlaskConical, CircleAlert, TriangleAlert, List, MousePointer, ChevronLeft, ChevronRight, Copy, Code, Check } from "lucide-preact";
import { api, type Section, type Diagnostic, type RenderedSection } from "../api";
import type { Route } from "../router";
import { HistorySidebar } from "../components/HistorySidebar";

interface Props {
  path: string;
  section: string | null;
  navigate: (r: Route) => void;
}

// TreeNode is a Section plus its child sections, rebuilt from the compiled
// IR's flat, depth-first list (Level/ParentID) into a nested tree so the
// sidebar can show the book's chapter/section/subsection hierarchy.
interface TreeNode {
  section: Section;
  children: TreeNode[];
}

function buildTree(sections: Section[]): TreeNode[] {
  const roots: TreeNode[] = [];
  const stack: TreeNode[] = [];
  for (const s of sections) {
    const node: TreeNode = { section: s, children: [] };
    while (stack.length > 0 && stack[stack.length - 1].section.Level >= s.Level) {
      stack.pop();
    }
    if (stack.length > 0) stack[stack.length - 1].children.push(node);
    else roots.push(node);
    stack.push(node);
  }
  return roots;
}

export function BookDetail({ path, section, navigate }: Props) {
  const [title, setTitle] = useState("");
  const [sections, setSections] = useState<Section[]>([]);
  const [rendered, setRendered] = useState<Record<string, RenderedSection>>({});
  const [diagnostics, setDiagnostics] = useState<Diagnostic[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [selectedID, setSelectedID] = useState<string | null>(null);
  const [dragID, setDragID] = useState<string | null>(null);
  const [newLawText, setNewLawText] = useState("");
  const [newChapter, setNewChapter] = useState(false);
  const [newSectionUnder, setNewSectionUnder] = useState<string | null>(null);
  const [showHistory, setShowHistory] = useState(false);
  const [historyFilter, setHistoryFilter] = useState<{ type: "section" | "law"; id: string; sectionId?: string } | null>(null);
  const [lawAuthors, setLawAuthors] = useState<Record<string, string>>({});
  const [sourceContent, setSourceContent] = useState<{ path: string; lineStart: number; lineEnd: number; content: string } | null>(null);
  const [copiedPath, setCopiedPath] = useState(false);
  const fetchingAuthors = useRef<Set<string>>(new Set());

  const reload = () => {
    setError(null);
    api
      .compile(path)
      .then((r) => {
        setTitle(r.lawbook.Metadata.Title);
        setSections(r.lawbook.Sections);
        setRendered(r.rendered ?? {});
        setDiagnostics(r.diagnostics ?? []);
        // Functional update: read current selection (the URL's section may
        // have just been restored by the section-sync effect), not the
        // stale closure value from this render.
        setSelectedID((prev) => prev ?? r.lawbook.Sections[0]?.ID ?? null);
      })
      .catch((e) => setError(String(e)));
  };

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(reload, [path]);

  // The URL carries the selected section (e.g. #/books/<book>/<section>), so
  // a refresh or back/forward restores the same page. Clicking a tree node
  // pushes the id into the hash via select(); this effect picks it back up
  // when the hash changes (or on first load after a refresh).
  useEffect(() => {
    if (section) setSelectedID(section);
  }, [section]);

  const select = (id: string) => {
    setSelectedID(id);
    navigate({ name: "book", path, section: id });
  };

  const currentIndex = selectedID ? sections.findIndex((s) => s.ID === selectedID) : -1;
  const hasPrev = currentIndex > 0;
  const hasNext = currentIndex >= 0 && currentIndex < sections.length - 1;

  const goPrev = () => { if (hasPrev) select(sections[currentIndex - 1].ID); };
  const goNext = () => { if (hasNext) select(sections[currentIndex + 1].ID); };

  const relativePath = (srcPath: string) => {
    const bookDir = path.includes(".") ? path.replace(/[/\\][^/\\]+$/, "") : path;
    if (srcPath.startsWith(bookDir)) return srcPath.slice(bookDir.length + 1);
    return srcPath;
  };

  const copyPath = (srcPath: string) => {
    navigator.clipboard.writeText(srcPath).then(() => {
      setCopiedPath(true);
      setTimeout(() => setCopiedPath(false), 1500);
    });
  };

  const loadSource = (srcPath: string, lineStart: number, lineEnd: number) => {
    api.source(srcPath, lineStart, lineEnd).then(setSourceContent).catch(() => setSourceContent(null));
  };

  // Recursive sidebar renderer: each section is a row (with the same
  // select/drag/history/remove handlers as before), and any children are
  // rendered as a nested <ul> indented under it.
  const renderTree = (nodes: TreeNode[]): VNode[] =>
    nodes.map((n) => {
      const s = n.section;
      return (
        <li key={s.ID} class="tree-branch">
          <div
            class="tree-node"
            data-level={s.Level}
            aria-selected={s.ID === selectedID}
            draggable
            onDragStart={() => setDragID(s.ID)}
            onDragOver={(e) => e.preventDefault()}
            onDrop={(e) => {
              e.preventDefault();
              const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
              const after = e.clientY - rect.top > rect.height / 2;
              drop(s.ID, after);
            }}
            onClick={() => select(s.ID)}
          >
            <span class="number">{s.Number}</span>
            <span class="node-title">{s.Title}</span>
            <span class="node-actions">
              <button
                class="icon-button"
                title="History"
                onClick={(e) => {
                  e.stopPropagation();
                  openSectionHistory(s.ID);
                }}
              >
                <Clock size={11} />
              </button>
              {s.Level > 0 && (
                <button
                  class="icon-button"
                  title="New section here"
                  onClick={(e) => {
                    e.stopPropagation();
                    setNewSectionUnder(newSectionUnder === s.ID ? null : s.ID);
                  }}
                >
                  <Plus size={11} />
                </button>
              )}
              <button
                class="icon-button"
                title="Remove"
                onClick={(e) => {
                  e.stopPropagation();
                  removeNode(s);
                }}
              >
                <X size={11} />
              </button>
            </span>
          </div>
          {n.children.length > 0 && (
            <ul class="tree-children">{renderTree(n.children)}</ul>
          )}
        </li>
      );
    });

  const tree = useMemo(() => buildTree(sections), [sections]);

  const selected = sections.find((s) => s.ID === selectedID) ?? null;
  const errorCount = diagnostics.filter((d) => d.Severity === "error").length;
  const warningCount = diagnostics.filter((d) => d.Severity === "warning").length;

  function openSectionHistory(sectionId: string) {
    setHistoryFilter({ type: "section", id: sectionId, sectionId });
    setShowHistory(true);
  }

  function openLawHistory(lawNumber: string, sectionId: string) {
    setHistoryFilter({ type: "law", id: lawNumber, sectionId });
    setShowHistory(true);
  }

  function fetchLawAuthor(sectionId: string, lawNumber: string) {
    const key = `${sectionId}:${lawNumber}`;
    if (lawAuthors[key] !== undefined || fetchingAuthors.current.has(key)) return;
    fetchingAuthors.current.add(key);
    api
      .history(path, lawNumber)
      .then((h) => {
        if (h.Modifications && h.Modifications.length > 0) {
          const last = h.Modifications[0];
          const name = last.Author.replace(/ <.*>$/, "");
          setLawAuthors((prev) => ({ ...prev, [key]: name }));
        } else {
          setLawAuthors((prev) => ({ ...prev, [key]: "" }));
        }
      })
      .catch(() => {
        setLawAuthors((prev) => ({ ...prev, [key]: "" }));
      });
  }

  async function drop(targetID: string, after: boolean) {
    if (!dragID || dragID === targetID) return;
    const dragged = sections.find((s) => s.ID === dragID);
    const target = sections.find((s) => s.ID === targetID);
    if (!dragged || !target) return;

    const kind = dragged.Level === 1 ? "chapter" : "section";
    const opts = after ? { after: targetID } : { before: targetID };
    if (kind === "section") {
      const newParentId = target.Level < dragged.Level ? target.ID : target.ParentID;
      await api.move(path, "section", dragID, { ...opts, newParentId });
    } else {
      await api.move(path, "chapter", dragID, opts);
    }
    setDragID(null);
    reload();
  }

  async function removeNode(s: Section) {
    const kind = s.Level === 1 ? "chapter" : "section";
    if (!confirm(`Remove ${kind} "${s.Title}" (${s.ID})?`)) return;
    await api.remove(path, kind, s.ID, true);
    if (selectedID === s.ID) {
      setSelectedID(null);
      navigate({ name: "book", path });
    }
    reload();
  }

  async function addLaw() {
    if (!selected || !newLawText.trim()) return;
    await api.addLaw(path, selected.ID, newLawText.trim());
    setNewLawText("");
    reload();
  }

  async function removeLaw(number: number) {
    if (!selected) return;
    await api.removeLaw(path, selected.ID, number, true);
    reload();
  }

  return (
    <div class="shell">
      <div class="titlebar">
        <button class="link-button" onClick={() => navigate({ name: "books" })}>
          <ArrowLeft size={12} /> Books
        </button>
        <span class="book-title">{title || path}</span>
        <span class="path">{path}</span>
        <div class="spacer" />
        <div class="export-group">
          <span class="export-group-label">Export this book:</span>
          <a class="link-button" href={api.exportURL(path, "html")} target="_blank" rel="noreferrer">
            <FileText size={12} /> HTML
          </a>
          <a class="link-button" href={api.exportURL(path, "pdf")} target="_blank" rel="noreferrer">
            <FileText size={12} /> PDF
          </a>
          <a class="link-button" href={api.exportURL(path, "md")} target="_blank" rel="noreferrer">
            <FileText size={12} /> Markdown
          </a>
        </div>
        <button class="link-button" onClick={() => navigate({ name: "books" })} title="Export every book, from the home view">
          <Download size={12} /> Export all books…
        </button>
        <button class="link-button" onClick={() => navigate({ name: "playground", path })}>
          <FlaskConical size={12} /> Playground
        </button>
        <button
          class={`link-button ${showHistory ? "active" : ""}`}
          onClick={() => { setShowHistory((v) => !v); if (showHistory) setHistoryFilter(null); }}
        >
          <History size={12} style={{ verticalAlign: "middle", marginRight: 4 }} />
          History
        </button>
      </div>

      {error && <p class="error-text"><CircleAlert size={12} /> {error}</p>}

      <div class="workbench">
        <nav class="sidebar" aria-label="Lawbook sections">
          <div class="sidebar-title">
            Lawbook
            <button class="icon-button" title="New chapter" onClick={() => setNewChapter((v) => !v)}>
              <Plus size={12} />
            </button>
          </div>

          {newChapter && <NewNodeForm onSubmit={async (file, t, id) => { await api.createChapter(path, file, t, id); setNewChapter(false); reload(); }} onCancel={() => setNewChapter(false)} />}

          <ul class="tree">{renderTree(tree)}</ul>
          {newSectionUnder && (
            <NewNodeForm
              parent={newSectionUnder}
              onSubmit={async (file, t, id) => {
                await api.createSection(path, file, t, id, newSectionUnder);
                setNewSectionUnder(null);
                reload();
              }}
              onCancel={() => setNewSectionUnder(null)}
            />
          )}
        </nav>

        <div class="divider" />

        <main class="detail">
          {selected ? (
            <>
              <div class="detail-header">
                <div class="detail-nav">
                  <button class="icon-button nav-arrow" title="Previous section" disabled={!hasPrev} onClick={goPrev}>
                    <ChevronLeft size={16} />
                  </button>
                  <button class="icon-button nav-arrow" title="Next section" disabled={!hasNext} onClick={goNext}>
                    <ChevronRight size={16} />
                  </button>
                </div>
                <h1>
                  {selected.Number} {selected.Title}
                </h1>
              </div>
              <div class="section-id">
                {selected.ID}
              </div>
              <div class="section-file">
                <Code size={11} />
                <span class="section-file-path" title={selected.Source.Path}>{relativePath(selected.Source.Path)}</span>
                <button class="icon-button" title="Copy path" onClick={() => copyPath(selected.Source.Path)}>
                  {copiedPath ? <Check size={11} /> : <Copy size={11} />}
                </button>
                <button class="icon-button" title="View source" onClick={() => loadSource(selected.Source.Path, selected.Source.LineStart, selected.Source.LineEnd)}>
                  <Code size={11} /> Source
                </button>
              </div>
              <div class="commentary" dangerouslySetInnerHTML={{ __html: rendered[selected.ID]?.CommentaryHTML ?? escapeHTML(selected.Commentary) }} />

              {selected.Laws && selected.Laws.length > 0 ? (
                <ol class="law-list">
                  {selected.Laws.map((law) => {
                    const authorKey = `${selected.ID}:${law.Number}`;
                    const author = lawAuthors[authorKey];
                    return (
                      <li key={law.Number} onMouseEnter={() => fetchLawAuthor(selected.ID, law.Number)}>
                        <span class="law-number">{law.Number}</span>
                        <span
                          class="law-text"
                          dangerouslySetInnerHTML={{ __html: rendered[selected.ID]?.LawHTML?.[law.Number] ?? escapeHTML(law.Text) }}
                        />
                        {author !== undefined && author !== "" && (
                          <span class="law-author">
                            <User size={10} /> {author}
                          </span>
                        )}
                        <span class="law-actions">
                          <button
                            class="icon-button"
                            title="History"
                            onClick={() => openLawHistory(law.Number, selected.ID)}
                          >
                            <Clock size={11} />
                          </button>
                          <button class="icon-button" title="Remove law" onClick={() => removeLaw(law.Index)}>
                            <X size={11} />
                          </button>
                        </span>
                      </li>
                    );
                  })}
                </ol>
              ) : (
                <p class="empty-state"><List size={14} /> This section has no laws of its own.</p>
              )}

              <div class="add-law-form">
                <input
                  placeholder="Add a law…"
                  value={newLawText}
                  onInput={(e) => setNewLawText((e.target as HTMLInputElement).value)}
                  onKeyDown={(e) => e.key === "Enter" && addLaw()}
                />
                <button class="btn" onClick={addLaw}>
                  <Plus size={12} /> Add
                </button>
              </div>
            </>
          ) : (
            <p class="empty-state"><MousePointer size={14} /> Select a section.</p>
          )}
        </main>
      </div>

      <HistorySidebar
        path={path}
        show={showHistory}
        onClose={() => { setShowHistory(false); setHistoryFilter(null); }}
        filterEntity={historyFilter}
        sections={sections}
      />

      {sourceContent && (
        <div class="source-overlay" onClick={() => setSourceContent(null)}>
          <div class="source-modal" onClick={(e) => e.stopPropagation()}>
            <div class="source-modal-header">
              <span class="source-modal-title">{relativePath(sourceContent.path)}:{sourceContent.lineStart}-{sourceContent.lineEnd}</span>
              <div class="source-modal-actions">
                <button class="icon-button" title="Copy source" onClick={() => { navigator.clipboard.writeText(sourceContent.content); }}>
                  <Copy size={11} />
                </button>
                <button class="icon-button" title="Close" onClick={() => setSourceContent(null)}>
                  <X size={11} />
                </button>
              </div>
            </div>
            <pre class="source-modal-content"><code>{sourceContent.content}</code></pre>
          </div>
        </div>
      )}

      <div class="statusbar">
        <span class={`diagnostic-count ${errorCount > 0 ? "error" : ""}`}><CircleAlert size={11} /> {errorCount} errors</span>
        <span class={`diagnostic-count ${warningCount > 0 ? "warning" : ""}`}><TriangleAlert size={11} /> {warningCount} warnings</span>
      </div>
    </div>
  );
}

function NewNodeForm(props: {
  parent?: string;
  onSubmit: (file: string, title: string, id: string) => void;
  onCancel: () => void;
}) {
  const [file, setFile] = useState("");
  const [t, setT] = useState("");
  const [id, setId] = useState("");
  return (
    <form
      class="new-node-form"
      onSubmit={(e) => {
        e.preventDefault();
        props.onSubmit(file, t, id);
      }}
    >
      <input placeholder="file.md" value={file} onInput={(e) => setFile((e.target as HTMLInputElement).value)} required autoFocus />
      <input placeholder="title" value={t} onInput={(e) => setT((e.target as HTMLInputElement).value)} required />
      <input placeholder="id" value={id} onInput={(e) => setId((e.target as HTMLInputElement).value)} required />
      <div class="new-node-form-actions">
        <button class="btn btn-primary" type="submit">
          <Plus size={12} /> Create
        </button>
        <button class="btn" type="button" onClick={props.onCancel}>
          <X size={12} /> Cancel
        </button>
      </div>
    </form>
  );
}

function escapeHTML(s: string): string {
  const div = document.createElement("div");
  div.textContent = s;
  return div.innerHTML;
}
