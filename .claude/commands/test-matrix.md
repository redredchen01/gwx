---
description: "Test Matrix — 用 Google Sheet 作為 S6 測試追蹤儀表板。觸發：「測試追蹤」「test matrix」「建測試追蹤表」「sync tests to sheet」"
allowed-tools: Read, Grep, Glob, Bash, Task
argument-hint: "<spec_folder 路徑> 或 <SHEET_ID>（已有 Sheet 時）"
---

# Test Matrix — Sheets × S6 測試追蹤

> **組合技**：`gwx sheets` × S6 測試階段
> 用 Google Sheet 作為活的測試追蹤儀表板，PM/QA 不進 repo 也能看進度。

## 輸入
參數：$ARGUMENTS

---

## 模式判斷

- `$ARGUMENTS` 含 `spec_folder` 路徑或 `sdd_context.json` → **初始化模式**（Step 1-3）
- `$ARGUMENTS` 含 `SHEET_ID` → **同步模式**（Step 4-6）
- `$ARGUMENTS` 為空 → 嘗試從當前 SOP context 自動偵測

---

## 初始化模式（S3 完成後調用）

### Step 1：讀取 S3 測試計畫

```
讀取 {spec_folder}/s3_implementation_plan.md
提取所有 tdd_plan 區塊：
  - task_id
  - test_file
  - test_cases（名稱 + 類型）
  - test_command
```

### Step 2：建立 Google Sheet

```bash
gwx sheets create --title "{feature_name} — Test Matrix" --json
```

記錄 SHEET_ID。

### Step 3：填充測試案例

設定表頭 + 批次寫入所有測試案例：

```bash
# 表頭
gwx sheets update {SHEET_ID} "A1:J1" --values '[["TC-ID", "Task", "Test Case", "Type", "Priority", "Status", "Result", "Error Summary", "Fixed In", "Last Run"]]' --json

# 測試案例（smart-append 自動驗證欄位結構）
gwx sheets smart-append {SHEET_ID} "A:J" --values '{extracted_test_cases}' --json
```

### Step 4：回寫 sdd_context

將 Sheet URL 和 ID 記錄到 `sdd_context.json`：
```json
{
  "stages": {
    "s6": {
      "output": {
        "test_matrix_sheet_id": "{SHEET_ID}",
        "test_matrix_url": "https://docs.google.com/spreadsheets/d/{SHEET_ID}"
      }
    }
  }
}
```

輸出 Sheet URL 給用戶。

---

## 同步模式（S6 執行中調用）

### 即時更新

每次測試執行後，更新對應行：

| 情境 | 更新欄位 |
|------|----------|
| 測試通過 | Status=passed, Result=✅, Last Run={now} |
| 測試失敗 | Status=failed, Result=❌, Error Summary={msg}, Last Run={now} |
| 修復後通過 | Status=passed, Result=✅, Fixed In={commit}, Last Run={now} |
| 跳過 | Status=skipped, Result=⏭️ |

```bash
gwx sheets update {SHEET_ID} "F{row}:J{row}" --values '[["{status}", "{result}", "{error}", "{commit}", "{timestamp}"]]' --json
```

### 進度統計

```bash
gwx sheets stats {SHEET_ID} --json
```

輸出格式：
```
測試進度：passed=12, failed=3, pending=5, skipped=1
通過率：12/21 = 57.1%
P0 未通過：TC-003, TC-007
```

### 修復迴圈對比

```bash
gwx sheets diff {SHEET_ID} --tab1 "Round1" --tab2 "Round2" --json
```

---

## 完成後（S6 Gate 通過）

### 匯出存檔

```bash
gwx sheets export {SHEET_ID} --format csv --output "{spec_folder}/s6_test_results.csv"
```

### 最終統計

```bash
gwx sheets stats {SHEET_ID} --json
```

將最終統計寫入 `sdd_context.json` 的 `stages.s6.output`。

---

## Fallback

- **gwx 未認證**：跳過 Sheet 同步，S6 正常執行（測試結果只存 sdd_context）
- **Sheet API 錯誤**：記錄錯誤，不阻塞 S6 流程
- **無 S3 tdd_plan**：提示用戶手動提供測試案例列表

> Test Matrix 是 **增強層**，不是 S6 的前提。Sheet 掛了不影響測試執行。
