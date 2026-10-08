# CLI Bridger

## Completed task: persistent settings database and reset (2026-10-08)

Goal: save settings in cli-bridger.db beside the app executable, restore on startup and reset in one click.
Acceptance: persistent target/runner/command/values/toggles plus cached validated descriptor; no CLI invocation on restore; external provenance warning retained; reset clears stored and visible settings; failures visible; tests/build/restart WebView check/independent review.
Knowns: use pure-Go bbolt embedded DB; no CGO/compiler changes. Persist configuration only, not terminal output/input. User explicitly authorized adjacent DB and reset. Backend owns storage/tests/dependency; main owns UI/docs/integration. No open questions.
- [x] Added bbolt v1.4.3 DB beside os.Executable, bounded settings and cached schema/source, serialized saves and recoverable storage errors.
- [x] Automatic restore without invoking CLI; restores runner/command/parameters/toggles and external warning. Reset drains saves, deletes DB, clears form; disabled during execution/discovery.
- [x] Backend tests, all Go tests, Vite/production builds and independent review passed.
- [x] Actual process restart smoke passed: custom runner, nested command, Unicode/environment values, zero, false, enabled toggles, cached schema even with changed sidecar, no startup CLI invocation, successful manual run, pending-write reset, empty restart and corrupt-DB recovery. 800px layout has no overflow; screenshot inspected.
Artifacts: .cache/settings-smoke.mjs, .cache/settings-preview.png; regression tests settings_windows_test.go; rebuilt build/bin/cli-bridger.exe. Test apps used isolated .cache/settings-smoke-* executable/DB/WebView directories and were closed afterward. Existing installed application and its settings were untouched. No commit/push; no required work remains.
Verification lesson: set CLI_BRIDGER_WEBVIEW_DATA to an isolated test directory when the user's installed app is already running, avoiding shared WebView profile locks.

## Completed task: external descriptions for third-party CLIs (2026-10-08)

Goal: load a user-authored descriptor beside a CLI that cannot respond to discovery, with a persistent accuracy/provenance warning.
Acceptance: exact <full-target-filename>.cli-bridger.json lookup for executables/scripts; sidecar first without discovery; bounded same-schema validation and clear errors; reload preserves source and previous valid state on failure; native discovery unchanged; visible warning/path; working example, backend tests, real WebView smoke and independent review.
Knowns: reuse v1 protocol and launch arguments; external-file source metadata comes from backend, never self-declared authorship. No new dependencies or schema expansion. No open questions.
Plan completed: backend/tests by sidecar_backend; main integrated UI/docs/example; sidecar_review independently reviewed the implementation and executed the example.
- [x] Exact native/script sidecar lookup; file-first discovery and fail-closed validation; source metadata and pinned reload with state retention.
- [x] Persistent warning and source path, sidecar reload even without env parameters, native source transitions clear warning; all content rendered as text.
- [x] Added examples/third-party.py and matching JSON, updated README/protocol docs.
- [x] Vite build, all Go tests, production executable build, actual WebView smoke and independent review passed.
Verification: .cache/sidecar-smoke.mjs exercised real third-party execution with Chinese argv, source warning, valid/invalid reload, retained input, safe text rendering, native discovery transition and 800px width. Screenshot .cache/sidecar-preview.png inspected. Regression tests in sidecar_windows_test.go cover native/script preview, malformed/oversized/nonregular/locked files, native fallback and deleted-sidecar reload. Build: build/bin/cli-bridger.exe. No commits/pushes or new dependencies; no required work remains.
Verification lesson: xterm may defer DOM rendering when below the viewport; scroll the terminal into view before asserting rendered output in WebView tests.

## Completed task: GitHub tag releases (2026-10-08)

Goal: pushing a version tag builds the Windows app and installer and publishes downloadable GitHub Release assets.
Acceptance: tag-triggered workflow; tested production build; compiled installer and portable ZIP/checksums; WebView2 handling; documented tag/release process; independent review.
Knowns: reuse build.ps1, Wails bootstrapper and runner-provided Inno Setup; x64 only, per-user installer. No actual tag push or GitHub publication authorized in this task.
Open questions: none.
- [x] Added v* tag workflow using official setup actions, build.ps1, per-user Inno installer, portable ZIP and SHA256SUMS; prerelease suffixes do not replace Latest.
- [x] Full build.ps1 passed (npm ci/Vite, Go tests, demo, Windows resources and production app).
- [x] Compiled v1.2.3-beta.1 installer with official Inno Setup 6.7.1 in project-local portable mode; verified ZIP contents, checksum values and rejection of invalid/oversized versions. Actionlint v1.7.12 passed.
- [x] Documented installation/tag publishing/local packaging and reviewed workflow/installer independently.
Verification scripts: .cache/prepare-inno.ps1 and .cache/check-release.ps1. Sample packages: build/bin/release/v1.2.3-beta.1/. This is a local test version, not a published release. GitHub workflow adds actual install/uninstall smoke checks before publishing; those hosted-runner checks and Release upload remain unexecuted until a tag is pushed. Local app installation and missing-WebView2 installation were not run (no user registry/runtime changes). No commit, push, tag or remote release performed.

