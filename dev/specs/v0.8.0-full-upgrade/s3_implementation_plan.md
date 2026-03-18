# S3 Implementation Plan: gwx v0.8.0 Full Upgrade

> **階段**: S3 執行計畫
> **建立時間**: 2026-03-18 18:15
> **Agents**: go-expert (frontend-developer 不適用，純 Go 後端)

---

## 1. 概述

### 1.1 功能目標
gwx v0.8.0 全面升級：MCP 工具 39→59、測試覆蓋、batch 操作、快取層、結構化日誌。

### 1.2 實作範圍
- **範圍內**: FA-1~FA-5 + version bump（見 s1_dev_spec.md）
- **範圍外**: Web UI、磁碟快取、新 Google API 服務

### 1.3 關聯文件
| 文件 | 路徑 | 狀態 |
|------|------|------|
| Brief Spec | `./s0_brief_spec.md` | ✅ |
| Dev Spec | `./s1_dev_spec.md` | ✅ |
| Review Report | `./review/s2_review_report.md` | ✅ |
| Implementation Plan | `./s3_implementation_plan.md` | 📝 當前 |

---

## 2. 實作任務清單

### 2.1 任務總覽

| # | 任務 | 類型 | Agent | 依賴 | 複雜度 | FA | TDD | 狀態 |
|---|------|------|-------|------|--------|-----|-----|------|
| 1 | slog 封裝 + 替換 | 後端 | `frontend-developer` | - | S | FA-5 | ✅ | ⬜ |
| 2 | LRU+TTL 快取實作 | 後端 | `frontend-developer` | - | M | FA-4 | ✅ | ⬜ |
| 3 | 快取注入 API Service | 後端 | `frontend-developer` | #2 | M | FA-4 | null | ⬜ |
| 4 | 18 個 MCP 工具定義 + handler | 後端 | `frontend-developer` | - | L | FA-1 | ✅ | ⬜ |
| 5 | drive_batch_upload 實作 | 後端 | `frontend-developer` | - | M | FA-3 | ✅ | ⬜ |
| 6 | sheets_batch_append 實作 | 後端 | `frontend-developer` | - | M | FA-3 | ✅ | ⬜ |
| 7 | batch MCP 工具定義 + handler | 後端 | `frontend-developer` | #5, #6 | S | FA-3 | null | ⬜ |
| 8 | API 層 unit test | 測試 | `test-engineer` | #2, #3, #5, #6 | M | FA-2 | ✅ | ⬜ |
| 9 | MCP 層 unit test | 測試 | `test-engineer` | #4, #7 | M | FA-2 | ✅ | ⬜ |
| 10 | Version bump + 整合驗證 | 後端 | `frontend-developer` | #1~#9 | S | VER | null | ⬜ |

> Agent 說明：專案無 `go-expert` agent 可用，使用 `frontend-developer`（支援 Read/Grep/Glob/Edit/Write/Bash）執行 Go 程式碼實作。

### 2.2 波次規劃

```
Wave 1（基礎設施 + 獨立功能，可並行）
├── T1: slog 封裝 + 替換 (FA-5, S)
├── T2: LRU+TTL 快取實作 (FA-4, M)
├── T4: 18 個 MCP 工具 (FA-1, L)
├── T5: drive_batch_upload (FA-3, M)
└── T6: sheets_batch_append (FA-3, M)

Wave 2（依賴 Wave 1）
├── T3: 快取注入 API Service (FA-4, M) ← depends T2
└── T7: batch MCP 工具 (FA-3, S) ← depends T5, T6

Wave 3（測試，依賴 Wave 1+2）
├── T8: API 層 unit test (FA-2, M) ← depends T2, T3, T5, T6
└── T9: MCP 層 unit test (FA-2, M) ← depends T4, T7

Wave 4（收尾）
└── T10: Version bump + 整合驗證 (VER, S) ← depends all
```

### 2.3 並行規則

| Wave | 可並行任務 | 說明 |
|------|----------|------|
| Wave 1 | T1, T2, T4, T5, T6 | 5 個任務無依賴，全部可並行 |
| Wave 2 | T3, T7 | 2 個任務無互相依賴，可並行 |
| Wave 3 | T8, T9 | 2 個任務無互相依賴，可並行 |
| Wave 4 | T10 | 單一任務 |

