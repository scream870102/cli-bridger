import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { Call, Events } from '@wailsio/runtime';
import '@xterm/xterm/css/xterm.css';
import './style.css';

const $ = id => document.getElementById(id);
const backend = Object.fromEntries(['Describe','Reload','PickPath','Preview','Run','Stop','Input','Resize','LoadSettings','GetToolSettings','SaveSettings','ResetSettings','LoadPreferences','SavePreferences'].map(name=>[name,(...args)=>Call.ByName(`main.App.${name}`,...args)]));
const api = () => backend;
const terminal = new Terminal({convertEol:false, fontFamily:'Cascadia Code, Consolas, monospace',fontSize:16,lineHeight:1.2,scrollback:5000,theme:{background:'#121212',foreground:'#f0f0f0',cursor:'#E28C91'}});
const fit = new FitAddon(); terminal.loadAddon(fit); terminal.open($('terminal')); fit.fit();
let descriptor, descriptionFile='', path=[], values={}, enabled={}, customEnv=[], running=false, revision=0, inputGeneration=0;
let settingsReady=false, settingsQueue=Promise.resolve(), settingsRevision=0, prefs={recordSettings:true,scrollback:5000}, envShown={count:0,required:false};
function settingsMessage(message, failed=false, detail='') { $('settings-message').textContent=message; $('settings-message').title=detail; $('settings-message').dataset.error=String(failed); $('settings-message').setAttribute('role',failed?'alert':'status'); }
function preferencesMessage(message, failed=false) { $('preferences-message').textContent=message; $('preferences-message').dataset.error=String(failed); $('preferences-message').setAttribute('role',failed?'alert':'status'); }
// Blank draft rows are not variables; they never reach the backend or the database.
function envList() { return customEnv.filter(v=>v.name||v.value).map(({name,value})=>({name,value})); }
function saveSettings() {
 if(!settingsReady||!prefs.recordSettings)return;
 const request=++settingsRevision;
 const settings={target:$('executable').value,runner:$('runner').value,customRunner:$('custom-runner').value,path:[...path],values:{...values},enabled:{...enabled},hasDescriptor:!!descriptor,customEnv:envList()};
 settingsMessage('正在儲存設定…');
 settingsQueue=settingsQueue.then(()=>api().SaveSettings(settings)).then(()=>{if(request===settingsRevision)settingsMessage(prefs.recordSettings?'設定已儲存':'未記錄設定');}).catch(e=>{if(request===settingsRevision)settingsMessage(`設定未儲存：${e}`,true);});
}
function applyPreferences() { $('record-settings').checked=prefs.recordSettings; $('scrollback').value=prefs.scrollback; terminal.options.scrollback=prefs.scrollback; }
// Queued with settings saves so a toggle never overtakes an earlier save.
function savePreferences() {
 const next={...prefs};
 preferencesMessage('正在儲存…');
 settingsQueue=settingsQueue.then(()=>api().SavePreferences(next)).then(()=>preferencesMessage('已儲存')).catch(async e=>{
  preferencesMessage(`未儲存：${e}`,true);
  try{prefs=await api().LoadPreferences();applyPreferences();}catch{}
 });
}
function setDescriptionSource(file='') { descriptionFile=file; $('source-warning').hidden=!file; $('source-path').textContent=file?`來源檔案：${file}`:''; }
function error(e) { $('error').hidden=!e; $('error').textContent=e ? String(e) : ''; }
function busy(value) { running=value; inputGeneration++; $('reset-settings').disabled=value||!descriptor; $('terminal-input').disabled=true; $('send-input').disabled=true; $('commands').inert=value||!descriptor; $('fields').disabled=value||!descriptor; $('run').disabled=value||!descriptor; $('reload').disabled=value||!descriptor; $('reload-file').disabled=value||!descriptor; $('stop').disabled=!value; for(const id of ['load','browse','executable','runner','custom-runner','browse-runner'])$(id).disabled=value; $('status').textContent=value?'執行中':descriptor?'已連接':'尚未連接'; }
async function attempt(fn) { try { error(''); await fn(); } catch(e) {error(e);} }
function invalidate() { descriptor=null;path=[];values={};enabled={};customEnv=[];setDescriptionSource();revision++;$('env-section').hidden=true;$('commands').replaceChildren();$('parameters').replaceChildren();$('env-parameters').replaceChildren();$('custom-env').replaceChildren();$('raw').textContent='尚未載入';$('preview').textContent='請讀取工具規格';$('description').textContent='載入工具後，這裡會顯示指令與參數。';busy(false);saveSettings(); }
function el(tag, text, cls) { const e=document.createElement(tag); if(text!==undefined)e.textContent=text; if(cls)e.className=cls; return e; }
function icon(name) { const ns='http://www.w3.org/2000/svg', svg=document.createElementNS(ns,'svg'), use=document.createElementNS(ns,'use'); svg.setAttribute('class','icon'); svg.setAttribute('aria-hidden','true'); use.setAttribute('href',`#i-${name}`); svg.append(use); return svg; }
function iconButton(name, label) { const b=el('button',undefined,'icon-button'); b.type='button'; b.title=label; b.setAttribute('aria-label',label); b.append(icon(name)); return b; }
function commandChain() { const list=[descriptor.root]; for(const id of path)list.push(list.at(-1).commands.find(c=>c.id===id)); return list; }
function restoreToolValues(saved, previous) {
 path=[];let command=descriptor.root;
 for(const id of saved.path||[]){const next=command.commands?.find(c=>c.id===id);if(!next)break;path.push(id);command=next;}
 const parameters=c=>[...(c.parameters||[]),...(c.commands||[]).flatMap(parameters)];
 const oldParameters=new Map(parameters(previous.root).map(p=>[p.id,p]));
 values={};enabled={};
 for(const p of parameters(descriptor.root)) {
  const old=oldParameters.get(p.id);
  if(old&&old.type===p.type&&old.flag===p.flag&&old.env===p.env) {
   if(Object.hasOwn(saved.values||{},p.id))values[p.id]=saved.values[p.id];
   if(saved.enabled?.[p.id]===true)enabled[p.id]=true;
  }
 }
 // Custom variables do not depend on the schema; collisions surface in the preview.
 customEnv=(saved.customEnv||[]).map(({name='',value=''})=>({name,value}));
}
function effective(p) { return Object.hasOwn(values,p.id)?values[p.id]:p.default; }
function activeParameters() {
 const active=new Map();
 for(const c of commandChain())for(const p of c.parameters||[]) {
  const dep=p.dependsOn;
  const visible=!dep || (active.has(dep.id) && effective(active.get(dep.id))===dep.value);
  if(visible&&(p.required||enabled[p.id]===true))active.set(p.id,p);
 }
 return active;
}
async function preview(persist=true) {
 if(persist)saveSettings();
 const request=++revision;
 try { const args=await api().Preview(path,values,enabled,envList()); if(request!==revision)return; $('preview').textContent=args.map(a=>JSON.stringify(a)).join(' '); $('run').disabled=running; }
 catch(e) { if(request!==revision)return; $('preview').textContent=String(e); $('run').disabled=true; }
}
function chip(label, value) { const c=el('span',label,'chip'); if(value!==undefined)c.append(el('b',String(value))); return c; }
function updateEnvSummary() {
 const summary=$('env-summary'); summary.replaceChildren();
 if(envShown.required)summary.append(el('span','含必填','badge required'));
 if(envShown.count)summary.append(chip('工具提供',envShown.count));
 if(envList().length)summary.append(chip('自訂',envList().length));
}
// Mirrors protocol.CustomEnvironment only to point at the row; the backend message explains it.
function checkCustomEnv() {
 const declared=new Set(commandChain().flatMap(c=>c.parameters||[]).filter(p=>p.env).map(p=>p.env.toUpperCase())), seen=new Set();
 [...$('custom-env').children].forEach((row,i)=>{
  const {name,value}=customEnv[i], key=name.toUpperCase();
  const bad=!!(name||value)&&(!name||name.trim()!==name||/[=\0]/.test(name)||declared.has(key)||seen.has(key));
  if(name)seen.add(key);
  row.querySelector('input').setAttribute('aria-invalid',String(bad));
 });
 updateEnvSummary();
}
function renderCustomEnv() {
 const list=$('custom-env'); list.replaceChildren();
 customEnv.forEach((v,i)=>{
  const row=el('div',undefined,'env-row'), name=el('input'), value=el('input'), remove=iconButton('x','移除此變數');
  name.value=v.name;value.value=v.value;name.placeholder='名稱';value.placeholder='值';name.spellcheck=value.spellcheck=false;
  name.setAttribute('aria-label',`自訂變數 ${i+1} 名稱`);value.setAttribute('aria-label',`自訂變數 ${i+1} 值`);
  name.oninput=()=>{v.name=name.value;checkCustomEnv();preview();};
  value.oninput=()=>{v.value=value.value;checkCustomEnv();preview();};
  remove.onclick=()=>{customEnv.splice(i,1);renderCustomEnv();preview();};
  row.append(name,el('span','='),value,remove);list.append(row);
 });
 checkCustomEnv();
}
function parameterCard(p, active) {
 const row=el('div',undefined,'parameter'), title=el('div',undefined,'parameter-title'), chips=el('span',undefined,'chips');
 const label=el('label',p.name||p.id); label.htmlFor=`value-${p.id}`;
 const help=el('p',p.description,'hint parameter-description');help.id=`help-${p.id}`;
 if(!p.required) {
  const toggle=el('input');toggle.type='checkbox';toggle.checked=enabled[p.id]===true;toggle.setAttribute('aria-label',`啟用 ${p.id}`);toggle.setAttribute('aria-describedby',help.id);
  toggle.onchange=()=>{enabled[p.id]=toggle.checked;if(toggle.checked&&p.type==='bool'&&effective(p)===undefined)values[p.id]=true;render();};title.append(toggle);
 }
 title.append(label);
 if(p.env||p.flag)title.append(el('span',p.env?`ENV: ${p.env}`:p.flag,'flag'));
 if(p.required)chips.append(el('span','必填','badge required'));
 chips.append(el('span',p.type,'badge'));
 if(p.default!==undefined)chips.append(chip('預設',p.default));
 if(p.examples?.length) {
  const examples=chip('例');
  p.examples.forEach(v=>{const b=el('button',String(v));b.type='button';b.title=`填入 ${v}`;b.onclick=()=>{values[p.id]=v;if(!p.required)enabled[p.id]=true;render();};examples.append(b);});
  chips.append(examples);
 }
 title.append(chips);row.append(title);
 if(active.has(p.id)) {
  const controls=el('div',undefined,'controls'); let input;
  if(p.enum?.length) {input=el('select');input.append(new Option('請選擇',''));p.enum.forEach(v=>input.append(new Option(String(v),String(v))));}
  else { input=el('input');input.type=p.type==='bool'?'checkbox':['int','float'].includes(p.type)?'number':'text'; }
  input.id=`value-${p.id}`;
  input.setAttribute('aria-describedby',help.id);
  const numeric=['int','float'].includes(p.type), current=effective(p);
  if(p.type==='bool'&&!p.enum?.length)input.checked=current===true;else input.value=current??'';
  if(numeric){input.step=p.type==='int'?'1':'any';if(p.limits?.min!=null)input.min=p.limits.min;if(p.limits?.max!=null)input.max=p.limits.max;}
  // Protocol length is Unicode codepoints; HTML maxlength counts UTF-16 units.
  input.required=!!p.required; controls.append(input);
  let slider;
  if(numeric && p.limits?.min!=null && p.limits?.max!=null) {
   slider=el('input');slider.type='range';slider.min=p.limits.min;slider.max=p.limits.max;slider.step=p.type==='int'?'1':'any';slider.value=current??p.limits.min;slider.setAttribute('aria-label',`${p.id} 滑桿`);
   slider.setAttribute('aria-describedby',help.id);
   slider.oninput=()=>{input.value=slider.value;values[p.id]=Number(slider.value);preview();};slider.onchange=()=>render();controls.append(slider);
  }
  input.oninput=()=>{ values[p.id]=p.type==='bool'?(p.enum?.length?input.value==='true':input.checked):numeric&&input.value!==''?Number(input.value):input.value; if(slider&&input.value!=='')slider.value=input.value;preview();};
  input.onchange=()=>render();
  if(p.type==='path'){const pick=iconButton('folder','選擇…');pick.onclick=()=>attempt(async()=>{const selected=await api().PickPath(p.pathKind||'file');if(selected){values[p.id]=selected;render();}});controls.append(pick);}
  row.append(controls);
 } else row.classList.add('inactive');
 row.append(help);
 return row;
}
function render() {
 const commands=$('commands'), parameters=$('parameters'), envParameters=$('env-parameters'); commands.replaceChildren(); parameters.replaceChildren(); envParameters.replaceChildren();
 const chain=commandChain(), active=activeParameters();
 envShown={count:0,required:false};
 $('env-section').hidden=false;
 $('reload').hidden=!!descriptionFile;
 chain.forEach((command,level)=>{
  if(command.commands?.length) {
   const row=el('div',undefined,'command-row'); row.append(el('span',level?'下一層子命令':'子命令','hint'));
   const root=el('button','此層命令',path.length===level?'selected':''); root.append(el('small',command.description,'command-help'));root.onclick=()=>{path=path.slice(0,level); render();};row.append(root);
   command.commands.forEach(c=>{const b=el('button',c.name,path[level]===c.id?'selected':'');b.append(el('small',c.description,'command-help'));b.onclick=()=>{path=[...path.slice(0,level),c.id];render();};row.append(b);});commands.append(row);
  }
  parameters.append(el('h3',command.name||descriptor.name),el('p',command.description,'hint command-description'));
  for(const p of command.parameters||[]) {
   const dep=p.dependsOn;
   if(dep&&(!active.has(dep.id)||effective(active.get(dep.id))!==dep.value))continue;
   if(p.env){envParameters.append(parameterCard(p,active));envShown.count++;envShown.required||=!!p.required;}
   else parameters.append(parameterCard(p,active));
  }
 });
 renderCustomEnv();
 preview();
}
$('executable').oninput=invalidate;
$('custom-runner').oninput=invalidate;
$('browse').onclick=()=>attempt(async()=>{const file=await api().PickPath('file');if(file){$('executable').value=file;invalidate();}});
$('runner').onchange=()=>{$('custom-runner-row').hidden=$('runner').value!=='custom';invalidate();};
$('browse-runner').onclick=()=>attempt(async()=>{const file=await api().PickPath('file');if(file){$('custom-runner').value=file;invalidate();}});
$('load').onclick=()=>attempt(async()=>{
 invalidate();busy(true);$('stop').disabled=true;$('status').textContent='讀取規格中…';$('preview').textContent='讀取中…';
 try {
  await settingsQueue;
  const runner=$('runner').value==='custom'?$('custom-runner').value:$('runner').value;
  if(!runner.trim())throw Error('請指定執行器路徑');
  const result=await api().Describe($('executable').value,runner);
  const cached=await api().GetToolSettings();
  descriptor=result.descriptor;setDescriptionSource(result.descriptionFile);path=[];values={};enabled={};customEnv=[];
  if(cached?.loaded)restoreToolValues(cached.settings,cached.loaded.descriptor);
  $('executable').value=result.target;$('description').textContent=`${descriptor.name} — ${descriptor.description||''}`;$('raw').textContent=result.raw;render();
 }
 finally{busy(false);}
});
$('add-env').onclick=()=>{customEnv.push({name:'',value:''});renderCustomEnv();$('custom-env').lastElementChild.querySelector('input').focus();};
$('run').onclick=()=>attempt(async()=>{busy(true);terminal.clear();$('copy-status').textContent='';$('terminal-input').value='';try{await api().Run(path,values,enabled,envList(),terminal.cols,terminal.rows);$('terminal-input').disabled=!running;$('send-input').disabled=!running;terminal.focus();}catch(e){busy(false);throw e;}});
$('reload').onclick=$('reload-file').onclick=()=>attempt(async()=>{
 // Freeze effective env defaults so reloading a dynamic descriptor keeps the applied value.
 const previous=new Map(commandChain().flatMap(c=>c.parameters||[]).map(p=>[p.id,p]));
 const retained={...values};for(const p of activeParameters().values())if(p.env&&effective(p)!==undefined)retained[p.id]=effective(p);
 revision++;busy(true);$('stop').disabled=true;$('status').textContent='重讀規格中…';
 try {
  await settingsQueue;
  const result=await api().Reload(path,values,enabled,envList());descriptor=result.descriptor;setDescriptionSource(result.descriptionFile);
  let command=descriptor.root;const nextPath=[];
  for(const id of path){const next=command.commands?.find(c=>c.id===id);if(!next)break;nextPath.push(id);command=next;}path=nextPath;
  const nextValues={},nextEnabled={};
  for(const c of commandChain())for(const p of c.parameters||[]){const old=previous.get(p.id);if(old&&old.type===p.type&&old.env===p.env&&old.flag===p.flag){if(Object.hasOwn(retained,p.id))nextValues[p.id]=retained[p.id];if(enabled[p.id]===true)nextEnabled[p.id]=true;}}
  values=nextValues;enabled=nextEnabled;$('raw').textContent=result.raw;$('description').textContent=`${descriptor.name} — ${descriptor.description}`;
 } finally {busy(false);render();}
});
$('stop').onclick=()=>attempt(()=>{inputGeneration++;$('terminal-input').disabled=true;$('send-input').disabled=true;return api().Stop();});
$('clear').onclick=()=>terminal.clear();
let inputQueue=Promise.resolve();
function sendInput(data) {
 if(!running||$('send-input').disabled)return Promise.resolve();
 const generation=inputGeneration;
 const sent=inputQueue.then(()=>{if(running&&generation===inputGeneration)return api().Input(data);});
 inputQueue=sent.catch(error);
 return sent;
}
$('terminal-input-form').onsubmit=event=>{
 event.preventDefault();
 if(!running)return;
 const input=$('terminal-input'), text=input.value;
 input.value='';
 attempt(()=>sendInput(text+'\r'));
 input.focus();
};
terminal.onData(data=>{sendInput(data).catch(error);});
terminal.onSelectionChange(()=>{$('copy-selection').disabled=!terminal.hasSelection();});
async function copyOutput(selectedOnly=false) {
 let text=terminal.getSelection();
 if(!selectedOnly) {
  const buffer=terminal.buffer.active, lines=[];
  for(let i=0;i<buffer.length;i++) {
   const line=buffer.getLine(i), next=buffer.getLine(i+1);
   lines.push(line.translateToString(!next?.isWrapped)+(next?.isWrapped?'':'\n'));
  }
  text=lines.join('').replace(/\n+$/,'');
 }
 if(!text){$('copy-status').textContent='沒有可複製的內容';return;}
 await navigator.clipboard.writeText(text);
 $('copy-status').textContent=selectedOnly?'已複製選取文字':'已複製保留的終端輸出';
}
$('copy-selection').onclick=()=>attempt(()=>copyOutput(true));
$('copy-output').onclick=()=>attempt(()=>copyOutput());
terminal.attachCustomKeyEventHandler(event=>{
 if(event.ctrlKey&&!event.altKey&&event.key.toLowerCase()==='c'&&(event.shiftKey||terminal.hasSelection())) {
  if(event.type==='keydown'){event.preventDefault();attempt(()=>copyOutput(terminal.hasSelection()));}
  return false;
 }
 return true;
});
new ResizeObserver(()=>{fit.fit();if(running)api().Resize(terminal.cols,terminal.rows).catch(error);}).observe($('terminal'));
Events.On('terminal:data',event=>terminal.write(Uint8Array.from(atob(event.data),c=>c.charCodeAt(0))));
Events.On('terminal:exit',event=>{terminal.write(`\r\n\x1b[90m${event.data}\x1b[0m\r\n`);busy(false);preview();});

