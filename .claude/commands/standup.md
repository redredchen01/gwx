---
description: "AI 站會報告 — 合併 Git 活動、SOP 進度、Google Workspace 資料，產出結構化站會報告。觸發：「standup」「站會」「daily standup」「站會報告」"
allowed-tools: Read, Grep, Glob, Bash, Task
argument-hint: "[chat:SPACE_NAME] [email:ADDR] — 可選推送目標"
---

# AI Standup Report

> **組合技**：gwx (Gmail + Calendar + Tasks + Chat) × Git × SDD Context
> 每天一個指令，站會報告自動生成。

## 輸入
推送目標：$ARGUMENTS

---

## Phase 1：本地資料蒐集（即時，無網路需求）

### 1a. Git 活動
```bash
git log --oneline --since="yesterday" --author="$(git config user.name)" 2>/dev/null || echo "no commits"
```
```bash
git branch --show-current
```
```bash
git status --porcelain 2>/dev/null | head -5
```

提取：
- 昨日 commit 數量 + 摘要
- 當前分支
- 未提交變更（有/無）

### 1b. SOP 進度

掃描所有活躍的 SDD Context：
```bash
find dev/specs -name "sdd_context.json" -maxdepth 3 2>/dev/null
```

對每個找到的 context，讀取並提取：
- `feature`：功能名稱
- `current_stage`：當前階段
- `status`：in_progress / completed
- `stages.{current}.output`：關鍵輸出（如 repair_loop_count）

若無活躍 SOP，此區段跳過。

---

## Phase 2：Google Workspace 蒐集（全 🟢 讀取）

> 若 gwx 未認證，跳過本 Phase，Phase 3 只用 Phase 1 資料。

以下命令**並行**執行：

```bash
# 2a. 昨日寄出的信
gwx gmail search "in:sent after:$(date -v-1d +%Y/%m/%d) before:$(date +%Y/%m/%d)" --limit 10 --json

# 2b. 昨日會議
gwx calendar list --from $(date -v-1d +%Y-%m-%d) --to $(date +%Y-%m-%d) --json

# 2c. 今日會議
gwx calendar agenda --days 1 --json

# 2d. 已完成任務
gwx tasks list --show-completed --json

# 2e. 待辦任務
gwx tasks list --json
```

---

## Phase 3：編譯站會報告

```markdown
# Daily Standup — {YYYY-MM-DD}

## Done（昨日完成）

### Development
- {N} commits on `{branch}`:
  - {commit message 1}
  - {commit message 2}
- SOP: {feature_name} — S{x} → S{y} {status}

### Communication
- 寄出 {N} 封信：{主旨列表}
- 參加 {N} 場會議：{會議標題}
- 完成 {N} 項任務：{任務名稱}

## Plan（今日計畫）

### Development
- {feature_name}：繼續 S{current_stage}
- {未提交變更狀態}

### Meetings
- {time}: {meeting title}

### Tasks
- [ ] {pending task 1}（due: {date}）
- [ ] {pending task 2}

## Blockers
- {SOP 修復迴圈超限 / 測試失敗 / 其他}
- （無）← 若無 blocker
```

### 資料缺失處理
- 無 commit → "No commits yesterday"
- 無 SOP → 整個 Development/SOP 區段省略
- gwx 未認證 → Communication 區段顯示 "(gwx not connected)"
- 無 blocker → "（無）"

---

## Phase 4：輸出（依 $ARGUMENTS 決定）

### 預設：終端機顯示（🟢）
直接 print 站會報告。

### `chat:SPACE_NAME` → 推送到 Google Chat（🔴 需確認）
```bash
gwx chat send {SPACE_NAME} --text "{standup_report}" --json
```
**必須顯示完整報告內容並取得用戶明確確認才能送出。**

### `email:ADDR` → 草稿信件（🟡 需確認）
```bash
gwx gmail draft --to "{ADDR}" --subject "Standup — $(date +%Y-%m-%d)" --body "{standup_report}" --json
```

---

## 搭配 /loop 自動化

可與 `/loop` 技能搭配，每天早上自動生成：
```
/loop 24h /standup
```

或定時推送到 Chat：
```
/loop 24h /standup chat:spaces/AAAA
```

---

## 安全性
- Phase 1-2 全部 🟢 讀取操作
- Phase 4 推送是 🔴/🟡，需明確確認
- 報告不寫入檔案（除非用戶要求）
- 不包含敏感資訊（信件只列主旨，不列內文）
