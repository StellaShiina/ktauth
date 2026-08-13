const copyButton = document.querySelector("[data-copy]")

copyButton?.addEventListener("click", async () => {
  try {
    await navigator.clipboard.writeText(copyButton.dataset.copy)
    const label = copyButton.querySelector("span")
    label.textContent = "已复制"
    window.setTimeout(() => {
      label.textContent = "复制"
    }, 1800)
  } catch {
    const selection = window.getSelection()
    const range = document.createRange()
    range.selectNodeContents(document.querySelector(".terminal code"))
    selection.removeAllRanges()
    selection.addRange(range)
  }
})
