# CLI Bridger

授權：[MIT License](LICENSE) · Copyright (c) 2026 scream870102

以 Go + **Wails v3.0.0-beta.28** 製作的 Windows 桌面 App。CLI 回傳一份 JSON，App 便產生子命令導覽與參數表單。採用暗黑開發者工作台設計：石墨黑、青綠重點色、緊湊表單與終端。

## 下載與安裝

到此 GitHub repository 的 **Releases** 頁面，下載 `CLI-Bridger-<版本>-windows-x64-setup.exe`，依精靈安裝後從開始功能表開啟。安裝在目前使用者的 `%LOCALAPPDATA%\Programs\CLI Bridger`，可從 Windows「已安裝的應用程式」解除安裝。

若缺少 WebView2 Runtime，安裝包會自動執行 Microsoft bootstrapper，需要網路連線；失敗時會顯示訊息並允許重試。也可下載 `*-portable.zip` 解壓縮後執行 `cli-bridger.exe`，但需自行備妥 [WebView2 Runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/)。兩種套件都包含 `demo-cli.exe`、README 與 MIT 授權。

目前發佈 Windows x64 版本，安裝包與程式尚未進行程式碼簽章。`SHA256SUMS.txt` 提供下載檔案的 SHA-256 校驗值。

## 自動發佈版本

將 `.github/workflows/release.yml` 與打包檔案合併到 GitHub 後，在要發佈的 commit 建立並推送版本 tag：

```powershell
git tag v1.0.0
git push origin v1.0.0
```

GitHub Actions 會執行完整前端建置、Go 測試、Windows exe 編譯，再產生 Inno Setup 安裝包、免安裝 ZIP 與校驗檔，全部上傳到該 tag 的 GitHub Release。使用內建 `GITHUB_TOKEN` 與 workflow 的 `contents: write` 權限，不需要另外設定 PAT；repository 必須允許執行 GitHub Actions。

Tag 格式為 `v主版.次版.修訂版`，例如 `v1.2.3`；可加預發佈後綴，如 `v1.2.3-beta.1`，這類 Release 會標示為 prerelease，且不取代 Latest。三段數字各不得超過 65535。正式 Release 的附件不會自動覆蓋：發佈後有修改請使用新 tag；建置失敗且尚未建立 Release 時可重跑 workflow。若發佈中斷而留下 draft，先到 Releases 檢查並處理該 draft 再重跑。