$('open-settings').onclick=()=>{preferencesMessage('');$('settings-dialog').showModal();};
$('settings-dialog').onclick=event=>{if(event.target===$('settings-dialog'))$('settings-dialog').close();};
$('record-settings').onchange=()=>{
 prefs.recordSettings=$('record-settings').checked;savePreferences();
 // An empty form must not replace the last recorded session.
 if(!prefs.recordSettings)settingsMessage('未記錄設定');else if(descriptor)saveSettings();else settingsMessage('');
};
$('scrollback').onchange=()=>{
 const lines=Math.round(Number($('scrollback').value));
 prefs.scrollback=Number.isFinite(lines)&&$('scrollback').value!==''?Math.min(100000,Math.max(1000,lines)):5000;
 applyPreferences();savePreferences();
};

$('reset-settings').onclick=async()=>{
 if(!descriptor)return;
 settingsReady=false;busy(true);$('stop').disabled=true;$('status').textContent='重置設定中…';
 try {
  await settingsQueue;
  await api().ResetSettings();
  path=[];values={};enabled={};customEnv=[];error('');render();settingsMessage('已清除此工具的快取，其他工具的設定不受影響');
 } catch(e) {settingsMessage(`重置失敗：${e}`,true);}
 finally {settingsReady=true;busy(false);if(descriptor)preview(false);}
};

