---
description: "Spec Health Dashboard — 用 Google Sheet 追蹤跨 feature 的 spec-audit 品質趨勢。觸發：「spec health」「品質儀表板」「spec 健康」「audit 追蹤」「quality dashboard」"
allowed-tools: Read, Grep, Glob, Bash, Task
argument-hint: "[SHEET_ID] [record:{spec_folder}]"
---

# Spec Health Dashboard — Sheets × Spec Audit

> **組合技**：`gwx sheets stats/diff` × `/spec-audit`
> 跨 feature 品質趨勢追蹤，一張 Sheet 看全局。

## 輸入
參數：$ARGUMENTS

---

## 模式判斷

- 無參數 → **檢視模式**（stats + insights）
- `SHEET_ID` → **檢視模式**（指定 Sheet）
- `record:{spec_folder}` → **記錄模式**（寫入 audit 結果）
- 首次使用 → **初始化模式**（建立 Sheet）

---

## 初始化模式（首次）

### Step 1：建立 Sheet（🟡）
```bash
gwx sheets create --title "Spec Health Dashboard" --json
```

### Step 2：建立 3 個 Tab（🟡）

**Audit Log**：每次 audit 一行
```bash
gwx sheets update {SHEET_ID} "Audit Log!A1:I1" --values '[["Date", "Feature", "Spec Mode", "Round", "P0", "P1", "P2", "Status", "Duration"]]' --json
```

**Feature Summary**：每個 feature 最新狀態
```bash
gwx sheets update {SHEET_ID} "Feature Summary!A1:H1" --values '[["Feature", "Total Audits", "Last Audit", "P0", "P1", "P2", "Rounds", "Health"]]' --json
```

**Trend**：週度聚合
```bash
gwx sheets update {SHEET_ID} "Trend!A1:F1" --values '[["Week", "Features", "P0", "P1", "P2", "Avg Rounds"]]' --json
```

輸出 Sheet URL，建議記錄到專案設定。

---

## 記錄模式（每次 audit 後）

讀取 `{spec_folder}/sdd_context.json`，提取 audit 結果：

```bash
gwx sheets smart-append {SHEET_ID} "Audit Log!A:I" --values '[
  ["{date}", "{feature}", "{spec_mode}", "{round}", {p0}, {p1}, {p2}, "{status}", "{duration}"]
]' --json
```

更新 Feature Summary 對應行：

Health 判斷邏輯：
- P0=0, P1=0, P2≤2 → `✅`
- P0=0, P1≤2 → `🟡`
- P0>0 → `🔴`

---

## 檢視模式（隨時查看）

### 整體統計（🟢）
```bash
gwx sheets stats {SHEET_ID} --tab "Feature Summary" --json
```

### 品質報告

```
Spec Health Report:
━━━━━━━━━━━━━━━━━━━━
Features: {N} tracked
  ✅ Healthy: {N}
  🟡 Acceptable: {N}
  🔴 Critical: {N}

Most audited: {feature} ({N} rounds)
Highest P0: {feature} — 需要根因分析
Avg convergence: {N} rounds
Trend: P0 {↑|↓|→} vs last week
```

### 跨期對比（🟢）
```bash
gwx sheets diff {SHEET_ID} --tab1 "{period1}" --tab2 "{period2}" --json
```

---

## 與 SOP 整合

- `/spec-audit` 完成後 → 自動提示：「記錄到 Spec Health Dashboard？」
- `/audit-converge` 完成後 → 記錄每輪結果 + 最終狀態
- 週會前 → 跑 `/spec-health` 產出品質報告

---

## Fallback

- **gwx 未認證**：跳過，audit 結果仍存在 sdd_context
- **Sheet 不存在**：提示初始化
- **寫入失敗**：記錄錯誤，不阻塞 audit 流程
