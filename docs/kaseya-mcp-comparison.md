# Which Kaseya MCP servers should Logiphys maintain?

**Decision, 20 November 2026. Maintain all nine product servers. Start with a read-only Datto RMM pilot.** Kaseya publishes 47 tools across three overlapping products. Even those three leave existing workflows without a replacement. For the other six, the first-party catalog and product documentation identify no product MCP to switch to. A portfolio cutover would lose working client interfaces and would put Autotask write attribution at risk. [Kaseya catalog and availability FAQ][catalog] · [Logiphys inventory at the comparison revision][baseline]

| Priority | Product | LGP / Kaseya tools | Replacement status | What to adopt, retain or skip |
|---|---|---:|---|---|
| 1 | Datto RMM | 55 / 16 | Partial candidate | Pilot patch detail and scoped device, alert and activity reads. Vendor measurements give concrete fixtures for comparison. Keep site administration, variables, historical job output and the current write interface. Defer official writes until preview, submission and audit can preserve client behavior. |
| 2 | IT Glue | 31 / 20 | Partial candidate | Pilot a per-user OAuth connection for organization, folder and asset lookup. Draft creation and section editing are candidates after access tests. Retain password, contact, expiration, publishing and deletion tools. Skip a shared broad key for the initial pilot. |
| 3 | Autotask PSA | 82 / 11 | Partial candidate | Pilot restricted contract service reads for the outstanding request for quantities and unit prices, then ticket history. Keep the PSA server for billing, quotes, child records, existing searches and attributable writes. The generic write API would add the most identity, policy and result-translation work. |
| 4 | Datto Networking | 13 / 0 | No official candidate | Keep the implementation available for validation. Prioritize one read-only DNA tenant check before funding expansion or enabling a gateway route. Revisit adoption upon publication of a networking MCP with device and traffic calls. |
| 5 | Datto EDR | 21 / 0 | No official candidate | Maintain EDR investigation and scoped response actions. RMM endpoint posture can enrich triage. Skip using it to replace scans, isolation or restoration. Reassess when a first-party EDR connector exposes agent actions. |
| 5 | RocketCyber | 13 / 0 | No official candidate | Maintain SOC event and incident investigation. RMM monitoring alerts provide a different record. Reassess when an official server can return RocketCyber incidents, events and suppression rules. |
| 5 | Datto Unified Continuity | 20 / 0 | No official candidate | Maintain BCDR appliance, SaaS Protection and Direct-to-Cloud inventory. Skip inferring backup state from installed agents. Reassess when the continuity portal publishes corresponding MCP calls. |
| 5 | Datto Backup | 12 / 0 | No official candidate | Maintain the backup portal reporting client for customers, protected assets, backup history and alerts. Its Unitrends API and credentials differ from Unified Continuity. Reassess on a release covering that API surface. |
| 5 | myITprocess | 11 / 0 | No official candidate | Maintain vCIO review and planning reads. A PSA company or IT Glue organization supplies no review, finding or initiative history. Reassess when Kaseya publishes vCIO reporting tools. |

Priorities order the **next engineering work**. The five tied products have no candidate with which to reduce backend maintenance today. No tenant usage, staffing or cost figures were assumed. Full replacement, directly or through a maintained adapter, requires every existing tool, input, result, side effect and user access rule to survive for existing clients. A maintained adapter can qualify, but still costs engineering time for schema changes, auth and regression testing. Using Kaseya for some workflows while retaining the LGP server for others is a partial replacement.

## How to read the comparison

The local baseline is `795eb82dbbab30edd6faa5df893899600db7a676`. We followed the nine `main()` entry points through `RegisterTools()`, their handlers and clients, plus `RegisterServerInfoTool()` once per binary. The 258 names include operational and dispatch tools. Kaseya's `512c355fc525233e894e013af1ed4756c67bedd3` catalog lists 11 Autotask, 16 RMM and 20 IT Glue calls. A skill can combine several calls. The three published MCPB packages contain client relays to hosted services. Their manifests and relay code establish transport behavior. The remote tool implementations and a complete tenant-specific schema snapshot are unavailable in those packages. [LGP entry points][entry] · [Shared server information][info] · [Kaseya catalog][catalog]

The tables use backend tool names. To reconstruct a client-visible name, append the listed name to the gateway prefix for the product, e.g. `autotask_create_ticket`. The reviewed gateway templates use `autotask_`, `datto_rmm_`, `itglue_`, `datto_edr_`, `datto_uc_`, `rocketcyber_`, `datto_backup_` and `myitprocess_`. Datto Networking has no enabled example route or shipped server template at that revision. An installed gateway can have different prefixes or enabled tools. The replacement gate covers the server inventory and the names actually configured at the gateway.

**Same workflow** means the sources document the same business operation. **Partial** identifies a narrower scope, a changed effect, or a required composition. **LGP only** and **Kaseya only** refer to the reviewed MCP surfaces. **Unproven** means the sources cannot establish the claimed behavior. A dash denotes an empty side of a mapping. For every row, local behavior is evidenced by handler **code** and official product behavior by **documentation**, unless a more specific test or observation is cited. The sections identify synthetic tests and dated live evidence separately. On 9 April 2026 Logiphys recorded real-credential tests of every product server except Datto Networking. That is server-level evidence for eight products. The record does not enumerate each tool, role or later change. This comparison used published evidence and static source inspection. [Logiphys validation record][issue1]

## Autotask PSA

The local `autotask_` tools implement named workflows. The official catalog offers entity discovery, schema, query, count, retrieval and mutation, plus ticket-specific calls. Its entity table enumerates 26 readable record types and permitted verbs. Use that table to bound a generic call. For example, `ServiceCalls` supports create and update, while deletion is limited to checklist items and ticket links. A second naming issue affects discovery itself. The catalog spells `create-entity` and `get-ticket` with hyphens, while the vendor's bundled skills refer to `create_entity` and `get_ticket`. Capture the tenant's `tools/list` before assigning aliases or role rules. The guarded-changes skill previews changes and asks for confirmation. It also describes deployments where generic writes are switched off. [LGP registrations][at-reg] · [Kaseya tool catalog][catalog] · [Entity/verb table][at-entities] · [Autotask skill package][at-zip]

