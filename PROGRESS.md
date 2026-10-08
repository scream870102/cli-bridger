# CLI Bridger

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
