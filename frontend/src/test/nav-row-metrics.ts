import { vi } from "vitest"

const DEFAULT_ROW_HEIGHT = 36
const DEFAULT_ROW_STRIDE = 40

/**
 * jsdom 不计算布局（`offsetTop`／`offsetHeight` 恒为 0）。
 * 这里按行序号给出确定的行高与偏移，让「滑动药丸」的位移可以被断言。
 *
 * @returns 恢复原始描述的清理函数
 */
export function stubNavRowMetrics(rowHeight = DEFAULT_ROW_HEIGHT, rowStride = DEFAULT_ROW_STRIDE) {
  const rowIndexOf = (element: HTMLElement) => {
    const item = element.closest("li")
    const list = item?.parentElement
    if (!item || !list) return 0
    return Array.prototype.indexOf.call(list.children, item)
  }

  const top = vi.spyOn(HTMLElement.prototype, "offsetTop", "get").mockImplementation(function (
    this: HTMLElement,
  ) {
    return rowIndexOf(this) * rowStride
  })
  const height = vi
    .spyOn(HTMLElement.prototype, "offsetHeight", "get")
    .mockImplementation(function (this: HTMLElement) {
      return this.tagName === "A" || this.tagName === "LI" ? rowHeight : 0
    })

  return () => {
    top.mockRestore()
    height.mockRestore()
  }
}
