import { useEffect } from "preact/hooks";

export interface Shortcut {
  key: string;
  description: string;
  action: () => void;
}

let registry: Shortcut[] = [];

export function getRegisteredShortcuts(): Shortcut[] {
  return [...registry];
}

function isInputFocused(): boolean {
  const el = document.activeElement;
  if (!el) return false;
  const tag = el.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || (el as HTMLElement).isContentEditable;
}

export function useShortcuts(shortcuts: Shortcut[]) {
  useEffect(() => {
    registry.push(...shortcuts);
    const handler = (e: KeyboardEvent) => {
      if (isInputFocused()) return;
      if (e.ctrlKey || e.metaKey || e.altKey) return;
      for (const s of shortcuts) {
        if (e.key === s.key) {
          e.preventDefault();
          s.action();
          return;
        }
      }
    };
    window.addEventListener("keydown", handler);
    return () => {
      window.removeEventListener("keydown", handler);
      for (const s of shortcuts) {
        const idx = registry.indexOf(s);
        if (idx !== -1) registry.splice(idx, 1);
      }
    };
  }, shortcuts.map((s) => s.action));
}

export function useEscape(handler: () => boolean) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        if (handler()) e.preventDefault();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [handler]);
}
