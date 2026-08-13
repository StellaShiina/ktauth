const root = document.documentElement
const languageButton = document.querySelector("#language-toggle")
const themeButton = document.querySelector("#theme-toggle")
const menuButton = document.querySelector("#menu-toggle")
const nav = document.querySelector("#site-nav")
const commandCode = document.querySelector("#command-code")
const copyButton = document.querySelector(".copy-button")
const toast = document.querySelector(".toast")
const backToTop = document.querySelector("#back-to-top")

const commandBase = "bash <(curl -fsSL https://ktauth.kaju.win)"
const translations = {
  en: {
    skip: "Skip to content", navStart: "Get started", navRoutes: "Two routes", navInside: "Highlights", navApi: "API",
    heroLead: "Keep complex access decisions outside your app.",
    heroSub: "A lightweight, self-hosted authentication and authorization service. Clear Forward Auth, IP policy, precise rate limiting, and identity sessions for your reverse proxy.",
    viewSource: "View source", quickOverline: "START HERE / ONE SCRIPT", quickTitle: "Hand the gate<br>to KTAUTH.",
    quickCopy: "The script walks through Docker checks, listening address, and core credentials. Pick an action, then copy and run.",
    install: "Install", uninstall: "Uninstall", update: "Update", config: "Configure", interactive: "INTERACTIVE SETUP",
    routesOverline: "TWO ROUTES. CLEAR DECISIONS.", routesTitle: "Send every request<br>through the right gate.",
    routeZero: "The general gate. Rejects blocked sources and applies a sliding-window limit to non-whitelisted requests.",
    routeOne: "The strict gate. Allows whitelisted sources only, built for admin surfaces and internal services.",
    blackReject: "BLOCKLIST REJECT", rateControlled: "RATE CONTROLLED", whiteOnly: "ALLOWLIST ONLY", adminReady: "ADMIN READY",
    routeCaption: "Success returns 204. Drop it into Caddy forward_auth or Nginx auth_request.",
    insideOverline: "SMALL FOOTPRINT. FULL INTENT.", insideTitle: "One gate.<br>Every critical check.",
    aclTitle: "Cached IP policy", aclCopy: "Allow, grey, and block rules cover individual IPv4 addresses and IPv6 /64 ranges; Redis keeps the decision path fast.",
    rateTitle: "Millisecond sliding window", rateCopy: "Redis Lua and Sorted Sets calculate request windows atomically, controlling bursts and sustained abuse with precision.",
    identityTitle: "Identity, end to end", identityCopy: "Invitation or email-code registration, bcrypt password hashing, JWT, and revocable Redis sessions working together.",
    architectureTitle: "A clean Go core", architectureCopy: "Layered Gin architecture, PostgreSQL persistence, and Redis-backed fast state. Clear boundaries make it easy to read and extend.",
    apiOverline: "GO DEEPER WHEN NEEDED", apiTitle: "An API kept<br>easy to scan.", apiCopy: "Only endpoints and intent live here. Request schemas and full technical detail remain in the GitHub docs.",
    apiLabel: "ENDPOINT INDEX", showEndpoints: "Reveal all endpoints", hideEndpoints: "Hide all endpoints", users: "USERS", tokens: "TOKENS", policies: "IP RULES",
    epRegister: "Invite or email registration", epSend: "Send email code", epVerify: "Verify and consume code", epLogin: "Sign in", epAuth: "Check session", epLogout: "End current session", epUsers: "User list · admin",
    epRestock: "Generate invite batch", epFlush: "Clear available invites", epToken: "Get one invite", epTokens: "List all invites", epIps: "List IP rules", epIpsNew: "Create IP rule", epIpsDelete: "Delete IP rule",
    fullDocs: "Read the full documentation on GitHub", closingTitle: "Guard the gate.<br>Build what matters.", footerLine: "Simple auth. Clear boundaries.", copied: "Command copied", topLabel: "Top"
  },
  zh: {
    skip: "跳至内容", navStart: "开始使用", navRoutes: "双路由", navInside: "项目亮点", navApi: "API",
    heroLead: "把复杂的访问判断，挡在应用之外。", heroSub: "一个轻量、自托管的认证与授权服务。为反向代理提供清晰的 Forward Auth、IP 策略、精确限流与身份会话。", viewSource: "查看源代码",
    quickOverline: "从这里开始 / ONE SCRIPT", quickTitle: "把入口交给<br>KTAUTH。", quickCopy: "脚本会引导完成 Docker 检查、监听地址与核心凭据配置。选择动作，然后复制执行。",
    install: "安装", uninstall: "卸载", update: "更新", config: "配置", interactive: "交互式配置",
    routesOverline: "两条路由，清晰决策", routesTitle: "让每个请求<br>走对入口。", routeZero: "综合入口。拒绝黑名单，对非白名单请求执行滑动窗口限流。", routeOne: "严格入口。只允许白名单来源，适合管理面与内部服务。",
    blackReject: "黑名单拒绝", rateControlled: "流量受控", whiteOnly: "仅白名单", adminReady: "管理面就绪", routeCaption: "成功即返回 204。可直接接入 Caddy forward_auth 或 Nginx auth_request。",
    insideOverline: "小体积，不是小能力", insideTitle: "在一道门后，<br>做足关键判断。", aclTitle: "IP 策略缓存", aclCopy: "白、灰、黑名单覆盖单个 IPv4 与 IPv6 /64；Redis 缓存让入口判断保持迅速。",
    rateTitle: "毫秒级滑动窗口", rateCopy: "Redis Lua 与 Sorted Set 原子计算请求窗口，精确控制突发流量与持续滥用。", identityTitle: "完整身份闭环", identityCopy: "邀请码或邮箱验证码注册，bcrypt 密码哈希，JWT 与可撤销 Redis 会话协作。",
    architectureTitle: "干净的 Go 内核", architectureCopy: "Gin 驱动的分层架构，PostgreSQL 持久化，Redis 承担高速状态；边界明确，便于阅读与扩展。",
    apiOverline: "需要时，再深入", apiTitle: "API，保持<br>简洁可查。", apiCopy: "这里只列出接口与职责。请求结构和完整技术细节请前往 GitHub 文档。", apiLabel: "端点索引", showEndpoints: "展开全部端点", hideEndpoints: "收起全部端点",
    users: "用户 / USERS", tokens: "令牌 / TOKENS", policies: "策略 / IP RULES", epRegister: "邀请码或邮箱注册", epSend: "发送邮箱验证码", epVerify: "验证并消费验证码", epLogin: "用户登录", epAuth: "检查登录状态", epLogout: "结束当前会话", epUsers: "用户列表 · 管理员",
    epRestock: "批量生成邀请码", epFlush: "清空可用邀请码", epToken: "获取一个邀请码", epTokens: "全部邀请码", epIps: "列出 IP 规则", epIpsNew: "创建 IP 规则", epIpsDelete: "删除 IP 规则",
    fullDocs: "在 GitHub 阅读完整文档", closingTitle: "守住入口。<br>专注你的应用。", footerLine: "简单认证，清晰边界。", copied: "命令已复制", topLabel: "顶部"
  }
}

