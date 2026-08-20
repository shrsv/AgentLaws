import { useEffect, useMemo, useState, useRef } from "preact/hooks";
import type { VNode } from "preact";
import { History, Plus, X, Clock, User, ArrowLeft, FileText, Download, FlaskConical, CircleAlert, TriangleAlert, List, MousePointer, ChevronLeft, ChevronRight, Copy, Code, Check, HelpCircle, Search, ChevronDown, Terminal, BookOpen, Layers } from "lucide-preact";
import { api, type Section, type Diagnostic, type RenderedSection, type PromptTemplate, type RenderedPrompt } from "../api";
import type { Route } from "../router";
import { HistorySidebar } from "../components/HistorySidebar";
import { HotkeyHelp } from "../components/HotkeyHelp";
import { SearchPanel } from "../components/SearchPanel";
import { useShortcuts, useEscape, getRegisteredShortcuts } from "../hooks/useKeyboardShortcuts";

interface Props {
  path: string;
  section: string | null;
  law?: string | null;
  navigate: (r: Route) => void;
  onSectionsChange?: (sections: Section[]) => void;
  onPromptsChange?: (prompts: PromptTemplate[]) => void;
  onOpenCommandPalette?: () => void;
  promptMode?: boolean;
  promptID?: string | null;
}

// TreeNode is a Section plus its child sections, rebuilt from the compiled
// IR's flat, depth-first list (Level/ParentID) into a nested tree so the
// sidebar can show the book's chapter/section/subsection hierarchy.
interface TreeNode {
  section: Section;
  children: TreeNode[];
}

function buildTree(sections: Section[]): TreeNode[] {
  const roots: TreeNode[] = [];
  const stack: TreeNode[] = [];
  for (const s of sections) {
    const node: TreeNode = { section: s, children: [] };
    while (stack.length > 0 && stack[stack.length - 1].section.Level >= s.Level) {
      stack.pop();
    }
    if (stack.length > 0) stack[stack.length - 1].children.push(node);
    else roots.push(node);
    stack.push(node);
  }
  return roots;
}

