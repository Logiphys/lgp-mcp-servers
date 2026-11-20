#!/usr/bin/env python3
"""Compare public Kaseya publications to a maintainer-reviewed snapshot."""

import argparse
import hashlib
import json
import re
from html.parser import HTMLParser
from pathlib import Path
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parents[1]
REVIEWED = ROOT / "config/kaseya-reviewed.json"
REPO = "https://github.com/kaseya/mcp-servers"
API = "https://api.github.com/repos/kaseya/mcp-servers"
RMM = "https://merlot-api.centrastage.net/api/v3/api-docs/Datto-RMM"
ITGLUE = "https://api.itglue.com/developer/"
AUTOTASK = "https://autotask.net/help/DeveloperHelp/Content/APIs/REST/Entities/"
INDEX = AUTOTASK + "_EntitiesOverview.htm"
METHOD = r"(?:GET|PUT|POST|PATCH|DELETE|HEAD|OPTIONS|TRACE)"
SOURCES = {
    "MCP tools": REPO + "#kaseya-mcp-tools",
    "MCP publications": REPO,
    "Release notes": REPO + "/releases",
    "RMM operations": RMM,
    "RMM models": RMM,
    "IT Glue operations": ITGLUE,
    "Autotask entities": INDEX,
    "Autotask field documentation": INDEX,
}


def fetch(url):
    request = Request(url, headers={"User-Agent": "Logiphys-Kaseya-catalog-review"})
    with urlopen(request, timeout=30) as response:
        return response.read().decode("utf-8")


def digest(value):
    canonical = json.dumps(value, sort_keys=True, separators=(",", ":")).encode()
    return hashlib.sha256(canonical).hexdigest()[:24]


def catalog(markdown):
    section = re.search(r"^## Kaseya MCP tools\s*$\n(.*?)(?=^## |\Z)", markdown, re.M | re.S)
    if not section:
        raise ValueError("Kaseya MCP catalog heading missing")
    tools = {}
    product = ""
    product_rows = {}
    for line in section[1].splitlines():
        if line.startswith("### "):
            product = line[4:].strip()
            if not product or product in product_rows:
                raise ValueError("unexpected product heading in catalog")
            product_rows[product] = 0
        if not line.startswith("|") or line.startswith(("| Tool |", "| ---", "|---")):
            continue
        row = re.fullmatch(r"\|\s*\*\*([a-zA-Z0-9_-]+)\*\*([^|]*)\|\s*([^|]+?)\s*\|\s*(.+?)\s*\|", line)
        if not row or not product:
            raise ValueError("unexpected published tool row")
        key = product + " / " + row[1]
        if key in tools:
            raise ValueError("duplicate published tool " + key)
        tools[key] = (row[3], " ".join((row[2] + " " + row[4]).split()))
        product_rows[product] += 1
    if not product_rows or not all(product_rows.values()):
        raise ValueError("product tool table missing or unreadable")
    return tools


class VisibleText(HTMLParser):
    def __init__(self):
        super().__init__()
        self.parts = []
        self.hidden = False

    def handle_starttag(self, tag, attrs):
        if tag in ("style", "script"):
            self.hidden = True
        if tag == "img" and not self.hidden:
            a = dict(attrs)
            self.parts.append(a.get("alt") or a.get("src") or "")

    def handle_endtag(self, tag):
        if tag in ("style", "script"):
            self.hidden = False

    def handle_data(self, data):
        if not self.hidden:
            self.parts.append(data)


def visible(fragment):
    parser = VisibleText()
    parser.feed(fragment)
    return " ".join(" ".join(parser.parts).split())


def itglue_operations(document):
    sections = re.split(r"<h1\b[^>]*>(.*?)</h1>", document, flags=re.I | re.S)
    operations = {}
    for heading, body in zip(sections[1::2], sections[2::2]):
        name = visible(heading)
        if re.match(METHOD + r"\s+/", name):
            if name in operations:
                raise ValueError("duplicate IT Glue operation " + name)
            operations[name] = digest(visible(body))
    return operations


def autotask_entities(document):
    entities = {}
    for row in re.findall(r"<tr\b[^>]*>(.*?)</tr>", document, re.S | re.I):
        match = re.search(r'href="([A-Za-z0-9_-]+Entity\.htm)"', row, re.I)
        if match:
            entities.setdefault(match[1], []).append(visible(row))
    return {name: digest(sorted(rows)) for name, rows in entities.items()}


