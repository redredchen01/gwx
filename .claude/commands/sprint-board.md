---
description: "Sprint Board — 用 Google Sheet 當看板，S0 建 ticket、S4 更新狀態、stats 算 burn-down。觸發：「sprint board」「看板」「kanban」「sprint sheet」「專案追蹤」"
allowed-tools: Read, Grep, Glob, Bash, Task
argument-hint: "[SHEET_ID] [ticket:{feature_name}] [update:{feature}:{status}] [stats] [archive]"
---

# Sprint Board — Sheets × SOP Pipeline

> **組合技**：`gwx sheets` 全家桶 × SOP S0-S7
> 零成本專案管理。gwx 就是你的 Jira。

## 輸入
參數：$ARGUMENTS

---

## 模式判斷

| 參數 | 模式 |
|------|------|
| 無參數 | 初始化（建立 Sheet）或檢視（若已存在） |
| `SHEET_ID` | 檢視模式（stats） |
| `ticket:{feature}` | 新增 ticket（從 S0 output） |
| `update:{feature}:{status}` | 更新 ticket 狀態 |
| `stats` | 顯示 burn-down |
| `archive` | 歸檔當前 sprint |

---

## 初始化模式

### Step 1：建立 Sheet（🟡）
```bash
gwx sheets create --title "Sprint Board — {sprint_name}" --json
```

### Step 2：設定表頭（🟡）
```bash
gwx sheets update {SHEET_ID} "Backlog!A1:J1" --values '[["Ticket", "Feature", "Type", "Priority", "Status", "Assignee", "Created", "Updated", "Branch", "Notes"]]' --json
```

輸出 Sheet URL。

---

## 新增 Ticket 模式（S0 後調用）

讀取 `sdd_context.json`，提取 S0 output：

```bash
gwx sheets smart-append {SHEET_ID} "Backlog!A:J" --values '[
  ["SOP-{N}", "{feature}", "{work_type}", "{priority}", "backlog", "{assignee}", "{date}", "{date}", "{branch}", ""]
]' --json
```

**SOP Stage → Status 對照表**：

| Stage | Status |
|-------|--------|
| S0 confirmed | backlog |
| S1-S3 | in-progress |
| S4 | in-progress |
| S5 | review |
| S5 fix_required | in-progress（Notes: "Review round N"） |
| S6 | testing |
| S6 blocked | blocked（Notes: "Repair loop 3x"） |
| S7 done | done |

---

## 更新 Status 模式

```bash
gwx sheets update {SHEET_ID} "Backlog!E{row}:J{row}" --values '[["{status}", "", "", "{date}", "", "{notes}"]]' --json
```

先用 `gwx sheets search` 或 `read` 找到 feature 對應的 row。

---

## 檢視 / Stats 模式（🟢）

```bash
gwx sheets stats {SHEET_ID} --json
```

```
Sprint Progress:
━━━━━━━━━━━━━━━━━━━━
Status → done: 7, in-progress: 3, review: 1, testing: 2, backlog: 4, blocked: 1
Type   → feature: 10, bugfix: 5, refactor: 3
Priority → P0: 2, P1: 8, P2: 8

Velocity: 7/18 = 38.9%
🔴 Blocked: SOP-012 — S6 repair loop
```

---

## 歸檔模式（Sprint 結束時）

```bash
# 複製當前 sprint 為歸檔 tab
gwx sheets copy-tab {SHEET_ID} --from "Backlog" --to "Sprint {N} Archive" --json
```

跨 sprint 對比：
```bash
gwx sheets diff {SHEET_ID} --tab1 "Sprint 1 Archive" --tab2 "Sprint 2 Archive" --json
```

---

## 與 SOP 自動整合

Sprint Board 是**被動接收者**，SOP 管線主動推送：

```
S0 Gate ✅ → smart-append 新 ticket
S4 開始   → update status = in-progress
S5 開始   → update status = review
S5 退回   → update status = in-progress + notes
S6 開始   → update status = testing
S6 卡住   → update status = blocked + notes
S7 完成   → update status = done
```

---

## 多人協作

```bash
# 🔴 需確認
gwx drive share {SHEET_ID} --email "team@co.com" --role writer --json
```

---

## Fallback

- **gwx 未認證**：SOP 正常執行，看板只是增強層
- **Sheet 不存在**：提示初始化
- **寫入失敗**：記錄錯誤，不阻塞 SOP
