export default {
    async fetch(request: Request, env: any): Promise<Response> {
        const url = new URL(request.url)
        const ua = request.headers.get("user-agent") || ""

        if (url.pathname == "/") {
            const isCli = /curl|wget|httpie/i.test(ua)

            if (isCli) {
                url.pathname = "/install.sh"
            } else {
                return Response.redirect("https://github.com/stellashiina/ktauth", 302)
            }

        }
        const resp = await env.ASSETS.fetch(url.toString(), request)
        if (resp.status === 404) {
            return resp
        }
        return new Response(resp.body, {
            headers: {
                "content-type": "text/plain; charset=utf-8",
            }
        })
    }
}