## Completed task: interactive input and copying output (2026-10-08)

Goal: continue running commands with replies such as y + Enter, and copy output for debugging.
Acceptance: multi-prompt real PTY interaction; explicit input form and terminal keyboard; selection/all-output clipboard with Ctrl+C preserving interrupt when unselected; production build and independent review.
Knowns: existing ConPTY Input/onData already provide interactive transport; reuse xterm selection/buffer and browser clipboard. No new dependencies. Copy all is limited to retained terminal buffer (5,000 scrollback lines).
Open questions: none.
- [x] Added explicit reply form; blank Enter and successive prompts verified through App.Input and actual WebView.
- [x] Selection/all-output clipboard buttons and Ctrl+C behavior verified in actual WebView; OS clipboard confirmed Chinese output and line breaks.
- [x] Go tests, Vite build, production binary build and independent review passed. 800px layout had no horizontal overflow.
Verification artifacts: .cache/terminal-smoke.mjs, .cache/terminal-keys.mjs, .cache/terminal-preview.png. Regression test: TestAppInputMultiplePrompts. Independent review found stale queued input crossing runs; fixed with generation invalidation and verified by executing the actual queue function. Production binary: build/bin/cli-bridger.exe. Diagnostic browser port exists only in an ignored test overlay; no production debug port, dependencies or commits added. No required work remains.

## Completed task: dark panel redesign
Goal: Spotify-inspired dark surfaces, distinct sections and parameter cards, accent #E28C91.
Acceptance: readable 16px body; separated rounded sections/cards; pink selected/focus/action states; no overflow at 800px; actual desktop smoke and production build pass.
Knowns: preserve current environment feature edits and behavior. Reuse CSS/DOM with no dependencies. Open questions: none.
Verification: all acceptance criteria passed. Vite build and production Go build succeeded. Actual WebView smoke verified discovery, environment edit/reload/run, 16px body and 14px help. At 800px cards stack and toolbar wraps with no horizontal overflow. Inspected .cache/pink-preview.png and .cache/pink-narrow.png. Main workspace scrolls as a whole so parameter cards remain readable. Production binary rebuilt; no dependencies, commits or external repo edits.

Latest completed task: environment parameters and larger typography; see "Environment parameters" below.

## Completed task: purpose descriptions (2026-10-08)

User requested purpose explanations for every command and parameter. Reuse description, require nonblank text for tool/root/commands/parameters, complete Go/Python demo responses and docs, and render visible help in sidebar/forms including disabled optional parameters. Preserve user's existing README manual build edits.

- [x] Protocol validation/tests, docs and demos complete.
- [x] Visible command/parameter help and accessible input descriptions implemented.
- [x] Rebuild and verify actual desktop help rendering; independent review.

Verification: all Go tests and Vite production build passed. Actual Wails WebView smoke loaded Python demo, verified root with no parameters, sidebar/current command explanations, optional help before enable, accessible input links and conditional help, then ran demo successfully. Screenshot .cache/help-preview.png inspected. Independent review passed all criteria. Rebuilt build/bin/cli-bridger.exe and demo-cli.exe; diagnostic port existed only in a temporary Go overlay, and test PID 47088 was closed. No outstanding work for this task.

Goal: Generate a desktop GUI from a versioned CLI JSON description.

Decision: Go + Wails v3 Beta (user selected v3 during implementation); vanilla JavaScript + xterm.js; Windows ConPTY first. Pin framework version and adapt desktop lifecycle/runtime before final verification.

Acceptance:
- [x] Versioned protocol, documentation, runnable example and argument validation tests.
- [x] Nested commands, required/optional/conditional typed inputs and bounded numeric sliders.
- [x] Native file/directory/save dialog integration and collapsed raw descriptor; actual file dialog opened and inspected. Directory/save modes not manually exercised.
- [x] Actual PTY terminal output, keyboard input, resize and stop. Three 180 KiB final bursts retained all tokens; stop/restart tests pass.
- [x] Build and run verification; independent review and limitations recorded.