---

## 3. 任務詳情

### Task #1: slog 封裝 + 替換

- **FA**: FA-5
- **Agent**: frontend-developer
- **依賴**: 無
- **複雜度**: S
- **受影響檔案**:
  - 新增: `internal/log/logger.go`
  - 修改: `internal/cmd/mcpserver.go`, `internal/cmd/root.go`, `internal/output/formatter.go`, `internal/mcp/protocol.go`
- **source_ref**: s1_dev_spec.md §4.8
- **DoD**:
  - [ ] `internal/log/logger.go` 建立，SetupMCPLogger() + SetupCLILogger()
  - [ ] 6 處 fmt.Fprintf(os.Stderr) 替換為 slog（含 protocol.go 2 處）
  - [ ] MCP 模式 slog handler 輸出到 os.Stderr（不碰 stdout）
  - [ ] CLI 模式根據 isatty 選擇 Text/JSON handler
  - [ ] oauth.go 2 處保持不動
  - [ ] `go build ./...` 通過
- **tdd_plan**:
  - test_file: `internal/log/logger_test.go`
  - test_cases: [TestSetupMCPLogger_WritesToStderr, TestSetupCLILogger_TTY_TextHandler, TestSetupCLILogger_Pipe_JSONHandler]
  - test_command: `go test ./internal/log/ -v`

### Task #2: LRU+TTL 快取實作

- **FA**: FA-4
- **Agent**: frontend-developer
- **依賴**: 無
- **複雜度**: M
- **受影響檔案**:
  - 新增: `internal/api/cache.go`, `internal/api/cache_test.go`
- **source_ref**: s1_dev_spec.md §4.7
- **DoD**:
  - [ ] Cache struct 實作完成（sync.Mutex + container/list + map）
  - [ ] Get/Set/Invalidate/InvalidatePrefix/Len 全部 thread-safe
  - [ ] Get 正確處理 TTL 過期（返回 miss）
  - [ ] LRU 淘汰邏輯正確
  - [ ] InvalidatePrefix 正確清除匹配 key
  - [ ] cache_test.go 覆蓋所有行為
- **tdd_plan**:
  - test_file: `internal/api/cache_test.go`
  - test_cases: [TestCache_SetGet, TestCache_TTLExpiry, TestCache_LRUEviction, TestCache_InvalidatePrefix, TestCache_ConcurrentAccess, TestCache_MaxEntries]
  - test_command: `go test ./internal/api/ -run TestCache -v`

### Task #3: 快取注入 API Service

- **FA**: FA-4
- **Agent**: frontend-developer
- **依賴**: #2
- **複雜度**: M
- **受影響檔案**:
  - 修改: `internal/api/client.go`, `internal/api/drive.go`, `internal/api/sheets.go`, `internal/api/contacts.go`, `internal/cmd/root.go`
- **source_ref**: s1_dev_spec.md §4.7
- **DoD**:
  - [ ] Client struct 加 cache *Cache + NoCache bool
  - [ ] NewClient 初始化 cache（MaxEntries=256, DefaultTTL=5min）
  - [ ] --no-cache flag 加入 CLI struct
  - [ ] drive: ListFiles, SearchFiles 加快取（TTL 5min）
  - [ ] sheets: ReadRange, DescribeSheet, GetInfo 加快取（TTL 10min）
  - [ ] contacts: SearchContacts, ListContacts 加快取（TTL 15min）
  - [ ] calendar: Agenda, ListEvents 加快取（TTL 2min）
  - [ ] 寫入方法加 InvalidatePrefix（API Service 層）
  - [ ] NoCache=true 時跳過快取
  - [ ] `go build ./...` 通過
- **tdd_plan**: null
- **skip_justification**: 快取注入是在各 service method 內加 3-5 行邏輯，無獨立可測單元。快取行為由 T2 的 cache_test.go 和 T8 的整合測試覆蓋。

### Task #4: 18 個 MCP 工具定義 + handler

- **FA**: FA-1
- **Agent**: frontend-developer
- **依賴**: 無
- **複雜度**: L
- **受影響檔案**:
  - 新增: `internal/mcp/tools_new.go`
  - 修改: `internal/mcp/tools.go` (ListTools 合併 + CallTool routing), `internal/api/drive.go` (DownloadFile Fields 加 size)
