export default {
    async fetch(request: Request, env: any): Promise<Response> {
        const url = new URL(request.url)
        if (url.pathname == "/") {
            url.pathname = "/install.sh"
        }
        const resp = await env.ASSETS.fetch(url.toString(), request)
        if (resp.status === 404) {
            return resp;
        }
        return new Response(resp.body, {
            headers: {
                "content-type": "text/plain; charset=utf-8",
            },
        })
    }
};
