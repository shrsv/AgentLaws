# Export Samples

This folder contains sample exports of the `examples/engineering` lawbook in every supported format. Use them to see what AgentLaws output looks like without installing or compiling anything.

| File | Format | Best for |
|---|---|---|
| [engineering-governance.html](engineering-governance.html) | HTML | Sharing in a browser, embedding in a wiki or intranet |
| [engineering-governance.pdf](engineering-governance.pdf) | PDF | Printing, attaching to compliance docs, offline review |
| [engineering-governance.md](engineering-governance.md) | Markdown | PR descriptions, wikis, feeding into other Markdown tools |
| [engineering-governance.json](engineering-governance.json) | JSON | Programmatic consumption, CI pipelines, diffing |
| [combined/all-governance.html](combined/all-governance.html) | Combined HTML | All three example books in one document |
| [combined/all-governance.pdf](combined/all-governance.pdf) | Combined PDF | Full governance package for offline review |
| [combined/all-governance.md](combined/all-governance.md) | Combined Markdown | All books in one Markdown file |

## Regenerate

```bash
# Single book
alaws compile examples/engineering --format html,pdf,md,json --out samples

# All books combined
alaws export examples --format html,pdf,md --title "All Governance" --out samples/combined
```
