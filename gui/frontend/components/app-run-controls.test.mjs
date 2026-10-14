import { strict as assert } from "node:assert"
import { test } from "node:test"
import { createElement } from "react"
import { renderToStaticMarkup } from "react-dom/server"

// The control imports the desktop event bridge, but rendering needs no daemon.
globalThis.window = { runtime: { EventsOnMultiple: () => () => {} } }
const { AppRunControls } = await import("./app-run-controls.tsx")

for (const variant of ["compact", "icon", "default"]) {
  for (const running of [false, true]) {
    test(`${variant} actions while ${running ? "running prod" : "idle"}`, () => {
      const html = renderToStaticMarkup(createElement(AppRunControls, {
        appId: 1, running, kind: "prod", variant, onStatus: () => {},
      }))
      assert(!html.includes(variant === "compact" ? 'aria-label="Start setup"' : 'aria-label="Setup"'))
      if (variant === "default") assert(!html.includes("Setup"))
      for (const label of ["Build", "Prod"]) {
        assert(html.includes(variant === "compact" ? `aria-label="Start ${label.toLowerCase()}"` : label))
      }
      assert(html.includes(variant === "compact" ? `aria-label="${running ? "Stop" : "Start dev"}"` : "Dev"))
      if (variant === "compact" && running) {
        for (const label of ["build", "prod"]) {
          const button = html.match(new RegExp(`<button[^>]*aria-label="Start ${label}"[^>]*>`))?.[0]
          assert(button && /\sdisabled=/.test(button), `${label} must be disabled`)
        }
        const stop = html.match(/<button[^>]*aria-label="Stop"[^>]*>/)?.[0]
        assert(stop && !/\sdisabled=/.test(stop), "Stop must remain enabled")
      }
    })
  }
  test(`${variant} shows Setup when requested by the app screen`, () => {
    const html = renderToStaticMarkup(createElement(AppRunControls, {
      appId: 1, running: false, variant, showSetup: true, onStatus: () => {},
    }))
    assert(html.includes(variant === "compact" ? 'aria-label="Start setup"' : "Setup"))
  })
}