let language = localStorage.getItem("ktauth-language") || (navigator.language.toLowerCase().startsWith("zh") ? "zh" : "en")
let toastTimer

function setLanguage(nextLanguage) {
  language = nextLanguage
  root.lang = language === "zh" ? "zh-CN" : "en"
  document.querySelectorAll("[data-i18n]").forEach((element) => {
    const value = translations[language][element.dataset.i18n]
    if (value) element.innerHTML = value
  })
  languageButton.textContent = language === "zh" ? "EN" : "中"
  languageButton.setAttribute("aria-label", language === "zh" ? "Switch to English" : "切换到中文")
  localStorage.setItem("ktauth-language", language)
  updateApiToggleLabel()
}

function resolveTheme(theme) {
  return theme === "system" ? (matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light") : theme
}

function setTheme(theme) {
  root.dataset.theme = theme
  root.dataset.resolvedTheme = resolveTheme(theme)
  localStorage.setItem("ktauth-theme", theme)
  const labels = language === "zh" ? { system: "主题：跟随系统", light: "主题：浅色", dark: "主题：深色" } : { system: "Theme: system", light: "Theme: light", dark: "Theme: dark" }
  const icons = {
    system: '<rect x="3" y="4" width="18" height="13" rx="1"/><path d="M8 21h8M12 17v4"/><path d="M8 10h3M13 10h3M8 13h5"/>',
    light: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.42 1.42M17.65 17.65l1.42 1.42M2 12h2M20 12h2M4.93 19.07l1.42-1.42M17.65 6.35l1.42-1.42"/>',
    dark: '<path d="M20 15.2A8.5 8.5 0 0 1 8.8 4 8.5 8.5 0 1 0 20 15.2Z"/>'
  }
  themeButton.querySelector(".theme-icon").innerHTML = icons[theme]
  themeButton.setAttribute("aria-label", labels[theme])
  themeButton.setAttribute("title", labels[theme])
  document.querySelector('meta[name="theme-color"]').content = root.dataset.resolvedTheme === "dark" ? "#11100e" : "#f2f0e9"
}

function updateApiToggleLabel() {
  const apiToggle = document.querySelector(".api-toggle")
  const label = apiToggle.querySelector("strong")
  label.textContent = translations[language][apiToggle.getAttribute("aria-expanded") === "true" ? "hideEndpoints" : "showEndpoints"]
}

languageButton.addEventListener("click", () => setLanguage(language === "zh" ? "en" : "zh"))

themeButton.addEventListener("click", () => {
  const themes = ["system", "light", "dark"]
  const current = themes.indexOf(root.dataset.theme)
  setTheme(themes[(current + 1) % themes.length])
})

matchMedia("(prefers-color-scheme: dark)").addEventListener("change", () => {
  if (root.dataset.theme === "system") setTheme("system")
})

document.querySelectorAll(".command-tabs button").forEach((tab) => {
  tab.addEventListener("click", () => {
    document.querySelectorAll(".command-tabs button").forEach((button) => button.setAttribute("aria-selected", String(button === tab)))
    commandCode.textContent = `${commandBase} ${tab.dataset.command}`
  })
})

async function copyCommand() {
  try {
    await navigator.clipboard.writeText(commandCode.textContent)
  } catch {
    const range = document.createRange()
    range.selectNodeContents(commandCode)
    const selection = getSelection()
    selection.removeAllRanges()
    selection.addRange(range)
  }
  copyButton.classList.add("copied")
  toast.classList.add("visible")
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => {
    copyButton.classList.remove("copied")
    toast.classList.remove("visible")
  }, 1800)
}

