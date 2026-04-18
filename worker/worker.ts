export default {
    async fetch(request: Request, env: any): Promise<Response> {
        const resp = await env.ASSETS.fetch(request)
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
