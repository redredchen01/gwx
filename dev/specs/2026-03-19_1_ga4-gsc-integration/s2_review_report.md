# S2 審查報告：GA4 & Google Search Console 整合

> 本文件由 S2 Spec Review 自動產出，記錄對 `s1_dev_spec.md` 的完整審查結果與修正軌跡。

## 審查摘要

| 項目 | 內容 |
|------|------|
| 審查模式 | Full Spec 對抗式審查（含 Phase 0 預審） |
| 引擎 | Codex (R1) + Claude architect (R2) + Claude Sonnet (R3) |
| 結論 | conditional_pass → 修正後 pass |
| R1 Short-Circuit | 否（P1=6，進入 R2+R3） |
| 審查日期 | 2026-03-19 |

---

## 預審摘要（Phase 0）

| 統計 | 數量 |
|------|------|
| P0 | 0（SR-PRE-008 路徑問題降為 P1 合併至 R1） |
| P1 | 2（scope 語意、macOS 路徑） |
| P2 | 3（go get 描述、flatten 邏輯、kong tag） |

預審獨有發現（SR-PRE-008 config.Dir() macOS 路徑）已合併至 R1 的 SR-P1-006。

---

## 問題清單與處置

### P0 問題（設計層）

| # | 問題描述 | R2 回應 | R3 裁決 | 處置 |
|---|---------|---------|---------|------|
| - | 無 P0 問題 | - | - | - |

### P1 問題（實作層）

| # | 問題描述 | R2 回應 | R3 裁決 | 處置 |
|---|---------|---------|---------|------|
| SR-P1-001 | OAuth scope 升級機制假設 auto-upgrade 但 EnsureAuth 不檢查 scope | 接受 | ✅ 接受 | 已修正 §3.4：改為明確要求 gwx auth login + 403 錯誤處理 |
| SR-P1-002 | Config 命令缺 MCP 工具但成功標準承諾全覆蓋 | 部分接受：建議補 T-13 | ⚠️ 部分接受 | 已新增 T-13 MCP config 工具任務 |
| SR-P1-003 | Sitemap scope 矛盾（list vs submit/管理） | 部分接受：矛盾在 s0 | ✅ 接受 | 已修正 s0 流程圖 + dev_spec 成功標準統一為「列出」 |
| SR-P1-004 | Circuit Breaker 無實作落點 | **反駁**：CB 已自動套用 | ✅ 接受反駁 | 已補 §3.5 CB 說明，確認自動繼承 |
| SR-P1-005 | Config 損壞風險 DoD 缺口 | 接受 | ✅ 接受 | 已補 T-01 DoD：malformed JSON 不 panic |
| SR-P1-006 | config.Dir() macOS 路徑不正確 | 接受 | ✅ 接受 | 已改為泛化描述 `config.Dir()/preferences.json` |

### P2 建議（改善）

| # | 建議描述 | R2 回應 | R3 裁決 | 是否採納 |
|---|---------|---------|---------|----------|
| SR-P2-001 | --services vs --scope 旗標不一致 | 接受 | ✅ | 已統一為 --services |
| SR-P2-002 | T-12 go get 描述誤導 | 部分接受 | ✅ | 已改為 go build 驗證 |
| SR-P2-003 | ListProperties 需 flatten AccountSummaries | 接受 | ✅ | 已補 T-04 flatten 說明 |

---

## s1_dev_spec.md 修正摘要

| 修正項 | 修正前（摘要） | 修正後（摘要） | 對應問題 |
|--------|--------------|--------------|----------|
| §3.4 OAuth 策略 | 假設 OAuth2 auto-upgrade | 明確要求 gwx auth login + 403 處理 | SR-P1-001 |
| 新增 §3.5 | 無 | CB 自動繼承說明 | SR-P1-004 |
| 新增 T-13 | 無 | MCP config 工具任務 | SR-P1-002 |
| T-01 DoD | 只覆蓋檔案不存在 | 補 malformed JSON 不 panic | SR-P1-005 |
| 全文路徑 | ~/.config/gwx | config.Dir()/preferences.json | SR-P1-006 |
| T-04 描述 | 未說明 flatten | 補 AccountSummaries flatten + pageToken | SR-P2-003 |
| T-12 描述 | go get 子包 | go build 驗證 import | SR-P2-002 |
| s0 流程圖 | Sitemaps.List/Submit | Sitemaps.List: Sitemap 列表 | SR-P1-003 |
| s0 成功標準 | 能管理 Sitemap | 能列出 Sitemap | SR-P1-003 |

---

## 完整性評分

| 檢查項目 | 評等 | 備註 |
|---------|------|------|
| 任務清單 & DoD | B | 修正後 13 個任務，DoD 已補 malformed JSON |
| 驗收標準 | B | 15 個 AC，覆蓋 happy + error path |
| 技術決策 | A | 5 個決策有方案比較，OAuth 策略已修正 |
| User/Data Flow | B | Mermaid 圖完整，CB 說明已補 |
| 影響範圍 | A | 8 個新增 + 9 個修改檔案，CB 確認自動繼承 |
| 風險評估 | B | 6 個風險 + 4 個回歸風險，DoD 已承接 |
| Codebase 一致性 | A | Phase 4 交叉驗證 0 Critical |

---

## 審查軌跡

| Round | 引擎/Agent | 中介檔案路徑 |
|-------|-----------|-------------|
| Phase 0 預審 | codebase-explorer (Sonnet) | （inline） |
| R1 挑戰 | Codex (gpt-5.4) | `/tmp/adversarial-review/20260319_104536/r1_findings.md` |
| R2 防禦 | architect (Sonnet) | `/tmp/adversarial-review/20260319_104536/r2_defense.md` |
| R3 裁決 | Sonnet | `/tmp/adversarial-review/20260319_104536/r3_verdict.md` |
