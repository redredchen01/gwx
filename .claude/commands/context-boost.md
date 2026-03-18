---
description: "Context Boost — 從 Google Workspace 蒐集背景後啟動 S0。觸發：「帶上下文開 SOP」「context boost」「先查再做」「蒐集背景再開始」"
allowed-tools: Read, Grep, Glob, Bash, Task, mcp__sequential-thinking__sequentialthinking
argument-hint: "<需求描述，含可搜尋的關鍵字>"
---

# Context Boost → S0

> **組合技**：`gwx context` × S0 需求討論
> 先從 Google Workspace 蒐集所有相關背景，再注入 S0 requirement-analyst，加速需求收斂。

## 輸入
需求描述：$ARGUMENTS

---

## Phase 1：關鍵字提取

從 `$ARGUMENTS` 提取 1-3 個搜尋關鍵字。

**規則**：
- 優先取名詞和專有名詞（產品名、功能名、人名）
- 排除通用動詞（做、加、改、新增）
- 中英文都保留

**範例**：
- "幫我做 invoice 自動寄送" → `invoice`, `寄送`
- "修復 OAuth token refresh 的 bug" → `OAuth`, `token refresh`
- "重構 sheets API 的 rate limiter" → `sheets`, `rate limiter`

---

## Phase 2：跨服務蒐集（全 🟢，自動執行）

依序執行以下 gwx 命令。若 gwx 未認證（exit code 10/11），跳過本 Phase，直接進入 Phase 4。

### Step 1：統一上下文蒐集
```bash
gwx context "{keywords}" --days 14 --json
```

### Step 2：高信號結果深挖

**Email（前 3 封最相關）**：
```bash
gwx gmail get {message_id} --json
```

**Drive 文件（前 2 份最相關）**：
```bash
gwx docs get {doc_id} --json
```

### Step 3：今日/明日行事曆
```bash
gwx calendar agenda --days 2 --json
```
過濾出標題含關鍵字的事件。

---

## Phase 3：編譯 Context Briefing

將蒐集結果結構化為以下格式（總長度 ≤ 2000 字元，超過則摘要）：

```markdown
## 🔍 Context Briefing（自動蒐集）

### 📬 相關 Email（{count} 封）
| 日期 | 寄件者 | 主旨 | 重點摘要 |
|------|--------|------|----------|

### 📄 相關文件（{count} 份）
| 文件名 | 最後修改 | 摘要 |
|--------|----------|------|

### 📅 相關會議（{count} 場）
| 日期 | 標題 | 出席者 |
|------|------|--------|

### 💡 自動洞察
- 涉及的利害關係人：{去重後的寄件者/出席者列表}
- 時程信號：{信件中提到的截止日或里程碑}
- 已有決策：{文件或信件中的明確結論}
```

若某服務無結果，該區段不顯示。

---

## Phase 4：注入 S0

將 Context Briefing + 原始需求一起傳入 S0：

```
Task(
  subagent_type: "requirement-analyst",
  model: "sonnet",
  prompt: "以下是自動蒐集的 Google Workspace 背景資料，請用來加速需求討論：\n\n--- BEGIN CONTEXT BRIEFING ---\n{context_briefing}\n--- END CONTEXT BRIEFING ---\n\n用戶原始需求：{$ARGUMENTS}\n\n指引：\n1. 引用具體 email/doc 來確認需求：「Alice 在 3/15 的信中提到 X，這仍是需求嗎？」\n2. 從出席者/寄件者預填利害關係人\n3. 發現矛盾時主動指出：「PRD 說 A，但信件討論傾向 B」\n4. 背景已回答的問題直接確認，不重複問\n5. 依標準 S0 流程走完六維度例外探測和 Spec Mode 判斷",
  description: "Context-boosted S0 需求討論"
)
```

---

## Fallback

- **gwx 未安裝**：跳過 Phase 2-3，直接以標準 S0 啟動
- **gwx 未認證**：提示 `gwx onboard`，同時以標準 S0 啟動（不阻塞）
- **無搜尋結果**：告知「未在 Google Workspace 找到相關資料」，以標準 S0 繼續
- **API 錯誤**：記錄錯誤，以標準 S0 繼續（context boost 是增強，不是前提）

---

## 安全性

- Phase 2 所有操作均為 🟢 Tier 1（純讀取），無需用戶確認
- Context briefing 不寫入檔案（僅存在於對話 context 中），除非用戶要求
- 若 briefing 包含敏感內容（密碼、token），自動遮蔽後再注入