| LGP tools | Kaseya tools (catalog spelling) | Operation and client consequence |
|---|---|---|
| `search_tickets`, `get_ticket_details` | `search-tickets`, `get-ticket`, `query-entity`, `get-entity` | Read, same ticket lookup, partial search. Local `searchTerm` matches a ticket number prefix, search can omit status, and detail/search can enrich IDs with names. Official ticket search defaults to open tickets. Preserve closed tickets, time filters, assigned resources, compact fields and empty results for old calls. [Tickets][at-tickets] |
| `create_ticket`, `update_ticket` | `create-entity`, `update-entity` | Write, same workflow for Tickets. Local create requires numeric `companyID` and string `title` and `description`. Update requires `ticketId`, while local detail requires `ticketID`. Official calls select an entity and use field values or a row ID. Read-back, attribution and the legacy result require translation. [Tickets][at-tickets] |
| `get_ticket_note`, `search_ticket_notes`, `create_ticket_note` | `get-entity`, `query-entity`, `add-ticket-note`, `create-entity` | Read/write, same ticket-note workflow. Local lookups enforce the parent ticket context and create accepts title, description and note visibility/type. Preserve the parent check, publication semantics and the two IDs. A global note search is outside the published scope. [Tickets][at-tickets] |
| `search_companies`, `create_company`, `update_company` | `query-entity`, `create-entity`, `update-entity` | Read/write, same company operations. Numeric and cursor paging, company-name matching and the exact response envelope are client dependencies. [Companies][at-companies] |
| `search_contacts`, `create_contact` | `query-entity`, `create-entity` | Read/write, same contact operations. Translate the local search term/company/active filters and required company, first and last names into Contacts fields. Kaseya also permits updating a contact. [Contacts][at-contacts] |
| `get_opportunity`, `search_opportunities`, `create_opportunity`, `search_contracts` | `get-entity`, `query-entity`, `create-entity` | Same opportunity reads/create and contract reads. Preserve company/status filters, required opportunity ownership, stage and dates. Kaseya adds opportunity update and contract create/update. [Financial handlers][at-financial] |
| `search_projects`, `create_project`, `search_tasks`, `create_task` | `query-entity`, `create-entity` | Read/write, same project/task workflows. Scope tasks to their project and retain status, assignment and the local ID envelopes. Kaseya adds updates to both. [Projects][at-projects] |
| `get_service_call`, `search_service_calls`, `create_service_call`, `update_service_call` | `get-entity`, `query-entity`, `create-entity`, `update-entity` | Same ServiceCalls reads/create/update. Preserve numeric IDs and scheduling fields. [Service calls][at-calls] |
| `search_time_entries`, `create_time_entry` | `query-entity`, `create-entity` | Read/write, same time-entry workflow. The booked `resourceID` and the person who performed the write are distinct. Preserve ticket, hours, date and proxy-time permissions. Kaseya adds update. [Time and billing][at-time] |
| `search_resources`, `search_configuration_items`, `get_service`, `search_services`, `get_service_bundle`, `search_service_bundles` | `get-entity`, `query-entity` | Same record reads on Resources, ConfigurationItems, Services and ServiceBundles. Local filters and pagination vary by resource. Resource search has a recorded upstream failure mode. [Resources and configurations][at-tickets] · [Catalog items][at-products] · [API quirks][quirks] |
| `get_field_info`, `list_queues`, `list_ticket_statuses`, `list_ticket_priorities` | `describe-entity` | Read, partial. Local field aliases, optional single-field selection and active picklist projections are separate interfaces. Kaseya exposes live field flags/operators for supported entities. Local metadata lookup can address entities outside that list. [Utility handlers][at-utility] · [Picklists][at-picklist] |
| `get_ticket_charge`, `search_ticket_charges`, `create_ticket_charge`, `update_ticket_charge`, `delete_ticket_charge`, `get_ticket_attachment`, `search_ticket_attachments` | — | LGP only. Ticket charges have individual reads/writes/deletes and attachment reads use the parent ticket. Neither entity appears in the official list. [Tickets][at-tickets] |
| `get_company_note`, `search_company_notes`, `create_company_note`, `get_project_note`, `search_project_notes`, `create_project_note`, `list_phases`, `create_phase` | — | LGP only, read/create company notes, project notes and phases. The vendor's supported TicketNotes entity has a different parent. [Companies][at-companies] · [Projects][at-projects] |
| `get_quote`, `search_quotes`, `create_quote`, `get_quote_item`, `search_quote_items`, `create_quote_item`, `update_quote_item`, `delete_quote_item`, `search_invoices` | — | LGP only, quote headers/lines, pricing, line deletion and invoice lookup. The local invoice company filter has a documented reliability limitation. [Financial handlers][at-financial] · [API quirks][quirks] |
| `search_billing_items`, `get_billing_item`, `search_billing_item_approval_levels`, `get_expense_report`, `search_expense_reports`, `create_expense_report`, `create_expense_item`, `get_product`, `search_products` | — | LGP only, billing approval, expenses and product catalog. The official Services entity covers services. [Time and billing][at-time] · [Catalog items][at-products] |
| `delete_service_call`, `search_service_call_tickets`, `create_service_call_ticket`, `delete_service_call_ticket`, `search_service_call_ticket_resources`, `create_service_call_ticket_resource`, `delete_service_call_ticket_resource` | — | LGP only. Kaseya explicitly excludes ServiceCalls delete. Its ticket-to-contact/device link tools do not operate on service-call ticket/resource children. [Service calls][at-calls] · [Permitted verbs][at-entities] |
| `list_categories`, `list_category_tools`, `router`, `execute_tool`, `test_connection`, `server_info` | — | LGP only, operations and discovery. The dispatcher takes `toolName` plus arguments at the top level and passes actor context to the target. Entity discovery would not preserve these interfaces. [Meta-tools][at-meta] · [Utility handlers][at-utility] · [Server information][info] |
| — | `search-entities`, `count-entities` | Kaseya only, discover an entity by description and count matching rows without enumerating a result set. These are useful constrained reads. [Kaseya catalog][catalog] |
| — | `delete-entity` | Kaseya only for checklist item and extra-contact/device link deletion. The bundled skill describes a result containing the pre-delete record. LGP deletes different entities and returns a message. [Entity/verb table][at-entities] · [Guarded changes][at-zip] |

**Additional entity operations.** Official generic retrieval by ID adds direct access where LGP offers search alone. `query-entity` also supports field selection and a filter disjunction. The following additions share the generic tool names already counted above. Each new entity and verb needs its own access decision. [Kaseya tool catalog][catalog] · [Supported entities and verbs][at-entities]

| Official entities | Additional documented workflows |
|---|---|
| Contacts, Opportunities, Projects, Tasks, TimeEntries, Resources | Update existing records. An employee-facing read facade must exclude the corresponding write capability. |
| ConfigurationItems, Contracts | Create and update. Local clients currently search these entities. |
| ContractServices, ContractServiceUnits, ContractServiceBundles, ContractServiceBundleUnits, Departments, TicketHistory, ChecklistLibraries, TicketCategories | Read contract composition and units, departments, history and checklist/category metadata. A restricted ContractServices query could serve an existing request for line-level quantity and pricing. [LGP use case][issue3] |
| TicketChecklistItems, TicketChecklistLibraries, TicketAdditionalContacts, TicketAdditionalConfigurationItems | Read and create each. Update and delete checklist items. Delete extra-contact and extra-device links. Evaluate each verb separately before exposing a generic tool. |

**Write attribution and identifiers.** `WithActorContext` reads `_meta["logiphys.de/actor"]`. With a positive `autotask_resource_id`, `writeContext` adds `ImpersonationResourceId` to Create, CreateChild, Update, Delete and DeleteChild. Reads omit it. Without a resource ID the API user performs the operation. The caller cannot establish a trusted identity by supplying its own metadata. Gateway-side derivation and stripping must be proven in the deployed path. The backend does not retry a denied impersonation without the header. The product's behavior on denial and delete attribution remain outstanding live checks. The June 2026 journal records a successful ticket status update and read-back. The October mock tests cover direct and dispatched writes and ID formatting. They establish different facts. [Actor handling][actor] · [Write context][impersonation] · [Autotask client][at-client] · [Synthetic tests][at-tests] · [Journal][journal] · [Open test-object check][issue4]

Local creates return MCP text containing JSON like `{"message":"Ticket created successfully","itemId":42,"id":42}`. Existing clients can read `id` and the audit contract can extract `itemId`. Kaseya describes create/update returning the enriched row after reading it back and records API-credential writes under the API identity. Its relay forwards MCP messages. There is no published mapping from LGP actor metadata to a per-write Autotask header, nor a guarantee of the two numeric top-level keys. A facade can duplicate a reliably returned ID, but cannot repair a misattributed product history entry. Retain the LGP write path until a test proves both identity and output. On an ambiguous write, reconcile the record before retrying. [Result format][at-format] · [Actor and audit contract][access] · [Kaseya credentials and attribution][catalog] · [Guarded changes][at-zip]

**Read and error contracts.** `search_companies` accepts integer `page` (default 1), `pageSize` (default 25, maximum 200), optional string `searchTerm`, boolean `isActive` and string `nextPageUrl`. It returns JSON text with `items` and `summary.{returned,page,pageSize,hasMore,nextPageUrl}`. An empty page may have a continuation and a full page may be final. Numbered calls without a cursor replay earlier pages. For the official generic query, map `searchTerm` to a `contains` filter on `Companies.companyName`. The tool accepts filters, fields and a limit, and returns an ID-ordered `nextPage` hint. Kaseya's briefing instructions describe a default limit of 25 and a tenant-dependent default projection of eight fields. Request the fields an existing caller needs. Kaseya documents no translation for the local cursor. Preserve in-flight local cursors or finish them on the original route. Other local searches use tool-specific compact projections and `FormatNotFound` can be successful text containing an `error` key. Compare that separately from MCP `isError`, RPC errors and official `get-entity` returning `found: false`. [Company pagination and fixtures][at-pages] · [Formats][at-format] · [API quirks][quirks] · [Kaseya catalog][catalog] · [Briefing skill][at-zip]

## Datto RMM

