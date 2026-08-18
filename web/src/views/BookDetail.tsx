import { useEffect, useState } from "preact/hooks";
import { api, type Section, type Diagnostic, type RenderedSection } from "../api";
import type { Route } from "../router";

interface Props {
  path: string;
  navigate: (r: Route) => void;
}

export function BookDetail({ path, navigate }: Props) {
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

  const reload = () => {
    setError(null);
    api
      .compile(path)
      .then((r) => {
        setTitle(r.lawbook.Metadata.Title);
        setSections(r.lawbook.Sections);
        setRendered(r.rendered ?? {});
        setDiagnostics(r.diagnostics ?? []);
        if (!selectedID && r.lawbook.Sections.length > 0) setSelectedID(r.lawbook.Sections[0].ID);
      })
      .catch((e) => setError(String(e)));
  };

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(reload, [path]);

  const selected = sections.find((s) => s.ID === selectedID) ?? null;
  const errorCount = diagnostics.filter((d) => d.Severity === "error").length;
  const warningCount = diagnostics.filter((d) => d.Severity === "warning").length;

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
    if (selectedID === s.ID) setSelectedID(null);
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
          ← Books
        </button>
        <span class="book-title">{title || path}</span>
        <span class="path">{path}</span>
        <div class="spacer" />
        <div class="export-group">
          <span class="export-group-label">Export this book:</span>
          <a class="link-button" href={api.exportURL(path, "html")} target="_blank" rel="noreferrer">
            HTML
          </a>
          <a class="link-button" href={api.exportURL(path, "pdf")} target="_blank" rel="noreferrer">
            PDF
          </a>
          <a class="link-button" href={api.exportURL(path, "md")} target="_blank" rel="noreferrer">
            Markdown
          </a>
        </div>
        <button class="link-button" onClick={() => navigate({ name: "books" })} title="Export every book, from the home view">
          Export all books…
        </button>
        <button class="link-button" onClick={() => navigate({ name: "playground", path })}>
          Playground
        </button>
      </div>

      {error && <p class="error-text">{error}</p>}

      <div class="workbench">
        <nav class="sidebar" aria-label="Lawbook sections">
          <div class="sidebar-title">
            Lawbook
            <button class="icon-button" title="New chapter" onClick={() => setNewChapter((v) => !v)}>
              +
            </button>
          </div>

          {newChapter && <NewNodeForm onSubmit={async (file, t, id) => { await api.createChapter(path, file, t, id); setNewChapter(false); reload(); }} onCancel={() => setNewChapter(false)} />}

          <ul class="tree">
            {sections.map((n) => (
              <li
                key={n.ID}
                class="tree-node"
                data-level={Math.min(n.Level, 2)}
                aria-selected={n.ID === selectedID}
                draggable
                onDragStart={() => setDragID(n.ID)}
                onDragOver={(e) => e.preventDefault()}
                onDrop={(e) => {
                  e.preventDefault();
                  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
                  const after = e.clientY - rect.top > rect.height / 2;
                  drop(n.ID, after);
                }}
                onClick={() => setSelectedID(n.ID)}
              >
                <span class="number">{n.Number}</span>
                <span class="node-title">{n.Title}</span>
                {n.Level === 1 && (
                  <button
                    class="icon-button"
                    title="New section here"
                    onClick={(e) => {
                      e.stopPropagation();
                      setNewSectionUnder(newSectionUnder === n.ID ? null : n.ID);
                    }}
                  >
                    +
                  </button>
                )}
                <button
                  class="icon-button"
                  title="Remove"
                  onClick={(e) => {
                    e.stopPropagation();
                    removeNode(n);
                  }}
                >
                  ×
                </button>
              </li>
            ))}
          </ul>
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
              <h1>
                {selected.Number} {selected.Title}
              </h1>
              <div class="section-id">{selected.ID}</div>
              {/* Commentary is Markdown (docs/PLAN1.md §7); rendered[...] is
                  the server's goldmark-rendered HTML (same pipeline as the
                  HTML export), not the raw source, so code spans/lists/
                  highlighted code blocks show up formatted instead of as
                  literal backticks and asterisks. */}
              <div class="commentary" dangerouslySetInnerHTML={{ __html: rendered[selected.ID]?.CommentaryHTML ?? escapeHTML(selected.Commentary) }} />

              {selected.Laws && selected.Laws.length > 0 ? (
                <ol class="law-list">
                  {selected.Laws.map((law) => (
                    <li key={law.Number}>
                      <span class="law-number">{law.Number}</span>
                      <span
                        class="law-text"
                        dangerouslySetInnerHTML={{ __html: rendered[selected.ID]?.LawHTML?.[law.Number] ?? escapeHTML(law.Text) }}
                      />
                      <button class="icon-button" title="Remove law" onClick={() => removeLaw(law.Index)}>
                        ×
                      </button>
                    </li>
                  ))}
                </ol>
              ) : (
                <p class="empty-state">This section has no laws of its own.</p>
              )}

              <div class="add-law-form">
                <input
                  placeholder="Add a law…"
                  value={newLawText}
                  onInput={(e) => setNewLawText((e.target as HTMLInputElement).value)}
                  onKeyDown={(e) => e.key === "Enter" && addLaw()}
                />
                <button class="btn" onClick={addLaw}>
                  Add
                </button>
              </div>
            </>
          ) : (
            <p class="empty-state">Select a section.</p>
          )}
        </main>
      </div>

      <div class="statusbar">
        <span class={`diagnostic-count ${errorCount > 0 ? "error" : ""}`}>{errorCount} errors</span>
        <span class={`diagnostic-count ${warningCount > 0 ? "warning" : ""}`}>{warningCount} warnings</span>
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
          Create
        </button>
        <button class="btn" type="button" onClick={props.onCancel}>
          Cancel
        </button>
      </div>
    </form>
  );
}

// escapeHTML is only a fallback for the brief window before the server's
// rendered HTML has loaded (or if it's missing for some reason) - it must
// escape, not pass through, since the string it's given is raw Markdown
// source, not markup.
function escapeHTML(s: string): string {
  const div = document.createElement("div");
  div.textContent = s;
  return div.innerHTML;
}
