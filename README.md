# CLI Bridger

授權：[MIT License](LICENSE) · Copyright (c) 2026 scream870102

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

## 建置與開發

### 環境需求

| 項目 | 需求與用途 |
| --- | --- |
| 作業系統 | Windows 10 1809+ 或 Windows 11；目前程式包含 Windows 專用 PTY 後端 |
| Go | 1.25 以上，用於後端、測試及 Windows 執行檔編譯 |
| Node.js / npm | 建議 Node.js 22 以上並包含 npm，用於前端依賴與 Vite 建置 |
| WebView2 Runtime | 執行桌面 App 時需要 Microsoft Edge WebView2 Runtime |
| Python 等執行器 | 僅在使用對應腳本 CLI 時需要；建置 App 不需要 Python |

以下指令使用 **PowerShell**，從專案根目錄（含 `go.mod` 的資料夾）開始執行。首次建置需要網路下載依賴。每個步驟成功後，再進行下一步。

可先確認工具版本：

```powershell
go version
node --version
npm.cmd --version
```

### 手動建置（不需 build.ps1）

**1. 安裝依賴並建置前端**

```powershell
go mod download

cd frontend
npm.cmd ci
npm.cmd run build
cd ..
```

`npm.cmd ci` 依照 `package-lock.json` 安裝固定依賴；`npm.cmd run build` 產生 `frontend/dist/`。必須先產生此目錄，Go 才能將前端嵌入執行檔。使用 `npm.cmd` 可直接呼叫 npm，避免 PowerShell 選到 `npm.ps1`。

**2. 執行測試**

```powershell
go test ./...
```

測試包含協定驗證、啟動器解析及實際 Windows ConPTY 程序。若系統有 Python，也會驗證 Python 示範的描述協定；未安裝時該項測試會跳過。

**3. 產生 Windows 資源**

```powershell
$buildArch = go env GOARCH
go run github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.28 generate syso `
  -manifest build/windows/app.manifest `
  -icon build/windows/icon.ico `
  -arch $buildArch `
  -out "rsrc_windows_$buildArch.syso"
```

這一步將應用程式圖示與 DPI manifest 編譯成 Go linker 會讀取的 `.syso`。Wails CLI 固定為 `v3.0.0-beta.28`，透過 `go run` 執行，不需全域安裝 Wails 或 Task。PowerShell 的續行符號是行尾的反引號，後面不可接空白。

**4. 編譯正式版 App**

```powershell
New-Item -ItemType Directory -Force build/bin | Out-Null
go build -tags production -ldflags '-H windowsgui' -o build/bin/cli-bridger.exe .
```

`production` 使用正式版模式；`-H windowsgui` 讓 App 啟動時不額外開啟命令列視窗。產出為 `build/bin/cli-bridger.exe`，已包含前端資源，執行時不需要另外啟動 Vite。

**5. 建立示範 CLI（可選）**

```powershell
go build -o build/bin/demo-cli.exe ./examples/demo
```

也可直接使用 `examples/demo.py`，不需編譯它，但執行時需要 Python。

**6. 啟動**

```powershell
./build/bin/cli-bridger.exe
```

在 App 中選擇 `build/bin/demo-cli.exe` 或 `examples/demo.py`，按「讀取規格」開始操作。提供給其他 Windows 使用者時，App 執行檔已包含 UI；對方仍須有 WebView2 Runtime，以及要操作的 CLI 和該 CLI 所需的執行環境。

### 開發模式

首次開發先完成上面的依賴／前端建置及 Windows 資源生成，再從根目錄執行：

```powershell
go run .
```

此模式保留開發紀錄，方便在終端查看錯誤。Go 程式碼修改後需停止並重新執行；前端修改後，先重建資源，再重新啟動 App：

```powershell
cd frontend
npm.cmd run build
cd ..
go run .
```

本專案尚未配置桌面熱重載。`frontend` 的 `npm.cmd run dev` 只啟動 Vite 伺服器，不會同時啟動 Go 桌面後端；完整 CLI 操作請使用桌面 App。

### 可選：使用 build.ps1

想一次完成上述流程時，可從根目錄執行：

```powershell
./build.ps1
```

`build.ps1` 是便利腳本，不是必要建置工具。它會安裝前端依賴、建置 UI、執行測試、編譯示範 CLI、生成 Windows 資源，再編譯正式版 App。腳本將 Go 模組、Go build 與 npm 快取放在專案 `.cache/`，結束時還原原本的 Go 快取環境變數；手動指令則使用各工具的預設快取位置。

若 PowerShell 禁止執行 `.ps1`，可直接使用前面的手動步驟，無須調整系統執行原則。

### 常見建置問題

| 狀況 | 處理方式 |
| --- | --- |
| Go 回報 `frontend/dist` 找不到檔案 | 先在 `frontend` 執行 `npm.cmd ci` 與 `npm.cmd run build` |
| 提示找不到 Go、Node 或 npm | 確認已安裝並加入 PATH，重新開啟終端後檢查版本 |
| 無法覆寫 `cli-bridger.exe` | 關閉正在執行的 App，再重新編譯 |
| 修改前端後，App 仍顯示舊畫面 | 重建 `frontend/dist`，再重新執行 `go run .` 或重新編譯正式版 |
| App 無法建立 WebView2 | 確認 WebView2 Runtime 已安裝，且 WebView profile 目錄可寫入 |

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

工具、每個指令（含根指令）與每個參數都必須回覆非空白的 `description`，說明用途與操作效果。例如 `--steps` 的說明應是「設定進度更新次數；數值越大，更新越細」，而不只是重複參數名稱。App 會在指令導覽、指令標題下方與參數旁顯示這些純文字說明，選填參數未勾選時也能閱讀。缺少說明的描述會被拒絕，錯誤訊息會指出對應指令或參數；先前實作此原型協定的 CLI 需補齊欄位。

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