copyButton.addEventListener("click", copyCommand)

function updateBackToTop() {
  backToTop.classList.toggle("visible", window.scrollY > window.innerHeight * 0.72)
}

backToTop.addEventListener("click", () => window.scrollTo({ top: 0, behavior: "smooth" }))
window.addEventListener("scroll", updateBackToTop, { passive: true })
updateBackToTop()

const apiToggle = document.querySelector(".api-toggle")
const apiPanel = document.querySelector("#api-panel")
apiToggle.addEventListener("click", () => {
  const expanded = apiToggle.getAttribute("aria-expanded") === "true"
  apiToggle.setAttribute("aria-expanded", String(!expanded))
  apiPanel.hidden = expanded
  updateApiToggleLabel()
})

menuButton.addEventListener("click", () => {
  const open = menuButton.getAttribute("aria-expanded") === "true"
  menuButton.setAttribute("aria-expanded", String(!open))
  nav.classList.toggle("open", !open)
})

nav.querySelectorAll("a").forEach((link) => link.addEventListener("click", () => {
  menuButton.setAttribute("aria-expanded", "false")
  nav.classList.remove("open")
}))

const revealObserver = new IntersectionObserver((entries) => {
  entries.forEach((entry) => {
    if (entry.isIntersecting) {
      entry.target.classList.add("is-visible")
      revealObserver.unobserve(entry.target)
    }
  })
}, { threshold: 0.12 })

document.querySelectorAll(".reveal").forEach((element) => revealObserver.observe(element))
document.querySelector("#year").textContent = new Date().getFullYear()
setTheme(localStorage.getItem("ktauth-theme") || "system")
setLanguage(language)
