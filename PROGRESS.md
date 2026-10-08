# CLI Bridger

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
