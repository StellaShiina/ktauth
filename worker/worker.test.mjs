import assert from "node:assert/strict"
import { test } from "node:test"

import worker from "./worker.ts"

const installScript = "#!/usr/bin/env bash\necho install\n"
const indexHtml = "<!doctype html><title>KTAUTH</title>"

function createAssets() {
    return {
        async fetch(request) {
            const url = new URL(request.url)
            const body = url.pathname === "/install.sh" ? installScript : indexHtml
            const contentType = url.pathname === "/install.sh" ? "application/octet-stream" : "text/html"
            return new Response(body, { headers: { "content-type": contentType } })
        },
    }
}

for (const hostname of ["ktauth.kaju.win", "localhost", "127.0.0.1", "[::1]"]) {
    test(`${hostname} root returns install.sh`, async () => {
        const response = await worker.fetch(new Request(`http://${hostname}/`), { ASSETS: createAssets() })

        assert.equal(await response.text(), installScript)
        assert.equal(response.headers.get("content-type"), "text/plain; charset=utf-8")
    })
}

test("a non-script host root returns index.html", async () => {
    const response = await worker.fetch(new Request("https://www.kaju.win/"), { ASSETS: createAssets() })

    assert.equal(await response.text(), indexHtml)
    assert.equal(response.headers.get("content-type"), "text/html")
})

test("the explicit install.sh path returns the script on every host", async () => {
    const response = await worker.fetch(new Request("https://www.kaju.win/install.sh"), { ASSETS: createAssets() })

    assert.equal(await response.text(), installScript)
    assert.equal(response.headers.get("content-type"), "text/plain; charset=utf-8")
})
