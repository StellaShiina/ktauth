const SCRIPT_PATH = "/install.sh"
type AssetBinding = {
    fetch(input: Request | string, init?: RequestInit): Promise<Response>
}

function isCommandLineRequest(request: Request): boolean {
    const userAgent = request.headers.get("user-agent") || ""
    const accept = request.headers.get("accept") || ""
    const secFetchDest = request.headers.get("sec-fetch-dest") || ""
    const secFetchMode = request.headers.get("sec-fetch-mode") || ""

    if (/curl|wget|httpie|aria2|fetch/i.test(userAgent)) {
        return true
    }

    // curl/wget commonly send `Accept: */*` (or no Accept header), while a
    // browser navigation advertises HTML and carries Sec-Fetch headers.
    const browserNavigation = /text\/html/i.test(accept) || Boolean(secFetchDest || secFetchMode)
    return !browserNavigation && (!accept || /\*\/\*/.test(accept))
}

function assetRequest(request: Request, pathname: string): Request {
    const url = new URL(request.url)
    url.pathname = pathname
    return new Request(url.toString(), request)
}

async function fetchScript(request: Request, env: { ASSETS: AssetBinding }): Promise<Response> {
    const response = await env.ASSETS.fetch(assetRequest(request, SCRIPT_PATH))
    const contentType = response.headers.get("content-type") || ""

    // With `not_found_handling = "404-page"`, a missing asset can otherwise
    // look like a successful HTML response to curl.
    if (!response.ok || /text\/html/i.test(contentType)) {
        return new Response("KTAUTH install script is unavailable.\n", {
            status: 404,
            headers: { "content-type": "text/plain; charset=utf-8" },
        })
    }

    const headers = new Headers(response.headers)
    headers.set("content-type", "text/plain; charset=utf-8")
    headers.set("content-disposition", "inline; filename=install.sh")
    return new Response(response.body, {
        status: response.status,
        statusText: response.statusText,
        headers,
    })
}

export default {
    async fetch(request: Request, env: { ASSETS: AssetBinding }): Promise<Response> {
        const url = new URL(request.url)

        if (url.pathname === SCRIPT_PATH) {
            return fetchScript(request, env)
        }

        if (url.pathname === "/") {
            if (isCommandLineRequest(request)) {
                return fetchScript(request, env)
            }

            return env.ASSETS.fetch(assetRequest(request, "/index.html"))
        }

        return env.ASSETS.fetch(request)
    },
}
