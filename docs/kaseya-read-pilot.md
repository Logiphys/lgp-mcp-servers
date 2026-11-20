# Datto RMM and Autotask production reads

The two opt-in tools run inside the existing stdio binaries and use the product REST credentials configured for those binaries. Datto RMM exposes patch status for a device or site. Autotask exposes recurring contract service lines for the controlling team's billed-versus-installed comparison. [Nine-product decision](kaseya-mcp-comparison.md) · [Team request](https://github.com/Logiphys/lgp-mcp-servers/issues/3)

## Tool contracts

| Gateway tool | Example arguments | Result |
|---|---|---|
| `datto_rmm_list_patches` | `{"deviceUid":"device-1","installStatus":"APPROVED_PENDING","max":50}` | `items` contains documented patch fields including `patchId` and `installStatus`. `pagination` carries the upstream counts and page links. |
| `autotask_search_contract_services` | `{"contractID":7,"asOf":"2026-04-15","pageSize":25}` | `asOf`, one `items` entry per contract service line, and `summary` with `returned`, `page`, `pageSize`, `hasMore` and `nextPageUrl`. |

**Patch inventory.** Supply exactly one of `deviceUid` and `siteUid`. Site rows aggregate a patch across devices and include `totalDevices`. The optional status filter accepts `INSTALLED`, `APPROVED_PENDING` or `NOT_APPROVED`. Omit `page` for the first page. Use the page number from the provider's next link with the original scope, status and size. An upstream access denial returns a tool error. [Datto RMM OpenAPI](https://merlot-api.centrastage.net/api/v3/api-docs/Datto-RMM)

**Contract services.** Supply `contractID`, `contractIDs` or an exact line `id`. Narrow the result with `serviceID` or `serviceIDs`. Singular IDs use equality and lists use membership. The default page size is 25 and the maximum is 500. Start at page 1. To resume efficiently, send the returned `nextPageUrl`, original filters, `pageSize` and `asOf` back to the tool. Pass the next page number if a client displays it. An omitted date defaults to the current UTC calendar date.

| Value | Meaning for the controlling team |
|---|---|
| `id`, `contractID`, `serviceID` | Identify the contract line, contract and service. The same service can occur twice on the same contract. Join period rows by `ContractServiceUnits.contractServiceID`. |
| `unitPrice`, `internalCurrencyUnitPrice` | Prices on the contract line when queried. The internal currency field can be null. |
| `quantity`, `period.price`, `period.internalCurrencyPrice` | Quantity and the provider's price fields for the selected date. Check the period price and currency in the portal before treating it as a historical unit rate for monthly recurring revenue. The result includes the period ID and bounds. |
| `quantity: null`, `period: null` | The provider returned the line but no applicable period. Preserve it as missing evidence. Autotask omits zero-unit periods. |

