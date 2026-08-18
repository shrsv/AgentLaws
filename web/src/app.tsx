import { useEffect, useState } from "preact/hooks";
import "./app.css";
import { api } from "./api";
import { useRoute } from "./router";
import { BookPicker } from "./views/BookPicker";
import { BookDetail } from "./views/BookDetail";
import { Playground } from "./views/Playground";
import { WatchPanel } from "./components/WatchPanel";

export function App() {
  const [route, navigate] = useRoute();
  const [watchOpen, setWatchOpen] = useState(false);
  const [root, setRoot] = useState<string | null>(null);

  useEffect(() => {
    api.root().then((r) => setRoot(r.root)).catch(() => {});
  }, []);

  const currentPath = route.name === "books" ? null : route.path;

  return (
    <div class="app-shell">
      <div class="app-body">
        {route.name === "books" && <BookPicker navigate={navigate} />}
        {route.name === "book" && <BookDetail path={route.path} navigate={navigate} />}
        {route.name === "playground" && <Playground path={route.path} navigate={navigate} />}
      </div>

      <div class="app-footer">
        <button class="link-button" onClick={() => setWatchOpen((v) => !v)}>
          {watchOpen ? "Hide watch" : currentPath ? `Watch ${currentPath}` : "Watch all books"}
        </button>
      </div>

      <WatchPanel path={currentPath} root={root} open={watchOpen} onClose={() => setWatchOpen(false)} />
    </div>
  );
}