本機打包需先安裝 [Inno Setup 6](https://jrsoftware.org/isinfo.php)，再執行：

```powershell
./build.ps1
./build/windows/package.ps1 -Tag v1.0.0
```

產物位於 `build/bin/release/v1.0.0/`。Inno Setup 不在預設路徑時，可用 `-Compiler '路徑/ISCC.exe'` 指定。打包不會推送 tag 或發佈 Release。

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

沒有握手功能的第三方 CLI 也可使用下方的外部規格檔，不需要修改 CLI 本身。

### 自動儲存與重置設定

CLI Bridger 會把目前工具路徑、執行方式、自訂執行器、子命令、參數值及勾選狀態自動存入 **CLI Bridger 執行檔同目錄的 `cli-bridger.db`**。這是 bbolt 本機資料庫；不存放在被操作的 CLI 目錄，也不依目前工作目錄決定位置。請等畫面顯示「設定已儲存」再關閉程式。

右上角齒輪「設定」可關閉「記錄工具設定」：關閉後不讀取也不寫入紀錄（啟動時不還原、讀取規格時不套用舊值），已存在的紀錄保留，重新開啟後即可繼續使用。此開關與終端捲動行數屬於全域偏好，同樣存在 `cli-bridger.db`。

**每個工具各自保存一份設定**：A 填好的值不會被 B 覆蓋。選回 A 並按「讀取規格」後，會載入 A 的最新規格，再恢復 A 上次的子命令、參數、勾選狀態與自訂環境變數；B 也同樣保留自己的值，重開程式後仍有效。紀錄以工具的完整路徑區分，因此不同目錄的同名工具、使用同一個 Python／Node 執行器的不同腳本也不會混用。新工具採用預設值；規格更新後只恢復 ID、型別及旗標／環境綁定仍相容的參數。舊版只保存最後一個工具的資料會自動保留為該工具的紀錄。

下次啟動會使用已儲存的規格恢復表單，**不執行 CLI，也不自動查詢規格**。工具或外部 JSON 更新後，請手動讀取最新規格；外部規格的來源警告會保留。若原工具已搬移或無法找到，會保留工具路徑並提示重新選取／讀取。

按工具列的重置圖示（「重置此工具」）只清除**目前工具**的快取，保留目前工具與規格，將它的子命令、參數及勾選狀態恢復預設值並清空自訂環境變數；其他工具的紀錄不受影響。不影響 CLI、外部 JSON 或終端輸出，也不會刪除整個資料庫；未載入工具、指令執行或規格讀取中停用此按鈕。重置當下不會立即重新儲存被清除的紀錄，之後修改設定才再次儲存。讀寫資料庫失敗時會顯示錯誤，不會默默改存到其他目錄或清除其他工具的資料。

資料庫包含參數與使用者填入的環境變數值，以本機明文資料儲存；不保存終端互動輸入與輸出。搬移免安裝版時，連同 `cli-bridger.db` 搬移即可保留設定。安裝包更新不覆蓋這個檔案。

### 第三方 CLI：同目錄外部規格

在工具旁邊建立 UTF-8 JSON 檔，名稱為「完整工具檔名 + `.cli-bridger.json`」，保留原始副檔名：

| 選取的工具 | 同目錄規格檔 |
| --- | --- |
| `tool.exe` | `tool.exe.cli-bridger.json` |
| `tool.py` | `tool.py.cli-bridger.json` |
| `tool.cjs` | `tool.cjs.cli-bridger.json` |

檔案使用相同的 [v1 規格格式](docs/protocol.md)，可以由使用者自行撰寫或生成，不需要 CLI 作者配合。在 App 選取工具本身、按「讀取規格」即可；不是選取 JSON 檔。以 PATH 工具名稱載入時，會到實際找到的執行檔旁尋找；腳本則找在腳本旁，而非 Python／Node 執行器旁。

找到外部規格時會優先讀取，**不執行 `--cli-bridger-describe`**。只有檔案不存在時才使用原有握手。外部檔案格式錯誤、無法讀取或超過 1 MiB 時會直接顯示錯誤，不會改成呼叫 CLI。App 會持續顯示來源檔案路徑，以及「可能非 CLI 作者提供、準確性未經確認」的警告；檔案不能透過自訂作者欄位取消警告。

修改檔案後按「重新讀取規格檔」，會保留仍相容的輸入。若重新讀取失敗，保留上次成功載入的規格及警告，不呼叫第三方 CLI。外部 JSON 是靜態規格；其中 `env` 欄位仍會在執行 CLI 時套用，但重讀檔案不會依環境重新生成內容。

可直接試用 [`examples/third-party.py`](examples/third-party.py)：它只有一般 argparse 參數，沒有握手功能；旁邊的 [`third-party.py.cli-bridger.json`](examples/third-party.py.cli-bridger.json) 提供 UI 規格。安裝 Python 後，在 App 選取該 `.py`，輸入訊息並執行即可。

請依實際 CLI 版本核對參數。格式驗證僅確認 JSON 與欄位合法，不保證外部說明正確；例如目前帶值旗標使用 `--flag=value`，不接受這種語法的工具仍需配接程式。

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

CLI 作者可實作以下握手；stdout 只能包含一份描述 JSON，診斷訊息寫入 stderr。沒有握手功能時，可改用上述同目錄外部規格檔：

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

### 環境變數

CLI 可在參數中使用 `env` 指定環境變數名稱，取代 `flag`。例如在 `media.cjs` 的根指令 `parameters` 加入：

```json
{
  "id": "media-root",
  "name": "素材根目錄",
  "description": "設定原始錄影與封存成品的儲存位置。",
  "env": "ZZZ_MEDIA_ROOT",
  "type": "path",
  "pathKind": "directory",
  "required": false,
  "default": "Z:/Video/zzz-lab",
  "examples": ["D:/Media/zzz-lab"]
}
```

讀取規格後，展開參數區下方預設收合的「ENVIRONMENT 環境變數」區塊（標題會標示是否含必填欄位），勾選該欄位並輸入路徑或按資料夾圖示選擇。執行時 Bridger 只覆寫此子程序的環境，不把值加入命令列，也不修改 Windows 或 Bridger 自身的環境。未勾選時繼承 Bridger 啟動時的環境；沒有繼承值則由工具自己的程式邏輯決定預設值。JSON 的 `default` 只在欄位啟用後套用，不會自動覆寫既有環境。

第一次「讀取規格」使用繼承環境。若工具會根據環境回覆不同規格，設定後按「套用環境並重讀規格」；此操作只使用目前指令層級的有效環境欄位，不要求其他命令列必填欄位已填完。重讀後保留仍相容的欄位；停用環境欄位再重讀即可恢復繼承值。工具應能在尚未設定環境時回覆基本規格。工具宣告放在根指令時適用所有子命令，放在子命令時只適用該指令分支。

同一區塊的「自訂環境變數」可加入工具未宣告的變數（例如 `HTTP_PROXY`），每個工具各自保存，執行與「套用環境並重讀規格」都會套用。名稱不可空白、前後不可有空白、不可含 `=`；與目前指令層級中工具宣告的環境變數同名（不分大小寫）或彼此重複時會顯示錯誤並停用執行。使用外部規格檔的工具，改在來源警告中按「重新讀取規格檔」。

`media.cjs` 使用的是 `ZZZ_MEDIA_ROOT`；需要 `ZZZ_SITE_ROOT` 的工具可另外宣告同型態欄位。外部工具需自行加入描述，目前 Bridger 不會猜測程式使用了哪些環境變數。Go/Python demo 的 `BRIDGER_DEMO_ROOT` 可用來安全測試這個流程，只顯示值、不操作檔案。

使用 Windows ConPTY + xterm.js，支援 ANSI 顏色、游標移動、原地更新進度、鍵盤輸入、視窗尺寸同步及停止。輸出以原始位元組傳遞，避免中文 UTF-8 分段被破壞；stdout/stderr 合併為終端串流。

指令等待確認時，點擊終端輸入 `y` 並按 Enter，或使用下方「回覆指令」欄位送出。欄位留空可直接送出 Enter，支援同一程序連續多次提示。

拖曳選取文字後可按「複製選取」或 Ctrl+C；沒有選取文字時，Ctrl+C 仍會送往程序以中斷執行。「複製全部」會複製目前保留的終端文字（含捲動記錄，行數上限可在「設定」調整，預設 5,000 行），適合貼上除錯；這不是完整的歷史紀錄檔。

執行前由 Go 重新驗證參數，直接啟動執行檔或選定的腳本執行器，不拼接 shell 指令。Discovery 與執行共用同一組執行器／腳本引數。預覽以引數邊界顯示，並非可直接貼入所有 shell 的指令字串。工作目錄繼承 App 啟動目錄。沒有外部規格檔時，讀取規格也會執行工具；請選擇可信任的 CLI。

## 範圍與限制

- 第一版實作及測試 Windows；尚未提供 macOS/Linux PTY 後端或安裝包。
- v3 尚為 Beta；Go 與前端 runtime 均鎖定 beta.28。
- 一次執行一個程序。停止會關閉 pseudoconsole；刻意脫離終端的背景程序不保證一併結束。
- Discovery 最多 5 秒、stdout/stderr 各 1 MiB。`.cmd`／`.bat` 不支援；自訂執行器需為原生 `.exe`。目前不提供 `python -m module` 或任意前置啟動參數欄位。
- v1 不支援重複參數、複合布林依賴、互斥群組；帶值旗標採 `--flag=value`。詳見協定文件。
- 終端回捲預設保留 5,000 行，可在「設定」調整為 1,000–100,000 行；不將完整輸出持久儲存。

受限測試環境可在程序啟動前設定 `CLI_BRIDGER_WEBVIEW_DATA`，將 WebView profile 指到可寫入的測試目錄。一般啟動使用 Wails 的預設 profile 位置。

## 驗證

`go test ./...` 包含格式／參數驗證與實際 ConPTY 測試：互動輸入、resize、中文與引號引數、ANSI、停止後重新執行，以及三輪約 180 KiB 尾端輸出的完整性檢查。UI 另以桌面 WebView 實測 discovery、預設值、滑桿、條件欄位、執行與原始 JSON 摺疊；原生檔案對話框已開啟確認。
