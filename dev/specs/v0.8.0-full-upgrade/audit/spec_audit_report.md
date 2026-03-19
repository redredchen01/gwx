# Spec 深度審計報告：gwx v0.8.0-full-upgrade

> **審計 ID**: SA-2026-03-18T22-00-00
> **審計日期**: 2026-03-18
> **Spec 路徑**: `dev/specs/v0.8.0-full-upgrade/`
> **模式**: 獨立模式（standalone）
> **Engine 狀態**: completed（4/4 agents，D1/D3 跳過）

---

## 1. 審計摘要

| 項目 | 數值 |
|------|------|
| 審計維度 | 4/6（D1 Frontend、D3 Database 不適用） |
| 總審計項 | 63 |
| P0 | 0 |
| P1 | 6 |
| P2 | 5 |
| Info | 4 |

---

## 2. 六維度覆蓋矩陣

| 維度 | Total | Passed | Partial | Failed | 覆蓋率 |
|------|-------|--------|---------|--------|--------|
| D1 Frontend | — | — | — | — | N/A（純 CLI/MCP） |
| **D2 Backend** | 32 | 28 | 0 | **4** | 87.5% |
| D3 Database | — | — | — | — | N/A（無 DB） |
| **D4 User Flow** | 3 | 3 | 0 | 0 | 100% |
| **D5 Business Logic** | 10 | 6 | 3 | **1** | 60% |
| **D6 Test Coverage** | 18 | 8 | 7 | **3** | 44% |

---

## 3. D2 Backend 審計明細

### FA-1: MCP 工具補齊（18 個新工具）— 全部 PASSED

| ID | 工具 | 狀態 | 證據 |
|----|------|------|------|
| SA-D2-001 | gmail_reply | ✅ | tools_new.go:17, gmail.go:306 |
| SA-D2-002 | calendar_list | ✅ | tools_new.go:32, calendar.go:49 |
| SA-D2-003 | calendar_update | ✅ | tools_new.go:45, calendar.go:217 |
| SA-D2-004 | contacts_list | ✅ | tools_new.go:63, contacts.go:36 |
| SA-D2-005 | contacts_get | ✅ | tools_new.go:73, contacts.go:133 |
| SA-D2-006 | drive_download | ✅ | tools_new.go:85, drive.go:193+310 (100MB pre-check) |
| SA-D2-007 | sheets_update | ✅ | tools_new.go:98, sheets.go:108 |
| SA-D2-008 | sheets_import | ✅ | tools_new.go:111, sheets_ops.go:374 |
| SA-D2-009 | sheets_create | ✅ | tools_new.go:125, sheets.go:144 |
| SA-D2-010 | tasks_lists | ✅ | tools_new.go:137, tasks.go:40 |
| SA-D2-011 | tasks_complete | ✅ | tools_new.go:144, tasks.go:139 |
| SA-D2-012 | tasks_delete | ✅ | tools_new.go:156, tasks.go:174 |
| SA-D2-013 | docs_template | ✅ | tools_new.go:169, docs.go:284 |
| SA-D2-014 | docs_from_sheet | ✅ | tools_new.go:182, docs.go:253 |
| SA-D2-015 | docs_export | ✅ | tools_new.go:195, docs.go:307 |
| SA-D2-016 | chat_spaces | ✅ | tools_new.go:209, chat.go:37 |
| SA-D2-017 | chat_send | ✅ | tools_new.go:219, chat.go:75 |
| SA-D2-018 | chat_messages | ✅ | tools_new.go:231, chat.go:106 |

### FA-3: Batch 操作 — SPEC 偏移（4 項 FAILED）

| ID | Spec 要求 | 實際實作 | 狀態 |
|----|----------|---------|------|
| SA-D2-019 | drive_batch_meta | drive_batch_upload | ❌ |
| SA-D2-020 | drive_batch_share | (不存在) | ❌ |
| SA-D2-021 | sheets_batch_read | sheets_batch_append | ❌ |
| SA-D2-022 | sheets_batch_update | (不存在) | ❌ |

> **判定**：實際 batch 工具為 `drive_batch_upload` + `sheets_batch_append`，與 Spec 定義的 4 個工具名稱和功能語義完全不同。這是 Spec 與實作的偏移，建議更新 Spec。

