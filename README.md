# CLI Bridger

以 Go + **Wails v3.0.0-beta.28** 製作的 Windows 桌面 App。CLI 回傳一份 JSON，App 便產生子命令導覽與參數表單。採用暗黑開發者工作台設計：石墨黑、青綠重點色、緊湊表單與終端。

## 試用

1. 開啟 `build/bin/cli-bridger.exe`。
2. 選擇同資料夾的 `demo-cli.exe`，按「讀取規格」。
3. 選擇 `render`，調整必填 Steps；勾選 Color output 後，可啟用 Progress label。
4. 按「執行指令」，終端會顯示進度條。示範工具只輸出文字，不讀寫所選檔案。

也可以選擇 `examples/demo.py`，保持「自動辨識」，App 會透過已安裝的 Python 讀取描述並執行。

| 工具類型 | 自動執行方式 |
| --- | --- |
| 原生執行檔 | `tool.exe ...` |
| `.py` | `python -u tool.py ...` |
| `.js` / `.mjs` / `.cjs` | `node tool.js ...` |
| `.ps1` | `powershell -NoProfile -File tool.ps1 ...` |

執行器須已安裝且位於 PATH。可手動選擇 Python／Node.js／PowerShell，或選「自訂執行器」指定例如 `.venv/Scripts/python.exe`；一般自訂模式使用 `執行器 腳本 ...`，不解析 shell 指令字串。自訂 PowerShell/pwsh 仍會使用 `-NoProfile -File`；不接受 cmd/wsl 作為自訂執行器。不變更 PowerShell execution policy。

已有工具不會自動支援本協定。例如原版 Git 沒有此握手旗標，需工具作者實作，或另寫配接 CLI。

## 建置

Windows 10 1809+／Windows 11、Go 1.25+、Node.js 22+、Microsoft Edge WebView2 Runtime。

```powershell
./build.ps1
```

腳本安裝前端鎖定依賴、編譯 UI、執行 Go 測試、建立 demo、產生 Windows manifest/icon 資源並編譯桌面執行檔。Go 模組與 npm 快取放在專案 `.cache/`；Wails CLI 固定版本，透過 `go run` 執行，不需全域安裝。

開發時先建置前端，再執行 `go run .`；前端變更後需重建 `frontend/dist` 並重啟。這個最小專案使用 PowerShell 建置腳本，不依賴 Wails Taskfile 或全域 CLI。

## CLI 作者

實作以下握手；stdout 只能包含一份描述 JSON，診斷訊息寫入 stderr：

```text
your-cli.exe --cli-bridger-describe
```

- [通用格式 v1 與 JSON 範例](docs/protocol.md)
- [可執行的 Go 示範](examples/demo/main.go)
- [可執行的 Python 示範](examples/demo.py)
- [格式與參數驗證實作](internal/protocol/protocol.go)

描述提供必填／選填、string／int／float／path／bool、數值與 Unicode 字數限制、預設值、常用範例、列舉、子命令和條件依賴。原始 JSON 預設摺疊。數值上下界完整時顯示滑桿；路徑可手輸或使用系統檔案／資料夾／儲存對話框。

## 終端與執行

使用 Windows ConPTY + xterm.js，支援 ANSI 顏色、游標移動、原地更新進度、鍵盤輸入、視窗尺寸同步及停止。輸出以原始位元組傳遞，避免中文 UTF-8 分段被破壞；stdout/stderr 合併為終端串流。

執行前由 Go 重新驗證參數，直接啟動執行檔或選定的腳本執行器，不拼接 shell 指令。Discovery 與執行共用同一組執行器／腳本引數。預覽以引數邊界顯示，並非可直接貼入所有 shell 的指令字串。工作目錄繼承 App 啟動目錄。讀取格式也會執行工具，請選擇可信任的 CLI。

## 範圍與限制

- 第一版實作及測試 Windows；尚未提供 macOS/Linux PTY 後端或安裝包。
- v3 尚為 Beta；Go 與前端 runtime 均鎖定 beta.28。
- 一次執行一個程序。停止會關閉 pseudoconsole；刻意脫離終端的背景程序不保證一併結束。
- Discovery 最多 5 秒、stdout/stderr 各 1 MiB。`.cmd`／`.bat` 不支援；自訂執行器需為原生 `.exe`。目前不提供 `python -m module` 或任意前置啟動參數欄位。
- v1 不支援重複參數、複合布林依賴、互斥群組；帶值旗標採 `--flag=value`。詳見協定文件。
- 終端回捲保留 5,000 行；不將完整輸出持久儲存。

受限測試環境可在程序啟動前設定 `CLI_BRIDGER_WEBVIEW_DATA`，將 WebView profile 指到可寫入的測試目錄。一般啟動使用 Wails 的預設 profile 位置。

## 驗證

`go test ./...` 包含格式／參數驗證與實際 ConPTY 測試：互動輸入、resize、中文與引號引數、ANSI、停止後重新執行，以及三輪約 180 KiB 尾端輸出的完整性檢查。UI 另以桌面 WebView 實測 discovery、預設值、滑桿、條件欄位、執行與原始 JSON 摺疊；原生檔案對話框已開啟確認。
