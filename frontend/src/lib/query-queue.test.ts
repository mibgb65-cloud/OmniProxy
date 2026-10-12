import { describe, expect, it, vi } from "vitest"
import { createQueryQueue } from "./query-queue"

describe("query queue", () => {
  it("limits real concurrency and advances after a failed request", async () => {
    const run = createQueryQueue(1)
    let fail!: (error: Error) => void
    const first = run(
      () =>
        new Promise<void>((_, reject) => {
          fail = reject
        }),
    )
    const secondOperation = vi.fn(async () => 2)
    const second = run(secondOperation)
    await vi.waitFor(() => expect(fail).toBeDefined())
    expect(secondOperation).not.toHaveBeenCalled()
    const rejected = expect(first).rejects.toThrow("failure")
    fail(new Error("failure"))
    await rejected
    expect(await second).toBe(2)
  })

  it("does not execute a cancelled queued operation or release a running RPC early", async () => {
    const run = createQueryQueue(1)
    const runningSignal = new AbortController()
    let finish!: () => void
    const first = run(
      () =>
        new Promise<void>((resolve) => {
          finish = resolve
        }),
      runningSignal.signal,
    )
    const waitingSignal = new AbortController()
    const operation = vi.fn(async () => 2)
    const second = run(operation, waitingSignal.signal)
    const cancelled = expect(second).rejects.toMatchObject({ name: "AbortError" })
    waitingSignal.abort()
    await cancelled
    await vi.waitFor(() => expect(finish).toBeDefined())
    runningSignal.abort()
    const third = run(operation)
    expect(operation).not.toHaveBeenCalled()
    finish()
    await first
    expect(await third).toBe(2)
    expect(operation).toHaveBeenCalledTimes(1)
  })

  it("starts two operations at a time and rejects already cancelled work", async () => {
    const run = createQueryQueue(2)
    const finish: Array<() => void> = []
    const operation = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          finish.push(resolve)
        }),
    )
    const jobs = [run(operation), run(operation), run(operation)]
    await vi.waitFor(() => expect(operation).toHaveBeenCalledTimes(2))
    finish[0]!()
    await vi.waitFor(() => expect(operation).toHaveBeenCalledTimes(3))
    finish[1]!()
    finish[2]!()
    await Promise.all(jobs)
    const cancelled = new AbortController()
    cancelled.abort()
    await expect(run(operation, cancelled.signal)).rejects.toMatchObject({ name: "AbortError" })
    expect(operation).toHaveBeenCalledTimes(3)
  })
})