### FA-4: 快取層 — 全部 PASSED

| ID | 項目 | 狀態 |
|----|------|------|
| SA-D2-023 | Cache struct (LRU+TTL) | ✅ |
| SA-D2-024 | Client.cache 欄位 | ✅ |
| SA-D2-025 | Drive reads 快取 | ✅ |
| SA-D2-026 | Sheets reads 快取 | ✅ |
| SA-D2-027 | 寫入 InvalidatePrefix | ✅ |
| SA-D2-028 | --no-cache flag | ✅ |

### FA-5: 結構化日誌 — 全部 PASSED

| ID | 項目 | 狀態 |
|----|------|------|
| SA-D2-029 | internal/log/logger.go | ✅ |
| SA-D2-030 | SetupMCPLogger + SetupCLILogger | ✅ |
| SA-D2-031 | fmt.Fprintf → slog 替換 | ✅ (onboard.go 17 處保留，非 MCP path) |
| SA-D2-032 | MCP slog 只寫 stderr | ✅ |

---

## 4. D4 User Flow 追蹤明細

### FLOW1: MCP gmail_reply 呼叫 — ✅ PASSED

| Step | 節點 | 狀態 | 證據 |
|------|------|------|------|
| 1 | stdin JSON-RPC → Server.Run() | ✅ | protocol.go:118-139 |
| 2 | handleRequest → tools/call | ✅ | protocol.go:162-177 |
| 3 | CallTool → CallNewTool → gmailReply | ✅ | tools.go:362-376, tools_new.go |
| 4 | gmailReply → svc.ReplyMessage | ✅ | tools_new.go:310-323, gmail.go:306 |
| 5 | API → Google Gmail | ✅ | gmail.go:229-264 |
| 6 | ToolResult → sendResult → stdout | ✅ | protocol.go:187-201 |

### FLOW2: drive_batch_upload — ✅ PASSED

| Step | 節點 | 狀態 | 證據 |
|------|------|------|------|
| 1 | MCP → CallBatchTool | ✅ | tools.go:362-376, tools_batch.go:45 |
| 2 | 解析 paths/folder/concurrency | ✅ | tools_batch.go:60-77 |
| 3 | BatchUploadFiles | ✅ | drive_batch.go:57-126 |
| 4 | goroutine fan-out + semaphore | ✅ | drive_batch.go:60-109 |
| 5 | succeeded + failed 收集 | ✅ | drive_batch.go:111-125 |
| 6 | ToolResult 回傳 | ✅ | tools_batch.go:76 |

### FLOW3: 快取讀取（drive_list）— ✅ PASSED

| Step | 節點 | 狀態 | 證據 |
|------|------|------|------|
| 1 | MCP → driveList | ✅ | tools.go:330 |
| 2 | DriveService.ListFiles | ✅ | tools.go:455-462 |
| 3 | cache.Get → hit → return | ✅ | drive.go:41-46 |
| 4 | miss → Google API → cache.Set | ✅ | drive.go:48-93 |
| 5 | upload → InvalidatePrefix("drive:") | ✅ | drive.go:186-188 |

---

## 5. D5 Business Logic 審計明細

| ID | 項目 | 狀態 | 備註 |
|----|------|------|------|
| SA-D5-001 | E1 batch partial failure | ✅ | succeeded + failed 清單正確 |
| SA-D5-002 | E2 cache graceful degradation | ❌ | 未實作。TTL 到期直接刪除，無 stale-read 路徑 |
| SA-D5-003 | E3 drive_download 100MB | ✅ | CheckDownloadSize pre-check 正確 |
| SA-D5-004 | E4 rate limit retry + CB | ✅ | transport.go + circuitbreaker.go |
| SA-D5-005 | E5 gmail_reply not-found | ⚠️ | 錯誤有傳播但訊息不夠明確 |
| SA-D5-006 | E6 sheets_batch partial | ✅ | 結構對稱正確 |
| SA-D5-007 | 100MB 約束 (MCP) | ✅ | tools_new.go:408-411 |
| SA-D5-008 | 安全層級一致 | ✅ | CAUTION 標記對應 Tier 3 |
| SA-D5-009 | --no-cache flag | ✅ | 三層串通完整 |
| SA-D5-010 | 100MB 約束 (CLI) | ⚠️ | CLI 路徑未確認 |

