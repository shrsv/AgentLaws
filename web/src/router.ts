// Minimal hash router - no dependency, on purpose (this app has three
// routes; a router library would be more code than it saves).
import { useEffect, useState } from "preact/hooks";

export type Route =
  | { name: "books" }
  | { name: "book"; path: string; section?: string }
  | { name: "playground"; path: string };

function parseHash(hash: string): Route {
  const parts = hash.replace(/^#\/?/, "").split("/").filter(Boolean);
  if (parts.length === 0 || parts[0] !== "books") return { name: "books" };
  if (parts.length === 1) return { name: "books" };
  const path = decodeURIComponent(parts[1]);
  if (parts[2] === "playground") return { name: "playground", path };
  if (parts.length >= 3) return { name: "book", path, section: decodeURIComponent(parts[2]) };
  return { name: "book", path };
}

export function useRoute(): [Route, (r: Route) => void] {
  const [route, setRoute] = useState<Route>(() => parseHash(window.location.hash));

  useEffect(() => {
    const onHashChange = () => setRoute(parseHash(window.location.hash));
    window.addEventListener("hashchange", onHashChange);
    return () => window.removeEventListener("hashchange", onHashChange);
  }, []);

  const navigate = (r: Route) => {
    const hash =
      r.name === "books"
        ? "#/books"
        : r.name === "book"
          ? `#/books/${encodeURIComponent(r.path)}${r.section ? `/${encodeURIComponent(r.section)}` : ""}`
          : `#/books/${encodeURIComponent(r.path)}/playground`;
    window.location.hash = hash;
  };

  return [route, navigate];
}
