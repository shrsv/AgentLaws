// Minimal hash router - no dependency, on purpose (this app has three
// routes; a router library would be more code than it saves).
import { useEffect, useState } from "preact/hooks";

export type Route =
  | { name: "books" }
  | { name: "book"; path: string; section?: string; law?: string }
  | { name: "prompts"; path: string; id?: string }
  | { name: "playground"; path: string };

// The hash format for law links uses ~ to separate section ID from law slug:
//   #/books/<path>/<sectionID>~<lawSlug>
// Both section IDs and law slugs are dot-separated, so ~ is unambiguous.
function parseHash(hash: string): Route {
  const parts = hash.replace(/^#\/?/, "").split("/").filter(Boolean);
  if (parts.length === 0 || parts[0] !== "books") return { name: "books" };
  if (parts.length === 1) return { name: "books" };
  const path = decodeURIComponent(parts[1]);
  if (parts[2] === "playground") return { name: "playground", path };
  if (parts[2] === "prompts") {
    const id = parts[3] ? decodeURIComponent(parts[3]) : undefined;
    return { name: "prompts", path, id };
  }
  if (parts.length >= 3) {
    const raw = decodeURIComponent(parts[2]);
    const tildeIdx = raw.indexOf("~");
    if (tildeIdx >= 0) {
      return { name: "book", path, section: raw.slice(0, tildeIdx), law: raw.slice(tildeIdx + 1) };
    }
    return { name: "book", path, section: raw };
  }
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
          ? `#/books/${encodeURIComponent(r.path)}${r.section ? `/${encodeURIComponent(r.section)}${r.law ? `~${encodeURIComponent(r.law)}` : ""}` : ""}`
          : r.name === "prompts"
            ? `#/books/${encodeURIComponent(r.path)}/prompts${r.id ? `/${encodeURIComponent(r.id)}` : ""}`
            : `#/books/${encodeURIComponent(r.path)}/playground`;
    window.location.hash = hash;
  };

  return [route, navigate];
}
