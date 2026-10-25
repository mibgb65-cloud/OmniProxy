import { describe, expect, it } from "vitest"
import { isTransientQuotaError, QUOTA_KEEP_LAST_GOOD_MS, retainQuota } from "./quota-retention"

const quota = { plan: "plus", fetchedAt: "2026-10-25T00:00:00Z" }
const now = Date.parse("2026-10-25T00:20:00Z")
const recent = now - 60_000
const expired = now - QUOTA_KEEP_LAST_GOOD_MS

describe("额度保留策略", () => {
  it("成功查询原样返回数据，不标记保留", () => {
    expect(
      retainQuota({ data: quota, dataUpdatedAt: recent, isError: false, error: null, now }),
    ).toEqual({ data: quota, error: null, retained: false })
  })

  it("瞬时失败在窗口内沿用上次成功结果并隐藏错误", () => {
    const error = new Error("subscription: quota_timeout private-quota-marker")
    expect(retainQuota({ data: quota, dataUpdatedAt: recent, isError: true, error, now })).toEqual({
      data: quota,
      error: null,
      retained: true,
    })
  })

  it("瞬时失败超出窗口时不再保留，错误重新透出", () => {
    const error = new Error("subscription: quota_busy")
    expect(retainQuota({ data: quota, dataUpdatedAt: expired, isError: true, error, now })).toEqual(
      { data: quota, error, retained: false },
    )
  })

  it("quota_unavailable 属于瞬时失败：窗口内保留旧额度，超窗口才透出", () => {
    // 生产日志里用户反复看到的“额度查询失败”就是这一码（上游 request_failed）。
    const error = new Error("subscription: quota_unavailable private-quota-marker")
    expect(retainQuota({ data: quota, dataUpdatedAt: recent, isError: true, error, now })).toEqual({
      data: quota,
      error: null,
      retained: true,
    })
    expect(retainQuota({ data: quota, dataUpdatedAt: expired, isError: true, error, now })).toEqual(
      { data: quota, error, retained: false },
    )
  })

  it("确定性失败即使数据很新也立即透出", () => {
    const error = new Error("subscription: quota_not_authenticated")
    expect(retainQuota({ data: quota, dataUpdatedAt: recent, isError: true, error, now })).toEqual({
      data: quota,
      error,
      retained: false,
    })
  })

  it("没有上次成功数据时无法保留，错误照原样返回", () => {
    const error = new Error("subscription: quota_timeout")
    expect(retainQuota({ data: undefined, dataUpdatedAt: 0, isError: true, error, now })).toEqual({
      data: undefined,
      error,
      retained: false,
    })
  })

  it("自定义窗口按传入的 keepMs 判定", () => {
    const error = new Error("subscription: operation_timeout")
    expect(
      retainQuota({
        data: quota,
        dataUpdatedAt: recent,
        isError: true,
        error,
        now,
        keepMs: 30_000,
      }),
    ).toMatchObject({ retained: false })
  })

  it.each([
    ["quota_timeout", true],
    ["quota_busy", true],
    ["quota_refresh_failed", true],
    ["operation_timeout", true],
    ["quota_unavailable", true],
    ["quota_not_authenticated", false],
    ["identity_changed", false],
    ["invalid_account", false],
    ["codex_unavailable", false],
    ["quota_cache_unavailable", false],
  ])("把 %s 归类为瞬时失败：%s", (code, transient) => {
    expect(isTransientQuotaError(new Error(`subscription: ${code}`))).toBe(transient)
  })

  it("无法识别的错误按确定性失败处理", () => {
    expect(isTransientQuotaError(new Error("private-authorization-value"))).toBe(false)
    expect(isTransientQuotaError(null)).toBe(false)
  })
})
