# Pitfalls Registry

> 自動追加於 S5/S6/S7 階段。詳見 sop-full-spec.md 知識管理章節。

## new-service-key-mismatch
- **Tags**: go, api, auth, ratelimiter
- **Severity**: P1
- **Description**: 新增 Google 服務時，API 層 `WaitRate(ctx, "service")` 和 `ClientOptions(ctx, "service")` 的 service key 必須與 `auth/scopes.go` 的 `ServiceScopes` map key 和 `ratelimiter.go` 的 `defaultRates` map key 完全一致。否則 rate limiter 和 OAuth scope 無法生效。
- **Source**: GA4 & Google Search Console 整合 (2026-03)

## mcp-cli-validation-parity
- **Tags**: mcp, cli, validation
- **Severity**: P1
- **Description**: MCP handler 必須比照對應 CLI 命令做相同的輸入驗證（必填參數空值檢查、數值範圍限制）。不能依賴底層 API 回傳的晦澀錯誤。
- **Source**: GA4 & Google Search Console 整合 (2026-03)