For `datto_rmm_`, inventory is the strongest pilot. Kaseya's plugin includes **vendor live observations** on a connector identified as drmm v0.4.0. Its 30 September 2026 notes report list sizes for sites, devices, alerts, software, patches and components. They also document field omissions on device detail and limitations on audit data for non-agent devices. The activity-log fields were checked on 5 October 2026 on one tenant after another returned HTTP 500. These observations apply to those reads on that connector. The shipped bridge has a separate version, 0.1.0. Official writes have catalog evidence here. LGP's client tests exercise HTTP verbs, OAuth, pagination and failures with a test server. [RMM skill package, especially `drmm-device-quick-look` and `drmm-device-timeline` connector notes][rmm-zip] · [Kaseya catalog][catalog] · [LGP registrations][rmm-reg] · [LGP tests][rmm-tests]

| LGP tools | Kaseya tools | Operation and client consequence |
|---|---|---|
| `list_sites`, `get_site` | `list_sites`, `get_site` | Same read workflow. Local site lookup takes string `siteUid` and listing supports a partial `siteName`. Official lists expose counts and device totals. Confirm name filtering, paging base and the full detail shape. [Account][rmm-account] · [Sites][rmm-sites] |
| `list_devices`, `list_site_devices` | `list_devices` | Partial inventory/search. Local account filters include OS, device type, hostname and site name. Vendor instructions select one scope per call, use `site_uid` or `filter_id`, and use the “All Devices” filter for a full account inventory. The official hostname search can match several records. [Account][rmm-account] · [Sites][rmm-sites] · [Vendor lookup examples][rmm-zip] |
| `get_device`, `get_device_by_id`, `get_device_by_mac` | `get_device`, `list_devices` | Same UID lookup, partial numeric/MAC lookup. Translate string `deviceUid` to `device_uid`, numeric `deviceId` to `device_id` and normalize `macAddress` for `mac_address`. Local ID/MAC calls return one object. Official list-by-MAC can return several. Its measured device projection omitted numeric ID and UDFs. [Devices][rmm-devices] · [Vendor observations][rmm-zip] |
| `get_device_audit`, `get_device_audit_by_mac`, `get_esxi_audit`, `get_printer_audit` | `get_device_audit`, `list_devices` | Partial. Generic audit supplies hardware/OS/network facts. MAC lookup needs resolution. Dedicated printer attributes and ESXi-equivalent detail are unproven. The vendor records reduced ESXi coverage and class-dependent errors. [Device audit][rmm-devices] · [Specialized audit][rmm-audit] · [Vendor audit notes][rmm-zip] |
| `get_device_software` | `list_device_software` | Same installed-software read. Local device UID yields a list result. Vendor pages by device and reports name/version. Preserve completeness and fields for large inventories. [Audit][rmm-audit] |
| `list_default_filters`, `list_custom_filters`, `list_site_filters` | `list_device_filters` | Same three filter catalogs through one selector. Preserve no-argument default/custom calls and the required site UID for site-specific callers. [Filter handlers][rmm-filters] · [Sites][rmm-sites] |
| `list_open_alerts`, `list_resolved_alerts`, `list_device_open_alerts`, `list_device_resolved_alerts`, `list_site_open_alerts`, `list_site_resolved_alerts`, `get_alert` | `list_alerts`, `get_alert` | Same status and scope reads. LGP exposes separate tools and a boolean muted filter. Kaseya uses an account/site/device scope and status. Verify muted filtering, detail fields, times and empty pages. [Account][rmm-account] · [Devices][rmm-devices] · [Sites][rmm-sites] · [Alerts][rmm-alerts] |
| `get_activity_logs` | `list_activity_logs` | Partial account read plus new device/filter-scoped reads. Local `size`, `order`, `from` and `until` are optional. Official account-wide calls specify both times, then feed `_enhanced.next_search_after` back as `search_after`. The observed activity timestamp uses seconds and alert timestamps use milliseconds. [Activity][rmm-activity] · [Vendor timeline notes][rmm-zip] |
| `list_components`, `create_quick_job` | `list_components`, `create_job` | Same catalog and component execution workflow, different write contract. LGP takes device/component UIDs and a job name, forwards supplied component variables and immediately submits. Kaseya previews, requires confirmation to submit, then returns a job UID. Preserve variables, original result and single execution before attempting substitution. [Account][rmm-account] · [Devices][rmm-devices] |
| `get_job`, `get_job_results`, `get_job_stdout`, `get_job_stderr` | `get_job_status` | Partial. LGP accepts an existing job UID, and device UID for per-device results/streams. Stdout and stderr are separate plain-text calls. Kaseya combines status, results and available output. The vendor timeline notes limit lookup to jobs from `create_job`. Keep the historical-job route. [Job handlers][rmm-jobs] · [Vendor known gaps][rmm-zip] |
| `set_device_udf`, `resolve_alert` | `update_device_udfs`, `resolve_alert` | Same writes, different effect on first call. Local calls immediately submit selected `udf1` through `udf30` or the alert UID. Kaseya previews by default, leaves omitted UDFs untouched and treats an already-resolved alert as unchanged. Preserve the user's approval boundary as well as the result. [Devices][rmm-devices] · [Alerts][rmm-alerts] |
| `get_account`, `list_users`, `get_dnet_site_mappings`, `get_site_settings`, `get_site_network_interfaces`, `get_job_components` | — | LGP only, account/user administration reads, networking integration mappings, site settings/interfaces and the components attached to a particular job. The official component catalog lists runnable components. [Account][rmm-account] · [Sites][rmm-sites] · [Jobs][rmm-jobs] |
| `list_account_variables`, `list_site_variables`, `create_account_variable`, `update_account_variable`, `delete_account_variable`, `create_site_variable`, `update_site_variable`, `delete_site_variable` | — | LGP only, variable listing and lifecycle, including masked values. Device UDFs represent different data. [Account][rmm-account] · [Sites][rmm-sites] · [Variable handlers][rmm-vars] |
| `create_site`, `update_site`, `update_site_proxy`, `delete_site_proxy`, `move_device`, `set_device_warranty` | — | LGP only, site/proxy administration and device changes. Site/device reads do not perform them. [Sites][rmm-sites] · [Variable and proxy handlers][rmm-vars] · [Devices][rmm-devices] |
| `get_system_status`, `get_rate_limit`, `get_pagination_config`, `server_info` | — | LGP only, service diagnostics, limits and local binary identity. The bridge can expose `datto_rmm_configuration_error` on setup failure. Count it as a diagnostic and require a successful product read for readiness. [System handlers][rmm-system] · [Server information][info] · [Bridge package][rmm-bundle] |
| — | `list_patches` | Kaseya only, individual patches with approval/install status by device or site. A local device audit can supply a summary. This is the first additive read to pilot. [Kaseya catalog][catalog] · [Vendor patch observations][rmm-zip] |

**Result and paging contract.** Local list handlers return MCP text containing `items` and, if present, `pagination.{page,total_pages,count}`. They pass positive `page` and `max` values to the REST API and otherwise leave them to its defaults. Detail tools return JSON text, while some mutations and both job streams return plain text. Kaseya's measured reads include top-level counts, selected fields and `_enhanced` values, with response-size constraints in the tested client. A same-name official tool does not establish the same fields, page boundaries, error envelope or byte-for-byte stdout/stderr. [LGP list formatting][rmm-helpers] · [Job results][rmm-jobs] · [Manufacturer measurements][rmm-zip]

## IT Glue

For `itglue_`, retaining the user's IT Glue organization scope is a prerequisite for the OAuth pilot. The strongest candidate additions are folder navigation and flexible asset field metadata. Both sides expose document edits, but publishing and deletion remain LGP dependencies. The local JSON:API client flattens resources. A typical list returns `items` and `pagination.{current_page,total_pages,total_count}` in JSON text and returns a human-readable text message when empty. Most paged searches use `page_number=1` and `page_size=50`. The official skills use pagination totals, filters and optional relationship includes. Compare data visible through those relationships as well as the top-level object. Existing client tests use fixtures for list/get/create/update/delete. The reviewed official docs provide workflows and permission claims without a dated tenant execution trace for those writes. [Local registrations][ig-reg] · [Client and flattening][ig-client] · [Client tests][ig-tests] · [IT Glue product help][ig-help] · [Official skill package][ig-zip]

