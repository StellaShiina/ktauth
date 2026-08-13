export default {
    async fetch(request: Request, env: any): Promise<Response> {
        const url = new URL(request.url)
        const ua = request.headers.get("user-agent") || ""

        if (url.pathname === "/") {
            const isCli = /curl|wget|httpie/i.test(ua)

            if (isCli) {
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
