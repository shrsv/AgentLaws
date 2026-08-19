import { X } from "lucide-preact";
import type { Shortcut } from "../hooks/useKeyboardShortcuts";

interface Props {
  show: boolean;
  onClose: () => void;
  shortcuts: Shortcut[];
}

function formatKey(key: string): string {
  switch (key) {
    case "ArrowLeft":
      return "\u2190";
    case "ArrowRight":
      return "\u2192";
    case "ArrowUp":
      return "\u2191";
    case "ArrowDown":
      return "\u2193";
    case " ":
      return "Space";
    case "Escape":
      return "Esc";
    default:
      return key.length === 1 ? key.toUpperCase() : key;
  }
}

export function HotkeyHelp({ show, onClose, shortcuts }: Props) {
  if (!show) return null;

  return (
    <div class="source-overlay" onClick={onClose}>
      <div class="hotkey-help" onClick={(e) => e.stopPropagation()}>
        <div class="hotkey-help-header">
          <span class="hotkey-help-title">Keyboard Shortcuts</span>
          <button class="icon-button" title="Close" onClick={onClose}>
            <X size={11} />
          </button>
        </div>
        <div class="hotkey-help-body">
          {shortcuts.map((s) => (
            <div key={s.key} class="hotkey-help-row">
              <kbd class="hotkey-key">{formatKey(s.key)}</kbd>
              <span class="hotkey-desc">{s.description}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
