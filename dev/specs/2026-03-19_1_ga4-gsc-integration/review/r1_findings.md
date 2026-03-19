無法寫入 [r1_findings.md](/Users/mac/dex_2026/gwx%202603/dev/specs/2026-03-19_1_ga4-gsc-integration/review/r1_findings.md)。目前環境是唯讀 sandbox，兩次 `apply_patch` 都被拒絕。

以下是應寫入的內容：

```md
# Code Review Findings

Decision: fix_required

Finding ID: CR-P1-001
Severity: P1
Category: 架構合規 / Spec 一致性
File: internal/api/analytics.go; internal/auth/scopes.go; internal/api/ratelimiter.go
Line: 70-76, 147-153, 219-275; 32-55; 18-27
Description: Analytics 功能在三層之間使用了彼此不一致的 service key。CMD/Auth/Scope/RateLimiter 只註冊了 `analytics`，但 API layer 實際呼叫 `ClientOptions` / `WaitRate` 時使用 `analyticsdata` 與 `analyticsadmin`。這破壞了既有「同一 service key 貫穿 auth/config/API」的架構契約，導致新加的 scope 與限流設定無法可靠套用到真正的 API 呼叫路徑。
Evidence: `internal/auth/scopes.go` 只新增 `analytics` scope；`internal/api/ratelimiter.go` 也只新增 `analytics` rate。相對地，`internal/api/analytics.go` 的 `RunReport` / `RunRealtimeReport` 呼叫 `WaitRate(ctx, "analyticsdata")` 與 `ClientOptions(ctx, "analyticsdata")`，`ListProperties` / `ListAudiences` 則改用 `analyticsadmin`。目前沒有任何對應的 `analyticsdata` / `analyticsadmin` 註冊。
Recommendation: 統一使用單一 `analytics` service key，把 Data/Admin API 的差異留在 API 實作內部處理；或若框架確實要求按 API 分拆，則必須同步把 `analyticsdata` / `analyticsadmin` 完整註冊到 scope、read-only scope、rate limiter 與 auth 流程。
Status: open

Finding ID: CR-P1-002
Severity: P1
Category: 錯誤處理 / Spec 一致性 / 輸入驗證
File: internal/api/searchconsole.go; internal/cmd/searchconsole.go; internal/mcp/tools_searchconsole.go
Line: 86-111; 23-30, 67-74; 22-29, 134-141
Description: Search Console query 路徑沒有落實 spec 的輸入契約。S1 spec 明確要求 `EndDate` 為 `YYYY-MM-DD`，且 `Limit` 最大值為 `25000`，但目前 CLI 與 MCP 都允許缺少 `end_date`，service 也沒有對超過上限的 `limit` 做任何拒絕或裁切。
Evidence: Spec 中 `SearchQueryRequest` 定義 `EndDate` 為必填欄位，`Limit` 註明 `max 25000`。但 `internal/cmd/searchconsole.go` 把 `EndDate` 宣告成 `default:""`，送進 service 時也原樣傳遞；`internal/mcp/tools_searchconsole.go` 的 `searchconsole_query` 只把 `start_date` 設為 required。`internal/api/searchconsole.go` 僅在 `limit <= 0` 時改成 100，對 `EndDate == ""` 與 `limit > 25000` 都沒有驗證。
Recommendation: 在 CLI/MCP 層與 API service 層都補齊驗證。若產品決策允許省略 `end_date`，就把 spec 改成明確預設 `today`；否則應直接在輸入層拒絕。`limit` 也應在進 API 前強制限制到 `1..25000`。
Status: open

Finding ID: CR-P1-003
Severity: P1
Category: 錯誤處理 / 架構一致性
File: internal/mcp/tools_searchconsole.go; internal/cmd/searchconsole.go
Line: 102-149, 169-204; 41-51, 163-173, 214-220
Description: MCP Search Console handlers 對缺少 site 的情況沒有比照 CLI 做顯式驗證，導致 agent 路徑與 CLI 路徑出現不一致行為。當 args 未提供 `site` 且 config 也沒有 `searchconsole.default-site` 時，MCP 仍會把空字串送進 API，而不是回傳可操作的使用者錯誤。
Evidence: `resolveSearchConsoleSite()` 在 config 沒值時直接 `return val, nil`；`searchconsoleQuery` / `searchconsoleSitemaps` / `searchconsoleIndexStatus` 取得空字串後沒有再次檢查，直接建立 request。相比之下，CLI 的 `SearchConsoleQueryCmd`、`SearchConsoleSitemapsCmd`、`SearchConsoleIndexStatusCmd` 都在 fallback 後執行 `if site == "" { ... ErrExit(...) }`。
Recommendation: 把「arg 與 config 都缺失時回傳明確錯誤」抽成共用 helper，讓 CLI 與 MCP 走同一套驗證邏輯，避免 agent 呼叫落到遠端 API 才得到晦澀錯誤。
Status: open

Finding ID: CR-P1-004
Severity: P1
Category: 測試品質 / 錯誤處理
File: internal/api/analytics.go; internal/api/searchconsole.go; internal/cmd/analytics.go; internal/cmd/searchconsole.go; internal/mcp/tools_analytics.go; internal/mcp/tools_searchconsole.go; internal/config/preferences_test.go
Line: 1; 1; 1; 1; 1; 1; 1-78
Description: 這批變更新增了兩個 service、兩組 CLI、兩組 MCP tools 與新的 config 命令，但 diff 內唯一新增的測試只覆蓋 `preferences.go`。核心 error path 完全沒有自動化保護，和 Review Standards 要求的「happy + error path」不符。
Evidence: `internal/config/preferences_test.go` 只驗證 preferences 檔案讀寫；沒有任何對 `analytics` / `searchconsole` CLI、MCP handler、預設 property/site fallback、無效日期、超額 limit、或 API error wrapping 的測試變更。
Recommendation: 至少補上 table-driven tests 覆蓋以下情境：default property/site 缺失、Search Console query 缺少 `end_date`、`limit > 25000`、API 建立失敗時的錯誤包裝、MCP handler 在缺少必要 config 時回傳明確錯誤。
Status: open
```

如果你開可寫環境，我可以直接把這份內容落到 [r1_findings.md](/Users/mac/dex_2026/gwx%202603/dev/specs/2026-03-19_1_ga4-gsc-integration/review/r1_findings.md)。