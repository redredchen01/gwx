# gwx 使用指南

gwx 是一個 Google Workspace CLI 工具，同時支援兩種使用方式：
- **人類** — 在終端機直接下指令操作 Gmail、Calendar、Drive 等 8 個 Google 服務
- **AI Agent** — 作為 MCP Server 或 Bash 工具，讓 Claude Code / Codex 等 LLM 代理程式直接操作 Google Workspace

---

## 目錄

1. [前置準備（人類 & Agent 共用）](#前置準備)
2. [人類使用指南](#人類使用指南)
3. [Agent 使用指南](#agent-使用指南)
4. [安全機制](#安全機制)
5. [故障排除](#故障排除)

---

## 前置準備

不管你是人類還是 Agent，都需要先完成以下兩步。

### Step 1：安裝 gwx

三種安裝方式，選一個：

```bash
# 方式 A：npm（推薦，自動下載預編譯 binary）
npm install -g gwx-cli

# 方式 B：Go
go install github.com/redredchen01/gwx/cmd/gwx@latest

# 方式 C：從原始碼
git clone https://github.com/redredchen01/gwx.git
cd gwx && make install
```

驗證安裝成功：
```bash
gwx version
```

### Step 2：設定 Google Cloud 憑證

gwx 需要你自己的 Google Cloud OAuth 憑證。這是一次性的設定。

```bash
gwx onboard
```

互動式精靈會引導你完成三步：

1. **提供 OAuth 憑證** — 到 [Google Cloud Console](https://console.cloud.google.com/apis/credentials) 建立 OAuth 2.0 Client ID（類型選 Desktop App），下載 JSON 檔案，貼上路徑
2. **選擇服務** — 預設全選 8 個服務（Gmail, Calendar, Drive, Docs, Sheets, Tasks, Contacts, Chat），直接按 Enter
3. **登入** — 開瀏覽器完成 Google OAuth 授權

完成後，你的 OAuth token 會存在作業系統的 Keyring（macOS Keychain / Linux Secret Service / Windows Credential Manager），**不會寫到檔案裡**。

驗證認證狀態：
```bash
gwx auth status
```

---

## 人類使用指南

你在終端機下 `gwx <service> <command>` 就能操作 Google Workspace。

### 基本語法

```
gwx <服務> <指令> [參數] [--flags]
```

### 快速上手：5 個最常用的操作

```bash
# 看最近 5 封信
gwx gmail list --limit 5

# 今天有什麼會議
gwx calendar agenda

# 列出 Drive 檔案
gwx drive list

# 讀 Google Sheets 資料
gwx sheets read SPREADSHEET_ID "A1:C10"

# 跨服務搜尋（同時搜 Gmail + Drive）
gwx find "報價單"
```

### 快捷指令

不想打完整的 `service command`？gwx 有幾個常用快捷：

```bash
gwx ls                    # = gwx drive list
gwx search "keyword"      # = gwx gmail search
gwx send --to a@b.com ... # = gwx gmail send
gwx find "topic"          # = 同時搜 Gmail + Drive + Contacts
gwx context "project"     # = 彙整 Gmail + Drive + Calendar 的相關上下文
```

### 各服務指令速查

#### Gmail（9 個指令）

```bash
gwx gmail list [--limit N] [--unread] [--label LABEL]     # 列出信件
gwx gmail get MESSAGE_ID                                    # 讀取單封信
gwx gmail search "from:boss subject:urgent"                 # 搜尋信件
gwx gmail labels                                            # 列出標籤
gwx gmail send --to a@b.com --subject "Hi" --body "..."    # 寄信
gwx gmail draft --to a@b.com --subject "Hi" --body "..."   # 建草稿
gwx gmail reply MESSAGE_ID --body "收到"                    # 回信
gwx gmail digest --limit 30                                 # 智慧摘要
gwx gmail archive "subject:CI failed" --limit 50           # 批次封存
```

#### Calendar（6 個指令）

```bash
gwx calendar agenda [--days N]                              # 近期行程
gwx calendar list --from 2026-03-01 --to 2026-03-31        # 日期範圍查詢
gwx calendar create --title "週會" --start ... --end ...   # 建立事件
gwx calendar update EVENT_ID --title "改名"                # 修改事件
gwx calendar delete EVENT_ID                                # 刪除事件
gwx calendar find-slot --attendees a@b.com,c@d.com         # 找空檔
```

#### Drive（6 個指令）

```bash
gwx drive list [--folder FOLDER_ID] [--limit N]            # 列出檔案
gwx drive search "name contains 'report'"                   # 搜尋檔案
gwx drive upload file.pdf [--folder FOLDER_ID]              # 上傳檔案
gwx drive download FILE_ID [-o output.pdf]                  # 下載檔案
gwx drive share FILE_ID --email user@x.com --role reader   # 分享檔案
gwx drive mkdir "新資料夾" [--parent FOLDER_ID]            # 建資料夾
```

#### Docs（8 個指令）

```bash
gwx docs get DOC_ID                                         # 讀取文件
gwx docs create --title "新文件" [--body "內容"]            # 建立文件
gwx docs append DOC_ID --text "追加內容"                    # 追加內容
gwx docs search "keyword"                                   # 搜尋文件
gwx docs replace DOC_ID --find "舊" --replace "新"         # 尋找取代
gwx docs template TEMPLATE_ID -v '{"name":"Alice"}'        # 範本套用
gwx docs from-sheet SHEET_ID [--template TEMPLATE_ID]      # 從 Sheet 生成
gwx docs export DOC_ID --format pdf -o report.pdf          # 匯出
```

#### Sheets（15 個指令）

```bash
gwx sheets read SHEET_ID "A1:C10"                           # 讀取儲存格
gwx sheets info SHEET_ID                                     # 基本資訊
gwx sheets describe SHEET_ID                                 # 欄位結構分析
gwx sheets stats SHEET_ID                                    # 欄位統計
gwx sheets search SHEET_ID "keyword"                        # 搜尋內容
gwx sheets filter SHEET_ID --column "狀態" --value "完成"  # 篩選
gwx sheets diff SHEET_ID --from "第1周" --to "第2周"       # 比較兩個分頁
gwx sheets append SHEET_ID "A:C" --values '[["a",1,"b"]]'  # 新增列
gwx sheets smart-append SHEET_ID "A:F" --values '[...]'    # 驗證後新增
gwx sheets update SHEET_ID "A1:B2" --values '[["x","y"]]'  # 更新儲存格
gwx sheets clear SHEET_ID "A2:Z"                            # 清空範圍
gwx sheets copy-tab SHEET_ID --source "第1周" --name "第2周"  # 複製分頁
gwx sheets export SHEET_ID "A:D" --export-format csv -o r.csv  # 匯出
gwx sheets import SHEET_ID "A1" -i data.csv --import-format csv  # 匯入
gwx sheets create --title "新試算表"                        # 建立試算表
```

#### Tasks（5 個指令）

```bash
gwx tasks list [--list LIST_ID] [--show-completed]          # 列出待辦
gwx tasks lists                                              # 列出清單
gwx tasks create --title "買牛奶" [--due 2026-03-20]       # 建立待辦
gwx tasks complete TASK_ID                                   # 完成待辦
gwx tasks delete TASK_ID                                     # 刪除待辦
```

#### Contacts（3 個指令）

```bash
gwx contacts list [--limit N]                                # 列出聯絡人
gwx contacts search "John"                                   # 搜尋聯絡人
gwx contacts get people/c123456                              # 取得聯絡人詳情
```

#### Chat（3 個指令）

```bash
gwx chat spaces                                              # 列出聊天空間
gwx chat send spaces/AAAA --text "Hello"                    # 發送訊息
gwx chat messages spaces/AAAA [--limit N]                   # 讀取訊息
```

### 輸出格式

所有指令預設輸出 JSON。可用 `--format` 切換：

```bash
gwx gmail list --format json     # JSON（預設，適合 pipe 到 jq）
gwx gmail list --format plain    # 純文字（適合快速瀏覽）
gwx gmail list --format table    # 表格（適合人類閱讀）
```

用 `--fields` 只取特定欄位：

```bash
gwx gmail list --fields "subject,from,date"
```

### 多帳號

```bash
gwx auth login -a work           # 登入工作帳號
gwx auth login -a personal       # 登入個人帳號
gwx gmail list -a work           # 用工作帳號操作
gwx gmail list -a personal       # 用個人帳號操作
```

### Dry Run

不確定指令會做什麼？加 `--dry-run`：

```bash
gwx gmail send --to boss@co.com --subject "Hi" --body "..." --dry-run
# 只驗證參數，不會真的寄出
```

---

## Agent 使用指南

gwx 從設計之初就考慮了 AI Agent 的使用場景。有兩種整合方式：

### 方式 A：MCP Server（推薦）

MCP（Model Context Protocol）讓 Claude 直接呼叫 gwx 的工具，不需要透過 Bash。

#### 設定

在 `~/.claude/settings.json`（全域）或專案的 `.mcp.json` 加入：

```json
{
  "mcpServers": {
    "gwx": {
      "command": "gwx",
      "args": ["mcp-server"]
    }
  }
}
```

#### 運作方式

啟動後，Claude 可以直接呼叫 39 個 MCP tool，例如：
- `gmail_list` — 列出信件
- `gmail_search` — 搜尋信件
- `calendar_agenda` — 查看行程
- `sheets_read` — 讀取試算表
- `sheets_describe` — 分析欄位結構
- `sheets_stats` — 欄位統計
- `context_gather` — 跨服務彙整上下文

Agent 直接傳 JSON 參數呼叫，不需要組裝 CLI 指令字串。

#### MCP Tool 命名規則

CLI 指令對應到 MCP tool 的命名：`<service>_<command>`

| CLI 指令 | MCP Tool |
|---------|----------|
| `gwx gmail list` | `gmail_list` |
| `gwx gmail search` | `gmail_search` |
| `gwx calendar agenda` | `calendar_agenda` |
| `gwx sheets read` | `sheets_read` |
| `gwx sheets describe` | `sheets_describe` |
| `gwx drive search` | `drive_search` |
| `gwx docs get` | `docs_get` |
| `gwx find` | `unified_search` |
| `gwx context` | `context_gather` |

### 方式 B：Bash 工具呼叫

如果你的 Agent 沒有 MCP 支援（例如 Codex、GPT、自建 Agent），可以直接用 Bash 呼叫 gwx CLI。

#### Agent 友善設計

gwx 針對程式化呼叫做了以下設計：

**1. 結構化 JSON 輸出**

所有指令預設輸出 JSON，格式固定：

```json
// 成功
{"status": "ok", "data": { ... }}

// 失敗（輸出到 stderr）
{"status": "error", "error": {"code": 10, "name": "auth_required", "message": "..."}}
```

**2. 穩定的 Exit Code**

Agent 可以用 exit code 判斷結果，而不用 parse 錯誤訊息：

| Exit Code | 名稱 | 意義 | Agent 該怎麼做 |
|-----------|------|------|----------------|
| 0 | success | 成功 | parse `data` 欄位 |
| 10 | auth_required | 未認證 | 提示使用者跑 `gwx onboard` |
| 11 | auth_expired | Token 過期 | 提示使用者跑 `gwx auth login` |
| 12 | permission_denied | 權限不足 | 可能需要重新授權更多 scope |
| 20 | not_found | 資源不存在 | 幫使用者搜尋正確的 ID |
| 21 | conflict | 衝突 | 處理衝突邏輯 |
| 30 | rate_limited | 被限流 | 等 30 秒後重試 |
| 31 | circuit_open | 熔斷器開啟 | Google API 不穩定，等 30 秒 |
| 40 | invalid_input | 參數錯誤 | 修正參數後重試 |
| 50 | dry_run_success | Dry run 成功 | 參數驗證通過，可以執行 |

查看完整 exit code 列表：
```bash
gwx agent exit-codes
```

**3. Schema 自省**

Agent 可以查詢所有指令的 schema，包含安全等級：

```bash
gwx schema
```

回傳完整的指令清單，每個指令包含 `name`、`description`、`safety_tier`、`example_args`。

**4. 非 TTY 自動 JSON**

當 gwx 不在終端機裡跑（例如被 Agent 透過 subprocess 呼叫），設定環境變數即可強制 JSON：

```bash
export GWX_AUTO_JSON=1
```

**5. 無互動模式**

禁用所有互動式提示（Agent 不能打字回答問題）：

```bash
gwx gmail list --no-input
```

### 方式 C：Claude Code Skill（深度整合）

如果你使用 Claude Code，gwx 提供了一套 Skill 系統，讓 Claude 在對話中自動觸發 Google Workspace 操作。

#### 安裝

```bash
# 在 gwx 專案目錄下
./install.sh           # 全域安裝（~/.claude/commands/）
./install.sh --project # 專案級安裝（.claude/commands/）
```

這會安裝：
- 1 個主 Skill（`google-workspace.md`）— 指令路由 + 安全分級
- 4 個 Agent（`gmail-agent.md`、`calendar-agent.md`、`drive-agent.md`、`workspace-router.md`）
- 13 個 Recipe（跨服務工作流，見下方）

#### 自動觸發

安裝後，在 Claude Code 對話中用自然語言就能觸發：

```
你：看一下我的信
Claude：（自動呼叫 gwx gmail list）

你：明天有什麼會
Claude：（自動呼叫 gwx calendar agenda --days 1）

你：幫我找 John 的信箱
Claude：（自動呼叫 gwx contacts search "John"）
```

支援中文和英文觸發詞：`email/信件`、`calendar/行事曆`、`drive/雲端硬碟`、`sheets/試算表` 等。

#### Combo Skills — 跨服務工作流

gwx 和 Claude Code 的開發管線（SOP S0~S7）有深度整合：

| Skill | 說明 | 用法 |
|-------|------|------|
| `/context-boost` | 開 SOP 前先蒐集 Google Workspace 上下文 | `/context-boost 幫我做 invoice 功能` |
| `/test-matrix` | 用 Google Sheets 追蹤測試進度 | `/test-matrix dev/specs/my-feature` |
| `/standup` | 合併 Git + Google Workspace 產生站會報告 | `/standup` |
| `/bug-intake` | 從 Gmail 抓 bug 報告，轉為 SOP | `/bug-intake` |
| `/spec-health` | Spec 品質追蹤儀表板 | `/spec-health` |
| `/sprint-board` | Google Sheets 當看板 | `/sprint-board` |
| `/review-notify` | Review 結果推送到 Chat/Email | `/review-notify chat:spaces/AAAA` |
| `/parallel-schedule` | 為並行開發排 review 會議 | `/parallel-schedule --reviewers a@b.com` |

### Agent 權限沙箱

可以限制 Agent 只能用特定指令：

```bash
# 只允許讀取類操作
export GWX_ENABLE_COMMANDS="gmail.list,gmail.get,gmail.search,calendar.agenda,sheets.read"

# 允許特定服務的所有操作
export GWX_ENABLE_COMMANDS="gmail.*,calendar.*,sheets.read,sheets.describe"

# 允許全部（預設）
export GWX_ENABLE_COMMANDS="*"
```

被禁止的指令會回傳 exit code 2（usage_error）和明確的錯誤訊息。

---

## 安全機制

### 操作安全分級

每個指令都有安全等級，這同時影響人類使用和 Agent 行為：

| 等級 | 類型 | 行為 | 範例 |
|------|------|------|------|
| 🟢 Green | 唯讀 | 直接執行 | `gmail list`、`sheets read`、`calendar agenda` |
| 🟡 Yellow | 建立/修改 | 確認後執行 | `calendar create`、`sheets append`、`docs create` |
| 🔴 Red | 破壞性/對外發送 | 必須明確同意 | `gmail send`、`drive share`、`calendar delete` |
| ⛔ Blocked | 永久刪除 | 永遠不執行 | 永久刪除、ownership transfer |

Agent 在使用 Skill 時，會自動遵守這個分級：
- 🟢 直接執行，不問
- 🟡 先顯示操作摘要，等使用者確認
- 🔴 顯示完整細節（收件人、主旨、全文），等使用者明確說「好」

### Token 安全

- OAuth token 存在 OS Keyring（macOS Keychain / Linux Secret Service / Windows Credential Manager）
- **永遠不會寫到磁碟檔案**
- CSRF 保護：OAuth flow 使用 128-bit crypto/rand state
- CI/CD 環境可用環境變數：`export GWX_ACCESS_TOKEN="ya29.xxx"`

### 輸入安全

- Sheets 公式注入防護：自動轉義 `=`、`+`、`-`、`@` 開頭的值
- Drive 查詢注入防護：Folder ID 驗證
- 附件大小限制：25MB

### 穩定性

- Rate Limiter：每個服務獨立的 token bucket（Sheets 0.8 QPS、Gmail 4 QPS、Drive 8 QPS）
- Retry：429 指數退避（尊重 Retry-After header）、5xx 固定間隔重試
- Circuit Breaker：連續 5 次失敗後熔斷，30 秒後自動恢復

---

## 故障排除

### 常見問題

**Q: `gwx: command not found`**

```bash
# 如果用 Go 安裝，確認 GOPATH/bin 在 PATH 裡
export PATH="$PATH:$(go env GOPATH)/bin"

# 如果用 npm 安裝，確認全域 bin 在 PATH 裡
npm bin -g
```

**Q: `not authenticated` 錯誤**

```bash
# 檢查認證狀態
gwx auth status

# 重新認證
gwx auth login

# 完整重設
gwx onboard
```

**Q: `permission_denied` 錯誤（exit code 12）**

可能是初次 onboard 時沒有授權所有服務。重跑 `gwx onboard` 並選擇需要的服務。

**Q: MCP Server 連不上**

```bash
# 確認 gwx 在 PATH 裡
which gwx

# 手動測試 MCP server 能否啟動
gwx mcp-server
# 應該會等待 stdio 輸入，按 Ctrl+C 結束

# 確認 settings.json 設定正確
cat ~/.claude/settings.json | grep -A 5 gwx
```

**Q: Rate limit 錯誤（exit code 30）**

gwx 內建了 rate limiter 和自動重試，正常情況下你不會看到這個。如果看到了，代表你的 Google API quota 真的用完了。等幾分鐘再試。

### 移除 gwx

```bash
./uninstall.sh
```

這會移除 binary 和 Claude Code skill 檔案。OAuth token 留在 OS Keyring 裡，要手動移除的話先跑 `gwx auth logout`。

---

## 授權

MIT License. 詳見 [LICENSE](LICENSE)。