---

## 6. D6 Test Coverage 審計明細

| ID | Spec 項目 | Test 覆蓋 | 狀態 |
|----|----------|----------|------|
| SA-D6-001 | AC1: MCP ≥ 59 tools | TestListTools_Count | ✅ |
| SA-D6-002 | AC2: go test all pass | CI 層級 | ⚠️ |
| SA-D6-003 | AC3: batch partial failure | TestBatchAppend_PartialFailure | ✅ |
| SA-D6-004 | AC4: cache avoids API | Cache unit test only | ⚠️ |
| SA-D6-005 | AC5: slog stderr only | TestSetupMCPLogger_WritesToStderr | ✅ |
| SA-D6-006 | AC6: version 0.8.0 | TestCLI_Version | ✅ |
| SA-D6-007 | E1: batch partial | TestBatchAppend_PartialFailure | ✅ |
| SA-D6-008 | E3: 100MB limit | **無任何測試** | ❌ |
| SA-D6-009 | E5: reply not-found | **無任何測試** | ❌ |
| SA-D6-010 | SC1: MCP ≥ 55 | TestListTools_Count | ✅ |
| SA-D6-011 | SC2: 新工具可呼叫 | Schema test only | ⚠️ |
| SA-D6-012 | SC3: go test pass | CI 層級 | ⚠️ |
| SA-D6-013 | SC4: 新測試檔 ≥ 5 | 23 個 _test.go | ✅ |
| SA-D6-014 | SC5: drive_batch multi | 結構測試，無執行路徑 | ⚠️ |
| SA-D6-015 | SC6: sheets_batch multi | TestBatchAppend_AllSuccess | ✅ |
| SA-D6-016 | SC7: cache hit | Cache unit test only | ⚠️ |
| SA-D6-017 | SC8: slog 全覆蓋 | Logger test only | ⚠️ |
| SA-D6-018 | SC9: version 0.8.0 | TestCLI_Version | ✅ |

---

## 7. 交叉驗證結果

### 7.1 Backend x Spec：API 契約
- 18 個新 MCP 工具 InputSchema ✅ 全部一致
- Batch 工具 ⚠️ Spec 偏移（4 → 2，名稱不同）
- Cache TTL ⚠️ contacts/calendar 快取未按 Spec 實作

### 7.2 Business Logic x Test Coverage

| 邊界情境 | Code | Test | 結果 |
|----------|------|------|------|
| E1 batch partial | ✅ | ✅ | 完整 |
| E2 cache degradation | ❌ | ❌ | 雙重缺失 |
| E3 100MB limit | ✅ | ❌ | 未測試 |
| E4 rate limit | ✅ | ✅ | 完整 |
| E5 reply not-found | ⚠️ | ❌ | 未測試 |
| E6 sheets partial | ✅ | ✅ | 完整 |

### 7.3 成功標準錨定

| SC# | 結論 |
|-----|------|
| SC1 MCP ≥ 55 | **通過**（59 工具） |
| SC2 新工具可呼叫 | **部分**（無 E2E 測試） |
| SC3 go test pass | **部分**（需 CI 驗證） |
| SC4 測試檔 ≥ 5 | **通過**（23 個） |
| SC5 batch multi-file | **部分**（缺執行路徑測試） |
| SC6 batch multi-range | **通過** |
| SC7 cache hit | **部分**（無整合測試） |
| SC8 slog 全覆蓋 | **部分** |
| SC9 version 0.8.0 | **通過** |

---

## 8. 觀察項

| # | 描述 |
|---|------|
| 1 | 三層 fallthrough 路由效率隨工具數增長下降，建議未來改 map dispatch |
| 2 | `ReplyMessage` 未設定 `In-Reply-To` / `References` RFC 2822 標頭 |
| 3 | `sheets_batch_append` semaphore 不支援 ctx.Done() 取消 |
| 4 | `cmd/onboard.go` 保留 17 處 fmt.Fprintln（互動式 UI，非 MCP） |