| LGP tools | Kaseya tools | Operation and client consequence |
|---|---|---|
| `search_organizations`, `get_organization` | `list_organizations`, `search_organizations`, `get_organization` | Same enumeration/detail, partial name search. LGP offers paginated filters on type, status, name and PSA ID. Official experimental search ranks a small set of matches. Use listing for enumeration. Official examples pair PSA ID with integration type and can return related resources. [Organizations][ig-orgs] · [Vendor examples][ig-zip] |
| `search_configurations`, `get_configuration` | `list_configurations`, `get_configuration` | Same device inventory/detail. Map organization, type, status, serial and integration IDs. Kaseya adds MAC/asset-tag lookup, sorting, archived state and includes. Confirm local filters, ID types, field projection and empty results. [Configurations][ig-configs] |
| `search_documents`, `get_document` | `list_documents`, `search_documents`, `get_document` | Partial. LGP search takes numeric organization ID, optional name and pages. Its handler issues one organization-scoped list call. Kaseya splits folder metadata listing, experimental ranked search and reading all sections. Body search depends on tenant AI Search/Kaseya Assist. Validate nested folders, matching and order. [Documents][ig-docs] · [Folder limitation][quirks] · [IT Glue help][ig-help] |
| `list_document_sections`, `get_document_section` | `get_document` | Partial read. Local list takes a document ID and requests up to 100 sections. Local detail takes only a section ID. The handler strips HTML only under nested `attributes.content`, while `FlattenResource` normally lifts `content` to the top level. Capture the actual text/HTML shape. Official retrieval includes ordered sections for a known document and offers a content format. Section-ID-only lookup needs a proven parent resolution. [Documents][ig-docs] · [Resource flattening][flatten] · [Vendor examples][ig-zip] |
| `create_document`, `update_document` | `create_document`, `update_document` | Same create/rename, partial contract. Local create accepts an organization ID, name and optional HTML content. The official create takes `organization_id`, `name`, optional `document_folder_id` and ordered `sections`, and creates a draft. Local rename needs an ID plus an actual name change field. Kaseya limits update to renaming. Verify draft visibility, returned IDs and content. [Documents][ig-docs] · [Vendor document workflows][ig-skills] · [Create-documents skill][ig-zip] |
| `create_document_section`, `update_document_section` | `update_document_sections` | Partial write. LGP creates a single heading/text section in a document or updates content by section ID. The official operation edits a document's draft and can add, reorder or change several sections. Preserve the parent binding and the individual section result. Reconcile a timed-out add before retrying. [Documents][ig-docs] · [Vendor examples][ig-zip] |
| `delete_document`, `delete_document_section`, `publish_document` | — | LGP only, explicit deletes of documents/sections and publishing. Creating or editing an official draft cannot satisfy a publish call. [Documents][ig-docs] · [Vendor document workflows][ig-skills] |
| `search_locations`, `get_location` | `list_locations`, `get_location` | Same read. Retain name, city, region, country and organization filters. Check whether optional official notes/relationships expand data returned by a lower-tier read. [Locations][ig-locations] |
| `search_flexible_assets`, `list_flexible_asset_types` | `list_flexible_assets`, `list_flexible_asset_types` | Same type-scoped asset search, partial type listing. LGP requires a string type ID to search and optionally filters type lists by organization at a fixed page of up to 100. Kaseya supports archived assets and type metadata filters. Compare field visibility and organization scope. [Flexible assets][ig-flex] |
| `search_contacts`, `get_contact`, `search_passwords`, `get_password`, `list_password_categories`, `search_domains`, `list_expirations`, `search_configuration_interfaces`, `list_configuration_types`, `list_configuration_statuses` | — | LGP only, contact and credential workflows, expiring resources, domains, device interfaces and lookup dictionaries. `get_password` defaults `show_password` to false. Classify official documents/related objects independently for possible sensitive content. [Contacts][ig-contacts] · [Passwords][ig-passwords] · [Metadata][ig-meta] · [Domains][ig-domains] · [Expirations][ig-expire] · [Interfaces][ig-iface] |
| `health_check`, `server_info` | — | LGP only, product connection check and binary metadata. Preserve a distinct operator diagnostic if a relay or OAuth session fails. [Health][ig-health] · [Server information][info] |
| — | `list_document_folders`, `get_flexible_asset`, `get_flexible_asset_type`, `list_flexible_asset_fields`, `get_flexible_asset_field` | Kaseya only, navigate folder hierarchy, fetch an individual flexible asset/type by ID and inspect ordered field definitions. Pilot an organization-restricted read before exposing arbitrary fields. [Kaseya catalog][catalog] |

## Products without a cataloged first-party MCP

On 20 November 2026 we checked Kaseya's list of supported MCP products, the pinned availability FAQ, and the product-specific sources below for an MCP server or tool catalog. The FAQ names three Early Access products and gives no timeline for others. The six sections identify what would be lost at cutover and the closest overlap through those three servers. For all six the **official MCP** has no evidenced transport, auth flow, tool schema or live result to evaluate. The linked REST APIs and integrations establish product identity. Recheck the product list when the capability named in the priority table ships. [Kaseya supported products][directory] · [Availability FAQ][catalog]

### Datto EDR

| LGP tools | Kaseya tools | Workflow and evidence |
|---|---|---|
| `list_agents`, `get_agent`, `get_agent_count`, `list_organizations`, `list_locations`, `list_device_groups`, `list_policies` | — | LGP only, read EDR agents, tenant structure, groups and policy inventory. RMM device or installed-software reads can indicate an agent is present. They do not recover its EDR ID or policy. |
| `list_alerts`, `get_alert`, `list_alerts_archive`, `get_alert_count`, `get_dashboard`, `list_rules`, `list_suppression_rules`, `list_extensions`, `list_quarantined_files` | — | LGP only, current/archived detections, dashboard, controls and quarantine inventory. Local list results carry `data` and `count`. Historical release notes record the dashboard array fix and entity-specific filter formats. |
| `scan_agent`, `isolate_host`, `restore_host` | — | LGP only, write/execute actions on a string `agentId`. RMM patch or AV status offers no equivalent EDR control. Keep the response actions gated separately from read-only inventory. |
| `test_connection`, `server_info` | — | LGP only, connection and binary diagnostics. |

The Go client targets a tenant-specific Infocyte/EDR instance and uses a query token. LoopBack filtering and paging vary by entity. Keep credentials out of logged URLs. Release notes record corrections to observed shapes and endpoints. A successful live trace for each of the three response actions is absent from the reviewed material. [EDR registrations][edr-reg] · [EDR client][edr-client] · [Recorded release fixes][changes] · [Datto EDR product integrations][edr-help]

### Datto Unified Continuity

| LGP tools | Kaseya tools | Workflow and evidence |
|---|---|---|
| `list_devices`, `get_device`, `list_device_assets`, `list_device_agents`, `list_device_shares`, `list_device_alerts`, `list_device_vm_restores`, `get_device_volume_assets`, `list_agents` | — | LGP only, read appliances by serial, protected assets, shares, alerts and restore records. VM restore history is a read. |
| `list_saas_domains`, `get_saas_seats`, `get_saas_applications`, `list_dtc_assets`, `list_dtc_rmm_templates`, `get_dtc_storage_pool`, `list_dtc_client_assets`, `get_dtc_asset`, `get_activity_log` | — | LGP only, read SaaS Protection, Direct-to-Cloud inventory/storage/templates and continuity activity. An RMM software inventory can identify installed backup software but cannot verify a recovery point or seat. |
| `test_connection`, `server_info` | — | LGP only, diagnostics. No mutation is registered for this product. |

The backend uses public/secret-key Basic auth against the Datto continuity API. `page`/`perPage` are translated to endpoint-specific pagination keys, including underscore-prefixed keys for agent and activity reads. The changelog documents that correction. LGP code and vendor REST documentation support the capability assessment. Tenant completeness remains a later check. [Continuity registrations][uc-reg] · [Continuity client][uc-client] · [Pagination quirks][quirks] · [Datto REST integration guide][uc-help]

### RocketCyber

| LGP tools | Kaseya tools | Workflow and evidence |
|---|---|---|
| `get_account`, `list_agents`, `list_incidents`, `list_events`, `get_event_summary` | — | LGP only, read the SOC account, agents, incidents and application-scoped events. Event listing needs `appId`. An RMM monitor alert has no corresponding SOC case history or verdict. |
| `list_firewalls`, `list_apps`, `get_defender`, `get_office`, `list_suppression_rules`, `get_suppression_rule` | — | LGP only, read firewall/app telemetry, Microsoft security status and suppression rules. |
| `test_connection`, `server_info` | — | LGP only, diagnostics. No SOC write is registered. |