Scope: New project files only. No commits, global configuration changes or installation.

User decisions: Wails v3 Beta pinned to beta.28; dark style A, Developer Workbench. Generated conceptual comparison image shown inline, not used as UI asset.

Independent review: fixed prototype-name optional activation, Unicode codepoint/native maxlength mismatch, unused display names and bool enums. Stop now displays a deliberate stopped status instead of a closed-handle error.

Desktop verification: real Wails WebView exercised discovery -> render -> required default/slider -> dependency -> Unicode label -> execute -> exit; passed. Native file picker opened. OS screenshot provider captured occluding apps, so visual verification used the test app's own DevTools screenshot. Temporary debug port must be removed before release.

Scope extension: user requested Python and other interpreted scripts. Add automatic Python/Node/PowerShell runners, explicit custom interpreter (including virtualenv), structured fixed script argv for both discovery and execution, tests and a Python demo.

Python script integration: passed real WebView discovery/form/dependency/Unicode argv/run/exit test. Python 3.14.6 via Windows App Execution Alias verified. Screenshot .cache/workbench-python.png was inspected. All Go tests passed after allowing executable aliases. Built-in and custom PowerShell must use -NoProfile -File; explicit command shells are rejected by launcher.

Release cleanup: removed temporary AdditionalBrowserArgs debug port from main.go and closed the debug app (PID 29700). Final `rtk proxy powershell -NoProfile -File ./build.ps1` passed on 2026-10-08: locked npm install, Vite build, all Go tests, demo build, Windows resources, production desktop build. Outputs reread: build/bin/cli-bridger.exe (14,193,152 bytes) and demo-cli.exe. Independent reviewer passed rechecks for discovery selection locking, custom PowerShell file-mode protection, and debug-port removal.

Done: requested Windows prototype, v1 protocol and Go/Python demos, dark workbench A, script runners. No required implementation work remains. Limits: Windows only; Wails beta; no repeated/compound/exclusive parameters or Python module launcher; directory/save dialogs and Node/PowerShell script runtime were not manually exercised. README records scope. No commits/pushes performed.

Lessons worth retaining:
- Windows App Execution Aliases are valid executables but may be nonregular reparse points; executable validation must not reject them solely by IsRegular. Script targets still require regular files.
- Custom powershell.exe/pwsh.exe needs explicit -File just like the built-in runner; shell-style command parsing would invalidate literal argv assumptions.
- Disabling only Load during discovery allows stale tool responses to overwrite newer selections; lock the complete target/runner selection while the handshake is pending.
## Environment parameters (current task)

Goal: add declared env parameters to Bridger using existing typed fields; child-only overrides.
Knowns: initial handshake inherits app environment; ZZZ_MEDIA_ROOT belongs to media.cjs, ZZZ_SITE_ROOT to other tools. External repository stays unchanged.
Open questions: none; implement explicit environment-aware reload for dynamic descriptors.
Scope update: user requested larger, more legible text; raise body/terminal to 16px, help to 14–15px, improve muted contrast and narrow-window wrapping. Verify desktop rendering at 1200px and 800px widths.
Acceptance:
- [x] Schema validates env bindings and keeps them out of argv; defaults/dependencies/types work.
- [x] UI identifies env fields and supports path picker and environment-aware reload.
- [x] Discovery reload and ConPTY execution receive overrides; disabling inherits; parent unchanged.
- [x] Docs/examples updated, tests/build and independent review pass.

Verification: go test ./... passed, including actual discovery reload and ConPTY environment tests. Vite build passed; rebuilt production App and Go demo. Actual diagnostic WebView loaded Python demo, enabled a Chinese path with spaces, reloaded keeping the value, ran successfully with that environment visible in terminal and absent from argv, then disabled/reloaded. At 800px the controls stack with no horizontal overflow; screenshots .cache/env-preview.png and .cache/env-narrow.png inspected. Body/terminal 16px; help 14–15px. Independent review passed after fixing successful argument preview clearing unrelated reload errors. External zzz-lab-tools unchanged; CLI authors must add env declarations. No commits/pushes. No outstanding required work.

Build lesson: frontend/dist is embedded by Go; complete Vite build before Go compilation/tests rather than running them concurrently (asset replacement invalidates embed inputs).