export function BookDetail({ path, section, law, navigate, onSectionsChange, onPromptsChange, onOpenCommandPalette, promptMode, promptID }: Props) {
  const [title, setTitle] = useState("");
  const [sections, setSections] = useState<Section[]>([]);
  const [rendered, setRendered] = useState<Record<string, RenderedSection>>({});
  const [renderedPrompts, setRenderedPrompts] = useState<Record<string, RenderedPrompt>>({});
  const [prompts, setPrompts] = useState<PromptTemplate[]>([]);
  const [promptBacklinks, setPromptBacklinks] = useState<Record<string, string[]>>({});
  const [diagnostics, setDiagnostics] = useState<Diagnostic[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [selectedID, setSelectedID] = useState<string | null>(null);
  const [selectedPromptID, setSelectedPromptID] = useState<string | null>(null);
  const [isPromptMode, setIsPromptMode] = useState(promptMode ?? false);
  const [promptDisplayExpanded, setPromptDisplayExpanded] = useState(true);
  const [promptVars, setPromptVars] = useState<Record<string, string>>({});
  const [promptPreview, setPromptPreview] = useState<string>("");
  const [copiedGoUsage, setCopiedGoUsage] = useState(false);
  const [dragID, setDragID] = useState<string | null>(null);
  const [newLawText, setNewLawText] = useState("");
  const [newChapter, setNewChapter] = useState(false);
  const [newSectionUnder, setNewSectionUnder] = useState<string | null>(null);
  const [showHistory, setShowHistory] = useState(false);
  const [historyFilter, setHistoryFilter] = useState<{ type: "section" | "law"; id: string; sectionId?: string } | null>(null);
  const [lawAuthors, setLawAuthors] = useState<Record<string, string>>({});
  const [sourceContent, setSourceContent] = useState<{ path: string; lineStart: number; lineEnd: number; content: string } | null>(null);
  const [copiedPath, setCopiedPath] = useState(false);
  const [copiedLawLink, setCopiedLawLink] = useState<string | null>(null);
  const [showHelp, setShowHelp] = useState(false);
  const [showSearch, setShowSearch] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [highlightQuery, setHighlightQuery] = useState<string | null>(null);
  const [showExportMenu, setShowExportMenu] = useState(false);
  const fetchingAuthors = useRef<Set<string>>(new Set());
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    const saved = localStorage.getItem("sidebarWidth");
    return saved ? parseInt(saved, 10) : 280;
  });
  const [sidebarVisible, setSidebarVisible] = useState(() => localStorage.getItem("sidebarVisible") !== "false");
  const sidebarDragRef = useRef<{ startX: number; startWidth: number } | null>(null);

  const reload = () => {
    setError(null);
    api
      .compile(path)
      .then((r) => {
        setTitle(r.lawbook.Metadata.Title);
        setSections(r.lawbook.Sections);
        onSectionsChange?.(r.lawbook.Sections);
        setRendered(r.rendered ?? {});
        setRenderedPrompts(r.renderedPrompts ?? {});
        setPrompts(r.lawbook.Prompts ?? []);
        onPromptsChange?.(r.lawbook.Prompts ?? []);
        setPromptBacklinks(r.lawbook.PromptBacklinks ?? {});
        setDiagnostics(r.diagnostics ?? []);
        // Functional update: read current selection (the URL's section may
        // have just been restored by the section-sync effect), not the
        // stale closure value from this render.
        setSelectedID((prev) => prev ?? r.lawbook.Sections[0]?.ID ?? null);
      })
      .catch((e) => setError(String(e)));
  };

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(reload, [path]);

  // Live-reload: subscribe to watch events so the UI updates when any
  // source file changes (sections, prompt templates, alaws.toml) without
  // requiring a manual refresh. The SSE payload now includes rendered
  // HTML fragments, so we update state directly instead of calling reload().
  useEffect(() => {
    const stop = api.watch(path, (ev) => {
      if (!ev.ok) return;
      setTitle(ev.lawbook.Metadata.Title);
      setSections(ev.lawbook.Sections);
      onSectionsChange?.(ev.lawbook.Sections);
      setRendered(ev.rendered ?? {});
      setRenderedPrompts(ev.renderedPrompts ?? {});
      setPrompts(ev.lawbook.Prompts ?? []);
      onPromptsChange?.(ev.lawbook.Prompts ?? []);
      setPromptBacklinks(ev.lawbook.PromptBacklinks ?? {});
      setDiagnostics(ev.diagnostics ?? []);
    });
    return stop;
  }, [path]);

  // Clear parent sections and prompts on unmount
  useEffect(() => {
    return () => {
      onSectionsChange?.([]);
      onPromptsChange?.([]);
    };
  }, []);

  // The URL carries the selected section (e.g. #/books/<book>/<section>), so
  // a refresh or back/forward restores the same page. Clicking a tree node
  // pushes the id into the hash via select(); this effect picks it back up
  // when the hash changes (or on first load after a refresh).
  useEffect(() => {
    if (section) setSelectedID(section);
  }, [section]);

  // Sync prompt mode from route
  useEffect(() => {
    if (promptMode) setIsPromptMode(true);
  }, [promptMode]);

  // Sync prompt ID from route
  useEffect(() => {
    if (promptID) setSelectedPromptID(promptID);
  }, [promptID]);

  // Auto-select first prompt when in prompt mode but none selected
  useEffect(() => {
    if (isPromptMode && !selectedPromptID && prompts.length > 0) {
      setSelectedPromptID(prompts[0].ID);
    }
  }, [isPromptMode, selectedPromptID, prompts]);

  // Prompt preview: debounce variable changes and call API
  useEffect(() => {
    if (!isPromptMode || !selectedPromptID) return;
    const timer = setTimeout(() => {
      api.promptRender(path, selectedPromptID, { vars: promptVars, onMissing: "keep" }).then((r) => {
        setPromptPreview(r.text);
      }).catch(() => setPromptPreview("(render error)"));
    }, 300);
    return () => clearTimeout(timer);
  }, [isPromptMode, selectedPromptID, promptVars, path]);

  // Close export menu when clicking outside
  useEffect(() => {
    if (!showExportMenu) return;
    const close = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      if (!target.closest(".export-menu-container")) setShowExportMenu(false);
    };
    document.addEventListener("click", close);
    return () => document.removeEventListener("click", close);
  }, [showExportMenu]);

  // Auto-scroll to first search highlight when navigating from search
  useEffect(() => {
    if (highlightQuery && selectedID) {
      requestAnimationFrame(() => {
        const firstMark = document.querySelector(".search-highlight");
        if (firstMark) firstMark.scrollIntoView({ behavior: "smooth", block: "center" });
      });
    }
  }, [highlightQuery, selectedID]);

  // Auto-scroll to a specific law when navigating via alaws: link.
  // The law prop is just the slug (e.g. "every-change-reviewed-least-one"),
  // but the <li> id is "sectionid.slug". Reconstruct the full ID using
  // selectedID (the section) and flash-highlight the target.
  useEffect(() => {
    if (law && selectedID) {
      requestAnimationFrame(() => {
        const el = document.getElementById(selectedID + "." + law) || document.getElementById("law-" + law);
        if (el) {
          el.scrollIntoView({ behavior: "smooth", block: "center" });
          el.classList.remove("law-flash");
          // Force reflow to restart animation
          void (el as HTMLElement).offsetWidth;
          el.classList.add("law-flash");
          el.addEventListener("animationend", () => el.classList.remove("law-flash"), { once: true });
        }
      });
    }
  }, [law, selectedID]);

  // Persist sidebar width and visibility
  useEffect(() => {
    localStorage.setItem("sidebarWidth", String(sidebarWidth));
  }, [sidebarWidth]);

  useEffect(() => {
    localStorage.setItem("sidebarVisible", String(sidebarVisible));
  }, [sidebarVisible]);

  // Sidebar drag handlers
  useEffect(() => {
    const onMouseMove = (e: MouseEvent) => {
      if (!sidebarDragRef.current) return;
      const delta = e.clientX - sidebarDragRef.current.startX;
      const newWidth = Math.max(150, Math.min(600, sidebarDragRef.current.startWidth + delta));
      setSidebarWidth(newWidth);
    };
    const onMouseUp = () => {
      sidebarDragRef.current = null;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
    window.addEventListener("mousemove", onMouseMove);
    window.addEventListener("mouseup", onMouseUp);
    return () => {
      window.removeEventListener("mousemove", onMouseMove);
      window.removeEventListener("mouseup", onMouseUp);
    };
  }, []);

  const select = (id: string) => {
    setSelectedID(id);
    navigate({ name: "book", path, section: id });
  };

  const selectPrompt = (id: string) => {
    setSelectedPromptID(id);
    setPromptVars({});
    navigate({ name: "prompts", path, id });
  };

  const currentPromptIndex = selectedPromptID ? prompts.findIndex((p) => p.ID === selectedPromptID) : -1;
  const hasPrevPrompt = currentPromptIndex > 0;
  const hasNextPrompt = currentPromptIndex >= 0 && currentPromptIndex < prompts.length - 1;
  const goPrevPrompt = () => { if (hasPrevPrompt) selectPrompt(prompts[currentPromptIndex - 1].ID); };
  const goNextPrompt = () => { if (hasNextPrompt) selectPrompt(prompts[currentPromptIndex + 1].ID); };

  const currentIndex = selectedID ? sections.findIndex((s) => s.ID === selectedID) : -1;
  const hasPrev = currentIndex > 0;
  const hasNext = currentIndex >= 0 && currentIndex < sections.length - 1;

  const goPrev = () => { if (hasPrev) select(sections[currentIndex - 1].ID); };
  const goNext = () => { if (hasNext) select(sections[currentIndex + 1].ID); };

  const goPrevAny = () => { if (isPromptMode) goPrevPrompt(); else goPrev(); };
  const goNextAny = () => { if (isPromptMode) goNextPrompt(); else goNext(); };

  const toggleHistory = () => { setShowHistory((v) => !v); if (showHistory) setHistoryFilter(null); };
  const toggleHelp = () => { setShowHelp((v) => !v); };
  const toggleSearch = () => { setShowSearch((v) => { if (v) setHighlightQuery(null); return !v; }); };
  const toggleSidebar = () => { setSidebarVisible((v) => !v); };

  useEscape(() => {
    if (showHelp) { setShowHelp(false); return true; }
    if (sourceContent) { setSourceContent(null); return true; }
    if (showSearch) { setShowSearch(false); setHighlightQuery(null); return true; }
    if (showHistory) { setShowHistory(false); setHistoryFilter(null); return true; }
    return false;
  });

  useShortcuts([
    { key: "j", description: "Next section/prompt", action: goNextAny },
    { key: "k", description: "Previous section/prompt", action: goPrevAny },
    { key: "ArrowRight", description: "Next section/prompt", action: goNextAny },
    { key: "ArrowLeft", description: "Previous section/prompt", action: goPrevAny },
    { key: "h", description: "Toggle history panel", action: toggleHistory },
    { key: "b", description: "Toggle sidebar", action: toggleSidebar },
    { key: "s", description: "View source", action: () => {
      if (isPromptMode && selectedPromptID) {
        const p = prompts.find((pr) => pr.ID === selectedPromptID);
        if (p) loadSource(p.Source.Path, p.Source.LineStart, p.Source.LineEnd);
      } else if (selected) {
        loadSource(selected.Source.Path, selected.Source.LineStart, selected.Source.LineEnd);
      }
    }},
    { key: "c", description: "Copy file path", action: () => {
      if (isPromptMode && selectedPromptID) {
        const p = prompts.find((pr) => pr.ID === selectedPromptID);
        if (p) copyPath(p.Source.Path);
      } else if (selected) {
        copyPath(selected.Source.Path);
      }
    }},
    { key: "?", description: "Show keyboard shortcuts", action: toggleHelp },
    { key: "/", description: "Search in book", action: toggleSearch },
    { key: "p", description: "Quick jump (Ctrl+P also works)", action: () => onOpenCommandPalette?.() },
  ]);

  const relativePath = (srcPath: string) => {
    const bookDir = path.includes(".") ? path.replace(/[/\\][^/\\]+$/, "") : path;
    if (srcPath.startsWith(bookDir)) return srcPath.slice(bookDir.length + 1);
    return srcPath;
  };

  const copyPath = (srcPath: string) => {
    navigator.clipboard.writeText(srcPath).then(() => {
      setCopiedPath(true);
      setTimeout(() => setCopiedPath(false), 1500);
    });
  };

  const loadSource = (srcPath: string, lineStart: number, lineEnd: number) => {
    api.source(srcPath, lineStart, lineEnd).then(setSourceContent).catch(() => setSourceContent(null));
  };

  // Recursive sidebar renderer: each section is a row (with the same
  // select/drag/history/remove handlers as before), and any children are
  // rendered as a nested <ul> indented under it.
  const renderTree = (nodes: TreeNode[]): VNode[] =>
    nodes.map((n) => {
      const s = n.section;
      return (
        <li key={s.ID} class="tree-branch">
          <div
            class="tree-node"
            data-level={s.Level}
            aria-selected={s.ID === selectedID}
            draggable
            onDragStart={() => setDragID(s.ID)}
            onDragOver={(e) => e.preventDefault()}
            onDrop={(e) => {
              e.preventDefault();
              const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
              const after = e.clientY - rect.top > rect.height / 2;
              drop(s.ID, after);
            }}
            onClick={() => select(s.ID)}
          >
            <span class="number">{s.Number}</span>
            <span class="node-title">{s.Title}</span>
            <span class="node-actions">
              <button
                class="icon-button"
                title="History"
                onClick={(e) => {
                  e.stopPropagation();
                  openSectionHistory(s.ID);
                }}
              >
                <Clock size={11} />
              </button>
              {s.Level > 0 && (
                <button
                  class="icon-button"
                  title="New section here"
                  onClick={(e) => {
                    e.stopPropagation();
                    setNewSectionUnder(newSectionUnder === s.ID ? null : s.ID);
                  }}
                >
                  <Plus size={11} />
                </button>
              )}
              <button
                class="icon-button"
                title="Remove"
                onClick={(e) => {
                  e.stopPropagation();
                  removeNode(s);
                }}
              >
                <X size={11} />
              </button>
            </span>
          </div>
          {n.children.length > 0 && (
            <ul class="tree-children">{renderTree(n.children)}</ul>
          )}
        </li>
      );
    });

  const tree = useMemo(() => buildTree(sections), [sections]);

  const selected = sections.find((s) => s.ID === selectedID) ?? null;
  const errorCount = diagnostics.filter((d) => d.Severity === "error").length;
  const warningCount = diagnostics.filter((d) => d.Severity === "warning").length;

  function openSectionHistory(sectionId: string) {
    setHistoryFilter({ type: "section", id: sectionId, sectionId });
    setShowHistory(true);
  }

  function openLawHistory(lawNumber: string, sectionId: string) {
    setHistoryFilter({ type: "law", id: lawNumber, sectionId });
    setShowHistory(true);
  }

  function fetchLawAuthor(sectionId: string, lawNumber: string) {
    const key = `${sectionId}:${lawNumber}`;
    if (lawAuthors[key] !== undefined || fetchingAuthors.current.has(key)) return;
    fetchingAuthors.current.add(key);
    api
      .history(path, lawNumber)
      .then((h) => {
        if (h.Modifications && h.Modifications.length > 0) {
          const last = h.Modifications[0];
          const name = last.Author.replace(/ <.*>$/, "");
          setLawAuthors((prev) => ({ ...prev, [key]: name }));
        } else {
          setLawAuthors((prev) => ({ ...prev, [key]: "" }));
        }
      })
      .catch(() => {
        setLawAuthors((prev) => ({ ...prev, [key]: "" }));
      });
  }

  async function drop(targetID: string, after: boolean) {
    if (!dragID || dragID === targetID) return;
    const dragged = sections.find((s) => s.ID === dragID);
    const target = sections.find((s) => s.ID === targetID);
    if (!dragged || !target) return;

    const kind = dragged.Level === 1 ? "chapter" : "section";
    const opts = after ? { after: targetID } : { before: targetID };
    if (kind === "section") {
      const newParentId = target.Level < dragged.Level ? target.ID : target.ParentID;
      await api.move(path, "section", dragID, { ...opts, newParentId });
    } else {
      await api.move(path, "chapter", dragID, opts);
    }
    setDragID(null);
    reload();
  }

  async function removeNode(s: Section) {
    const kind = s.Level === 1 ? "chapter" : "section";
    if (!confirm(`Remove ${kind} "${s.Title}" (${s.ID})?`)) return;
    await api.remove(path, kind, s.ID, true);
    if (selectedID === s.ID) {
      setSelectedID(null);
      navigate({ name: "book", path });
    }
    reload();
  }

  async function addLaw() {
    if (!selected || !newLawText.trim()) return;
    await api.addLaw(path, selected.ID, newLawText.trim());
    setNewLawText("");
    reload();
  }

  async function removeLaw(number: number) {
    if (!selected) return;
    await api.removeLaw(path, selected.ID, number, true);
    reload();
  }

  return (
    <div class="shell">
      <div class="titlebar">
        <button class="link-button" onClick={() => navigate({ name: "books" })}>
          <ArrowLeft size={12} /> Books
        </button>
        <span class="book-title">{title || path}</span>
        <span class="path">{path}</span>
        <button class="link-button" onClick={toggleSidebar} title="Toggle sidebar (b)">
          <List size={12} />
        </button>
        <button class="link-button titlebar-search" onClick={toggleSearch} title="Search (/)">
          <Search size={12} /> Search
        </button>
        {onOpenCommandPalette && (
          <button class="link-button titlebar-cmd" onClick={onOpenCommandPalette} title="Quick jump (Ctrl+P)">
            <Terminal size={12} /> Jump
          </button>
        )}
        <div class="spacer" />
        <div class="export-menu-container">
          <button class="link-button" onClick={() => setShowExportMenu((v) => !v)} title="Export">
            <Download size={12} /> Export <ChevronDown size={10} />
          </button>
          {showExportMenu && (
            <div class="export-dropdown">
              <div class="export-dropdown-section">
                <span class="export-dropdown-label">This book</span>
                <a class="export-dropdown-item" href={api.exportURL(path, "html")} target="_blank" rel="noreferrer" onClick={() => setShowExportMenu(false)}>
                  <FileText size={12} /> HTML
                </a>
                <a class="export-dropdown-item" href={api.exportURL(path, "pdf")} target="_blank" rel="noreferrer" onClick={() => setShowExportMenu(false)}>
                  <FileText size={12} /> PDF
                </a>
                <a class="export-dropdown-item" href={api.exportURL(path, "md")} target="_blank" rel="noreferrer" onClick={() => setShowExportMenu(false)}>
                  <FileText size={12} /> Markdown
                </a>
              </div>
              <div class="export-dropdown-divider" />
              <button class="export-dropdown-item" onClick={() => { setShowExportMenu(false); navigate({ name: "books" }); }}>
                <Download size={12} /> Export all books…
              </button>
            </div>
          )}
        </div>
        <button class="link-button" onClick={() => navigate({ name: "playground", path })}>
          <FlaskConical size={12} /> Playground
        </button>
        <button
          class={`link-button ${showHistory ? "active" : ""}`}
          onClick={toggleHistory}
        >
          <History size={12} style={{ verticalAlign: "middle", marginRight: 4 }} />
          History
        </button>
        <button class="link-button" onClick={toggleHelp} title="Keyboard shortcuts (?)">
          <HelpCircle size={12} />
        </button>
      </div>

      {error && <p class="error-text"><CircleAlert size={12} /> {error}</p>}

      <div class={`workbench ${sidebarVisible ? "" : "sidebar-hidden"}`} style={sidebarVisible ? { gridTemplateColumns: `${sidebarWidth}px 1px 1fr` } : undefined}>
        <nav class="sidebar" aria-label="Lawbook sections">
          {showSearch ? (
            <SearchPanel
              bookPath={path}
              sections={sections}
              activeSectionID={selectedID}
              initialQuery={searchQuery}
              onQueryChange={setSearchQuery}
              onClose={() => setShowSearch(false)}
              onNavigate={(id) => { setHighlightQuery(searchQuery); select(id); }}
              onNavigatePrompt={(id) => { setHighlightQuery(searchQuery); selectPrompt(id); setShowSearch(false); }}
            />
          ) : (
            <>
              <div class="sidebar-title">
                <div class="sidebar-mode-toggle">
                  <button class={`mode-toggle-btn ${!isPromptMode ? "active" : ""}`} onClick={() => { setIsPromptMode(false); navigate({ name: "book", path, section: selectedID ?? undefined }); }} title="Laws">
                    <BookOpen size={12} /> Laws
                  </button>
                  <button class={`mode-toggle-btn ${isPromptMode ? "active" : ""}`} onClick={() => {
                    setIsPromptMode(true);
                    const targetID = selectedPromptID ?? prompts[0]?.ID;
                    if (targetID && !selectedPromptID) setSelectedPromptID(targetID);
                    navigate({ name: "prompts", path, id: targetID ?? undefined });
                  }} title="Prompts">
                    <Layers size={12} /> Prompts
                  </button>
                </div>
                {!isPromptMode && (
                  <button class="icon-button" title="New chapter" onClick={() => setNewChapter((v) => !v)}>
                    <Plus size={12} />
                  </button>
                )}
              </div>

              {isPromptMode ? (
                <ul class="tree prompt-list">
                  {prompts.map((p, i) => (
                    <li key={p.ID} class="tree-branch">
                      <div
                        class="tree-node"
                        aria-selected={p.ID === selectedPromptID}
                        onClick={() => selectPrompt(p.ID)}
                      >
                        <Layers size={12} style={{ flexShrink: 0, opacity: 0.6 }} />
                        <span class="number">P{i + 1}</span>
                        <span class="node-title">{p.Title}</span>
                      </div>
                    </li>
                  ))}
                  {prompts.length === 0 && (
                    <li class="empty-state-small">No prompt templates.</li>
                  )}
                </ul>
              ) : (
                <>
                  {newChapter && <NewNodeForm onSubmit={async (file, t, id) => { await api.createChapter(path, file, t, id); setNewChapter(false); reload(); }} onCancel={() => setNewChapter(false)} />}

                  <ul class="tree">{renderTree(tree)}</ul>
                  {newSectionUnder && (
                    <NewNodeForm
                      parent={newSectionUnder}
                      onSubmit={async (file, t, id) => {
                        await api.createSection(path, file, t, id, newSectionUnder);
                        setNewSectionUnder(null);
                        reload();
                      }}
                      onCancel={() => setNewSectionUnder(null)}
                    />
                  )}
                </>
              )}
            </>
          )}
        </nav>

        <div
          class="divider"
          onMouseDown={(e) => {
            e.preventDefault();
            sidebarDragRef.current = { startX: e.clientX, startWidth: sidebarWidth };
            document.body.style.cursor = "col-resize";
            document.body.style.userSelect = "none";
          }}
        />

        <main class="detail">
          {isPromptMode ? (
            selectedPromptID ? (() => {
              const p = prompts.find((pr) => pr.ID === selectedPromptID);
              if (!p) return <p class="empty-state"><MousePointer size={14} /> Prompt not found.</p>;
              const rp = renderedPrompts[p.ID];
              const goUsage = `book, _ := alaws.Load("${path}")
prompt, _ := book.Prompt("${p.ID}")
fmt.Println(prompt.Vars) // [${(p.Vars ?? []).map((v) => `"${v}"`).join(", ")}]
text, _ := prompt.Render(alaws.PromptRenderOptions{
    Vars: map[string]string{${(p.Vars ?? []).map((v) => `${v}: "..."`).join(", ")}},
})`;
              return (
                <>
                  <div class="detail-header">
                    <div class="detail-nav">
                      <button class="icon-button nav-arrow" title="Previous prompt" disabled={!hasPrevPrompt} onClick={goPrevPrompt}>
                        <ChevronLeft size={16} />
                      </button>
                      <button class="icon-button nav-arrow" title="Next prompt" disabled={!hasNextPrompt} onClick={goNextPrompt}>
                        <ChevronRight size={16} />
                      </button>
                    </div>
                    <h1>{p.Title}</h1>
                  </div>
                  <div class="section-id">{p.ID}</div>
                  <div class="section-file">
                    <Code size={11} />
                    <span class="section-file-path" title={p.Source.Path}>{relativePath(p.Source.Path)}</span>
                    <button class="icon-button" title="Copy path" onClick={() => copyPath(p.Source.Path)}>
                      {copiedPath ? <Check size={11} /> : <Copy size={11} />}
                    </button>
                    <button class="icon-button" title="View source" onClick={() => loadSource(p.Source.Path, p.Source.LineStart, p.Source.LineEnd)}>
                      <Code size={11} /> Source
                    </button>
                  </div>

                  <div class="commentary" dangerouslySetInnerHTML={{ __html: highlightMatches(rp?.CommentaryHTML ?? escapeHTML(p.Commentary), highlightQuery) }} />

                  <div class="prompt-template-section">
                    <div class="prompt-template-header">
                      <h3>Template</h3>
                      <div class="prompt-display-toggle">
                        <button class={`mode-toggle-btn ${promptDisplayExpanded ? "active" : ""}`} onClick={() => setPromptDisplayExpanded(true)}>Expanded</button>
                        <button class={`mode-toggle-btn ${!promptDisplayExpanded ? "active" : ""}`} onClick={() => setPromptDisplayExpanded(false)}>Compact</button>
                      </div>
                    </div>
                    <div class="prompt-template-content" dangerouslySetInnerHTML={{ __html: highlightMatches(promptDisplayExpanded ? (rp?.TemplateHTML ?? escapeHTML(p.Template)) : (rp?.CompactHTML ?? ""), highlightQuery) }} />
                  </div>

                  {p.Vars && p.Vars.length > 0 && (
                    <div class="prompt-vars-section">
                      <h3>Variables</h3>
                      <div class="prompt-vars-grid">
                        {p.Vars.map((v) => (
                          <div key={v} class="prompt-var-row">
                            <label class="prompt-var-label">{v}</label>
                            <input
                              class="prompt-var-input"
                              placeholder={v}
                              value={promptVars[v] ?? ""}
                              onInput={(e) => setPromptVars((prev) => ({ ...prev, [v]: (e.target as HTMLInputElement).value }))}
                            />
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  <div class="prompt-preview-section">
                    <h3>Preview</h3>
                    <pre class="prompt-preview-content"><code>{promptPreview || "(fill in variables above)"}</code></pre>
                  </div>

                  <div class="prompt-go-usage">
                    <h3>Go Usage</h3>
                    <div class="source-modal-content" style="position:relative">
                      <button class="icon-button" style="position:absolute;top:4px;right:4px" title="Copy" onClick={() => { navigator.clipboard.writeText(goUsage); setCopiedGoUsage(true); setTimeout(() => setCopiedGoUsage(false), 1500); }}>
                        {copiedGoUsage ? <Check size={11} /> : <Copy size={11} />}
                      </button>
                      <pre><code>{goUsage}</code></pre>
                    </div>
                  </div>

                  {p.ReferencedAnchors && p.ReferencedAnchors.length > 0 && (
                    <div class="prompt-refs-section">
                      <h3>References</h3>
                      <ul class="prompt-refs-list">
                        {p.ReferencedAnchors.map((anchor) => {
                          const sec = sections.find((s) => s.ID === anchor || s.Laws?.some((l) => (l.SectionID + "." + l.Slug) === anchor));
                          const label = sec ? (sec.ID === anchor ? `${sec.Number} ${sec.Title}` : anchor) : anchor;
                          return (
                            <li key={anchor}>
                              <a class="alaws-link" onClick={() => {
                                if (sec && sec.ID === anchor) {
                                  setIsPromptMode(false);
                                  navigate({ name: "book", path, section: anchor });
                                } else if (sec) {
                                  setIsPromptMode(false);
                                  navigate({ name: "book", path, section: sec.ID, law: anchor.split(".").pop() });
                                }
                              }}>{label}</a>
                            </li>
                          );
                        })}
                      </ul>
                    </div>
                  )}
                </>
              );
            })() : (
              <p class="empty-state"><MousePointer size={14} /> Select a prompt template.</p>
            )
          ) : selected ? (
            <>
              <div class="detail-header">
                <div class="detail-nav">
                  <button class="icon-button nav-arrow" title="Previous section" disabled={!hasPrev} onClick={goPrev}>
                    <ChevronLeft size={16} />
                  </button>
                  <button class="icon-button nav-arrow" title="Next section" disabled={!hasNext} onClick={goNext}>
                    <ChevronRight size={16} />
                  </button>
                </div>
                <h1>
                  {selected.Number} {selected.Title}
                </h1>
              </div>
              <div class="section-id">
                {selected.ID}
              </div>
              <div class="section-file">
                <Code size={11} />
                <span class="section-file-path" title={selected.Source.Path}>{relativePath(selected.Source.Path)}</span>
                <button class="icon-button" title="Copy path" onClick={() => copyPath(selected.Source.Path)}>
                  {copiedPath ? <Check size={11} /> : <Copy size={11} />}
                </button>
                <button class="icon-button" title="View source" onClick={() => loadSource(selected.Source.Path, selected.Source.LineStart, selected.Source.LineEnd)}>
                  <Code size={11} /> Source
                </button>
              </div>
              <div class="commentary" dangerouslySetInnerHTML={{ __html: highlightMatches(rendered[selected.ID]?.CommentaryHTML ?? escapeHTML(selected.Commentary), highlightQuery) }} />

              {/* Backlinks: show which prompts reference this section */}
              {promptBacklinks[selected.ID] && promptBacklinks[selected.ID].length > 0 && (
                <div class="backlinks">
                  <span class="backlinks-label">Used in prompts:</span>
                  {promptBacklinks[selected.ID].map((pid) => {
                    const pt = prompts.find((p) => p.ID === pid);
                    return (
                      <a key={pid} class="alaws-link" onClick={() => { setIsPromptMode(true); navigate({ name: "prompts", path, id: pid }); }}>
                        {pt?.Title ?? pid}
                      </a>
                    );
                  })}
                </div>
              )}

              {selected.Laws && selected.Laws.length > 0 ? (
                <ol class="law-list">
                  {selected.Laws.map((law) => {
                    const authorKey = `${selected.ID}:${law.Number}`;
                    const author = lawAuthors[authorKey];
                    return (
                      <li key={law.Number} id={`law-${law.Slug || String(law.Index)}`} onMouseEnter={() => fetchLawAuthor(selected.ID, law.Number)}>
                        <span class="law-number">{law.Number}</span>
                        <span
                          class="law-text"
                          dangerouslySetInnerHTML={{ __html: highlightMatches(rendered[selected.ID]?.LawHTML?.[law.Number] ?? escapeHTML(law.Text), highlightQuery) }}
                        />
                        {/* Per-law backlinks */}
                        {law.Slug && (() => {
                          const lawAnchor = selected.ID + "." + law.Slug;
                          const bl = promptBacklinks[lawAnchor];
                          if (!bl || bl.length === 0) return null;
                          return (
                            <span class="law-backlinks">
                              Used in: {bl.map((pid) => {
                                const pt = prompts.find((p) => p.ID === pid);
                                return (
                                  <a key={pid} class="alaws-link" onClick={() => { setIsPromptMode(true); navigate({ name: "prompts", path, id: pid }); }}>
                                    {pt?.Title ?? pid}
                                  </a>
                                );
                              })}
                            </span>
                          );
                        })()}
                        {author !== undefined && author !== "" && (
                          <span class="law-author">
                            <User size={10} /> {author}
                          </span>
                        )}
                        <span class="law-actions">
                          <button
                            class="icon-button"
                            title="Copy link"
                            onClick={() => {
                              const lawId = law.Slug || String(law.Index);
                              const hash = `#/books/${encodeURIComponent(path)}/${encodeURIComponent(selected.ID)}~${encodeURIComponent(lawId)}`;
                              const url = window.location.origin + window.location.pathname + hash;
                              navigator.clipboard.writeText(url);
                              window.location.hash = hash;
                              setCopiedLawLink(law.Number);
                              setTimeout(() => setCopiedLawLink((v) => v === law.Number ? null : v), 1500);
                            }}
                          >
                            {copiedLawLink === law.Number
                              ? <Check size={11} />
                              : <span style="font-size:11px;font-weight:600">#</span>}
                          </button>
                          <button
                            class="icon-button"
                            title="History"
                            onClick={() => openLawHistory(law.Number, selected.ID)}
                          >
                            <Clock size={11} />
                          </button>
                          <button class="icon-button" title="Remove law" onClick={() => removeLaw(law.Index)}>
                            <X size={11} />
                          </button>
                        </span>
                      </li>
                    );
                  })}
                </ol>
              ) : (
                <p class="empty-state"><List size={14} /> This section has no laws of its own.</p>
              )}

              <div class="add-law-form">
                <input
                  placeholder="Add a law…"
                  value={newLawText}
                  onInput={(e) => setNewLawText((e.target as HTMLInputElement).value)}
                  onKeyDown={(e) => e.key === "Enter" && addLaw()}
                />
                <button class="btn" onClick={addLaw}>
                  <Plus size={12} /> Add
                </button>
              </div>
            </>
          ) : (
            <p class="empty-state"><MousePointer size={14} /> Select a section.</p>
          )}
        </main>
      </div>

      <HistorySidebar
        path={path}
        show={showHistory}
        onClose={() => { setShowHistory(false); setHistoryFilter(null); }}
        filterEntity={historyFilter}
        sections={sections}
      />

      {sourceContent && (
        <div class="source-overlay" onClick={() => setSourceContent(null)}>
          <div class="source-modal" onClick={(e) => e.stopPropagation()}>
            <div class="source-modal-header">
              <span class="source-modal-title">{relativePath(sourceContent.path)}:{sourceContent.lineStart}-{sourceContent.lineEnd}</span>
              <div class="source-modal-actions">
                <button class="icon-button" title="Copy source" onClick={() => { navigator.clipboard.writeText(sourceContent.content); }}>
                  <Copy size={11} />
                </button>
                <button class="icon-button" title="Close" onClick={() => setSourceContent(null)}>
                  <X size={11} />
                </button>
              </div>
            </div>
            <pre class="source-modal-content"><code>{sourceContent.content}</code></pre>
          </div>
        </div>
      )}

      <HotkeyHelp
        show={showHelp}
        onClose={() => setShowHelp(false)}
        shortcuts={getRegisteredShortcuts()}
      />

      <div class="statusbar">
        <span class={`diagnostic-count ${errorCount > 0 ? "error" : ""}`}><CircleAlert size={11} /> {errorCount} errors</span>
        <span class={`diagnostic-count ${warningCount > 0 ? "warning" : ""}`}><TriangleAlert size={11} /> {warningCount} warnings</span>
      </div>
    </div>
  );
}

function NewNodeForm(props: {
  parent?: string;
  onSubmit: (file: string, title: string, id: string) => void;
  onCancel: () => void;
}) {
  const [file, setFile] = useState("");
  const [t, setT] = useState("");
  const [id, setId] = useState("");
  return (
    <form
      class="new-node-form"
      onSubmit={(e) => {
        e.preventDefault();
        props.onSubmit(file, t, id);
      }}
    >
      <input placeholder="file.md" value={file} onInput={(e) => setFile((e.target as HTMLInputElement).value)} required autoFocus />
      <input placeholder="title" value={t} onInput={(e) => setT((e.target as HTMLInputElement).value)} required />
      <input placeholder="id" value={id} onInput={(e) => setId((e.target as HTMLInputElement).value)} required />
      <div class="new-node-form-actions">
        <button class="btn btn-primary" type="submit">
          <Plus size={12} /> Create
        </button>
        <button class="btn" type="button" onClick={props.onCancel}>
          <X size={12} /> Cancel
        </button>
      </div>
    </form>
  );
}

function escapeHTML(s: string): string {
  const div = document.createElement("div");
  div.textContent = s;
  return div.innerHTML;
}

function highlightMatches(html: string, query: string | null): string {
  if (!query?.trim()) return html;
  const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const regex = new RegExp(`(${escaped})`, "gi");
  return html.replace(regex, '<mark class="search-highlight">$1</mark>');
}
