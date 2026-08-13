import { cpSync, mkdirSync, rmSync } from "node:fs"

rmSync("dist", { recursive: true, force: true })
mkdirSync("dist", { recursive: true })

cpSync("site", "dist", { recursive: true })
cpSync("../scripts/install.sh", "dist/install.sh")
cpSync("../docker-compose.yaml", "dist/docker-compose.yaml")
cpSync("../sql/00-init.sql", "dist/00-init.sql")
cpSync("../.env.example", "dist/.env.example")