The client selects a US or EU API origin and sends a bearer key. Local tests exercise regional routing, auth and list decoding with synthetic responses. Record a tenant result separately if a future RocketCyber MCP appears. [SOC registrations][rc-reg] · [SOC client][rc-client] · [Client tests][rc-tests] · [RocketCyber customer API][rc-help]

### Datto Networking

| LGP tools | Kaseya tools | Workflow and evidence |
|---|---|---|
| `get_whoami`, `list_devices`, `get_devices_overview`, `get_device`, `get_router`, `get_user_devices`, `get_reseller_overview` | — | LGP only, implemented identity, inventory, MAC, user and reseller lookups. RMM site mappings and monitored network devices are a different scope. |
| `get_device_clients_overview`, `get_device_clients_usage`, `get_device_wan_usage`, `get_device_applications` | — | LGP only, implemented client count/usage, WAN and application visibility by MAC/time window. |
| `test_connection`, `server_info` | — | The first invokes `/whoami`. The second reports binary metadata without contacting DNA. Neither registration proves connectivity. |

**All 13 names belong to a server explicitly marked “not yet validated against live API” in the LGP README.** The client constructs a DNA v1 origin, uses Basic auth with public/secret keys, and expects arrays or a `data` wrapper on list calls. The product help distinguishes general Datto Networking/CloudTrax documentation from DNA-specific endpoints. Confirm the intended tenant, key family and device class before making any operational promise. The shipped gateway examples leave networking disabled. [README inventory][readme-servers] · [Network registrations][net-reg] · [Network client][net-client] · [Datto Networking and DNA API guide][net-help]

### Datto Backup

| LGP tools | Kaseya tools | Workflow and evidence |
|---|---|---|
| `list_customers`, `list_appliances`, `list_assets`, `list_backups`, `list_alerts`, `get_agent_version` | — | LGP only, customer and appliance inventory, protected assets, backup history, typed backup alerts and agent version. Asset tags, time windows and backup-specific alert filters cannot be reconstructed from RMM alerts. |
| `list_endpoint_assets`, `list_spanning_domains`, `list_spanning_domain_users`, `list_entra_domains` | — | LGP only, endpoint protection, Spanning domain/user and Entra backup inventory. IT Glue can document a backup appliance. Its documentation record cannot answer backup-history calls. |
| `test_connection`, `server_info` | — | LGP only, diagnostics. The local surface is read-only. |

The LGP README links a Datto Backup host. The implementation identifies the **Unitrends public API** and defaults to `public-api.backup.net` with OAuth client credentials at `login.backup.net`. It renews the bearer token ahead of expiry. Kaseya's MSP portal auth guide confirms this endpoint family. Treat Unified Continuity's Basic-auth API as a separate integration. [Backup registrations][backup-reg] · [Actual client defaults][backup-client] · [Kaseya public API documentation][backup-help] · [Portal authentication][backup-auth] · [IT Glue backup integration][ig-backup]

### myITprocess

| LGP tools | Kaseya tools | Workflow and evidence |
|---|---|---|
| `list_clients`, `list_users`, `list_reviews`, `list_overdue_reviews`, `list_findings` | — | LGP only, read vCIO clients, reviewer identity, review status and findings. PSA contacts and companies cannot supply review evidence. |
| `list_recommendations`, `get_recommendation_configurations`, `list_initiatives`, `list_meetings` | — | LGP only, read recommendations, linked configurations, planned initiatives and meetings. |
| `test_connection`, `server_info` | — | LGP only, diagnostics. No planning write is registered. |

Although the local quirks document mentions GraphQL, `Client.GetList` uses the reporting **REST** API with `mitp-api-key`. It reads `items`, `page`, `pageSize` and `totalCount`, and the tool returns data and pagination. The vendor's product help also identifies this as a REST API. The code and docs do not establish tenant-level validation of every report. [myITprocess registrations][mitp-reg] · [REST client][mitp-client] · [Earlier description][quirks] · [Vendor reporting API][mitp-help]

## Connecting through lgp-mcp-gateway

**Baseline.** The deployment journal records a gateway process spawning separately deployed backend binaries. A gateway upgrade alone did not deliver an Autotask binary fix. In the gateway source reviewed at `7789e359275917eb91ef7f442938d9a9336630f7`, the `external/lgp-mcp-servers` gitlink points to `5d78696a`. The `795eb82d` backend inventory is newer. Record the installed versions, effective role definitions, prefixes and tool lists before any pilot. The journal proves the historical deployment it describes. The gateway's `Toolset.handler` records `key.AuditActor()` in its own audit event. Verify the running versions, actor binding in Autotask history and extraction of the create ID in the deployed path. [Deployment journal][journal] · [Backend contract][access]

In that gateway revision, `internal/subprocess/manager.go` (`Manager.createClient`, `Manager.Start`) can launch a binary/stdio command with arguments and environment, or open an HTTP backend with configured headers. It initializes one client per configured backend and discovers its tools. `internal/mcphttp/client.go` (`NewClient`) supports Streamable HTTP with JSON/SSE responses and rejects redirects. Connection-level headers suit a dedicated service identity. Interactive login, token renewal, revocation and isolation between product users require an explicit identity design for each route.

| Product | Existing connection | First-party route and operating decision |
|---|---|---|
| Autotask | Go stdio → PSA REST using API username, secret, integration tracking code and the configured API origin. Per-call write headers are generated in the backend. | Prefer isolated gateway HTTP to the account's zoned HTTPS `/mcp` endpoint for a read pilot. Provision the vendor's MCP API user with its tracking-vendor selection. Both the MCPB 0.4.1 relay's `readSettings`/`buildCredentialFetch` and the helper's `Build-HeaderJson` send `UserName` and `Secret`. The setup prose also mentions an integration code. Verify the actual tenant handshake. Static headers can represent that account, but they provide no per-write resource mapping. The alternative Node 20+ bundle declares macOS/Windows and the CLI helper uses Windows DPAPI. [LGP client][at-client] · [Credential guide][credentials] · [AT relay][at-bundle] · [AT helper][at-helper] |
| Datto RMM | Go stdio → regional RMM REST with API key/secret exchanged for a cached bearer token. | Best first trial is gateway stdio launching the Node 22+ bridge, MCPB 0.1.0, which declares Linux support. Supply the platform and dedicated API credentials through its environment. `McpBridge.forwardRequest` sends `x-drmm-api-key`, `x-drmm-api-secret` and `x-request-origin` to the platform-specific HTTPS MCP endpoint, handles JSON/SSE and tracks the session. Direct gateway HTTP is a second candidate after verifying the bridge's headers, protocol negotiation and session handling. The keys guide warns of broad key access while the FAQ describes security-level-dependent 403s. Use the least-privileged test account and prove its effective scope. [LGP client][rmm-client] · [Bridge and manifest][rmm-bundle] · [Key guide][credentials] · [RMM FAQ][catalog] |
| IT Glue | Go stdio → regional IT Glue REST with an API key. Gateway roles govern what each caller can request. | Prefer the regional hosted MCP with the user's OAuth 2.1 identity. Establish one authorized product session per user/tenant, including renewal and revocation. The existing static-header route alone cannot implement that lifecycle. A dedicated `x-api-key` HTTP route can test transport but can grant account-wide API data. MCPB 0.1.0 wraps the key route in Node 20+ on macOS/Windows. Its `buildCredentialFetch` binds requests to the endpoint. The API-key CLI helper uses Windows DPAPI. [LGP client][ig-client] · [Regional OAuth and rollout][setup] · [IT Glue relay][ig-bundle] · [Key helper][ig-helper] |

**Availability and support.** Confirm product subscription, API entitlement, enabled MCP account, target region and network allowlist. The November rollout note marks IT Glue OAuth available in EU/AU, with NA expected during 19–24 November. IT Glue help also conditions access on the Enterprise plan. The vendor FAQ's general statement about available regions cannot establish an individual account's OAuth entitlement. Autotask endpoints vary by zone and RMM by platform. Kaseya names Claude Desktop and Claude Code as supported clients. Gateway use, including a Linux relay, needs an operational owner. The Pre-Release Terms distinguish an internal/private integration from offering a connector-based integration to multiple Kaseya customers. Check that distinction and the applicable agreement before wider deployment. [Setup and support][setup] · [IT Glue product help][ig-help] · [Pre-release terms, sections 2 and 6A][terms]

