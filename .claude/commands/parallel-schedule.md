---
description: "Parallel Schedule — 為並行開發的 worktree 自動排 review 會議。觸發：「parallel schedule」「排 review 會議」「schedule reviews」「並行排程」"
allowed-tools: Read, Grep, Glob, Bash, Task
argument-hint: "[--reviewers a@co.com,b@co.com] [--duration 30m]"
---

# Parallel Schedule — Calendar × Parallel Develop

> **組合技**：`gwx calendar find-slot` × `/parallel-develop`
> 所有 worktree 到 S4 → 自動找空檔 → 建 review 會議 → 附 briefing doc。

## 輸入
參數：$ARGUMENTS

---

## Phase 1：掃描 Worktree 狀態（local）

```bash
git worktree list --porcelain
```

對每個 worktree，讀取 `sdd_context.json`：
- Feature name
- Current stage
- Brief spec §1 一句話描述

過濾出 **S4 completed 或 S5 pending** 的 worktree。

若無 worktree 需要排程 → 「目前沒有待 review 的 worktree」。

---

## Phase 2：蒐集 Reviewer 資訊

**從參數**：`--reviewers a@co.com,b@co.com`
**從對話**：詢問 reviewer emails
**預設**：`--duration 30m`、`--days 3`

---

## Phase 3：找空檔（🟢）

對每個待 review 的 feature：
```bash
gwx calendar find-slot --attendees "{reviewer_emails}" --duration {duration} --days 3 --json
```

編排時間表，避免連續排（至少間隔 15 分鐘）：

```
建議 Review 時間表：

| # | Feature | 時間 | 出席者 |
|---|---------|------|--------|
| 1 | invoice | Mon 10:00-10:30 | alice, bob |
| 2 | auth-refactor | Mon 14:00-14:30 | alice, bob |
| 3 | dashboard | Tue 10:00-10:30 | alice, bob |

確認？（yes / 調整）
```

等待用戶確認。

---

## Phase 4：建立日曆事件（🟡 需確認）

```bash
gwx calendar create \
  --title "Code Review: {feature_name}" \
  --start "{start}" \
  --end "{end}" \
  --attendees "{reviewer_emails}" \
  --json
```

---

## Phase 5：建立 Briefing Doc（🟡 可選）

```bash
gwx docs create \
  --title "Review Brief: {feature_name}" \
  --body "## {feature_name}\n\n{brief_spec_summary}\n\n**Branch**: {branch}\n**Changed files**: {file_count}\n**Key decisions**: {from dev_spec}" \
  --json
```

在 calendar event 描述中附上 Doc 連結。

---

## Phase 6：輸出總覽

```
Review Schedule Created:

| Feature | Date | Time | Brief Doc |
|---------|------|------|-----------|
| invoice | Mon 3/19 | 10:00 | [link] |
| auth    | Mon 3/19 | 14:00 | [link] |
| dashboard | Tue 3/20 | 10:00 | [link] |

所有 reviewer 已收到日曆邀請。
```

---

## Fallback

- **無共同空檔**：建議延長搜尋天數或縮短 review 時長
- **gwx 未認證**：列出待 review 的 feature，讓用戶手動排程
- **單一 worktree**：簡化為一個 find-slot + create
