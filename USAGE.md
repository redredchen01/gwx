# gwx 使用指南

gwx 是一個統一的 CLI + MCP Server 工具，支援兩種使用方式：
- **人類** — 在終端機直接下指令操作 Gmail、Calendar、Drive、Docs、Sheets、Tasks、Contacts、Chat、Analytics、Search Console、Slides、Forms、BigQuery、GitHub、Slack、Notion 共 16 個服務
- **AI Agent** — 作為 MCP Server 或 Bash 工具，讓 Claude Code / Codex 等 LLM 代理程式直接操作

---

## 目錄

1. [前置準備](#前置準備)
2. [人類使用指南](#人類使用指南)
3. [Agent 使用指南](#agent-使用指南)
4. [Skill DSL](#skill-dsl)
5. [多供應商認證](#多供應商認證)
6. [Shell 自動補全](#shell-自動補全)
7. [健康檢查](#健康檢查)
8. [安全機制](#安全機制)
9. [故障排除](#故障排除)

---

## 前置準備

不管你是人類還是 Agent，都需要先完成以下兩步。

### Step 1：安裝 gwx

五種安裝方式，選一個：

```bash
# 方式 A：一鍵安裝（macOS/Linux — 下載預編譯 binary 到 /usr/local/bin）
curl -fsSL https://raw.githubusercontent.com/redredchen01/gwx/main/install-bin.sh | sudo bash

# 方式 B：npm（推薦，自動下載預編譯 binary）
npm install -g gwx-cli

# 方式 C：Go
go install github.com/redredchen01/gwx/cmd/gwx@latest

# 方式 D：Homebrew
brew install redredchen01/tap/gwx

# 方式 E：從原始碼
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

1. **提供 OAuth 憑證** — 到 [Google Cloud Console](https://console.cloud.google.com/apis/credentials) 建立 OAuth 2.0 Client ID（類型選 Desktop App），下載 JSON 檔案。兩種方式擇一：
   - **本地**：貼上檔案路徑（如 `~/Downloads/credentials.json`）
   - **VPS/遠端**：直接貼上 JSON 內容（以 `{` 開頭，自動偵測）
2. **選擇服務** — 預設全選（Gmail, Calendar, Drive, Docs, Sheets, Tasks, Contacts, Chat, Analytics, Search Console），直接按 Enter
3. **登入** — 三種模式：
   - **(b)rowser**（預設）：自動開瀏覽器完成授權
   - **(m)anual**：啟動 localhost redirect，手動複製 URL
   - **(r)emote**：VPS 專用 — 在本機瀏覽器開 URL、授權後複製 redirect URL 貼回終端

> **VPS 用戶推薦流程**：選 `r`（remote）。授權後瀏覽器會顯示「無法連線」，這是正常的。複製瀏覽器網址列的完整 URL，貼回 VPS 終端即可。

完成後，OAuth token 存在作業系統的 Keyring（macOS Keychain / Linux Secret Service / Windows Credential Manager），**不會寫到檔案裡**。

驗證認證狀態：
```bash
gwx auth status
```

---

## 人類使用指南

你在終端機下 `gwx <service> <command>` 就能操作。

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

```bash
gwx ls                    # = gwx drive list
gwx search "keyword"      # = gwx gmail search
gwx send --to a@b.com ... # = gwx gmail send
gwx find "topic"          # = 同時搜 Gmail + Drive + Contacts
gwx context "project"     # = 彙整 Gmail + Drive + Calendar 的相關上下文
```

### 各服務指令速查

#### Gmail（11 個指令）

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
gwx gmail label "from:github" --add CI --remove INBOX       # 批量標籤
gwx gmail forward MESSAGE_ID --to colleague@co.com          # 轉發信件
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

#### Analytics（4 個指令）

```bash
gwx analytics report --metrics sessions,activeUsers --dimensions date   # GA4 報表查詢
    # --start-date 7daysAgo --end-date today --limit 100 --property properties/123
gwx analytics realtime --metrics activeUsers                            # 即時數據
gwx analytics properties                                                # 列出 GA4 Property
gwx analytics audiences --property properties/123                       # 列出受眾
```

> 首次使用前設定預設 Property：`gwx config set analytics.default-property properties/123456`

#### Search Console（5 個指令）

```bash
gwx searchconsole query --start-date 2026-03-01                        # 搜尋成效
    # --dimensions query,page --query-filter "keyword" --limit 100 --site https://...
gwx searchconsole sites                                                 # 列出已驗證網站
gwx searchconsole inspect --site https://example.com URL                # 網址索引檢查
gwx searchconsole sitemaps --site https://example.com                   # 列出 Sitemap
gwx searchconsole index-status --site https://example.com               # 索引覆蓋率
```

> 首次使用前設定預設 Site：`gwx config set searchconsole.default-site https://example.com`

#### Slides（6 個指令）

```bash
gwx slides get PRESENTATION_ID                              # 讀取簡報結構
gwx slides list [--limit N]                                  # 列出簡報
gwx slides create --title "報告"                             # 建立簡報
gwx slides duplicate PRESENTATION_ID --title "副本"           # 複製簡報
gwx slides export PRESENTATION_ID --export-format pdf -o r.pdf  # 匯出 PDF/PPTX
gwx slides from-sheet --template ID --sheet-id ID --range "A:D" # 從 Sheet 套範本
```

#### Forms（3 個指令）

```bash
gwx forms get FORM_ID                                        # 讀取表單結構
gwx forms responses FORM_ID --limit 50                       # 列出所有回覆
gwx forms response FORM_ID RESPONSE_ID                       # 取得單筆回覆
```

#### BigQuery（4 個指令）

```bash
gwx bigquery query "SELECT * FROM dataset.table LIMIT 10" --project my-project   # 執行 SQL 查詢
gwx bigquery datasets --project my-project                                         # 列出資料集
gwx bigquery tables --project my-project --dataset my_dataset                      # 列出資料表
gwx bigquery describe my_table --project my-project --dataset my_dataset           # 資料表 schema
```

> 首次使用前設定預設 Project：`gwx config set bigquery.default-project my-project`

#### GitHub（10 個指令）

```bash
gwx github login --token ghp_xxx                             # 認證
gwx github logout                                            # 登出
gwx github status                                            # 認證狀態
gwx github repos --limit 10                                  # 列出 repo
gwx github issues owner/repo --state open                    # 列出 issue
gwx github pulls owner/repo                                  # 列出 PR
gwx github pull owner/repo 42                                # 取得 PR 詳情
gwx github runs owner/repo                                   # 列出 CI 執行
gwx github notify                                            # 列出通知
gwx github create issue owner/repo --title "Bug" --labels bug  # 建立 issue
```

#### Slack（7 個指令）

```bash
gwx slack login xoxb-xxx                                     # 認證
gwx slack status                                             # 認證狀態
gwx slack channels                                           # 列出頻道
gwx slack send "Hello" --channel "#general"                  # 發送訊息
gwx slack messages C01234567 --limit 20                      # 讀取歷史
gwx slack search "deploy error"                              # 搜尋
gwx slack users                                              # 列出成員
```

#### Notion（7 個指令）

```bash
gwx notion login ntn_xxx                                     # 認證
gwx notion status                                            # 認證狀態
gwx notion search "project plan"                             # 搜尋頁面
gwx notion page PAGE_ID                                      # 取得頁面
gwx notion create --parent DATABASE_ID --title "New item"    # 建立頁面
gwx notion databases                                         # 列出資料庫
gwx notion query DATABASE_ID --filter '{"property":"Status","select":{"equals":"Done"}}'  # 查詢
```

#### Workflow（13 個指令）

```bash
# 頂層命令（高頻）
gwx standup [--days N] [--execute --push chat:spaces/XXX]  # 每日站會報告
gwx meeting-prep "會議關鍵字" [--days N]                    # 會議準備資料

# gwx workflow 子命令群組
gwx workflow weekly-digest [--weeks N]                      # 每週摘要
gwx workflow context-boost "主題" [--days N] [--limit N]    # 主題上下文彙整
gwx workflow bug-intake [--bug-id "BUG-123"] [--after 2026/03/15]  # Bug 相關資料
gwx workflow test-matrix init --feature "功能名"             # 建立測試追蹤 Sheet
gwx workflow test-matrix sync --file results.json            # 同步測試結果
gwx workflow test-matrix stats                               # 測試統計
gwx workflow spec-health init --feature "功能名"             # Spec 品質追蹤 Sheet
gwx workflow spec-health record --spec-folder dev/specs/xxx  # 記錄 spec 狀態
gwx workflow sprint-board init --feature "Sprint Q2"         # 建立看板 Sheet
gwx workflow sprint-board ticket --title "修 Bug" --priority P1  # 新增 ticket
gwx workflow sprint-board stats                              # 看板統計
gwx workflow review-notify --spec-folder xxx --reviewers a@co.com  # 審查通知預覽
gwx workflow email-from-doc --doc-id XXX --recipients a@co.com     # Doc 轉 Email 預覽
gwx workflow sheet-to-email --sheet-id XXX --range "A:F" \
    --email-col 0 --subject-col 1 --body-col 2              # 批量 Email 預覽（上限 50 列）
gwx workflow parallel-schedule --title "Review" \
    --attendees a@co.com,b@co.com --duration 30m             # 排程 1-on-1 預覽
```

> 所有 workflow 預設輸出 JSON（唯讀模式）。加 `--execute` 才會真的執行動作（發信、建事件等）。MCP 工具（`workflow_standup` 等）永遠是唯讀，不會執行動作。

#### Pipeline（1 個指令）

```bash
gwx pipe "gmail search 'invoice' | sheets append SHEET_ID A:C"
    # 每個階段的 JSON 輸出自動傳給下一階段的 stdin
    # 用 | 分隔多個 gwx 子命令
```

#### Config（3 個指令）

```bash
gwx config set analytics.default-property properties/123456  # 設定偏好
gwx config get analytics.default-property                     # 讀取偏好
gwx config list                                               # 列出所有偏好
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

gwx 從設計之初就考慮了 AI Agent 的使用場景。有三種整合方式。

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

啟動後，Claude 可以直接呼叫 123 個 MCP tool。Agent 直接傳 JSON 參數呼叫，不需要組裝 CLI 指令字串。

#### MCP Tool 命名規則

CLI 指令對應到 MCP tool 的命名：`<service>_<command>`

| CLI 指令 | MCP Tool |
|---------|----------|
| `gwx gmail list` | `gmail_list` |
| `gwx gmail search` | `gmail_search` |
| `gwx gmail digest` | `gmail_digest` |
| `gwx gmail archive` | `gmail_archive` |
| `gwx gmail label` | `gmail_batch_label` |
| `gwx gmail forward` | `gmail_forward` |
| `gwx gmail reply` | `gmail_reply` |
| `gwx calendar agenda` | `calendar_agenda` |
| `gwx calendar list` | `calendar_list` |
| `gwx calendar find-slot` | `calendar_find_slot` |
| `gwx sheets read` | `sheets_read` |
| `gwx sheets describe` | `sheets_describe` |
| `gwx sheets smart-append` | `sheets_smart_append` |
| `gwx drive search` | `drive_search` |
| `gwx docs get` | `docs_get` |
| `gwx analytics report` | `analytics_report` |
| `gwx searchconsole query` | `searchconsole_query` |
| `gwx slides from-sheet` | `slides_from_sheet` |
| `gwx forms get` | `forms_get` |
| `gwx bigquery query` | `bigquery_query` |
| `gwx github repos` | `github_repos` |
| `gwx github issues` | `github_issues` |
| `gwx github pulls` | `github_pulls` |
| `gwx slack channels` | `slack_channels` |
| `gwx slack send` | `slack_send` |
| `gwx notion search` | `notion_search` |
| `gwx notion query` | `notion_query` |
| `gwx find` | `unified_search` |
| `gwx context` | `context_gather` |
| `gwx standup` | `workflow_standup` |
| `gwx config set` | `config_set` |

#### 完整 MCP 工具參考（123 tools）

##### Gmail（11 tools）

| 工具 | 說明 |
|------|------|
| `gmail_list` | 列出信件 |
| `gmail_get` | 讀取單封 |
| `gmail_search` | 搜尋 |
| `gmail_send` | 寄信 |
| `gmail_draft` | 建草稿 |
| `gmail_reply` | 回信 |
| `gmail_digest` | 智慧摘要 |
| `gmail_archive` | 批次封存 |
| `gmail_labels` | 列出標籤 |
| `gmail_batch_label` | 批量標籤 |
| `gmail_forward` | 轉發 |

##### Calendar（6 tools）

| 工具 | 說明 |
|------|------|
| `calendar_agenda` | 近期行程 |
| `calendar_list` | 日期範圍查詢 |
| `calendar_create` | 建立事件 |
| `calendar_update` | 修改事件 |
| `calendar_delete` | 刪除事件 |
| `calendar_find_slot` | 找空檔 |

##### Drive（7 tools）

| 工具 | 說明 |
|------|------|
| `drive_list` | 列出檔案 |
| `drive_search` | 搜尋 |
| `drive_upload` | 上傳 |
| `drive_download` | 下載 |
| `drive_share` | 分享 |
| `drive_mkdir` | 建資料夾 |
| `drive_batch_upload` | 批次上傳 |

##### Docs（8 tools）

| 工具 | 說明 |
|------|------|
| `docs_get` | 讀取文件 |
| `docs_create` | 建立文件 |
| `docs_append` | 追加內容 |
| `docs_search` | 搜尋文件 |
| `docs_replace` | 尋找取代 |
| `docs_template` | 範本套用 |
| `docs_from_sheet` | 從 Sheet 生成 |
| `docs_export` | 匯出 |

##### Sheets（16 tools）

| 工具 | 說明 |
|------|------|
| `sheets_read` | 讀取儲存格 |
| `sheets_info` | 基本資訊 |
| `sheets_describe` | 欄位結構分析 |
| `sheets_stats` | 欄位統計 |
| `sheets_search` | 搜尋內容 |
| `sheets_filter` | 篩選 |
| `sheets_diff` | 比較分頁 |
| `sheets_append` | 新增列 |
| `sheets_smart_append` | 驗證後新增 |
| `sheets_update` | 更新儲存格 |
| `sheets_clear` | 清空範圍 |
| `sheets_copy_tab` | 複製分頁 |
| `sheets_export` | 匯出 |
| `sheets_import` | 匯入 |
| `sheets_create` | 建立試算表 |
| `sheets_batch_append` | 批次新增 |

##### Tasks（5 tools）

| 工具 | 說明 |
|------|------|
| `tasks_list` | 列出待辦 |
| `tasks_lists` | 列出清單 |
| `tasks_create` | 建立待辦 |
| `tasks_complete` | 完成待辦 |
| `tasks_delete` | 刪除待辦 |

##### Contacts（3 tools）

| 工具 | 說明 |
|------|------|
| `contacts_list` | 列出聯絡人 |
| `contacts_search` | 搜尋 |
| `contacts_get` | 取得詳情 |

##### Chat（3 tools）

| 工具 | 說明 |
|------|------|
| `chat_spaces` | 列出空間 |
| `chat_send` | 發送訊息 |
| `chat_messages` | 讀取訊息 |

##### Analytics / GA4（4 tools）

| 工具 | 說明 |
|------|------|
| `analytics_report` | 報表查詢 |
| `analytics_realtime` | 即時數據 |
| `analytics_properties` | 列出 Property |
| `analytics_audiences` | 列出受眾 |

##### Search Console（5 tools）

| 工具 | 說明 |
|------|------|
| `searchconsole_query` | 搜尋成效 |
| `searchconsole_sites` | 列出已驗證網站 |
| `searchconsole_inspect` | 網址索引檢查 |
| `searchconsole_sitemaps` | 列出 Sitemap |
| `searchconsole_index_status` | 索引覆蓋率 |

##### Slides（6 tools）

| 工具 | 說明 |
|------|------|
| `slides_get` | 讀取簡報 |
| `slides_list` | 列出簡報 |
| `slides_create` | 建立簡報 |
| `slides_duplicate` | 複製簡報 |
| `slides_export` | 匯出 |
| `slides_from_sheet` | 從 Sheet 套範本 |

##### Forms（3 tools）

| 工具 | 說明 |
|------|------|
| `forms_get` | 讀取表單 |
| `forms_responses` | 列出回覆 |
| `forms_response` | 取得單筆回覆 |

##### BigQuery（4 tools）

| 工具 | 說明 |
|------|------|
| `bigquery_query` | 執行 SQL 查詢 |
| `bigquery_datasets` | 列出資料集 |
| `bigquery_tables` | 列出資料表 |
| `bigquery_describe` | 資料表 schema |

##### GitHub（7 tools）

| 工具 | 說明 |
|------|------|
| `github_repos` | 列出 repo |
| `github_issues` | 列出 issue |
| `github_create_issue` | 建立 issue |
| `github_pulls` | 列出 PR |
| `github_pull` | 取得 PR 詳情 |
| `github_runs` | 列出 CI 執行 |
| `github_notifications` | 列出通知 |

##### Slack（6 tools）

| 工具 | 說明 |
|------|------|
| `slack_channels` | 列出頻道 |
| `slack_send` | 發送訊息 |
| `slack_messages` | 讀取歷史 |
| `slack_search` | 搜尋 |
| `slack_users` | 列出成員 |
| `slack_user` | 取得成員詳情 |

##### Notion（5 tools）

| 工具 | 說明 |
|------|------|
| `notion_search` | 搜尋頁面 |
| `notion_page` | 取得頁面 |
| `notion_create_page` | 建立頁面 |
| `notion_databases` | 列出資料庫 |
| `notion_query` | 查詢資料庫 |

##### Config（3 tools）

| 工具 | 說明 |
|------|------|
| `config_set` | 設定偏好 |
| `config_get` | 讀取偏好 |
| `config_list` | 列出所有偏好 |

##### 跨服務（2 tools）

| 工具 | 說明 |
|------|------|
| `unified_search` | 跨服務搜尋 |
| `context_gather` | 上下文彙整 |

##### Workflow（19 tools）

| 工具 | 說明 |
|------|------|
| `workflow_standup` | 站會報告 |
| `workflow_meeting_prep` | 會議準備資料 |
| `workflow_weekly_digest` | 每週摘要 |
| `workflow_context_boost` | 主題上下文彙整 |
| `workflow_bug_intake` | Bug 相關資料 |
| `workflow_test_matrix_init` | 建立測試追蹤 Sheet |
| `workflow_test_matrix_sync` | 同步測試結果 |
| `workflow_test_matrix_stats` | 測試統計 |
| `workflow_spec_health_init` | Spec 品質追蹤 Sheet |
| `workflow_spec_health_record` | 記錄 spec 狀態 |
| `workflow_spec_health_stats` | Spec 統計 |
| `workflow_sprint_board_init` | 建立看板 Sheet |
| `workflow_sprint_board_ticket` | 新增 ticket |
| `workflow_sprint_board_stats` | 看板統計 |
| `workflow_review_notify` | 審查通知 |
| `workflow_email_from_doc` | Doc 轉 Email |
| `workflow_sheet_to_email` | 批量 Email |
| `workflow_parallel_schedule` | 排程 1-on-1 |
| `workflow_digest` | 信件摘要 |

> 所有 `workflow_*` 工具為唯讀，不會執行寫入動作。

##### 偽工具（用於 Skill DSL）

| 工具 | 說明 |
|------|------|
| `transform` | 本地資料轉換（pick, flatten, sort_by, limit, count） |

### 方式 B：Bash 工具呼叫

如果你的 Agent 沒有 MCP 支援（例如 Codex、GPT、自建 Agent），可以直接用 Bash 呼叫 gwx CLI。

#### Agent 友善設計

**1. 結構化 JSON 輸出**

所有指令預設輸出 JSON，格式固定：

```json
// 成功
{"status": "ok", "data": { ... }}

// 失敗（輸出到 stderr）
{"status": "error", "error": {"code": 10, "name": "auth_required", "message": "..."}}
```

**2. 穩定的 Exit Code**

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

```bash
gwx schema    # 回傳所有指令的 schema（name, description, safety_tier, example_args）
```

**4. 非 TTY 自動 JSON**

```bash
export GWX_AUTO_JSON=1
```

**5. 無互動模式**

```bash
gwx gmail list --no-input
```

### 方式 C：Claude Code Skill（深度整合）

```bash
./install.sh             # 全域安裝（~/.claude/commands/）
./install.sh --project   # 專案級安裝（.claude/commands/）
```

安裝內容：
- 1 個主 Skill（`google-workspace.md`）— 指令路由 + 安全分級
- 4 個 Agent（`gmail-agent.md`、`calendar-agent.md`、`drive-agent.md`、`workspace-router.md`）
- Workflow Recipe 文件

安裝後，在 Claude Code 對話中用自然語言就能觸發：

```
你：看一下我的信
Claude：（自動呼叫 gwx gmail list）

你：明天有什麼會
Claude：（自動呼叫 gwx calendar agenda --days 1）
```

#### Combo Skills — 跨服務工作流

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

## Skill DSL

### 概覽

Skill DSL 讓你用 YAML 定義多步驟工作流 — 不需要寫 Go、不需要重新編譯。每個 skill 就是一個 `.yaml` 檔案，描述一系列 MCP tool 呼叫，依序（或並行）執行，步驟之間可以互相引用輸出。

Skills 自動從兩個目錄載入：

| 目錄 | 用途 | 優先級 |
|------|------|--------|
| `~/.config/gwx/skills/` | 個人全域 skills | 低（被專案覆蓋） |
| `./skills/` | 專案本地 skills | 高（覆蓋全域同名 skill） |

載入後的 skills 自動註冊為 MCP 工具（名稱格式：`skill_<name>`），Claude 可直接呼叫。

### YAML 格式完整參考

```yaml
# === 基本資訊 ===
name: my-skill              # (必填) skill 名稱，也是 MCP tool 名稱的一部分 → skill_my-skill
version: "1.0"               # (選填) 版本號
description: "做什麼用的"     # (選填) 描述，會顯示為 MCP tool 的 description

# === 輸入參數 ===
inputs:                       # (選填) 使用者傳入的參數列表
  - name: query               # (必填) 參數名稱
    type: string              # (選填) string | int | bool，預設 string
    required: true            # (選填) 是否必填，預設 false
    default: "keyword"        # (選填) 預設值（required + 無 default → 執行時必須提供）
    description: "搜尋關鍵字"  # (選填) 參數說明

# === 執行步驟 ===
steps:                        # (必填) 至少一個步驟
  - id: step1                 # (選填) 步驟 ID，省略會自動產生 step_1, step_2...
    tool: gmail_search        # (必填) 要呼叫的 MCP tool 名稱，或 "transform" / "skill:<name>"
    args:                     # (選填) 傳給 tool 的參數，值可用模板表達式
      query: "{{.input.query}}"
      limit: "10"
    store: emails             # (選填) 結果存放的 key，預設用 id
    on_fail: skip             # (選填) 失敗處理：skip（跳過繼續）| abort（中斷，預設）
    parallel: true            # (選填) 與相鄰的 parallel 步驟並行執行
    each: "{{.steps.data}}"   # (選填) 迭代列表，對每個元素執行一次
    if: "{{.steps.check}}"    # (選填) 條件式，表達式為 falsy 時跳過此步驟

# === 輸出 ===
output: "{{.steps.step1}}"   # (選填) 最終輸出模板，省略則回傳最後一個步驟的結果

# === 元資料 ===
meta:                         # (選填) 任意 key-value，用於分類、標記
  author: gwx
  category: finance
```

### 模板表達式

所有 `args` 值和 `output` 都支援 `{{...}}` 模板語法：

| 語法 | 說明 |
|------|------|
| `{{.input.name}}` | 使用者輸入值 |
| `{{.steps.id}}` | 步驟 `id` 的完整輸出 |
| `{{.steps.id.field}}` | 步驟輸出中的特定欄位 |
| `{{.steps.id.field.nested}}` | 多層巢狀存取 |
| `{{.item}}` | `each` 迴圈中的當前元素（完整物件） |
| `{{.item.field}}` | `each` 迴圈中當前元素的欄位 |

```yaml
# 引用輸入參數
args:
  query: "{{.input.keyword}}"

# 引用前一步驟的輸出
args:
  input: "{{.steps.search}}"

# 在 each 迴圈中引用當前元素
each: "{{.steps.contacts}}"
args:
  to: "{{.item.email}}"
  subject: "Hello {{.item.name}}"
```

當整個值是單一模板（如 `"{{.steps.data}}"`），引擎會保留原始型別（map、array）。混合文字 + 模板（如 `"Hello {{.item.name}}"`）會被轉為字串。

### 功能詳解

#### 並行執行

連續標記 `parallel: true` 的步驟會同時執行：

```yaml
steps:
  - id: emails
    tool: gmail_search
    args: { query: "{{.input.query}}" }
    parallel: true

  - id: files
    tool: drive_search
    args: { query: "{{.input.query}}" }
    parallel: true

  # 等上面兩個都完成後才執行
  - id: combine
    tool: transform
    args:
      input: "{{.steps.emails}}"
      pick: "from,subject"
```

並行步驟共用一份 store snapshot，不互相干擾。結果按定義順序合併回主 store。

> `parallel` 和 `each` 不能同時用在同一個步驟。

#### Each 迴圈

對列表中的每個元素重複執行一個步驟：

```yaml
steps:
  - id: contacts
    tool: sheets_read
    args:
      spreadsheet_id: "{{.input.sheet_id}}"
      range: "Contacts!A:C"

  - id: notify
    tool: gmail_send
    each: "{{.steps.contacts}}"
    args:
      to: "{{.item.email}}"
      subject: "Hi {{.item.name}}"
      body: "This is an automated update."
    on_fail: skip
```

`each` 的結果會收集成一個 array，存入 store。

#### Transform 偽工具

`transform` 不呼叫 MCP，而是在本地做資料轉換：

```yaml
- id: extract
  tool: transform
  args:
    input: "{{.steps.search}}"   # (必填) 要處理的資料
    pick: "from,subject,date"     # 只保留指定欄位
    flatten: "true"               # 攤平巢狀 array
    sort_by: "date"               # 依欄位排序（字串比較）
    limit: "10"                   # 只取前 N 筆
    count: "true"                 # 回傳元素數量（數字）
```

執行順序：`pick` → `flatten` → `sort_by` → `limit` → `count`。

#### 條件式執行

`if` 表達式為 falsy 時跳過步驟：

```yaml
steps:
  - id: check
    tool: gmail_search
    args: { query: "urgent" }

  - id: alert
    tool: chat_send
    if: "{{.steps.check.count}}"    # 0 或空 → falsy → 跳過
    args:
      space: "spaces/AAAA"
      text: "有緊急信件！"
```

Falsy 值：`nil`、空字串 `""`、`"false"`、`"0"`、`0`、空 array `[]`、空 map `{}`。其他都是 truthy。

#### Skill 組合

一個 skill 可以呼叫另一個 skill 作為步驟：

```yaml
steps:
  - id: brief
    tool: skill:google-morning-brief
    args:
      email-limit: "5"
```

最大嵌套深度 5 層。

#### 錯誤處理

```yaml
on_fail: skip    # 失敗時跳過，繼續執行後續步驟
on_fail: abort   # 失敗時中斷整個 skill（預設）
```

### CLI 指令

```bash
gwx skill list                      # 列出所有已載入的 skills
gwx skill inspect <name>            # 顯示 skill 的完整結構（inputs, steps, output, meta）
gwx skill validate <file>           # 驗證 YAML 檔案語法與結構
gwx skill run <name> -p key=value   # 執行 skill
gwx skill create <name>             # 建立新 skill 骨架
gwx skill test <name>               # 用 mock 資料測試
gwx skill install <url|path>        # 從本地檔案或 URL 安裝到 ~/.config/gwx/skills/
gwx skill remove <name>             # 從 ~/.config/gwx/skills/ 移除
```

### 內建 Skills（19 個）

#### Google 系列（8 個）

| Skill | 說明 |
|-------|------|
| `google-morning-brief` | 每日簡報 — 未讀信 + 行程 + 待辦 |
| `google-seo-daily` | SEO 每日快照 — Search Console + GA4，結果寫入 Sheets |
| `google-invoice-log` | 搜尋 Gmail 發票信件，記錄到 Sheet |
| `google-meeting-notes` | 抓取近期會議，產生 Google Doc 會議記錄範本 |
| `google-contact-export` | 匯出聯絡人到 Google Sheet |
| `google-doc-from-sheet` | 從 Sheet 資料生成 Google Doc |
| `google-bq-to-sheet` | 執行 BigQuery SQL，結果寫入 Sheet |
| `google-task-report` | 待辦事項報告 — 列出所有清單與待辦 |

#### GitHub 系列（3 個）

| Skill | 說明 |
|-------|------|
| `github-ci-alert` | CI 失敗警報 — 檢查 GitHub Actions，發 Slack/Email 通知 |
| `github-issue-to-sheet` | 匯出 GitHub Issue 到 Google Sheet |
| `github-pr-to-slack` | 待審 PR 摘要推送到 Slack |

#### Cross-platform 系列（8 個）

| Skill | 說明 |
|-------|------|
| `cross-client-360` | 客戶 360 度視圖 — 信件 + Drive 檔案 + 聯絡人 |
| `cross-daily-report` | 綜合日報 — morning-brief + seo-daily |
| `cross-full-context` | 全平台搜尋 — Gmail + Drive + Slack + Notion + GitHub 並行 |
| `cross-github-standup` | 開發者站會 — GitHub PR + 未讀信 + 行程 |
| `cross-weekly-sync` | 自動週報 — GA4 + 信件摘要 + 行程 + GitHub PR |
| `cross-form-to-slack` | Google Forms 回覆通知到 Slack |
| `cross-notion-to-sheet` | Notion 資料庫同步到 Google Sheet |
| `cross-onboard-checklist` | 新人到職 — 建 Drive 資料夾 + 歡迎文件 + 排會議 + Slack 通知 |

### 建立 Skill — 實戰範例

目標：建立一個 skill，搜尋 Gmail 中的發票信件，同時搜尋 Drive 中的相關檔案，合併結果。

**Step 1：建立 YAML 檔案**

```yaml
# skills/invoice-finder.yaml
name: invoice-finder
version: "1.0"
description: "搜尋 Gmail 和 Drive 中的發票相關資料"

inputs:
  - name: keyword
    type: string
    required: true
    description: "搜尋關鍵字（如：發票、invoice）"
  - name: limit
    type: int
    default: "10"
    description: "每個服務的最大結果數"

steps:
  # 並行搜尋 Gmail 和 Drive
  - id: emails
    tool: gmail_search
    args:
      query: "{{.input.keyword}} has:attachment"
      limit: "{{.input.limit}}"
    parallel: true

  - id: files
    tool: drive_search
    args:
      query: "name contains '{{.input.keyword}}'"
      limit: "{{.input.limit}}"
    parallel: true
    on_fail: skip

  # 從 email 結果只取關鍵欄位
  - id: summary
    tool: transform
    args:
      input: "{{.steps.emails}}"
      pick: "from,subject,date"
      sort_by: "date"
      limit: "5"

output: "{{.steps.summary}}"
```

**Step 2：驗證**

```bash
gwx skill validate skills/invoice-finder.yaml
# → {"valid": true, "name": "invoice-finder", "inputs": 2, "steps": 3}
```

**Step 3：使用**

放在 `./skills/` 或 `~/.config/gwx/skills/` 即可。重新啟動 MCP server 後，Claude 可直接呼叫 `skill_invoice-finder`。

```bash
gwx skill run invoice-finder -p keyword=invoice -p limit=5
```

---

## 多供應商認證

gwx 支援四個供應商，每個供應商的 token 獨立存放在 OS keyring。

### Google（OAuth 流程）

```bash
gwx onboard                       # 互動式設定精靈
gwx auth status                    # 檢查狀態
gwx auth login                     # 重新登入
gwx auth logout                    # 登出
```

CI/CD 環境：
```bash
# 方式 A：環境變數提供 OAuth JSON
export GWX_OAUTH_JSON='{"installed":{"client_id":"...","client_secret":"..."}}'
gwx onboard                       # 自動偵測 env var，使用 remote auth

# 方式 B：直接提供 access token（跳過 OAuth）
export GWX_ACCESS_TOKEN="ya29.xxx"
```

### GitHub

```bash
gwx github login --token ghp_xxx  # Personal Access Token
gwx github status                  # 檢查狀態
gwx github logout                  # 登出
```

### Slack

```bash
gwx slack login xoxb-xxx          # Bot Token
gwx slack status                   # 檢查狀態
```

### Notion

```bash
gwx notion login ntn_xxx          # Integration API Key
gwx notion status                  # 檢查狀態
```

所有 token 存在 OS Keyring（macOS Keychain / Linux Secret Service / Windows Credential Manager），每個供應商使用獨立的 keyring entry，不會互相影響。

---

## Shell 自動補全

```bash
# Bash
eval "$(gwx completion bash)"

# Zsh（加到 ~/.zshrc）
eval "$(gwx completion zsh)"

# Fish
gwx completion fish | source
```

---

## 健康檢查

```bash
gwx doctor
```

一次檢查所有狀態：
- gwx 版本、OS、Go 版本
- Config 目錄狀態
- 各供應商認證狀態（Google / GitHub / Slack / Notion）
- 已載入的 skills 數量

---

## 安全機制

### 操作安全分級

每個指令都有安全等級，同時影響人類使用和 Agent 行為：

| 等級 | 類型 | 行為 | 範例 |
|------|------|------|------|
| Green | 唯讀 | 直接執行 | `gmail list`、`sheets read`、`calendar agenda` |
| Yellow | 建立/修改 | 確認後執行 | `calendar create`、`sheets append`、`docs create` |
| Red | 破壞性/對外發送 | 必須明確同意 | `gmail send`、`drive share`、`calendar delete` |
| Blocked | 永久刪除 | 永遠不執行 | 永久刪除、ownership transfer |

Agent 在使用 Skill 時，會自動遵守這個分級：
- Green：直接執行，不問
- Yellow：先顯示操作摘要，等使用者確認
- Red：顯示完整細節（收件人、主旨、全文），等使用者明確同意

### Token 安全

- OAuth token 存在 OS Keyring，**永遠不會寫到磁碟檔案**
- 多供應商 token 互相隔離，使用獨立 keyring entry
- CSRF 保護：OAuth flow 使用 128-bit crypto/rand state
- CI/CD 環境可用環境變數：`export GWX_ACCESS_TOKEN="ya29.xxx"`

### 輸入安全

- Sheets 公式注入防護：自動轉義 `=`、`+`、`-`、`@` 開頭的值
- Drive 查詢注入防護：Folder ID 驗證
- 附件大小限制：25MB

### 穩定性

- Rate Limiter：每個服務獨立的 token bucket（Sheets 0.8 QPS、Gmail 4 QPS、Drive 8 QPS、Analytics 2 QPS、Search Console 2 QPS）
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
gwx auth status          # 檢查認證狀態
gwx auth login           # 重新認證
gwx onboard              # 完整重設
```

**Q: `permission_denied` 錯誤（exit code 12）**

可能是初次 onboard 時沒有授權所有服務。重跑 `gwx onboard` 並選擇需要的服務。

**Q: MCP Server 連不上**

```bash
which gwx                # 確認 gwx 在 PATH 裡
gwx mcp-server           # 手動測試能否啟動（Ctrl+C 結束）
```

**Q: Rate limit 錯誤（exit code 30）**

gwx 內建了 rate limiter 和自動重試，正常情況下你不會看到這個。如果看到了，代表你的 Google API quota 真的用完了。等幾分鐘再試。

### 移除 gwx

```bash
./uninstall.sh
```

移除 binary 和 Claude Code skill 檔案。OAuth token 留在 OS Keyring 裡，要手動移除的話先跑 `gwx auth logout`。

---

## 授權

MIT License. 詳見 [LICENSE](LICENSE)。
