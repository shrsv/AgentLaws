import { useState } from 'preact/hooks'
import './app.css'

/**
 * Placeholder shell for the AgentLaws local UI (docs/PLAN1.md §28-§31).
 *
 * This is not wired to the Go server's Lawbook API yet - the compiler,
 * discovery, and server packages are still stubs (docs/PLAN1.md §64). The
 * static tree below mirrors the README's own worked example so the VS
 * Code-style navigation/detail layout can be reviewed before the API
 * exists.
 */

interface LawEntry {
  number: string
  text: string
}

interface SectionEntry {
  id: string
  number: string
  title: string
  level: 1 | 2
  commentary: string
  laws: LawEntry[]
}

const SECTIONS: SectionEntry[] = [
  {
    id: 'engineering.principles',
    number: '1',
    title: 'Principles',
    level: 1,
    commentary: 'General engineering principles agents should follow.',
    laws: [],
  },
  {
    id: 'engineering.security',
    number: '2',
    title: 'Security',
    level: 1,
    commentary:
      'This section defines the security requirements for agents working with the repository.',
    laws: [],
  },
  {
    id: 'engineering.security.secrets',
    number: '2.5',
    title: 'Credentials',
    level: 2,
    commentary:
      'Rules governing how agents handle credentials discovered in or introduced into the repository.',
    laws: [
      { number: '2.5.1', text: 'Credentials must never be committed to source control.' },
      { number: '2.5.2', text: 'Agents must not print credentials into logs.' },
      { number: '2.5.3', text: 'Credentials discovered in source must be treated as compromised.' },
    ],
  },
  {
    id: 'engineering.coding',
    number: '3',
    title: 'Coding',
    level: 1,
    commentary: 'Rules for making code changes.',
    laws: [],
  },
]

export function App() {
  const [selectedId, setSelectedId] = useState(SECTIONS[2].id)
  const selected = SECTIONS.find((s) => s.id === selectedId)

  return (
    <div class="shell">
      <div class="titlebar">
        <span class="book-title">Engineering Governance</span>
        <span class="path">./governance</span>
      </div>

      <div class="workbench">
        <nav class="sidebar" aria-label="Lawbook sections">
          <div class="sidebar-title">Lawbook</div>
          <ul class="tree">
            {SECTIONS.map((s) => (
              <li
                key={s.id}
                class="tree-node"
                data-level={s.level}
                aria-selected={s.id === selectedId}
                onClick={() => setSelectedId(s.id)}
              >
                <span class="number">{s.number}</span>
                <span>{s.title}</span>
              </li>
            ))}
          </ul>
        </nav>

        <div class="divider" />

        <main class="detail">
          {selected ? (
            <>
              <h1>
                {selected.number} {selected.title}
              </h1>
              <div class="section-id">{selected.id}</div>
              <p>{selected.commentary}</p>
              {selected.laws.length > 0 ? (
                <ul class="law-list">
                  {selected.laws.map((law) => (
                    <li key={law.number}>
                      <span class="law-number">{law.number}</span>
                      <span>{law.text}</span>
                    </li>
                  ))}
                </ul>
              ) : (
                <p class="empty-state">This section has no laws of its own.</p>
              )}
            </>
          ) : (
            <p class="empty-state">Select a section.</p>
          )}
        </main>
      </div>

      <div class="statusbar">
        <span class="diagnostic-count">0 errors</span>
        <span class="diagnostic-count warning">0 warnings</span>
      </div>
    </div>
  )
}