**Access rules.** The gateway's `config/roles.yaml` maps user roles to servers and tiers. `internal/config/server.go` (`ServerConfig.PrefixedToolName`, `ToolTier`, `ToolAllowed`) and `internal/proxy/toolset.go` (`Toolset.filter`, `allowed`, `handler`) apply them to listing and calling. A missing server grant gives no access. Tier 1 is basic read, tier 2 adds sensitive reads, and tier 3 allows configured writes. For example, `config/servers/itglue.yaml` assigns `itglue_get_password` to tier 2 and `itglue_create_document` to tier 3. The Autotask and RMM templates assign `autotask_create_ticket` and `datto_rmm_create_quick_job` to tier 3. A registered name missing from the higher-tier lists defaults to tier 1 in this revision. The startup warning about write tools does not assign a tier. Inventory every official name in an isolated prefix before publishing it. Treat new or changed remote schemas as pending review.

The gateway's `internal/proxy/dispatch.go` (`checkDispatch`) checks the target tool's tier when a dispatcher names another registered tool. A generic `query-entity` or `update-entity` instead names an entity in its payload. Constrain the adapter to the particular entities, fields, related objects, organizations and verbs allowed to that role, or withhold the entire tool. Apply the same rule to counts, nested relationships, IT Glue flexible fields and RMM activity/stream output. Preserve allowed narrow reads as well as denied sensitive calls. Product scopes and the gateway's policy both apply. Tool annotations and skill confirmation prompts supply no server-side authorization. [LGP access model][access] · [Kaseya credential and approval model][catalog]

**Continuing maintenance.** New reads under a separate prefix need the least translation once authentication and privacy checks pass. A compatibility adapter adds ownership of aliases and argument conversion, result shaping, pagination, errors and schema-change fixtures. Per-user OAuth adds session storage, refresh, revocation and isolation. Autotask attribution and ID extraction require product-side proof on every relevant write path. Rebuilding billing operations via a separately maintained product client would retain significant backend maintenance. The current Go clients also provide timeouts and product-specific retry, rate-limit and circuit-breaker behavior. Compare failure and recovery behavior before retiring any of it. Pin local relay builds by release/checksum, monitor remote schema drift independently and use a harmless read or discovery for health. The RMM bridge can register a configuration-error tool when misconfigured. A running process or a nonempty tool list alone is insufficient. [Shared resilience][resilience] · [LGP HTTP client][http-client] · [RMM bridge][rmm-bundle]

## Pilot sequence and acceptance evidence