- **source_ref**: s1_dev_spec.md §4.4
- **DoD**:
  - [ ] 18 個工具 InputSchema 定義完成，參數名/類型/必填與 spec §4.4 一致
  - [ ] 18 個 handler 方法呼叫對應 API Service 方法
  - [ ] gmail_reply: 支援 reply_all
  - [ ] drive_download: DownloadFile Fields 加 size，100MB pre-check
  - [ ] docs_template: vars JSON string → map[string]string
  - [ ] docs_from_sheet: headers + rows JSON string → 對應類型
  - [ ] calendar_list vs calendar_agenda: 差異備註
  - [ ] CallTool 正確路由所有 18 個新工具
  - [ ] MCP tools/list 回傳 59+ 工具
  - [ ] `go build ./...` 通過
- **tdd_plan**:
  - test_file: `internal/mcp/tools_new_test.go`
  - test_cases: [TestNewTools_Count, TestNewTools_AllHaveDescription, TestNewTools_RequiredFields]
  - test_command: `go test ./internal/mcp/ -run TestNewTools -v`

### Task #5: drive_batch_upload 實作

- **FA**: FA-3
- **Agent**: frontend-developer
- **依賴**: 無
- **複雜度**: M
- **受影響檔案**:
  - 新增: `internal/api/drive_batch.go`
- **source_ref**: s1_dev_spec.md §4.5, §4.6
- **DoD**:
  - [ ] BatchUploadFiles() 實作完成
  - [ ] semaphore 控制並行度（上限 5）
  - [ ] 部分失敗不中止，回傳 succeeded + failed
  - [ ] BatchUploadResult struct 符合 spec §4.6 格式
  - [ ] `go build ./...` 通過
- **tdd_plan**:
  - test_file: `internal/api/drive_batch_test.go`
  - test_cases: [TestBatchUpload_AllSuccess, TestBatchUpload_PartialFailure, TestBatchUpload_ConcurrencyLimit]
  - test_command: `go test ./internal/api/ -run TestBatchUpload -v`

### Task #6: sheets_batch_append 實作

- **FA**: FA-3
- **Agent**: frontend-developer
- **依賴**: 無
- **複雜度**: M
- **受影響檔案**:
  - 新增: `internal/api/sheets_batch.go`
- **source_ref**: s1_dev_spec.md §4.5, §4.6
- **DoD**:
  - [ ] BatchAppendValues() 實作完成
  - [ ] BatchAppendEntry struct: {Range, Values}
  - [ ] semaphore 控制並行度（上限 5）
  - [ ] 部分失敗回傳 succeeded + failed
  - [ ] BatchAppendResult struct 符合 spec §4.6 格式
  - [ ] `go build ./...` 通過
- **tdd_plan**:
  - test_file: `internal/api/sheets_batch_test.go`
  - test_cases: [TestBatchAppend_AllSuccess, TestBatchAppend_PartialFailure, TestBatchAppend_ConcurrencyLimit]
  - test_command: `go test ./internal/api/ -run TestBatchAppend -v`

### Task #7: batch MCP 工具定義 + handler

- **FA**: FA-3
- **Agent**: frontend-developer
- **依賴**: #5, #6
- **複雜度**: S
- **受影響檔案**:
  - 新增: `internal/mcp/tools_batch.go`
  - 修改: `internal/mcp/tools.go` 或 `tools_extended.go`（routing）
- **source_ref**: s1_dev_spec.md §4.5
- **DoD**:
  - [ ] drive_batch_upload + sheets_batch_append InputSchema 定義
  - [ ] handler 解析 paths（逗號分隔）和 entries（JSON 陣列）
  - [ ] concurrency cap 為 5
  - [ ] CallTool 正確路由
  - [ ] `go build ./...` 通過
- **tdd_plan**: null
- **skip_justification**: batch handler 是 thin wrapper（解析參數 + 呼叫 API 方法），主邏輯已在 T5/T6 覆蓋。handler routing 由 T9 MCP 層測試覆蓋。

### Task #8: API 層 unit test

