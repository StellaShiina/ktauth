export default {
    async fetch(request: Request, env: any): Promise<Response> {
        const url = new URL(request.url)

        if (url.pathname === "/") {
            const scriptHosts = new Set(["ktauth.kaju.win", "localhost", "127.0.0.1", "[::1]", "dev-ktauth.kaju.workers.dev"])
            const isScriptHost = scriptHosts.has(url.hostname.toLowerCase())

            if (isScriptHost) {
                url.pathname = "/install.sh"
            } else {
                url.pathname = "/index.html"
            }
        }

        const response = await env.ASSETS.fetch(new Request(url.toString(), request))
        if (url.pathname === "/install.sh") {
            const headers = new Headers(response.headers)
            headers.set("content-type", "text/plain; charset=utf-8")
            return new Response(response.body, {
                status: response.status,
                statusText: response.statusText,
                headers,
            })
        }

        return response
    }
}
