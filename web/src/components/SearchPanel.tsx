import { useState, useRef, useEffect, useCallback } from "preact/hooks";
import type { VNode } from "preact";
import { X, Search, CaseSensitive, WholeWord, Regex, ChevronDown, ChevronRight, FileText } from "lucide-preact";
import { api, type Section, type SearchMatch } from "../api";

interface Props {
  bookPath: string;
  sections: Section[];
  activeSectionID: string | null;
  initialQuery?: string;
  onQueryChange?: (query: string) => void;
  onClose: () => void;
  onNavigate: (sectionId: string) => void;
  onNavigatePrompt?: (promptId: string) => void;
}

interface ScopeState {
  mode: "all" | "custom";
  selected: Set<string>;
  expanded: Set<string>;
}

export function SearchPanel({ bookPath, sections, activeSectionID, initialQuery, onQueryChange, onClose, onNavigate, onNavigatePrompt }: Props) {
  const [query, setQuery] = useState(initialQuery ?? "");
  const [caseSensitive, setCaseSensitive] = useState(false);
  const [wholeWord, setWholeWord] = useState(false);
  const [regex, setRegex] = useState(false);
  const [results, setResults] = useState<SearchMatch[] | null>(null);
  const [searching, setSearching] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [scope, setScope] = useState<ScopeState>({ mode: "all", selected: new Set(), expanded: new Set() });
  const [showScope, setShowScope] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    inputRef.current?.focus();
    if (initialQuery?.trim()) doSearch(initialQuery);
  }, []);

  const doSearch = useCallback(
    (q: string) => {
      if (!q.trim()) {
        setResults(null);
        setError(null);
        return;
      }
      setSearching(true);
      setError(null);
      const sectionIds = scope.mode === "custom" ? [...scope.selected] : undefined;
      api
        .search(bookPath, q, { caseSensitive, wholeWord, regex, sectionIds })
        .then((r) => { setResults(r); setSearching(false); })
        .catch((e) => { setError(String(e)); setSearching(false); });
    },
    [bookPath, caseSensitive, wholeWord, regex, scope],
  );

  const onInput = (val: string) => {
    setQuery(val);
    onQueryChange?.(val);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => doSearch(val), 250);
  };

  const onKey = (e: KeyboardEvent) => {
    if (e.key === "Escape") {
      e.preventDefault();
      onClose();
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (debounceRef.current) clearTimeout(debounceRef.current);
      doSearch(query);
    }
  };

  const toggleSection = (id: string) => {
    setScope((prev) => {
      const next = new Set(prev.selected);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return { ...prev, mode: "custom", selected: next };
    });
  };

  const toggleChapter = (_chapterID: string, childIDs: string[]) => {
    setScope((prev) => {
      const next = new Set(prev.selected);
      const allSelected = childIDs.every((id) => next.has(id));
      if (allSelected) {
        childIDs.forEach((id) => next.delete(id));
      } else {
        childIDs.forEach((id) => next.add(id));
      }
      return { ...prev, mode: "custom", selected: next };
    });
  };

  const toggleExpand = (id: string) => {
    setScope((prev) => {
      const next = new Set(prev.expanded);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return { ...prev, expanded: next };
    });
  };

  const setScopeMode = (mode: "all" | "custom") => {
    if (mode === "all") {
      setScope((prev) => ({ ...prev, mode, selected: new Set() }));
    } else {
      setScope((prev) => ({ ...prev, mode, selected: activeSectionID ? new Set([activeSectionID]) : new Set() }));
    }
  };

  useEffect(() => {
    if (query.trim()) doSearch(query);
  }, [scope.mode, caseSensitive, wholeWord, regex, sections]);

  const scopeCount = scope.mode === "all" ? sections.length : scope.selected.size;
  const totalCount = results?.length ?? 0;

  const grouped = groupResults(results ?? []);

  return (
    <div class="search-panel">
      <div class="search-panel-header">
        <span class="search-panel-title">
          <Search size={13} /> Search
        </span>
        <button class="icon-button" title="Close (Esc)" onClick={onClose}>
          <X size={14} />
        </button>
      </div>

      <div class="search-input-row">
        <input
          ref={inputRef}
          class="search-input"
          type="text"
          placeholder="Search…"
          value={query}
          onInput={(e) => onInput((e.target as HTMLInputElement).value)}
          onKeyDown={onKey}
        />
        {query && (
          <button class="icon-button search-clear" title="Clear" onClick={() => { setQuery(""); setResults(null); onQueryChange?.(""); inputRef.current?.focus(); }}>
            <X size={12} />
          </button>
        )}
      </div>

      <div class="search-options">
        <button
          class={`icon-button search-toggle ${caseSensitive ? "active" : ""}`}
          title="Match Case"
          onClick={() => setCaseSensitive((v) => !v)}
        >
          <CaseSensitive size={14} />
        </button>
        <button
          class={`icon-button search-toggle ${wholeWord ? "active" : ""}`}
          title="Match Whole Word"
          onClick={() => setWholeWord((v) => !v)}
        >
          <WholeWord size={14} />
        </button>
        <button
          class={`icon-button search-toggle ${regex ? "active" : ""}`}
          title="Use Regular Expression"
          onClick={() => setRegex((v) => !v)}
        >
          <Regex size={14} />
        </button>
        <div class="search-options-spacer" />
        <button
          class={`icon-button search-toggle ${showScope ? "active" : ""}`}
          title="Search scope"
          onClick={() => setShowScope((v) => !v)}
        >
          <FileText size={13} /> {scopeCount === sections.length ? "All" : `${scopeCount} of ${sections.length}`}
        </button>
      </div>

      {showScope && (
        <div class="search-scope">
          <div class="search-scope-mode">
            <label class="search-scope-radio">
              <input type="radio" name="scope" checked={scope.mode === "all"} onChange={() => setScopeMode("all")} />
              <span>All sections</span>
            </label>
            <label class="search-scope-radio">
              <input type="radio" name="scope" checked={scope.mode === "custom"} onChange={() => setScopeMode("custom")} />
              <span>Selected sections</span>
            </label>
          </div>
          {scope.mode === "custom" && (
            <div class="search-scope-tree">
              {buildScopeTree(sections).map((node) => (
                <ScopeNode
                  key={node.section.ID}
                  node={node}
                  scope={scope}
                  onToggle={toggleSection}
                  onToggleChapter={toggleChapter}
                  onToggleExpand={toggleExpand}
                />
              ))}
            </div>
          )}
        </div>
      )}

      <div class="search-results-area">
        {searching && <div class="search-status">Searching…</div>}
        {error && <div class="search-status search-error">{error}</div>}
        {!searching && !error && results !== null && (
          <div class="search-status">
            {totalCount} {totalCount === 1 ? "result" : "results"} in {grouped.length} {grouped.length === 1 ? "file" : "files"}
          </div>
        )}
        {grouped.map((g) => (
          <div key={g.sectionId} class="search-result-group">
            <div
              class="search-result-file"
              onClick={() => {
                if (scope.mode === "custom" && !scope.selected.has(g.sectionId)) {
                  toggleSection(g.sectionId);
                }
              }}
            >
              <ChevronDown size={12} />
              <span class="search-result-file-title">{g.isPrompt ? "📝 " : ""}{g.sectionNumber} {g.sectionTitle}</span>
              <span class="search-result-file-count">{g.matches.length}</span>
            </div>
            {g.matches.map((m, i) => (
              <div
                key={`${g.sectionId}-${m.line}-${i}`}
                class="search-result-item"
                onClick={() => {
                  if (m.isPrompt && onNavigatePrompt) {
                    onNavigatePrompt(m.sectionId);
                  } else {
                    onNavigate(m.sectionId);
                  }
                }}
              >
                <span class="search-result-line">:{m.line}</span>
                <div class="search-result-context">
                  {m.before && <div class="search-context-line search-context-faded">{m.before}</div>}
                  <div class="search-context-line search-context-match">{m.match}</div>
                  {m.after && <div class="search-context-line search-context-faded">{m.after}</div>}
                </div>
              </div>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}

interface ScopeNodeData {
  section: Section;
  children: ScopeNodeData[];
}

function buildScopeTree(sections: Section[]): ScopeNodeData[] {
  const roots: ScopeNodeData[] = [];
  const stack: ScopeNodeData[] = [];
  for (const s of sections) {
    const node: ScopeNodeData = { section: s, children: [] };
    while (stack.length > 0 && stack[stack.length - 1].section.Level >= s.Level) {
      stack.pop();
    }
    if (stack.length > 0) stack[stack.length - 1].children.push(node);
    else roots.push(node);
    stack.push(node);
  }
  return roots;
}

function ScopeNode({ node, scope, onToggle, onToggleChapter, onToggleExpand }: {
  node: ScopeNodeData;
  scope: ScopeState;
  onToggle: (id: string) => void;
  onToggleChapter: (chapterID: string, childIDs: string[]) => void;
  onToggleExpand: (id: string) => void;
}): VNode {
  const s = node.section;
  const hasChildren = node.children.length > 0;
  const isExpanded = scope.expanded.has(s.ID);

  if (hasChildren) {
    const childIDs = collectChildIDs(node);
    const allSelected = childIDs.every((id) => scope.selected.has(id));
    const someSelected = childIDs.some((id) => scope.selected.has(id));
    return (
      <div class="scope-node">
        <div class="scope-node-row">
          <button
            class="icon-button scope-expand"
            onClick={() => onToggleExpand(s.ID)}
          >
            {isExpanded ? <ChevronDown size={10} /> : <ChevronRight size={10} />}
          </button>
          <label class="scope-checkbox">
            <input
              type="checkbox"
              checked={allSelected}
              ref={(el) => { if (el) el.indeterminate = someSelected && !allSelected; }}
              onChange={() => onToggleChapter(s.ID, childIDs)}
            />
            <span class="scope-label-title">{s.Title}</span>
          </label>
        </div>
        {isExpanded && node.children.map((c) => (
          <ScopeNode key={c.section.ID} node={c} scope={scope} onToggle={onToggle} onToggleChapter={onToggleChapter} onToggleExpand={onToggleExpand} />
        ))}
      </div>
    );
  }

  return (
    <div class="scope-node">
      <div class="scope-node-row scope-leaf">
        <label class="scope-checkbox">
          <input
            type="checkbox"
            checked={scope.selected.has(s.ID)}
            onChange={() => onToggle(s.ID)}
          />
          <span class="scope-label-title">{s.Title}</span>
        </label>
      </div>
    </div>
  );
}

function collectChildIDs(node: ScopeNodeData): string[] {
  const ids: string[] = [];
  const walk = (n: ScopeNodeData) => {
    if (n.children.length === 0) {
      ids.push(n.section.ID);
    } else {
      for (const c of n.children) walk(c);
    }
  };
  walk(node);
  return ids;
}

interface GroupedResult {
  sectionId: string;
  sectionTitle: string;
  sectionNumber: string;
  isPrompt: boolean;
  matches: SearchMatch[];
}

function groupResults(matches: SearchMatch[]): GroupedResult[] {
  const map = new Map<string, GroupedResult>();
  for (const m of matches) {
    let g = map.get(m.sectionId);
    if (!g) {
      g = { sectionId: m.sectionId, sectionTitle: m.sectionTitle, sectionNumber: m.sectionNumber, isPrompt: !!m.isPrompt, matches: [] };
      map.set(m.sectionId, g);
    }
    g.matches.push(m);
  }
  return [...map.values()];
}
