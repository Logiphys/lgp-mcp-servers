"""Offline fixtures for the scheduled public capability comparison."""

import copy
import json
import unittest
from unittest.mock import patch

import kaseya_watch as watch


SHA = "a" * 40
CATALOG = """## Kaseya MCP tools

### Datto RMM
| Tool | Operation | Description |
| --- | --- | --- |
| **list_patches** | Read | List patches |

### Autotask
| Tool | Operation | Description |
| --- | --- | --- |
| **query-entity** (Preview) | Read | Query an entity |
## Another section
| text outside catalog |
"""


def publications():
    field_page = "<h1>Service</h1><h2>Field definitions</h2>units<h2>Additional resources</h2>"
    return {
        watch.API + "/commits/HEAD": json.dumps({"sha": SHA}),
        watch.API + "/git/trees/" + SHA + "?recursive=1": json.dumps({
            "truncated": False, "tree": [{"type": "blob", "path": "RMM/relay.mcpb", "sha": "b" * 40}]}),
        "https://raw.githubusercontent.com/kaseya/mcp-servers/" + SHA + "/README.md": CATALOG,
        watch.API + "/releases?per_page=100&page=1": "[]",
        watch.RMM: json.dumps({"paths": {"/v2/patches": {"get": {"summary": "Patches"}}},
                               "components": {"schemas": {"Patch": {"type": "object"}}}}),
        watch.ITGLUE: "<h1>About</h1>Intro<h1>GET /assets</h1><p>Read an asset</p>",
        watch.INDEX: '<tr><td><a href="ContractServicesEntity.htm">Services</a></td></tr>',
        watch.AUTOTASK + "ContractServicesEntity.htm": field_page,
        watch.AUTOTASK + "ContractServiceUnitsEntity.htm": field_page,
    }


class CapabilityReviewTest(unittest.TestCase):
    def test_bidirectional_report_and_acknowledgement(self):
        docs = publications()
        with patch.object(watch, "fetch", side_effect=docs.__getitem__):
            baseline, _ = watch.snapshot()
            readme = next(key for key in docs if key.endswith("/README.md"))
            docs[readme] = docs[readme].replace("List patches", "Read patch details").replace(
                "**query-entity** (Preview) | Read | Query an entity", "**new-tool** | Write | @team `literal`")
            docs[watch.RMM] = docs[watch.RMM].replace('"object"', '"array"')
            docs[watch.AUTOTASK + "ContractServiceUnitsEntity.htm"] = docs[watch.AUTOTASK + "ContractServiceUnitsEntity.htm"].replace("units", "quantity")
            current, tools = watch.snapshot()
        report = watch.report(baseline, current, tools)
        self.assertIn("**Added** ` Autotask / new-tool `", report)
        self.assertIn("**Removed** ` Autotask / query-entity `", report)
        self.assertIn("**Changed** ` Datto RMM / list_patches `", report)
        self.assertIn("`` Write: @team `literal` ``", report)
        self.assertIn("**Changed** ` Patch `", report)
        self.assertIn("**Changed** ` ContractServiceUnits `", report)
        self.assertEqual(watch.report(current, copy.deepcopy(current), tools), "")

    def test_new_product_and_stable_normalization(self):
        extended = CATALOG.replace("## Another section", "### New Product\n| **lookup** | Read | Find an item |\n## Another section")
        self.assertIn("New Product / lookup", watch.catalog(extended))
        self.assertEqual(watch.catalog(CATALOG), watch.catalog(CATALOG.replace("List patches", "List    patches")))
        old = "<h1>GET /assets</h1>Read <b>asset</b> <script>timestamp</script>"
        new = "<h1>GET /assets</h1>Read asset"
        self.assertEqual(watch.itglue_operations(old), watch.itglue_operations(new))
        self.assertEqual(watch.digest({"b": 2, "a": 1}), watch.digest({"a": 1, "b": 2}))

    def test_fails_on_missing_evidence(self):
        docs = publications()
        for name, bad in [
            (watch.API + "/git/trees/" + SHA + "?recursive=1", '{"truncated":true}'),
            (watch.ITGLUE, "<h1>Access temporarily unavailable</h1>"),
            (watch.AUTOTASK + "ContractServicesEntity.htm", "<h1>Login</h1>"),
        ]:
            with self.subTest(source=name), patch.object(watch, "fetch", side_effect={**docs, name: bad}.__getitem__):
                with self.assertRaises(ValueError):
                    watch.snapshot()
        for bad in [CATALOG.replace("| **list_patches** | Read | List patches |", "missing table"),
                    CATALOG.replace("## Kaseya MCP tools", "## Other heading"),
                    CATALOG.replace("### Autotask", "| **list_patches** | Read | Duplicate |\n### Autotask")]:
            with self.assertRaises(ValueError):
                watch.catalog(bad)
        with self.assertRaises(ValueError):
            watch.report({"sources": {}}, {"sources": {}}, {})

    def test_release_pages_and_report_bound(self):
        docs = publications()
        docs[watch.API + "/releases?per_page=100&page=1"] = json.dumps([
            {"tag_name": str(i), "name": "Release", "body": "New tools", "prerelease": False} for i in range(100)])
        docs[watch.API + "/releases?per_page=100&page=2"] = '[{"tag_name":"last","name":"Last","body":null,"prerelease":true}]'
        with patch.object(watch, "fetch", side_effect=docs.__getitem__):
            current, tools = watch.snapshot()
        self.assertEqual(len(current["sources"]["Release notes"]), 101)
        baseline = copy.deepcopy(current)
        for group in baseline["sources"].values():
            for i in range(200):
                group[str(i) + "🗃" * 180] = "fixture"
        report = watch.report(baseline, current, tools)
        self.assertIn("further differences", report)
        self.assertLess(len(report.encode()), 60000)


if __name__ == "__main__":
    unittest.main()
