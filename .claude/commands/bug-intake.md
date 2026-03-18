---
description: "Bug Intake — 從 Gmail 撈 bug 報告，自動轉為 S0 bugfix SOP。觸發：「bug intake」「bug email」「信件裡的 bug」「從 email 開 bug」「bug 報告轉 SOP」"
allowed-tools: Read, Grep, Glob, Bash, Task
argument-hint: "[搜尋關鍵字] [--after YYYY/MM/DD]"
---

# Bug Intake — Gmail Digest × S0 Bugfix

> **組合技**：`gwx gmail digest` × S0 需求討論
> 從 email 撈 bug 報告 → 結構化提取 → 自動注入 S0 bugfix 流程。

## 輸入
參數：$ARGUMENTS

---

## Phase 1：搜尋 Bug 報告 Email（🟢）

### 搜尋策略

**有關鍵字**：
```bash
gwx gmail search "subject:({keyword} AND (bug OR error OR issue OR 問題 OR crash)) -in:sent" --limit 10 --json
```

**無關鍵字（預設掃描）**：
```bash
gwx gmail search "subject:(bug OR error OR issue OR 問題 OR 壞了 OR crash OR failed) -in:sent" --limit 10 --json
```

**有日期過濾**：加上 `after:{date}`

### 分類呈現

```
找到 {N} 封可能的 Bug 報告：

1. [2026-03-17] alice@co.com — "API returns 500 on invoice creation"
2. [2026-03-16] bob@co.com — "Dashboard chart not loading after deploy"
3. [2026-03-15] ci@github.com — "Build failed: test_auth_refresh"

選擇要處理的（1-3, 或 'all'）：
```

等待用戶選擇。

---

## Phase 2：提取 Bug 詳情（🟢）

對選中的 email：
```bash
gwx gmail get {message_id} --json
```

從 email body 提取：
- **Reporter**：寄件者
- **Summary**：主旨
- **Description**：內文前 500 字
- **Reproduction Steps**：搜尋 numbered list 或 "steps to reproduce"
- **Expected**：搜尋 "expected" / "should" / "應該"
- **Actual**：搜尋 "actual" / "instead" / "but" / "但是" / "結果"
- **Environment**：版本號、OS、瀏覽器資訊
- **Severity**：blocker / critical / minor 關鍵字

---

## Phase 3：編譯 S0 輸入

```markdown
## Bug Report（from email）

**Reporter**: {sender}
**Date**: {email_date}
**Subject**: {subject}

### Description
{extracted description}

### Reproduction Steps
{extracted or "未提供 — 需要與 reporter 確認"}

### Expected vs Actual
- Expected: {extracted or "需要確認"}
- Actual: {extracted or "需要確認"}

### Environment
{extracted or "未指定"}
```

---

## Phase 4：注入 S0

```
Skill(
  skill: "s0-understand",
  args: "{compiled bug report}\n\nwork_type: bugfix\nspec_mode_hint: quick"
)
```

**S0 行為調整**：
- `work_type` 預設為 `bugfix`，跳過類型判斷
- `spec_mode_hint` 建議 Quick（除非跨模組）
- 聚焦「重現步驟」和「預期 vs 實際」
- 提取不足的欄位由 requirement-analyst 主動詢問

---

## Batch Mode

選擇多封 email 時：
1. 逐封處理：提取 → 編譯
2. 檢測相關性（相同 component / 相同 reporter）
3. 相關 bug → 建議合併為單一 SOP
4. 不相關 → 逐個建立獨立 S0

---

## Fallback

- **gwx 未認證**：提示 `gwx onboard`
- **無搜尋結果**：「未找到 bug 報告 email，請手動描述 bug 或調整搜尋條件」
- **email 內文不足**：requirement-analyst 會補問