async function restoreSettings() {
 busy(true);$('stop').disabled=true;$('status').textContent='還原設定中…';
 try {
  try {prefs=await api().LoadPreferences();}
  finally {applyPreferences();}
  if(!prefs.recordSettings){settingsMessage('未記錄設定');return;}
  const restored=await api().LoadSettings();
  if(!restored)return;
  const saved=restored.settings;
  $('executable').value=saved.target||'';$('runner').value=saved.runner||'auto';$('custom-runner').value=saved.customRunner||'';
  $('custom-runner-row').hidden=$('runner').value!=='custom';
  if(restored.loaded) {
   const loaded=restored.loaded;descriptor=loaded.descriptor;setDescriptionSource(loaded.descriptionFile);
   restoreToolValues(saved,descriptor);
   $('raw').textContent=loaded.raw;$('description').textContent=`${descriptor.name} — ${descriptor.description}`;render();
  }
  if(restored.warning)settingsMessage(restored.warning,true);
  else if(restored.loaded)settingsMessage('已還原上次設定',false,'規格使用儲存的版本，需要更新時請重新讀取。');
  else settingsMessage('已還原上次選取的工具',false,'請讀取規格。');
 } catch(e) {settingsMessage(`無法還原設定：${e}`,true);}
 finally {settingsReady=true;busy(false);if(descriptor)preview(false);}
}
restoreSettings();