def snapshot():
    sha = json.loads(fetch(API + "/commits/HEAD"))["sha"]
    if not re.fullmatch(r"[0-9a-f]{40}", sha):
        raise ValueError("invalid manufacturer revision")
    upstream = json.loads(fetch(API + "/git/trees/" + sha + "?recursive=1"))
    if upstream["truncated"]:
        raise ValueError("incomplete manufacturer file tree")
    publications = {f["path"]: f["sha"] for f in upstream["tree"] if f["type"] == "blob"}
    tools = catalog(fetch("https://raw.githubusercontent.com/kaseya/mcp-servers/" + sha + "/README.md"))
    releases = {}
    page = 1
    while True:
        batch = json.loads(fetch(f"{API}/releases?per_page=100&page={page}"))
        for entry in batch:
            releases[entry["tag_name"]] = digest([entry["name"], entry["body"], entry["prerelease"]])
        if len(batch) < 100:
            break
        page += 1
    rmm = json.loads(fetch(RMM))
    operations = {method.upper() + " " + path: digest([obj.get("parameters", []), operation])
                  for path, obj in rmm["paths"].items() for method, operation in obj.items()
                  if re.fullmatch(METHOD, method.upper())}
    field_docs = {}
    for entity in ("ContractServices", "ContractServiceUnits"):
        body = fetch(AUTOTASK + entity + "Entity.htm")
        content = re.search(r"<h1\b[^>]*>(.*?)<h2\b[^>]*>\s*Additional resources", body, re.S | re.I)
        if not content or "Field definitions" not in visible(content[1]):
            raise ValueError("missing field definitions for " + entity)
        field_docs[entity] = digest(visible(content[1]))
    sources = {
        "MCP tools": {name: op + " " + digest(description) for name, (op, description) in tools.items()},
        "MCP publications": publications,
        "Release notes": releases,
        "RMM operations": operations,
        "RMM models": {name: digest(schema) for name, schema in rmm["components"]["schemas"].items()},
        "IT Glue operations": itglue_operations(fetch(ITGLUE)),
        "Autotask entities": autotask_entities(fetch(INDEX)),
        "Autotask field documentation": field_docs,
    }
    for source, entries in sources.items():
        if not entries and source != "Release notes":
            raise ValueError("empty source: " + source)
    return {"mcpRevision": sha, "sources": sources}, tools


def text_cell(value):
    # Render remote text as one literal code span, including embedded backticks.
    value = " ".join(value.split())[:180]
    fence = "`" * (1 + max((len(run) for run in re.findall(r"`+", value)), default=0))
    return f"{fence} {value} {fence}"


def report(previous, current, tools):
    before, after = previous["sources"], current["sources"]
    if set(before) != set(SOURCES) or set(after) != set(SOURCES):
        raise ValueError("incomplete baseline or snapshot")
    sections = []
    for source, url in SOURCES.items():
        first, last = before[source], after[source]
        lines = []
        for name in sorted(first.keys() | last.keys()):
            if first.get(name) == last.get(name):
                continue
            action = "Added" if name not in first else "Removed" if name not in last else "Changed"
            detail = ""
            if source == "MCP tools" and name in tools:
                op, description = tools[name]
                detail = " — " + text_cell(op + ": " + description)
            lines.append(f"- **{action}** {text_cell(name)}{detail}")
        if lines:
            sections.extend([f"### [{source}]({url})", ""])
            used = shown = 0
            for line in lines[:25]:
                used += len(line.encode("utf-8")) + 1
                if used > 5000:
                    break
                sections.append(line)
                shown += 1
            if shown < len(lines):
                sections.append(f"{len(lines)-shown} further differences in the candidate snapshot.")
            sections.append("")
    if not sections:
        return ""
    old, new = previous["mcpRevision"], current["mcpRevision"]
    return "\n".join([
        "## Kaseya changes awaiting review", "",
        f"[Reviewed MCP revision]({REPO}/tree/{old}) · [Current MCP revision]({REPO}/tree/{new})", "",
        f"Snapshot comparison: `{digest(before)}` → `{digest(after)}`. The workflow artifact contains the full candidate.", "",
        *sections, "## Maintainer decision", "",
        "Identify the user workflow and its existing Logiphys equivalent. Evaluate direct product API support, "
        "a scoped official route, or skipping a duplicate. Record the read/write effects, role and privacy tier, "
        "compatibility and a permitted/denied test before approving an implementation.",
        "Accept the reviewed snapshot through a separate baseline change in a pull request.", "",
    ])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--snapshot", type=Path, required=True, help="write the candidate JSON here")
    parser.add_argument("--report", type=Path, required=True, help="write Markdown here (empty when unchanged)")
    parser.add_argument("--baseline", type=Path, default=REVIEWED)
    args = parser.parse_args()
    previous = json.loads(args.baseline.read_text())
    current, tools = snapshot()
    message = report(previous, current, tools)
    args.snapshot.write_text(json.dumps(current, indent=2, sort_keys=True) + "\n")
    args.report.write_text(message)


if __name__ == "__main__":
    main()
