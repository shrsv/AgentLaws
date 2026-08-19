import { useEffect, useState } from "preact/hooks";
import { Eye, EyeOff } from "lucide-preact";
import "./app.css";
import { api, type Section } from "./api";
import { useRoute } from "./router";
import { BookPicker } from "./views/BookPicker";
import { BookDetail } from "./views/BookDetail";
import { Playground } from "./views/Playground";
import { WatchPanel } from "./components/WatchPanel";
import { CommandPalette } from "./components/CommandPalette";

export function App() {
  const [route, navigate] = useRoute();
  const [watchOpen, setWatchOpen] = useState(false);
  const [watchMinimized, setWatchMinimized] = useState(() => localStorage.getItem("watchMinimized") === "true");
  const [root, setRoot] = useState<string | null>(null);
  const [cmdOpen, setCmdOpen] = useState(false);
  const [books, setBooks] = useState<{ Path: string; Title: string }[]>([]);
  const [sections, setSections] = useState<Section[]>([]);

  useEffect(() => {
    api.root().then((r) => setRoot(r.root)).catch(() => {});
    api.discover().then(setBooks).catch(() => {});
  }, []);

  const currentPath = route.name === "books" ? null : route.path;

  // Auto-watch when a book is opened; stop when navigated away
  useEffect(() => {
    if (currentPath) {
      setWatchOpen(true);
    } else {
      setWatchOpen(false);
    }
  }, [currentPath]);

  // Persist minimized state
  useEffect(() => {
    localStorage.setItem("watchMinimized", String(watchMinimized));
  }, [watchMinimized]);

  // Dynamic page title
  useEffect(() => {
    let pageTitle = "AgentLaws";
    if (route.name === "books") {
      pageTitle = "Lawbooks — AgentLaws";
    } else if (route.name === "book") {
      const bookTitle = books.find((b) => b.Path === route.path)?.Title || route.path;
      if (route.section) {
        const sec = sections.find((s) => s.ID === route.section);
        if (sec) pageTitle = `${sec.Number} ${sec.Title} — ${bookTitle} — AgentLaws`;
        else pageTitle = `${route.section} — ${bookTitle} — AgentLaws`;
      } else {
        pageTitle = `${bookTitle} — AgentLaws`;
      }
    } else if (route.name === "playground") {
      pageTitle = `Playground — ${route.path} — AgentLaws`;
    }
    document.title = pageTitle;
  }, [route, sections, books]);

  // Ctrl+P to open command palette
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "p") {
        e.preventDefault();
        setCmdOpen((v) => !v);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  // Refresh book list when command palette opens
  useEffect(() => {
    if (cmdOpen) api.discover().then(setBooks).catch(() => {});
  }, [cmdOpen]);

  const cmdItems = [
    ...books.map((b) => ({
      id: `book:${b.Path}`,
      label: b.Title || b.Path,
      detail: b.Path,
      icon: "book" as const,
      action: () => navigate({ name: "book", path: b.Path }),
    })),
    ...(route.name === "book"
      ? sections.map((s) => ({
          id: `section:${s.ID}`,
          label: `${s.Number} ${s.Title}`,
          detail: s.ID,
          icon: "section" as const,
          action: () => navigate({ name: "book", path: route.path, section: s.ID }),
        }))
      : []),
  ];

  return (
    <div class="app-shell">
      <div class="app-body">
        {route.name === "books" && <BookPicker navigate={navigate} />}
        {route.name === "book" && <BookDetail path={route.path} section={route.section ?? null} law={route.law ?? null} navigate={navigate} onSectionsChange={setSections} onOpenCommandPalette={() => setCmdOpen(true)} />}
        {route.name === "playground" && <Playground path={route.path} navigate={navigate} />}
      </div>

      <div class="app-footer">
        <button class="link-button" onClick={() => setWatchOpen((v) => !v)}>
          {watchOpen ? <EyeOff size={12} /> : <Eye size={12} />}
          {watchOpen ? (watchMinimized ? " Watch (minimized)" : " Hide watch") : currentPath ? ` Watch ${currentPath}` : " Watch all books"}
        </button>
      </div>

      <WatchPanel
        path={currentPath}
        root={root}
        open={watchOpen}
        minimized={watchMinimized}
        onClose={() => setWatchOpen(false)}
        onMinimize={() => setWatchMinimized((v) => !v)}
      />
      <CommandPalette open={cmdOpen} onClose={() => setCmdOpen(false)} items={cmdItems} />
    </div>
  );
}
