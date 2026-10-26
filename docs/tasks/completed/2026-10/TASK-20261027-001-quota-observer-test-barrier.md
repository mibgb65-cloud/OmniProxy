# TASK-20261027-001: 修复额度观察测试完成屏障

## Status

Completed

## Created

2026-10-27（Asia/Singapore）

## Request

正式发布收尾核验发现并发 main CI 40365457327 的额度观察测试竞态，需恢复可靠质量门，
不得跳过测试或改动已公开版本。

## Analysis / Root Cause

`TestObserveQuotaFeedsGatewayAvailability` 在假客户端 RateLimits 返回前关闭 done，
但生产 readQuota 随后才执行 normalize／publishQuota／rememberQuota。测试收到信号后立即查网关，
可能抢在异步状态发布之前；这不是业务承诺同步刷新失败。
readQuota 的 defer client.Close 在 publishQuota 返回后执行，可作为现有完成屏障，不需要改生产逻辑。

## Implementation Plan

1. 用仅测试的客户端 Close 屏障替换“读取结束”信号。
2. 保留按需刷新和网关阻塞断言，不改超时、不删除测试。
3. 重复定向测试并执行包测试，正常提交 main，不移动 v2.0.0 标签或替换资产。

## Implementation Result

仅增加测试客户端 Close 屏障，等待 normalize／publishQuota 后再检查网关状态；
原有业务断言保留，没有改生产代码、延长超时、移动标签或替换公开文件。

## Verification

远端失败：https://github.com/mibgb65-cloud/OmniProxy/actions/runs/40365457327 。
同源码的正式 Release 40366364069 完整质量门成功；该偶发失败需要独立修复，而不是以重跑隐藏。
- 定向 `TestObserveQuotaFeedsGatewayAvailability` 无缓存重复 100 次：PASS。
- `go test ./internal/application/subscription -count=1`：PASS。
- `gofmt`／`git diff --check`：PASS。主分支推送后由同一完整 CI 门禁复验。

## Git

`5d9dac4df30cfef37cb93a1b8b213c2c9e1b1daa` —
`test(subscription): 用额度发布完成屏障消除竞态`。仅测试与记录，不改 v2.0.0 标签源码及公开资产。

## Result

测试同步点已修正，生产行为与 v2.0.0 标签／资产不变。发布验证及测试稳定性记录分开保留。

## Completed

2026-10-27（Asia/Singapore）