The two native reads for Datto RMM patches and Autotask contract lines have a production rollout and portal acceptance procedure on the existing gateway. The sequence below evaluates prospective official server connections. [Native read rollout](kaseya-read-pilot.md#live-rollout-on-the-existing-gateway)

1. **Capture the reference.** In an isolated tenant/gateway, record the binary versions, protocol versions, account/region and authenticated role. Save the full paginated `tools/list` and sanitized call/result fixtures for each currently exposed name. Include unfiltered server discovery, client-visible prefixes, required fields, defaults, annotations, content type, error and empty results. Compare the effective deployment to the 258 registrations and document intentional role exclusions.
2. **Connect one product at a time.** Use a distinct prefix and restricted test grants. Start with the official RMM patch and inventory reads, then IT Glue OAuth/folder reads, then constrained Autotask reads (including contract service lines). Compare both sides on the same stable read fixtures. Classify each official addition before exposing it. Keep one backend responsible for each write.
3. **Prove the policy and writes.** Run the checks below on disposable records with an operator who can inspect product history. Keep identifiers and evidence inside the test environment. Capture sanitized schemas and expected outcomes for later regression checks.
4. **Decide per route.** A complete switch is eligible only after every legacy name passes for existing clients, authorization and audit match, and the maintained adapter has an owner. Under an approved cutover, retain the existing client prefix and credentials required for fallback, route a write to exactly one backend and keep in-flight jobs/cursors reachable. On an identity, schema, access or result regression, restore the original backend routing and grants. Reconcile the outcome of outstanding writes before any retry. Continue partial routing wherever a product gate stays open.

| Later check | Prerequisite and action | Passing evidence and decision on failure |
|---|---|---|
| Connection, identity lifecycle and failure recovery | Set up the intended regional/zone test accounts. Initialize/list/call behind the gateway. Exercise restart, expired or revoked credentials, negotiated protocol, denial, rate limit, timeout and recovery. For IT Glue interleave two users from different organization scopes and revoke one consent. | Expected schemas and one known read return from the correct tenant. Expiry/revocation does not select another user's token or a broader key. Errors and recovery are observable without credential data in logs. Keep that route disabled if any condition fails. |
| Roles and privacy tiers | Use test users with no server grant and with tiers 1, 2 and 3. Compare visible names and results for a permitted basic read, sensitive read and write. Exercise direct, meta-dispatched and generic/entity-scoped calls and a relationship include. Refresh the catalog after a simulated rename/addition. | Intended reads still work. Disallowed names and data stay undiscoverable and calls are refused before product effects. Unclassified additions stay unpublished. Withhold the affected generic route or constrain its fields/verbs until this holds. |
| Existing clients and errors | Replay sanitized, previously valid requests under their old names, including optional fields and boundary IDs. Compare successful, empty, missing, invalid, denied and upstream-error cases plus pagination to exhaustion. Collect `content`, `structuredContent` if present and `isError`. | Input schemas, visible values/types, error category and effects remain usable by the same clients. An unhandled difference requires an adapter fixture or retention of the existing tool. |
| Autotask reads | Use companies spanning several pages, with empty/intermediate and full/final fixtures. Exercise numeric pages and old/new cursors. Include open and closed tickets and a tenant-supported contract services query. | The `summary` contract, filters, IDs and completion conditions match. Unmatched or rejected queries remain distinct from zero results. Keep local search on any difference. |
| Autotask writes and audit | Arrange actor and API resources with the necessary entity and proxy-time permissions. Exercise create, child create, update and permitted deletion directly and through the dispatcher. Include missing resource ID, denied impersonation and untrusted client metadata. Inspect product history, gateway audit and an unchanged client extracting both IDs. | The gateway supplies the trusted actor. Eligible writes are attributed to that person, reads omit impersonation, absent-ID behavior matches the stated API-user policy, and denial never silently loses attribution. Numeric `itemId` and `id` identify the same stored row. Any ambiguity keeps writes local. [Synthetic baseline][at-tests] · [Autotask permissions][at-security] |
| RMM reads and execution | Compare scoped/filtered lists, last page, multiple MAC matches and activity cursor. Use an existing historical job and one disposable approved component job. Exercise preview, confirm, unknown outcome and stdout/stderr. Check omitted UDFs and an already-resolved test alert. | Preview has zero product effects, confirmation produces one intended effect, and the job UID/results/streams and filters satisfy their old calls. Keep historical jobs and any incompatible writes on LGP. |
| IT Glue roles and document lifecycle | Use two organizations, a nested-folder document with HTML sections, a restricted asset and a sample credential record. Compare name/body searches with tenant options recorded. Separately create, rename and edit a draft and inspect visibility, ordering and Activity Logs. | OAuth organization boundaries and LGP privacy tiers both hold, including optional relationship fields. Sections, formatting, IDs and draft state agree with the advertised route. Publishing, deletes and password workflows stay local while the catalog lacks equivalent calls. |
| Networking baseline | With an authorized DNA test tenant, verify identity via `test_connection`/`get_whoami`, then compare `list_devices` and one MAC detail with the portal. Record product family and response shape. Extend to router, client, WAN and application data with known fixtures. | Claim practical confirmation only for the operations exercised. If auth, host, MAC format or decoding differs, keep the server marked unvalidated for that scope and track the result in the existing validation issue. [Validation issue][issue1] |

The six products with no official server enter this sequence only after a dated first-party catalog or endpoint appears. Until then, retain their product-specific tests and baselines. Selective adoption can avoid implementing new reads or fields on the three overlapping products. Retiring an entire local binary would require closing the gaps recorded above.

## Legacy inputs for overlapping workflows

These are the registered LGP input fields for the candidate mappings above. Types are the advertised MCP JSON Schema types. Named defaults are handler defaults. Optional filters without a value are generally omitted from the upstream request. The product matrices describe stricter validation, effective limits and return values. Kaseya publishes selected arguments and examples for these workflows. Record remaining official required fields, types, defaults, error bodies and result schemas from the enabled tenant before treating any mapping as client-compatible.

<details>
<summary>Autotask PSA input fields</summary>

| LGP tools | Required | Optional |
|---|---|---|
| `search_tickets` | — | *string* `searchTerm`, `createdAfter`, `createdBefore`, `lastActivityAfter`<br>*number* `companyID`, `status`, `assignedResourceID`, `page` = 1, `pageSize` = 25 |
| `get_ticket_details` | *number* `ticketID` | — |
| `create_ticket` | *number* `companyID`<br>*string* `title`, `description` | *number* `status`, `priority`, `assignedResourceID`, `assignedResourceRoleID`, `contactID` |
| `update_ticket` | *number* `ticketId` | *string* `title`, `description`, `dueDateTime`<br>*number* `status`, `priority`, `assignedResourceID`, `assignedResourceRoleID`, `contactID` |
| `get_ticket_note` | *number* `ticketId`, `noteId` | — |
| `search_ticket_notes` | *number* `ticketId` | *number* `pageSize` = 25 |
| `create_ticket_note` | *number* `ticketId`<br>*string* `description` | *string* `title`<br>*number* `noteType` = 1, `publish` = 1 |
| `search_companies` | — | *string* `searchTerm`, `nextPageUrl`<br>*boolean* `isActive`<br>*number* `page` = 1, `pageSize` = 25 |
| `create_company` | *string* `companyName`<br>*number* `companyType` | *string* `phone`, `address1`, `city`, `state`, `postalCode`<br>*number* `ownerResourceID`<br>*boolean* `isActive` |
| `update_company` | *number* `id` | *string* `companyName`, `phone`, `address1`, `city`, `state`, `postalCode`<br>*boolean* `isActive` |
| `search_contacts` | — | *string* `searchTerm`<br>*number* `companyID`, `page` = 1, `pageSize` = 25<br>*boolean* `isActive` |
| `create_contact` | *number* `companyID`<br>*string* `firstName`, `lastName` | *string* `emailAddress`, `phone`, `title` |
| `get_opportunity` | *number* `opportunityId` | — |
| `search_opportunities` | — | *number* `companyId`, `status`, `page` = 1, `pageSize` = 25<br>*string* `searchTerm` |
| `create_opportunity` | *string* `title`, `projectedCloseDate`, `startDate`<br>*number* `companyId`, `ownerResourceId`, `status`, `stage` | *number* `probability`, `amount`, `cost`, `totalAmountMonths`, `contactId`, `opportunityCategoryID`<br>*boolean* `useQuoteTotals`<br>*string* `description` |
| `search_contracts` | — | *string* `searchTerm`<br>*number* `companyID`, `status`, `page` = 1, `pageSize` = 25 |
| `search_projects` | — | *string* `searchTerm`<br>*number* `companyID`, `status`, `projectLeadResourceID`, `page` = 1, `pageSize` = 25 |
| `create_project` | *number* `companyID`, `status`, `projectType`<br>*string* `projectName` | *string* `description`, `startDate`, `endDate`<br>*number* `projectLeadResourceID`, `estimatedHours` |
| `search_tasks` | — | *string* `searchTerm`<br>*number* `projectID`, `status`, `assignedResourceID`, `page` = 1, `pageSize` = 25 |
| `create_task` | *number* `projectID`, `status`<br>*string* `title` | *string* `description`, `startDateTime`, `endDateTime`<br>*number* `assignedResourceID`, `estimatedHours`, `taskType` = 1 |
| `get_service_call`, `get_service`, `get_service_bundle` | *number* `id` | — |
| `search_service_calls` | — | *number* `companyID`, `status`, `page` = 1, `pageSize` = 25<br>*string* `startDateTime`, `endDateTime` |
| `create_service_call` | *number* `companyID`, `status`<br>*string* `startDateTime` | *string* `description`, `endDateTime`<br>*number* `duration` |
| `update_service_call` | *number* `id` | *number* `companyID`, `status`, `duration`<br>*string* `description`, `startDateTime`, `endDateTime` |
| `search_time_entries` | — | *number* `ticketID`, `resourceID`, `page` = 1, `pageSize` = 25<br>*string* `dateWorkedStart`, `dateWorkedEnd` |
| `create_time_entry` | *number* `ticketID`, `resourceID`, `hoursWorked`<br>*string* `dateWorked` | *string* `summaryNotes`, `internalNotes`, `startDateTime`, `endDateTime`<br>*number* `roleID`, `billingCodeID` |
| `search_resources`, `search_services`, `search_service_bundles` | — | *string* `searchTerm`<br>*boolean* `isActive`<br>*number* `page` = 1, `pageSize` = 25 |
| `search_configuration_items` | — | *string* `searchTerm`<br>*number* `companyID`, `productID`, `page` = 1, `pageSize` = 25<br>*boolean* `isActive` |
| `get_field_info` | *string* `entityType` | *string* `fieldName` |
| `list_queues`, `list_ticket_statuses`, `list_ticket_priorities` | — | — |

</details>

<details>
<summary>Datto RMM input fields</summary>

| LGP tools | Required | Optional |
|---|---|---|
| `list_sites` | — | *string* `siteName`<br>*number* `page`, `max` |
| `get_site` | *string* `siteUid` | — |
| `list_devices` | — | *string* `hostname`, `siteName`, `deviceType`, `operatingSystem`, `filterId`<br>*number* `page`, `max` |
| `list_site_devices` | *string* `siteUid` | *string* `filterId`<br>*number* `page`, `max` |
| `get_device`, `get_device_audit`, `get_esxi_audit`, `get_printer_audit`, `get_device_software` | *string* `deviceUid` | — |
| `get_device_by_id` | *number* `deviceId` | — |
| `get_device_by_mac`, `get_device_audit_by_mac` | *string* `macAddress` | — |
| `list_default_filters`, `list_custom_filters` | — | — |
| `list_site_filters` | *string* `siteUid` | *number* `page`, `max` |
| `list_open_alerts`, `list_resolved_alerts` | — | *boolean* `muted`<br>*number* `page`, `max` |
| `list_device_open_alerts`, `list_device_resolved_alerts` | *string* `deviceUid` | *boolean* `muted`<br>*number* `page`, `max` |
| `list_site_open_alerts`, `list_site_resolved_alerts` | *string* `siteUid` | *boolean* `muted`<br>*number* `page`, `max` |
| `get_alert`, `resolve_alert` | *string* `alertUid` | — |
| `get_activity_logs` | — | *number* `size`<br>*string* `order`, `from`, `until` |
| `list_components` | — | *number* `page`, `max` |
| `create_quick_job` | *string* `deviceUid`, `jobName`, `componentUid` | — |
| `get_job` | *string* `jobUid` | — |
| `get_job_results`, `get_job_stdout`, `get_job_stderr` | *string* `jobUid`, `deviceUid` | — |
| `set_device_udf` | *string* `deviceUid` | *string* `udf1` through `udf30` |

The quick-job handler also accepts an undeclared `variables` argument. Some list handlers, including `get_device_software`, read `page` and `max` through the pagination helper without advertising them. Include these established handler behaviors in client fixtures. [Device handlers][rmm-devices] · [Audit handlers][rmm-audit] · [List helper][rmm-helpers]

</details>

<details>
<summary>IT Glue input fields</summary>

| LGP tools | Required | Optional |
|---|---|---|
| `search_organizations` | — | *string* `name`, `organization_type_id`, `organization_status_id`, `psa_id`<br>*number* `page_number` = 1, `page_size` = 50 |
| `get_organization`, `get_configuration`, `get_location` | *number* `id` | — |
| `search_configurations` | — | *string* `organization_id`, `name`, `configuration_type_id`, `configuration_status_id`, `serial_number`, `rmm_id`, `psa_id`<br>*number* `page_number` = 1, `page_size` = 50 |
| `search_documents` | *number* `organization_id` | *string* `name`<br>*number* `page_number` = 1, `page_size` = 50 |
| `get_document` | *number* `organization_id`, `id` | — |
| `list_document_sections` | *number* `document_id` | — |
| `get_document_section` | *number* `section_id` | — |
| `create_document` | *number* `organization_id`<br>*string* `name` | *string* `content` |
| `update_document` | *number* `id` | *string* `name` |
| `create_document_section` | *number* `document_id`<br>*string* `section_type` | *string* `content` |
| `update_document_section` | *number* `section_id`<br>*string* `content` | — |
| `search_locations` | — | *string* `organization_id`, `name`, `city`, `region_id`, `country_id`<br>*number* `page_number` = 1, `page_size` = 50 |
| `search_flexible_assets` | *string* `flexible_asset_type_id` | *string* `organization_id`, `name`<br>*number* `page_number` = 1, `page_size` = 50 |
| `list_flexible_asset_types` | — | *string* `organization_id` |

</details>

[access]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/README.md#gateway-actor-and-autotask-impersonation
[actor]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/mcputil/actor.go
[at-bundle]: https://github.com/kaseya/mcp-servers/blob/512c355fc525233e894e013af1ed4756c67bedd3/AUTOTASK/autotask-0.4.1.mcpb
[at-calls]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/tools_service_calls.go
[at-client]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/client.go
[at-companies]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/tools_companies.go
[at-contacts]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/tools_contacts.go
[at-entities]: https://github.com/kaseya/mcp-servers/blob/512c355fc525233e894e013af1ed4756c67bedd3/AUTOTASK/Autotask-Skills-documentation.md#entities
[at-financial]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/tools_financial.go
[at-format]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/format.go
[at-helper]: https://github.com/kaseya/mcp-servers/blob/512c355fc525233e894e013af1ed4756c67bedd3/AUTOTASK/autotask-auth-0.1.0.ps1
[at-meta]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/metatools.go
[at-pages]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/tools_companies_test.go
[at-picklist]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/picklist.go
[at-products]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/tools_products.go
[at-projects]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/tools_projects.go
[at-reg]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/tools.go
[at-security]: https://ww15.autotask.net/help/DeveloperHelp/Content/APIs/REST/General_Topics/REST_Security_Auth.htm
[at-tests]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/impersonation_test.go
[at-tickets]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/tools_tickets.go
[at-time]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/tools_time_billing.go
[at-utility]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/tools_utility.go
[at-zip]: https://github.com/kaseya/mcp-servers/blob/512c355fc525233e894e013af1ed4756c67bedd3/AUTOTASK/autotask-plugin-0.1.0.zip
[backup-auth]: https://helpdesk.kaseya.com/hc/en-gb/articles/7006294343313-How-to-get-a-bearer-token-for-the-MSP-portal-API
[backup-client]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattobackup/client.go
[backup-help]: https://helpdesk.kaseya.com/hc/en-gb/articles/4407519479057-Unitrends-API-Documentation
[backup-reg]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattobackup/tools.go
[baseline]: https://github.com/Logiphys/lgp-mcp-servers/tree/795eb82dbbab30edd6faa5df893899600db7a676
[catalog]: https://github.com/kaseya/mcp-servers/blob/512c355fc525233e894e013af1ed4756c67bedd3/README.md
[changes]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/CHANGELOG.md
[credentials]: https://github.com/kaseya/mcp-servers/blob/512c355fc525233e894e013af1ed4756c67bedd3/GENERATING-API-KEYS.md
[directory]: https://www.kaseya.com/platform/mcp-servers/
[edr-client]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattoedr/client.go
[edr-help]: https://edr.datto.com/help/Content/06-understanding-integrations/integrations-intro.htm
[edr-reg]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattoedr/tools.go
[entry]: https://github.com/Logiphys/lgp-mcp-servers/tree/795eb82dbbab30edd6faa5df893899600db7a676/cmd
[flatten]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/apihelper/jsonapi.go
[http-client]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/apihelper/client.go
[ig-backup]: https://help.itglue.kaseya.com/help/Content/1-admin/other-integrations/integrating-with-datto-backup.html
[ig-bundle]: https://github.com/kaseya/mcp-servers/blob/512c355fc525233e894e013af1ed4756c67bedd3/ITGLUE/itglue-0.1.0.mcpb
[ig-client]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/itglue/client.go
[ig-configs]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/itglue/tools_configurations.go
[ig-contacts]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/itglue/tools_contacts.go
[ig-docs]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/itglue/tools_documents.go
[ig-domains]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/itglue/tools_domains.go
[ig-expire]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/itglue/tools_expirations.go
[ig-flex]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/itglue/tools_flexible_assets.go
[ig-health]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/itglue/tools_health.go
[ig-help]: https://help.itglue.kaseya.com/help/Content/it-glue-mcp-server.html
[ig-helper]: https://github.com/kaseya/mcp-servers/blob/512c355fc525233e894e013af1ed4756c67bedd3/ITGLUE/itglue-auth-0.1.0.ps1
[ig-iface]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/itglue/tools_configuration_interfaces.go
[ig-locations]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/itglue/tools_locations.go
[ig-meta]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/itglue/tools_metadata.go
[ig-orgs]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/itglue/tools_organizations.go
[ig-passwords]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/itglue/tools_passwords.go
[ig-reg]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/itglue/tools.go
[ig-skills]: https://github.com/kaseya/mcp-servers/blob/512c355fc525233e894e013af1ed4756c67bedd3/ITGLUE/ITGlue-Skills-documentation.md
[ig-tests]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/itglue/client_test.go
[ig-zip]: https://github.com/kaseya/mcp-servers/blob/512c355fc525233e894e013af1ed4756c67bedd3/ITGLUE/itglue-plugin-0.2.0.zip
[impersonation]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/autotask/impersonation.go
[info]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/mcputil/serverinfo.go
[issue1]: https://github.com/Logiphys/lgp-mcp-servers/issues/1
[issue3]: https://github.com/Logiphys/lgp-mcp-servers/issues/3
[issue4]: https://github.com/Logiphys/lgp-mcp-servers/issues/4
[journal]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/docs/journal.md
[mitp-client]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/myitprocess/client.go
[mitp-help]: https://help.myitprocess.kaseya.com/help/Content/setup-admin/myitprocess-api.html
[mitp-reg]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/myitprocess/tools.go
[net-client]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattonetwork/client.go
[net-help]: https://networkinghelp.datto.com/help/Content/kb/Networking/Datto%20Networking%20Appliance/DNA-API.htm
[net-reg]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattonetwork/tools.go
[quirks]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/docs/api-quirks.md
[rc-client]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/rocketcyber/client.go
[rc-help]: https://api-doc.rocketcyber.com/
[rc-reg]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/rocketcyber/tools.go
[rc-tests]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/rocketcyber/client_test.go
[readme-servers]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/README.md#servers
[resilience]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/resilience/middleware.go
[rmm-account]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattormm/tools_account.go
[rmm-activity]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattormm/tools_activity.go
[rmm-alerts]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattormm/tools_alerts.go
[rmm-audit]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattormm/tools_audit.go
[rmm-bundle]: https://github.com/kaseya/mcp-servers/blob/512c355fc525233e894e013af1ed4756c67bedd3/RMM/datto-rmm-0.1.0.mcpb
[rmm-client]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattormm/client.go
[rmm-devices]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattormm/tools_devices.go
[rmm-filters]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattormm/tools_filters.go
[rmm-helpers]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattormm/tools_helpers.go
[rmm-jobs]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattormm/tools_jobs.go
[rmm-reg]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattormm/tools.go
[rmm-sites]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattormm/tools_sites.go
[rmm-system]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattormm/tools_system.go
[rmm-tests]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattormm/client_test.go
[rmm-vars]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattormm/tools_variables.go
[rmm-zip]: https://github.com/kaseya/mcp-servers/blob/512c355fc525233e894e013af1ed4756c67bedd3/RMM/Datto-RMM-plugin-0.1.0.zip
[setup]: https://github.com/kaseya/mcp-servers/blob/512c355fc525233e894e013af1ed4756c67bedd3/README.md#getting-started
[terms]: https://github.com/kaseya/mcp-servers/blob/512c355fc525233e894e013af1ed4756c67bedd3/GOVERNANCE/MCP-PRERELEASE-TERMS.md
[uc-client]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattouc/client.go
[uc-help]: https://continuity.datto.com/help/Content/kb/siris-alto-nas/360016394931.html
[uc-reg]: https://github.com/Logiphys/lgp-mcp-servers/blob/795eb82dbbab30edd6faa5df893899600db7a676/pkg/dattouc/tools.go