- **FA**: FA-2
- **Agent**: test-engineer
- **依賴**: #2, #3, #5, #6
- **複雜度**: M
- **受影響檔案**:
  - 新增（若 T2/T5/T6 未含）: `internal/api/drive_batch_test.go`, `internal/api/sheets_batch_test.go`
  - 驗證: `internal/api/cache_test.go` 已在 T2 建立
- **source_ref**: s1_dev_spec.md §5.2 Task #8
- **DoD**:
  - [ ] cache_test.go 通過（T2 已建立）
  - [ ] drive_batch_test.go 覆蓋 batch upload 正常 + 部分失敗
  - [ ] sheets_batch_test.go 覆蓋 batch append 正常 + 部分失敗
  - [ ] `go test ./internal/api/... -count=1 -v` 全通過
- **tdd_plan**:
  - test_file: `internal/api/drive_batch_test.go`, `internal/api/sheets_batch_test.go`
  - test_cases: [TestBatchUpload_AllSuccess, TestBatchUpload_PartialFailure, TestBatchAppend_AllSuccess, TestBatchAppend_PartialFailure]
  - test_command: `go test ./internal/api/... -count=1 -v`

### Task #9: MCP 層 unit test

- **FA**: FA-2
- **Agent**: test-engineer
- **依賴**: #4, #7
- **複雜度**: M
- **受影響檔案**:
  - 新增: `internal/mcp/tools_test.go`
- **source_ref**: s1_dev_spec.md §5.2 Task #9
- **DoD**:
  - [ ] tools_test.go 建立
  - [ ] 測試 ListTools 回傳數量 >= 59
  - [ ] 測試 CallTool routing：已知工具分派正確、未知工具回傳 error
  - [ ] 測試 3 個代表性工具參數解析（gmail_reply, drive_batch_upload, sheets_create）
  - [ ] `go test ./internal/mcp/... -count=1 -v` 全通過
- **tdd_plan**:
  - test_file: `internal/mcp/tools_test.go`
  - test_cases: [TestListTools_Count, TestCallTool_KnownTool, TestCallTool_UnknownTool, TestGmailReply_ParseArgs, TestDriveBatchUpload_ParseArgs, TestSheetsCreate_ParseArgs]
  - test_command: `go test ./internal/mcp/... -count=1 -v`

### Task #10: Version bump + 整合驗證

- **FA**: VER
- **Agent**: frontend-developer
- **依賴**: #1~#9
- **複雜度**: S
- **受影響檔案**:
  - 修改: `internal/cmd/version.go`, `internal/cmd/cli_test.go`, `internal/mcp/protocol.go`
- **source_ref**: s1_dev_spec.md §5.2 Task #10
- **DoD**:
  - [ ] version.go: `const version = "0.8.0"`
  - [ ] protocol.go: ServerInfo Version `"0.8.0"`
  - [ ] cli_test.go: version 斷言更新
  - [ ] `go test ./...` 全部通過
  - [ ] `go build ./...` 通過
- **tdd_plan**: null
- **skip_justification**: version bump 是純字串替換，由既有 cli_test.go 的 version 斷言覆蓋。

---

## 4. 風險與緩解

| 風險 | 影響 | 緩解 |
|------|------|------|
| Wave 1 T4 (L) 耗時長，阻塞 Wave 3 T9 | 中 | T4 可拆分：先定義工具，後寫 handler。但 spec 設計為單一任務，保持簡單。 |
| T3 快取注入散射修改多檔 | 中 | 統一 cache.Get/Set/InvalidatePrefix 介面，每個 method 只加 3-5 行 |
| T5/T6 batch mock 困難 | 低 | 可 mock Client 的 UploadFile/AppendValues 方法，驗證 fan-out 邏輯 |

---

## 5. 驗收檢查表

| # | 驗收標準 | 覆蓋任務 | 驗證方式 |
|---|---------|---------|---------|
| AC-1 | MCP tools/list >= 59 | T4, T7 | T9 測試 |
| AC-2 | go test ./... all pass | T8, T9, T10 | CI |
| AC-3 | batch partial failure | T5, T6 | T8 測試 |
| AC-4 | cache hit avoids API | T2, T3 | T2 cache_test |
| AC-5 | slog only stderr | T1 | T1 logger_test |
| AC-6 | gwx version 0.8.0 | T10 | cli_test |
