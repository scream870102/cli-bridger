import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { Call, Events } from '@wailsio/runtime';
import '@xterm/xterm/css/xterm.css';
import './style.css';

const $ = id => document.getElementById(id);
const backend = Object.fromEntries(['Describe','Reload','PickPath','Preview','Run','Stop','Input','Resize'].map(name=>[name,(...args)=>Call.ByName(`main.App.${name}`,...args)]));
const api = () => backend;
const terminal = new Terminal({convertEol:false, fontFamily:'Cascadia Code, Consolas, monospace',fontSize:16,lineHeight:1.2,scrollback:5000,theme:{background:'#121212',foreground:'#f0f0f0',cursor:'#E28C91'}});
const fit = new FitAddon(); terminal.loadAddon(fit); terminal.open($('terminal')); fit.fit();
let descriptor, descriptionFile='', path=[], values={}, enabled={}, running=false, revision=0, inputGeneration=0;
function setDescriptionSource(file='') { descriptionFile=file; $('source-warning').hidden=!file; $('source-path').textContent=file?`來源檔案：${file}`:''; }
function error(e) { $('error').hidden=!e; $('error').textContent=e ? String(e) : ''; }
function busy(value) { running=value; inputGeneration++; $('terminal-input').disabled=true; $('send-input').disabled=true; $('commands').inert=value||!descriptor; $('fields').disabled=value||!descriptor; $('run').disabled=value||!descriptor; $('reload').disabled=value||!descriptor; $('stop').disabled=!value; for(const id of ['load','browse','executable','runner','custom-runner','browse-runner'])$(id).disabled=value; $('status').textContent=value?'執行中':descriptor?'已連接':'尚未連接'; }
async function attempt(fn) { try { error(''); await fn(); } catch(e) {error(e);} }
function invalidate() { descriptor=null;setDescriptionSource();revision++;$('reload').hidden=true;$('commands').replaceChildren();$('parameters').replaceChildren();$('raw').textContent='尚未載入';$('preview').textContent='請讀取工具規格';$('description').textContent='載入工具後，這裡會顯示指令與參數。';busy(false); }
function el(tag, text, cls) { const e=document.createElement(tag); if(text!==undefined)e.textContent=text; if(cls)e.className=cls; return e; }
function commandChain() { const list=[descriptor.root]; for(const id of path)list.push(list.at(-1).commands.find(c=>c.id===id)); return list; }
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
async function preview() {
 const request=++revision;
 try { const args=await api().Preview(path,values,enabled); if(request!==revision)return; $('preview').textContent=args.map(a=>JSON.stringify(a)).join(' '); $('run').disabled=running; }
 catch(e) { if(request!==revision)return; $('preview').textContent=String(e); $('run').disabled=true; }
}
function render() {
 const commands=$('commands'), parameters=$('parameters'); commands.replaceChildren(); parameters.replaceChildren();
 const chain=commandChain(), active=activeParameters();
 $('reload').hidden=!descriptionFile&&!chain.some(c=>(c.parameters||[]).some(p=>p.env));
 $('reload').textContent=descriptionFile?'重新讀取規格檔':'套用環境並重讀規格';
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
   const row=el('div',undefined,'parameter'), title=el('div',undefined,'parameter-title');
   const label=el('label',`${p.name||p.id}${p.env?` (ENV: ${p.env})`:p.flag?` (${p.flag})`:''}`); label.htmlFor=`value-${p.id}`;
   if(!p.required) {
    const toggle=el('input');toggle.type='checkbox';toggle.checked=enabled[p.id]===true;toggle.setAttribute('aria-label',`啟用 ${p.id}`);toggle.setAttribute('aria-describedby',`help-${p.id}`);
    toggle.onchange=()=>{enabled[p.id]=toggle.checked;if(toggle.checked&&p.type==='bool'&&effective(p)===undefined)values[p.id]=true;render();};title.append(toggle);
   }
   title.append(label,el('span',p.required?'必填':p.type,'badge'));row.append(title);
   const help=el('p',p.description+(p.env?'\n'+(p.required?'僅設定此子程序的環境變數。':'勾選後覆寫此子程序的環境；未勾選時繼承原值。'):''),'hint parameter-description');help.id=`help-${p.id}`;row.append(help);
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
    if(p.type==='path'){const pick=el('button','選擇…');pick.onclick=()=>attempt(async()=>{const selected=await api().PickPath(p.pathKind||'file');if(selected){values[p.id]=selected;render();}});controls.append(pick);}
    row.append(controls);
    if(p.default!==undefined)row.append(el('div',`預設：${String(p.default)}`,'hint'));
    if(p.examples?.length){const examples=el('div',undefined,'examples');examples.append(el('span','常用值','hint'));p.examples.forEach(v=>{const b=el('button',String(v));b.onclick=()=>{values[p.id]=v;render();};examples.append(b);});row.append(examples);}
   }
   parameters.append(row);
  }
 });
 preview();
}
$('executable').oninput=invalidate;
$('custom-runner').oninput=invalidate;
$('browse').onclick=()=>attempt(async()=>{const file=await api().PickPath('file');if(file){$('executable').value=file;invalidate();}});
$('runner').onchange=()=>{$('custom-runner-row').hidden=$('runner').value!=='custom';invalidate();};
$('browse-runner').onclick=()=>attempt(async()=>{const file=await api().PickPath('file');if(file){$('custom-runner').value=file;invalidate();}});
$('load').onclick=()=>attempt(async()=>{
 invalidate();$('preview').textContent='讀取中…';for(const id of ['load','browse','executable','runner','custom-runner','browse-runner'])$(id).disabled=true;
 try {const runner=$('runner').value==='custom'?$('custom-runner').value:$('runner').value;if(!runner.trim())throw Error('請指定執行器路徑');const result=await api().Describe($('executable').value,runner);descriptor=result.descriptor;setDescriptionSource(result.descriptionFile);path=[];values={};enabled={};$('executable').value=result.target;$('description').textContent=`${descriptor.name} — ${descriptor.description||''}`;$('raw').textContent=result.raw;render();}
 finally{busy(false);}
});
$('run').onclick=()=>attempt(async()=>{busy(true);terminal.clear();$('copy-status').textContent='';$('terminal-input').value='';try{await api().Run(path,values,enabled,terminal.cols,terminal.rows);$('terminal-input').disabled=!running;$('send-input').disabled=!running;terminal.focus();}catch(e){busy(false);throw e;}});
$('reload').onclick=()=>attempt(async()=>{
 // Freeze effective env defaults so reloading a dynamic descriptor keeps the applied value.
 const previous=new Map(commandChain().flatMap(c=>c.parameters||[]).map(p=>[p.id,p]));
 const retained={...values};for(const p of activeParameters().values())if(p.env&&effective(p)!==undefined)retained[p.id]=effective(p);
 revision++;busy(true);$('stop').disabled=true;$('status').textContent='重讀規格中…';
 try {
  const result=await api().Reload(path,values,enabled);descriptor=result.descriptor;setDescriptionSource(result.descriptionFile);
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
