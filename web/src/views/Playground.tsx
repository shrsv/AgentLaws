import { useEffect, useMemo, useState } from "preact/hooks";
import { api, type Operation } from "../api";
import type { Route } from "../router";

interface Props {
  path: string;
  navigate: (r: Route) => void;
}

// Substitutes {param} placeholders in a template with the user's actual
// entered values - the same mechanism for both the CLI-equivalent and the
// Go-snippet preview, so what you see is exactly what you could type.
function fill(template: string, values: Record<string, string>): string {
  return template.replace(/\{(\w+)\}/g, (_, k) => (k === "path" || k === "book" ? values[k] || values.path || "" : values[k] ?? `{${k}}`));
}

export function Playground({ path, navigate }: Props) {
  const [operations, setOperations] = useState<Operation[]>([]);
  const [selectedID, setSelectedID] = useState<string | null>(null);
  const [values, setValues] = useState<Record<string, string>>({});
  const [result, setResult] = useState<string | null>(null);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.operations().then((ops) => {
      setOperations(ops);
      if (ops.length > 0) setSelectedID(ops[0].id);
    });
  }, []);

  const op = operations.find((o) => o.id === selectedID) ?? null;
  const groups = useMemo(() => {
    const g: Record<string, Operation[]> = {};
    for (const o of operations) (g[o.group] ??= []).push(o);
    return g;
  }, [operations]);

  useEffect(() => {
    setValues((v) => ({ book: path, path, ...v }));
  }, [path]);

  async function run() {
    if (!op) return;
    setRunning(true);
    setError(null);
    setResult(null);
    try {
      const url = op.method === "GET" ? fillURL(op, values) : op.path;
      const res = await fetch(url, {
        method: op.method,
        headers: op.method === "GET" ? undefined : { "Content-Type": "application/json" },
        body: op.method === "GET" ? undefined : JSON.stringify(bodyFrom(op, values)),
      });
      const body = await res.json();
      setResult(JSON.stringify(body, null, 2));
    } catch (e) {
      setError(String(e));
    } finally {
      setRunning(false);
    }
  }

  return (
    <div class="shell">
      <div class="titlebar">
        <button class="link-button" onClick={() => navigate({ name: "book", path })}>
          ← {path}
        </button>
        <span class="book-title">Playground</span>
        <div class="spacer" />
        <span class="path">learn the CLI and pkg/alaws while you use the UI</span>
      </div>

      <div class="workbench playground-workbench">
        <nav class="sidebar" aria-label="Operations">
          {Object.entries(groups).map(([group, ops]) => (
            <div key={group}>
              <div class="sidebar-title">{group}</div>
              <ul class="tree">
                {ops.map((o) => (
                  <li key={o.id} class="tree-node" aria-selected={o.id === selectedID} onClick={() => setSelectedID(o.id)}>
                    <span class="node-title">{o.summary}</span>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </nav>

        <div class="divider" />

        <main class="detail playground-detail">
          {op && (
            <>
              <h1>{op.summary}</h1>
              <div class="section-id">
                {op.method} {op.path}
              </div>

              <div class="playground-form">
                {op.params.map((p) => (
                  <label key={p.name} class="playground-field">
                    <span>
                      {p.name}
                      {p.required && " *"}
                    </span>
                    <input
                      placeholder={p.description}
                      value={values[p.name] ?? ""}
                      onInput={(e) => setValues((v) => ({ ...v, [p.name]: (e.target as HTMLInputElement).value }))}
                    />
                  </label>
                ))}
                <button class="btn btn-primary" disabled={running} onClick={run}>
                  {running ? "Running…" : "Run"}
                </button>
              </div>

              <h2 class="playground-subheading">Equivalent CLI command</h2>
              <pre class="code-block">{fill(op.cliTemplate, values)}</pre>

              <h2 class="playground-subheading">Equivalent Go (pkg/alaws)</h2>
              <pre class="code-block">{fill(op.goTemplate, values)}</pre>

              {error && <p class="error-text">{error}</p>}
              {result && (
                <>
                  <h2 class="playground-subheading">Result</h2>
                  <pre class="code-block result-block">{result}</pre>
                </>
              )}
            </>
          )}
        </main>
      </div>
    </div>
  );
}

function fillURL(op: Operation, values: Record<string, string>): string {
  const q = new URLSearchParams();
  for (const p of op.params) {
    const v = values[p.name];
    if (v) q.set(p.name, v);
  }
  const qstr = q.toString();
  return qstr ? `${op.path}?${qstr}` : op.path;
}

function bodyFrom(op: Operation, values: Record<string, string>): Record<string, string> {
  const body: Record<string, string> = {};
  for (const p of op.params) {
    if (values[p.name]) body[p.name] = values[p.name];
  }
  return body;
}
