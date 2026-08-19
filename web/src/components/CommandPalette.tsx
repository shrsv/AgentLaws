import { useEffect, useRef, useState } from "preact/hooks";
import { Search, BookOpen, Hash } from "lucide-preact";

interface Item {
  id: string;
  label: string;
  detail: string;
  icon: "book" | "section";
  action: () => void;
}

interface Props {
  open: boolean;
  onClose: () => void;
  items: Item[];
}

function fuzzyMatch(query: string, text: string): boolean {
  const q = query.toLowerCase();
  const t = text.toLowerCase();
  let qi = 0;
  for (let ti = 0; ti < t.length && qi < q.length; ti++) {
    if (t[ti] === q[qi]) qi++;
  }
  return qi === q.length;
}

export function CommandPalette({ open, onClose, items }: Props) {
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  const filtered = query.trim() === ""
    ? items
    : items.filter((it) => fuzzyMatch(query, it.label) || fuzzyMatch(query, it.detail));

  useEffect(() => {
    if (open) {
      setQuery("");
      setSelected(0);
      setTimeout(() => inputRef.current?.focus(), 0);
    }
  }, [open]);

  useEffect(() => {
    if (selected >= filtered.length) setSelected(Math.max(0, filtered.length - 1));
  }, [filtered.length, selected]);

  const handleKey = (e: KeyboardEvent) => {
    if (e.key === "Escape") {
      e.preventDefault();
      onClose();
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelected((i) => (i + 1) % filtered.length);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelected((i) => (i - 1 + filtered.length) % filtered.length);
    } else if (e.key === "Enter") {
      e.preventDefault();
      filtered[selected]?.action();
      onClose();
    }
  };

  if (!open) return null;

  return (
    <div class="cmd-palette-backdrop" onClick={onClose}>
      <div class="cmd-palette" onClick={(e) => e.stopPropagation()}>
        <div class="cmd-palette-input-row">
          <Search size={14} class="cmd-palette-search-icon" />
          <input
            ref={inputRef}
            class="cmd-palette-input"
            placeholder="Jump to…"
            value={query}
            onInput={(e) => { setQuery((e.target as HTMLInputElement).value); setSelected(0); }}
            onKeyDown={handleKey}
          />
        </div>
        <ul class="cmd-palette-list">
          {filtered.length === 0 && (
            <li class="cmd-palette-empty">No matches</li>
          )}
          {filtered.map((it, i) => (
            <li
              key={it.id}
              class={`cmd-palette-item ${i === selected ? "selected" : ""}`}
              onClick={() => { it.action(); onClose(); }}
              onMouseEnter={() => setSelected(i)}
            >
              {it.icon === "book"
                ? <BookOpen size={12} class="cmd-palette-item-icon" />
                : <Hash size={12} class="cmd-palette-item-icon" />}
              <span class="cmd-palette-item-label">{it.label}</span>
              <span class="cmd-palette-item-detail">{it.detail}</span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
