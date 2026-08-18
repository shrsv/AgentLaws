import { useEffect, useRef, useState } from "preact/hooks";
import { api, type BookInfo } from "../api";

interface Props {
  path: string | null; // a specific book, or null to watch every book under root
  root: string | null;
  open: boolean;
  onClose: () => void;
}

interface Entry {
  time: string;
  book: string;
  ok: boolean;
  diagnostics: { Severity: string; Code: string; Message: string }[];
  error: string;
}

// A dockable panel that streams live recompile events, reachable from any
// view via the status bar toggle (App.tsx) - not a separate route, so
// it's available "at any level" per the design goal: pinned to one book
// while inside it, or every book under root from the home view. Uses the
// same SSE endpoint (/api/book/watch) that backs `alaws watch`'s own
// live-reload behavior, opening one connection per book being watched.
export function WatchPanel({ path, root, open, onClose }: Props) {
  const [entries, setEntries] = useState<Entry[]>([]);
  const stopsRef = useRef<(() => void)[]>([]);

  useEffect(() => {
    stopsRef.current.forEach((stop) => stop());
    stopsRef.current = [];
    if (!open) return;

    setEntries([]);

    function watchOne(bookPath: string) {
      const stop = api.watch(bookPath, (ev) => {
        setEntries((prev) => [
          ...prev.slice(-99),
          {
            time: new Date().toLocaleTimeString(),
            book: bookPath,
            ok: ev.ok,
            diagnostics: ev.diagnostics ?? [],
            error: ev.error,
          },
        ]);
      });
      stopsRef.current.push(stop);
    }

    if (path) {
      watchOne(path);
    } else {
      api.discover(root ?? undefined).then((books: BookInfo[]) => {
        for (const b of books) watchOne(b.Path);
      });
    }

    return () => {
      stopsRef.current.forEach((stop) => stop());
      stopsRef.current = [];
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, path, root]);

  if (!open) return null;

  return (
    <div class="watch-panel">
      <div class="watch-panel-header">
        <span>Watch {path ? `— ${path}` : root ? `— all books under ${root}` : ""}</span>
        <button class="icon-button" onClick={onClose}>
          ×
        </button>
      </div>
      <div class="watch-panel-body">
        {entries.length === 0 && <p class="empty-state">Waiting for changes…</p>}
        {entries.map((e, i) => (
          <div key={i} class="watch-event">
            <div class={`watch-event-summary ${e.ok ? "" : "error"}`}>
              <span class="watch-entry-time">{e.time}</span>
              {!path && <span class="watch-event-book">{e.book}</span>}
              {e.ok ? (
                <span>recompiled — {e.diagnostics.length === 0 ? "no diagnostics" : `${e.diagnostics.length} diagnostic(s)`}</span>
              ) : (
                <span>compile failed: {e.error}</span>
              )}
            </div>
            {e.diagnostics.map((d, j) => (
              <div key={j} class={`watch-diagnostic ${d.Severity}`}>
                [{d.Severity}] {d.Code}: {d.Message}
              </div>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}
