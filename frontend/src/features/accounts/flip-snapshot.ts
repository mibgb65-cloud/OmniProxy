export type FlipSnapshot = { rect: DOMRect; content: HTMLElement }

/**
 * 保留旧画面的只读副本，放在滚动裁剪之外过渡。
 * 连点时冻结屏幕上正在播放的层，而不是重新取已落位但不可见的真实节点。
 */
export function takeFlipSnapshot(source: HTMLElement): FlipSnapshot {
  const rect = source.getBoundingClientRect()
  const content = source.cloneNode(true) as HTMLElement
  const originals = [source, ...source.querySelectorAll<HTMLElement>("*")]
  const copies = [content, ...content.querySelectorAll<HTMLElement>("*")]
  originals.forEach((original, index) => {
    const copy = copies[index]
    copy.removeAttribute("id")
    copy.removeAttribute("autofocus")
    copy.removeAttribute("data-flip-id")
    copy.removeAttribute("data-flip-flying")
    copy.removeAttribute("data-motion-id")
    if (original.getAnimations?.().length) {
      const style = getComputedStyle(original)
      copy.style.transform = style.transform
      copy.style.opacity = style.opacity
    }
    // 数字更新、进度条等已有 CSS 动画不能在副本插入时重新播放，否则仍会局部闪烁。
    copy.style.animation = "none"
    copy.style.transition = "none"
  })
  Object.assign(content.style, {
    position: "absolute",
    left: "0",
    top: "0",
    width: `${source.offsetWidth || rect.width}px`,
    height: `${source.offsetHeight || rect.height}px`,
    margin: "0",
    opacity: "1",
    transformOrigin: "top left",
    transform: `scale(${rect.width / (source.offsetWidth || rect.width || 1)}, ${rect.height / (source.offsetHeight || rect.height || 1)})`,
  })
  return { rect, content }
}

// 在共同的外层几何下反向缩放内容。全程 old + new 的 opacity = 1，没有空白帧。
function contentFrames(scaleX: number, scaleY: number, incoming: boolean): Keyframe[] {
  const steps = 60
  return Array.from({ length: steps + 1 }, (_, index) => {
    const progress = index / steps
    const x = scaleX + (1 - scaleX) * progress
    const y = scaleY + (1 - scaleY) * progress
    const reveal = Math.min(1, progress / 0.65)
    return {
      offset: progress,
      transform: `scale(${1 / x}, ${1 / y})`,
      opacity: incoming ? reveal : 1 - reveal,
    }
  })
}

export function animateFlipSnapshot(
  first: FlipSnapshot,
  element: HTMLElement,
  last: DOMRect,
  options: KeyframeAnimationOptions,
) {
  const surface = document.createElement("div")
  surface.className = "bento-motion-surface"
  surface.setAttribute("aria-hidden", "true")
  surface.dataset.motionId = element.dataset.flipId
  surface.inert = true
  Object.assign(surface.style, {
    left: `${last.left}px`,
    top: `${last.top}px`,
    width: `${last.width}px`,
    height: `${last.height}px`,
  })
  const oldLayer = document.createElement("div")
  const newLayer = document.createElement("div")
  oldLayer.className = newLayer.className = "bento-motion-content"
  oldLayer.append(first.content)
  newLayer.append(takeFlipSnapshot(element).content)
  surface.append(oldLayer, newLayer)
  document.body.append(surface)
  const scaleX = first.rect.width / last.width
  const scaleY = first.rect.height / last.height
  const animation = surface.animate(
    [
      {
        transform: `translate(${first.rect.left - last.left}px, ${first.rect.top - last.top}px) scale(${scaleX}, ${scaleY})`,
      },
      { transform: "none" },
    ],
    options,
  )
  const animations = [
    animation,
    oldLayer.animate(contentFrames(scaleX, scaleY, false), options),
    newLayer.animate(contentFrames(scaleX, scaleY, true), options),
  ]
  return { surface, animations, animation }
}
