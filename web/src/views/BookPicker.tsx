import { useEffect, useState } from "preact/hooks";
import { Plus, X, FileText, BookOpen, Loader2, CircleAlert, Inbox } from "lucide-preact";
import { api, type BookInfo } from "../api";
import type { Route } from "../router";

interface Props {
  navigate: (r: Route) => void;
}

export function BookPicker({ navigate }: Props) {
  const [root, setRoot] = useState<string | null>(null);
  const [books, setBooks] = useState<BookInfo[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [newPath, setNewPath] = useState("");
  const [newTitle, setNewTitle] = useState("");

  const reload = () => {
    setError(null);
    api
      .root()
      .then((r) => setRoot(r.root))
      .catch(() => setRoot(null));
    api
      .discover()
      .then(setBooks)
      .catch((e) => setError(String(e)));
  };

  useEffect(reload, []);

  const create = async (e: Event) => {
    e.preventDefault();
    try {
      await api.createBook(newPath, newTitle);
      setCreating(false);
      setNewPath("");
      setNewTitle("");
      reload();
    } catch (e) {
      setError(String(e));
    }
  };

  return (
    <div class="book-picker">
      <div class="book-picker-header">
        <div>
          <h1>Lawbooks</h1>
          {root && <div class="book-picker-root">{root}</div>}
        </div>
        <div class="book-picker-actions">
          {books && books.length > 0 && (
            <div class="export-group">
              <span class="export-group-label">Export all books:</span>
              <a class="link-button" href={api.exportAllURL(undefined, "html")} target="_blank" rel="noreferrer">
                <FileText size={12} /> HTML
              </a>
              <a class="link-button" href={api.exportAllURL(undefined, "pdf")} target="_blank" rel="noreferrer">
                <FileText size={12} /> PDF
              </a>
              <a class="link-button" href={api.exportAllURL(undefined, "md")} target="_blank" rel="noreferrer">
                <FileText size={12} /> Markdown
              </a>
            </div>
          )}
          <button class="btn" onClick={() => setCreating((v) => !v)}>
            {creating ? <><X size={12} /> Cancel</> : <><Plus size={12} /> New book</>}
          </button>
        </div>
      </div>

      {creating && (
        <form class="new-book-form" onSubmit={create}>
          <input placeholder="path, e.g. ./governance" value={newPath} onInput={(e) => setNewPath((e.target as HTMLInputElement).value)} required />
          <input placeholder="title" value={newTitle} onInput={(e) => setNewTitle((e.target as HTMLInputElement).value)} required />
          <button class="btn btn-primary" type="submit">
            <Plus size={12} /> Create
          </button>
        </form>
      )}

      {error && <p class="error-text"><CircleAlert size={12} /> {error}</p>}

      {books === null && !error && <p class="empty-state"><Loader2 size={14} class="spin" /> Loading…</p>}

      {books !== null && books.length === 0 && (
        <p class="empty-state"><Inbox size={14} /> No lawbooks found under {root ?? "this root"}. Create one above, or run "alaws books create" from the CLI.</p>
      )}

      <div class="book-grid">
        {books?.map((b) => (
          <button key={b.Path} class="book-card" title={`${b.Title || "(untitled)"}\n${b.Path}`} onClick={() => navigate({ name: "book", path: b.Path })}>
            <BookOpen size={16} class="book-card-icon" />
            <div class="book-card-body">
              <div class="book-card-title">{b.Title || "(untitled)"}</div>
              <div class="book-card-path">{b.Path}</div>
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}
