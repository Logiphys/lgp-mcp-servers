# Datto RMM and Autotask read pilot

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

## Repeat the gateway test locally

From the repository root, point the test at an existing gateway executable. It builds both product servers, launches a separate gateway on loopback, generates two temporary keys and serves synthetic product responses. The test checks tool visibility, direct and dispatched calls, one inaccessible device and the number of requests reaching each product. The temporary environment is removed on exit.

```bash
LGP_PILOT_GATEWAY_BIN=/absolute/path/to/lgp-mcp-gateway go test -race -tags=pilot -count=1 -v ./tests/pilot
```

## Enable a test tenant when available

1. Use an isolated gateway with Autotask dispatch-target tier checks. Add the following entries to its existing server configuration **before** enabling either tool. Keep existing tier lists and dispatcher entries. Unclassified names default to tier 1. Use a higher tier if required by the customer's policy.

   | Server configuration | Append to `tiering.tier2` | Dispatcher mapping | Binary environment |
   |---|---|---|---|
   | `config/servers/autotask.yaml` | `autotask_search_contract_services` | `dispatchers.execute_tool: toolName` | `AUTOTASK_ENABLE_CONTRACT_SERVICES=true` |
   | `config/servers/datto-rmm.yaml` | `datto_rmm_list_patches` | — | `DATTO_ENABLE_PATCH_READS=true` |

2. Give the pilot identity the sensitive-read tier for both backends and a second identity tier 1. Use product credentials scoped to the intended RMM sites and Autotask contracts. The gateway grants access to tools. Everyone routed to the same binary shares its product credentials. If users require different customer visibility, give those groups separate backend credentials/routes and verify the effective scope with a known inaccessible record.
3. Install the new binaries on the test gateway, enable the flags and restart the corresponding processes. Confirm the two names in the pilot identity's `tools/list`. To withdraw a pilot, remove its flag and restart that process.
4. Record reference values independently in the two product portals. Choose an existing out-of-scope device and a contract line with a visible period. Store the following JSON shape in a private file outside the repository. Example values are synthetic.

   ```json
   {
     "deviceUID": "device-1", "deniedDeviceUID": "restricted",
     "patchID": "p1", "installStatus": "APPROVED_PENDING", "patchPage": 0,
     "contractID": 7, "lineID": 11, "asOf": "2026-04-15",
     "quantity": 4, "unitPrice": 9.99, "periodPrice": 4.04
   }
   ```

5. Supply the test gateway's HTTPS MCP endpoint in `LGP_PILOT_GATEWAY_URL`, the two keys in `LGP_PILOT_ALLOWED_KEY` and `LGP_PILOT_BASIC_KEY`, and the file path in `LGP_PILOT_FIXTURE`. Leave `LGP_PILOT_GATEWAY_BIN` unset to select the tenant test. Run `go test -tags=pilot -count=1 -v ./tests/pilot`. The basic identity must be able to call the Autotask dispatcher so its target check can be verified. Record the outcome and software revisions without customer records or keys. Tenant results and portal agreement remain to be verified.

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