The tool reads both entities, checks the line identities and follows every matching period page. When an adjustment takes effect on a shared boundary, the latest applicable start on that UTC day wins. Conflicting periods with the same latest start, unreadable pages and permission errors fail the call. Product prices retain their sign and currency fields. The response excludes cost fields, internal descriptions and invoice text. [ContractServices](https://autotask.net/help/DeveloperHelp/Content/APIs/REST/Entities/ContractServicesEntity.htm) · [ContractServiceUnits](https://autotask.net/help/DeveloperHelp/Content/APIs/REST/Entities/ContractServiceUnitsEntity.htm)

## Live rollout on the existing gateway

1. Record the two installed backend revisions, the active server configuration and the tools visible to the current basic and sensitive-read roles. Choose an existing authorized device, a known device the RMM credential is denied, and a contract line with a dated period. Record the expected values independently in the product portals. Keep that file and both role credentials in the deployment environment. Use the private fixture shape below.
2. Build only the two backends from the reviewed commit in a clean checkout. The production host runs Linux/amd64. Retain the revision and checksums with the deployment record.

   ```bash
   GOOS=linux GOARCH=amd64 CGO_ENABLED=0 make VERSION="$(git rev-parse HEAD)" build-autotask-mcp build-datto-rmm-mcp
   (cd dist && sha256sum autotask-mcp datto-rmm-mcp > SHA256SUMS.live)
   ```

3. Using the regular production admin path, save restorable copies of the two binaries and their server configurations with their existing permissions. Stage the binaries, verify `sha256sum -c SHA256SUMS.live` in the staging directory and compare the tool lists with the installed revisions. Account for any changes since the last backend deployment, including pre-existing writes that need a privacy tier. The two new tools start disabled.
4. Add the entries below to the **effective** server configuration and verify the dispatcher target check **before** enabling the flags. Preserve other entries and the current user grants. Every existing role at or above the configured tier can read these tools. The backend credential limits which customers it can see. Confirm that the existing roles are entitled to that scope and the financial fields. Use a higher tier or a separately scoped route where required.

   | Server | Append to `tiering.tier2` for the standard sensitive-read tier | Dispatcher mapping | Opt-in flag |
   |---|---|---|---|
   | Autotask | `autotask_search_contract_services` | `execute_tool: toolName` | `AUTOTASK_ENABLE_CONTRACT_SERVICES=true` |
   | Datto RMM | `datto_rmm_list_patches` | — | `DATTO_ENABLE_PATCH_READS=true` |

5. Replace each backend binary atomically on its installed filesystem. Enable its flag and restart that backend through the gateway's server controls. Check its health and its tool list in fresh role sessions before proceeding to the next backend. The gateway runs these binaries as separate processes. [Backend deployment history](journal.md)
6. Run the live acceptance command below. Record the UTC time, backend revisions, roles and pass/fail outcome in the private deployment record. If visibility, a denial, portal values or existing tools differ, disable the affected flag, restore its saved binary and configuration, restart that backend, and verify the previous tool list and health.

## Compare with the live portals

Provide the production HTTPS MCP endpoint as `LGP_PILOT_GATEWAY_URL`, the existing role credentials as `LGP_PILOT_ALLOWED_KEY` and `LGP_PILOT_BASIC_KEY`, and an absolute path to the private portal reference JSON as `LGP_PILOT_FIXTURE`. The basic role must be able to call the Autotask dispatcher to exercise its target check. Leave `LGP_PILOT_GATEWAY_BIN` unset. All values in this example are synthetic.

```json
{
  "deviceUID": "device-1", "deniedDeviceUID": "restricted",
  "patchID": "p1", "installStatus": "APPROVED_PENDING", "patchPage": 0,
  "contractID": 7, "lineID": 11, "asOf": "2026-04-15",
  "quantity": 4, "unitPrice": 9.99, "periodPrice": 4.04
}
```

```bash
go test -tags=pilot -count=1 -v -run '^TestPilot$' ./tests/pilot -args -live
```

Live mode requires HTTPS and rejects a local fixture gateway. It checks role visibility and basic-role denials, including the dispatcher, before reading the reference records. The RMM credential must reject the designated restricted device. The permitted patch status and both direct and dispatched contract reads must match the portal evidence. The command returns a failure on disagreement and keeps customer data out of test output.

## Optional local regression test

From the repository root, point the test at an existing gateway executable. It builds both product servers, launches a separate gateway on loopback, generates two temporary keys and serves synthetic product responses. The test checks tool visibility, direct and dispatched calls, one inaccessible device and the number of requests reaching each product. The temporary environment is removed on exit.

```bash
LGP_PILOT_GATEWAY_BIN=/absolute/path/to/lgp-mcp-gateway go test -race -tags=pilot -count=1 -v ./tests/pilot
```

## Weekly Kaseya review

Once the workflow is on the default branch, GitHub Actions schedules it for Mondays at 07:17 UTC. It also supports manual runs. It reads the Kaseya MCP catalog, published files and release notes across products. For Datto RMM it compares public OpenAPI operations and models. For IT Glue it compares documented HTTP operations. For Autotask it checks the REST entity index and the field documentation for both contract tables. A new product in the MCP catalog is picked up automatically. Product changes absent from these public sources require separate evidence. [Workflow](../.github/workflows/kaseya-watch.yml) · [Manufacturer catalog](https://github.com/kaseya/mcp-servers)

- On a difference the default-branch run opens or refreshes one review issue and attaches a link to the run containing the candidate inventory. A matching inventory closes the issue. An unavailable or incompatible source fails the run and preserves the reviewed baseline.
- Record the decision in an issue comment. Name the user workflow, equivalent local tool, read/write effects, user access and client contract. Choose a native implementation, a scoped official route, or a documented skip for a duplicate. An implementation goes through a pull request with allowed and denied fixtures.
- After reviewing the changes, copy the downloaded candidate `kaseya-reviewed.json` into `config/kaseya-reviewed.json` in a pull request. The next comparison uses the merged snapshot. The workflow has read-only repository contents access. It creates no tool or deployment.

To reproduce the comparison using public sources or run its offline tests:

```bash
python3 -B scripts/kaseya_watch.py --snapshot /tmp/kaseya-candidate.json --report /tmp/kaseya-review.md
python3 -B -m unittest discover -s scripts -p 'test_kaseya_watch.py'
```
