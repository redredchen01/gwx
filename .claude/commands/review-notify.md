---
description: "Review Notify — S5 Code Review 結果即時推送到 Google Chat 或 Email。觸發：「review notify」「推送 review 結果」「通知 review」「notify team」"
allowed-tools: Read, Grep, Glob, Bash, Task
argument-hint: "<chat:SPACE_NAME 或 email:ADDR> [spec_folder]"
---

# Review Notify — Chat/Email × S5 Code Review

> **組合技**：`gwx chat send` / `gwx gmail send` × S5 Code Review
> Review 完成 → 即時推送結果到團隊頻道，不用等開發者手動回報。

## 輸入
參數：$ARGUMENTS

---

## Phase 1：讀取 S5 結果（local）

從 `$ARGUMENTS` 中的 `spec_folder` 或自動掃描最近完成的 S5：

```bash
find dev/specs -name "sdd_context.json" -maxdepth 3 2>/dev/null
```

提取 `stages.s5.output`：
- `recommendation`：pass / conditional_pass / fix_required
- `findings`：severity + description + file + line
- `p0_count`、`p1_count`、`p2_count`

若找不到 S5 結果 → 「沒有已完成的 S5 review，請先跑 /s5-review 或 /code-review」

---

## Phase 2：編譯通知

```
📋 Code Review Complete: {feature_name}

Result: {✅ PASS | ⚠️ CONDITIONAL | ❌ FIX REQUIRED}
Branch: {branch}

Findings: {p0} P0 · {p1} P1 · {p2} P2
{if p0 > 0:}
🔴 P0 (blocking):
• {description} — {file}:{line}
{endif}
{if p1 > 0:}
🟡 P1:
• {description} — {file}:{line}
{endif}

Next: {S6 testing | Back to S4 for fix}
```

P0=P1=0 時簡化為一行：`✅ {feature_name} — Code Review PASSED, proceeding to S6`

---

## Phase 3：推送（🔴 需明確確認）

### `chat:SPACE_NAME`
```bash
gwx chat send {SPACE_NAME} --text "{notification}" --json
```

### `email:ADDR`
```bash
gwx gmail send --to "{ADDR}" --subject "Review: {feature_name} — {PASS|FAIL}" --body "{notification}" --json
```

### 無參數 → 詢問
「Review 結果要推送到哪裡？chat:SPACE_NAME 或 email:ADDR」

**推送前必須顯示完整通知內容並取得 "yes" 確認。**

---

## Phase 4：記錄

更新 `sdd_context.json`：
```json
{
  "stages.s5.output.notification_sent": true,
  "stages.s5.output.notification_channel": "{target}",
  "stages.s5.output.notification_at": "{ISO 8601}"
}
```

---

## Fallback

- **gwx 未認證**：顯示通知內容在終端機，讓用戶手動轉發
- **Chat space 不存在**：`gwx chat spaces --json` 列出可用 spaces
- **發送失敗**：記錄錯誤，通知內容仍顯示在終端機
