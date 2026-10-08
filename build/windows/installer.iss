#ifndef AppVersion
  #error AppVersion must be supplied by package.ps1
#endif
#ifndef NumericVersion
  #error NumericVersion must be supplied by package.ps1
#endif

[Setup]
AppId={{550C3B72-0BCE-44A2-850F-D61E61126A95}
AppName=CLI Bridger
AppVersion={#AppVersion}
VersionInfoVersion={#NumericVersion}
DefaultDirName={localappdata}\Programs\CLI Bridger
DefaultGroupName=CLI Bridger
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0.17763
OutputBaseFilename=CLI-Bridger-{#AppVersion}-windows-x64-setup
SetupIconFile=icon.ico
UninstallDisplayIcon={app}\cli-bridger.exe
LicenseFile=..\..\LICENSE
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
CloseApplications=yes

[Files]
Source: "..\bin\cli-bridger.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\bin\demo-cli.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\bin\MicrosoftEdgeWebview2Setup.exe"; Flags: dontcopy

[Icons]
Name: "{group}\CLI Bridger"; Filename: "{app}\cli-bridger.exe"; WorkingDir: "{app}"

[Run]
Filename: "{app}\cli-bridger.exe"; Description: "Launch CLI Bridger"; Flags: nowait postinstall skipifsilent

[Code]
function HasWebView2At(RootKey: Integer): Boolean;
var
  Version: String;
begin
  Result := RegQueryStringValue(RootKey,
    'Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}', 'pv', Version)
    and (Version <> '') and (Version <> '0.0.0.0');
end;

function HasWebView2: Boolean;
begin
  Result := HasWebView2At(HKLM32) or HasWebView2At(HKCU32);
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  ExitCode: Integer;
begin
  Result := '';
  if HasWebView2 then Exit;
  ExtractTemporaryFile('MicrosoftEdgeWebview2Setup.exe');
  if not Exec(ExpandConstant('{tmp}\MicrosoftEdgeWebview2Setup.exe'),
    '/silent /install', '', SW_HIDE, ewWaitUntilTerminated, ExitCode) then
    Result := 'Unable to start the Microsoft WebView2 installer. Install WebView2 Runtime and try again.'
  else if not HasWebView2 then
    Result := 'WebView2 Runtime installation failed (code ' + IntToStr(ExitCode) + '). Check your internet connection or install WebView2 Runtime manually, then retry.';
end;
