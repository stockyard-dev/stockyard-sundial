package server

import "net/http"

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashHTML))
}

const dashHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>Sundial</title>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;700&display=swap" rel="stylesheet">
<style>
:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#e8753a;--leather:#a0845c;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c94444;--blue:#5b8dd9;--mono:'JetBrains Mono',monospace}
*{margin:0;padding:0;box-sizing:border-box}
body{background:var(--bg);color:var(--cream);font-family:var(--mono);line-height:1.5}
.hdr{padding:1rem 1.5rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center;gap:1rem;flex-wrap:wrap}
.hdr h1{font-size:.9rem;letter-spacing:2px}
.hdr h1 span{color:var(--rust)}
.main{padding:1.5rem;max-width:960px;margin:0 auto}
.stats{display:grid;grid-template-columns:repeat(3,1fr);gap:.5rem;margin-bottom:1rem}
.st{background:var(--bg2);border:1px solid var(--bg3);padding:.7rem;text-align:center}
.st-v{font-size:1.3rem;font-weight:700;color:var(--gold)}
.st-l{font-size:.5rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-top:.2rem}
.toolbar{display:flex;gap:.5rem;margin-bottom:1rem;flex-wrap:wrap;align-items:center}
.search{flex:1;min-width:180px;padding:.4rem .6rem;background:var(--bg2);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.7rem}
.search:focus{outline:none;border-color:var(--leather)}
.filter-sel{padding:.4rem .5rem;background:var(--bg2);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.65rem}
.count-label{font-size:.6rem;color:var(--cm);margin-bottom:.5rem}
.item{background:var(--bg2);border:1px solid var(--bg3);padding:.8rem 1rem;margin-bottom:.5rem;transition:border-color .2s}
.item:hover{border-color:var(--leather)}
.item.billable{border-left:3px solid var(--gold)}
.item-top{display:flex;justify-content:space-between;align-items:flex-start;gap:.8rem}
.item-desc{font-size:.85rem;font-weight:700;flex:1}
.item-dur{font-size:1.05rem;font-weight:700;color:var(--gold);white-space:nowrap;text-align:right;font-variant-numeric:tabular-nums}
.item-meta{font-size:.55rem;color:var(--cm);margin-top:.4rem;display:flex;gap:.6rem;flex-wrap:wrap;align-items:center}
.item-meta-sep{color:var(--bg3)}
.item-actions{display:flex;gap:.3rem;flex-shrink:0;margin-left:.5rem}
.item-extra{font-size:.58rem;color:var(--cd);margin-top:.4rem;padding-top:.35rem;border-top:1px dashed var(--bg3);display:flex;flex-direction:column;gap:.15rem}
.item-extra-row{display:flex;gap:.4rem}
.item-extra-label{color:var(--cm);text-transform:uppercase;letter-spacing:.5px;min-width:90px}
.item-extra-val{color:var(--cream)}
.badge{font-size:.5rem;padding:.12rem .35rem;text-transform:uppercase;letter-spacing:1px;border:1px solid var(--bg3);color:var(--cm)}
.badge.billable{border-color:var(--gold);color:var(--gold)}
.tag{font-size:.5rem;padding:.1rem .3rem;background:var(--bg3);color:var(--cd)}
.btn{font-size:.6rem;padding:.25rem .5rem;cursor:pointer;border:1px solid var(--bg3);background:var(--bg);color:var(--cd);transition:all .2s;font-family:var(--mono)}
.btn:hover{border-color:var(--leather);color:var(--cream)}
.btn-p{background:var(--rust);border-color:var(--rust);color:#fff}
.btn-sm{font-size:.55rem;padding:.2rem .4rem}
.modal-bg{display:none;position:fixed;inset:0;background:rgba(0,0,0,.65);z-index:100;align-items:center;justify-content:center}
.modal-bg.open{display:flex}
.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:480px;max-width:92vw;max-height:90vh;overflow-y:auto}
.modal h2{font-size:.8rem;margin-bottom:1rem;color:var(--rust);letter-spacing:1px}
.fr{margin-bottom:.6rem}
.fr label{display:block;font-size:.55rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-bottom:.2rem}
.fr input,.fr select,.fr textarea{width:100%;padding:.4rem .5rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.7rem}
.fr input:focus,.fr select:focus,.fr textarea:focus{outline:none;border-color:var(--leather)}
.fr-checkbox{display:flex;align-items:center;gap:.5rem;margin-bottom:.6rem}
.fr-checkbox input{width:auto;margin:0}
.fr-checkbox label{display:inline;font-size:.65rem;color:var(--cd);text-transform:none;letter-spacing:0;margin:0}
.fr-section{margin-top:1rem;padding-top:.8rem;border-top:1px solid var(--bg3)}
.fr-section-label{font-size:.55rem;color:var(--rust);text-transform:uppercase;letter-spacing:1px;margin-bottom:.5rem}
.row2{display:grid;grid-template-columns:1fr 1fr;gap:.5rem}
.acts{display:flex;gap:.4rem;justify-content:flex-end;margin-top:1rem}
.empty{text-align:center;padding:3rem;color:var(--cm);font-style:italic;font-size:.85rem}
.dur-hint{font-size:.5rem;color:var(--cm);margin-top:.15rem}
@media(max-width:600px){.row2{grid-template-columns:1fr}.toolbar{flex-direction:column;align-items:stretch}.search{min-width:100%}.filter-sel{width:100%}}
</style>
</head>
<body>

<div class="hdr">
<h1 id="dash-title"><span>&#9670;</span> SUNDIAL</h1>
<button class="btn btn-p" onclick="openForm()">+ Log Time</button>
</div>

<div class="main">
<div class="stats" id="stats"></div>
<div class="toolbar">
<input class="search" id="search" placeholder="Search description, project, task, tags..." oninput="render()">
<select class="filter-sel" id="project-filter" onchange="render()">
<option value="">All Projects</option>
</select>
<select class="filter-sel" id="billable-filter" onchange="render()">
<option value="">All Entries</option>
<option value="yes">Billable Only</option>
<option value="no">Non-Billable</option>
</select>
</div>
<div class="count-label" id="count"></div>
<div id="list"></div>
</div>

<div class="modal-bg" id="mbg" onclick="if(event.target===this)closeModal()">
<div class="modal" id="mdl"></div>
</div>

<script>
var A='/api';
var RESOURCE='time_entries';

// Field defs drive the form, the rows, and the submit body.
// duration is type 'duration' — accepts "1h 30m", "1:30", "90m", "1.5h", or raw seconds.
// billable is type 'checkbox' (stored as 0/1).
var fields=[
{name:'description',label:'Description',type:'text',required:true,placeholder:'What did you work on?'},
{name:'project',label:'Project',type:'text',placeholder:'Project name'},
{name:'task',label:'Task',type:'text',placeholder:'Specific task'},
{name:'duration',label:'Duration',type:'duration',required:true,placeholder:'1h 30m'},
{name:'start_time',label:'Start',type:'datetime-local'},
{name:'end_time',label:'End',type:'datetime-local'},
{name:'billable',label:'Billable',type:'checkbox'},
{name:'tags',label:'Tags',type:'text',placeholder:'comma separated'}
];

var items=[],editId=null;

// ─── Duration helpers ─────────────────────────────────────────────
// Stored as integer seconds. Display as "1h 30m" or "45m" or "30s".
// Parse accepts: "1h 30m", "1h30m", "1:30", "1.5h", "90m", "5400" (raw seconds).

function fmtDuration(seconds){
var s=parseInt(seconds||0,10);
if(isNaN(s)||s<=0)return'0m';
var h=Math.floor(s/3600);
var m=Math.floor((s%3600)/60);
var sec=s%60;
var out=[];
if(h)out.push(h+'h');
if(m)out.push(m+'m');
if(!h&&!m&&sec)out.push(sec+'s');
return out.length?out.join(' '):'0m';
}

function fmtHours(seconds){
var s=parseInt(seconds||0,10);
if(isNaN(s)||s<=0)return'0h';
var hours=s/3600;
if(hours>=10)return Math.round(hours)+'h';
return hours.toFixed(1)+'h';
}

function parseDuration(str){
if(!str)return 0;
var s=String(str).trim().toLowerCase();
if(!s)return 0;
// Try "h:m" format first (e.g. "1:30")
if(/^\d+:\d+$/.test(s)){
var parts=s.split(':');
return parseInt(parts[0],10)*3600+parseInt(parts[1],10)*60;
}
// Match h/m/s tokens (e.g. "1h 30m", "2h", "45m", "90s", "1.5h")
var total=0;
var matched=false;
var re=/([\d.]+)\s*([hms])/g;
var m;
while((m=re.exec(s))!==null){
matched=true;
var n=parseFloat(m[1]);
if(isNaN(n))continue;
if(m[2]==='h')total+=Math.round(n*3600);
else if(m[2]==='m')total+=Math.round(n*60);
else if(m[2]==='s')total+=Math.round(n);
}
if(matched)return total;
// Fall back to raw integer (assume seconds)
var n=parseInt(s,10);
return isNaN(n)?0:n;
}

function fmtDateTime(s){
if(!s)return'';
try{
var d=new Date(s);
if(isNaN(d.getTime()))return s;
return d.toLocaleString('en-US',{month:'short',day:'numeric',hour:'numeric',minute:'2-digit'});
}catch(e){return s}
}

// Convert ISO 8601 timestamp to the format datetime-local input expects (YYYY-MM-DDTHH:MM)
function toDatetimeLocal(s){
if(!s)return'';
try{
var d=new Date(s);
if(isNaN(d.getTime()))return'';
var pad=function(n){return n<10?'0'+n:''+n};
return d.getFullYear()+'-'+pad(d.getMonth()+1)+'-'+pad(d.getDate())+'T'+pad(d.getHours())+':'+pad(d.getMinutes());
}catch(e){return''}
}

// ─── Loading and rendering ────────────────────────────────────────

async function load(){
try{
var r=await fetch(A+'/'+RESOURCE).then(function(r){return r.json()});
var list=r[RESOURCE]||[];
try{
var extras=await fetch(A+'/extras/'+RESOURCE).then(function(r){return r.json()});
list.forEach(function(it){
var ex=extras[it.id];
if(!ex)return;
Object.keys(ex).forEach(function(k){if(it[k]===undefined)it[k]=ex[k]});
});
}catch(e){}
items=list;
}catch(e){
console.error('load failed',e);
items=[];
}
populateProjectFilter();
renderStats();
render();
}

function populateProjectFilter(){
var sel=document.getElementById('project-filter');
if(!sel)return;
var current=sel.value;
var seen={};
var projects=[];
items.forEach(function(i){
if(i.project&&!seen[i.project]){seen[i.project]=true;projects.push(i.project)}
});
projects.sort();
sel.innerHTML='<option value="">All Projects</option>'+projects.map(function(p){return'<option value="'+esc(p)+'"'+(p===current?' selected':'')+'>'+esc(p)+'</option>'}).join('');
}

function renderStats(){
var totalSec=0;
var billableSec=0;
var weekSec=0;
var weekAgo=Date.now()-7*24*3600*1000;
items.forEach(function(i){
var d=parseInt(i.duration||0,10);
totalSec+=d;
if(i.billable)billableSec+=d;
// "This week" rolls 7 days back from now
var ts=null;
if(i.start_time){ts=new Date(i.start_time).getTime()}
else if(i.created_at){ts=new Date(i.created_at).getTime()}
if(ts&&!isNaN(ts)&&ts>=weekAgo)weekSec+=d;
});
document.getElementById('stats').innerHTML=
'<div class="st"><div class="st-v">'+fmtHours(totalSec)+'</div><div class="st-l">Total Tracked</div></div>'+
'<div class="st"><div class="st-v">'+fmtHours(billableSec)+'</div><div class="st-l">Billable</div></div>'+
'<div class="st"><div class="st-v">'+fmtHours(weekSec)+'</div><div class="st-l">This Week</div></div>';
}

function render(){
var q=(document.getElementById('search').value||'').toLowerCase();
var pf=document.getElementById('project-filter').value;
var bf=document.getElementById('billable-filter').value;
var f=items;
if(pf)f=f.filter(function(i){return i.project===pf});
if(bf==='yes')f=f.filter(function(i){return i.billable});
if(bf==='no')f=f.filter(function(i){return !i.billable});
if(q)f=f.filter(function(i){
return(i.description||'').toLowerCase().includes(q)||
       (i.project||'').toLowerCase().includes(q)||
       (i.task||'').toLowerCase().includes(q)||
       (i.tags||'').toLowerCase().includes(q);
});
document.getElementById('count').textContent=f.length+' entr'+(f.length!==1?'ies':'y');
if(!f.length){
var msg=window._emptyMsg||'No time entries yet. Click "Log Time" to start tracking.';
document.getElementById('list').innerHTML='<div class="empty">'+esc(msg)+'</div>';
return;
}
var h='';
f.forEach(function(i){h+=itemHTML(i)});
document.getElementById('list').innerHTML=h;
}

function itemHTML(i){
var cls='item';
if(i.billable)cls+=' billable';

var h='<div class="'+cls+'"><div class="item-top">';
h+='<div class="item-desc">'+esc(i.description)+'</div>';
h+='<div class="item-dur">'+esc(fmtDuration(i.duration))+'</div>';
h+='<div class="item-actions">';
h+='<button class="btn btn-sm" onclick="openEdit(\''+i.id+'\')">Edit</button>';
h+='<button class="btn btn-sm" onclick="del(\''+i.id+'\')" style="color:var(--red)">&#10005;</button>';
h+='</div></div>';

h+='<div class="item-meta">';
var parts=[];
if(i.project)parts.push('<span>'+esc(i.project)+'</span>');
if(i.task)parts.push('<span>'+esc(i.task)+'</span>');
if(i.start_time)parts.push('<span>'+esc(fmtDateTime(i.start_time))+'</span>');
h+=parts.join('<span class="item-meta-sep">·</span>');
if(i.billable)h+=' <span class="badge billable">BILLABLE</span>';
if(i.tags){
i.tags.split(',').forEach(function(t){
t=t.trim();
if(t)h+=' <span class="tag">#'+esc(t)+'</span>';
});
}
h+='</div>';

// Custom fields from personalization
var customRows='';
fields.forEach(function(f){
if(!f.isCustom)return;
var v=i[f.name];
if(v===undefined||v===null||v==='')return;
customRows+='<div class="item-extra-row">';
customRows+='<span class="item-extra-label">'+esc(f.label)+'</span>';
customRows+='<span class="item-extra-val">'+esc(String(v))+'</span>';
customRows+='</div>';
});
if(customRows)h+='<div class="item-extra">'+customRows+'</div>';

h+='</div>';
return h;
}

// ─── Form ─────────────────────────────────────────────────────────

function fieldByName(n){
for(var i=0;i<fields.length;i++)if(fields[i].name===n)return fields[i];
return null;
}

function fieldHTML(f,value){
var v=value;
if(v===undefined||v===null)v='';
var req=f.required?' *':'';
var ph='';
if(f.placeholder)ph=' placeholder="'+esc(f.placeholder)+'"';
else if(f.name==='description'&&window._placeholderName)ph=' placeholder="'+esc(window._placeholderName)+'"';

if(f.type==='checkbox'){
return '<div class="fr-checkbox"><input type="checkbox" id="f-'+f.name+'"'+(v?' checked':'')+'><label for="f-'+f.name+'">'+esc(f.label)+'</label></div>';
}

var h='<div class="fr"><label>'+esc(f.label)+req+'</label>';

if(f.type==='select'){
h+='<select id="f-'+f.name+'">';
if(!f.required)h+='<option value="">Select...</option>';
(f.options||[]).forEach(function(o){
var sel=(String(v)===String(o))?' selected':'';
var disp=(typeof o==='string')?(o.charAt(0).toUpperCase()+o.slice(1)):String(o);
h+='<option value="'+esc(String(o))+'"'+sel+'>'+esc(disp)+'</option>';
});
h+='</select>';
}else if(f.type==='textarea'){
h+='<textarea id="f-'+f.name+'" rows="2"'+ph+'>'+esc(String(v))+'</textarea>';
}else if(f.type==='duration'){
// Display the integer seconds as the friendly format in the input
var displayVal=v?fmtDuration(v):'';
h+='<input type="text" id="f-'+f.name+'" value="'+esc(displayVal)+'"'+ph+'>';
h+='<div class="dur-hint">Examples: 1h 30m · 1:30 · 90m · 1.5h</div>';
}else if(f.type==='datetime-local'){
var dtVal=v?toDatetimeLocal(v):'';
h+='<input type="datetime-local" id="f-'+f.name+'" value="'+esc(dtVal)+'">';
}else if(f.type==='number'||f.type==='integer'){
h+='<input type="number" id="f-'+f.name+'" value="'+esc(String(v))+'"'+ph+'>';
}else{
var inputType=f.type||'text';
h+='<input type="'+esc(inputType)+'" id="f-'+f.name+'" value="'+esc(String(v))+'"'+ph+'>';
}

h+='</div>';
return h;
}

function formHTML(item){
var i=item||{};
var isEdit=!!item;
var h='<h2>'+(isEdit?'EDIT TIME ENTRY':'NEW TIME ENTRY')+'</h2>';

// Description on its own row
h+=fieldHTML(fieldByName('description'),i.description);

// Project + task
h+='<div class="row2">'+fieldHTML(fieldByName('project'),i.project)+fieldHTML(fieldByName('task'),i.task)+'</div>';

// Duration on its own (it's important and has the hint)
h+=fieldHTML(fieldByName('duration'),i.duration);

// Start + end time
h+='<div class="row2">'+fieldHTML(fieldByName('start_time'),i.start_time)+fieldHTML(fieldByName('end_time'),i.end_time)+'</div>';

// Billable checkbox + tags
h+=fieldHTML(fieldByName('billable'),i.billable);
h+=fieldHTML(fieldByName('tags'),i.tags);

// Custom fields injected by personalization
var customFields=fields.filter(function(f){return f.isCustom});
if(customFields.length){
var sectionLabel=window._customSectionLabel||'Additional Details';
h+='<div class="fr-section"><div class="fr-section-label">'+esc(sectionLabel)+'</div>';
customFields.forEach(function(f){h+=fieldHTML(f,i[f.name])});
h+='</div>';
}

h+='<div class="acts">';
h+='<button class="btn" onclick="closeModal()">Cancel</button>';
h+='<button class="btn btn-p" onclick="submit()">'+(isEdit?'Save':'Log Entry')+'</button>';
h+='</div>';
return h;
}

function openForm(){
editId=null;
document.getElementById('mdl').innerHTML=formHTML();
document.getElementById('mbg').classList.add('open');
var d=document.getElementById('f-description');
if(d)d.focus();
}

function openEdit(id){
var x=null;
for(var j=0;j<items.length;j++){if(items[j].id===id){x=items[j];break}}
if(!x)return;
editId=id;
document.getElementById('mdl').innerHTML=formHTML(x);
document.getElementById('mbg').classList.add('open');
}

function closeModal(){
document.getElementById('mbg').classList.remove('open');
editId=null;
}

// ─── Submit ───────────────────────────────────────────────────────

async function submit(){
var descEl=document.getElementById('f-description');
if(!descEl||!descEl.value.trim()){alert('Description is required');return}

var durEl=document.getElementById('f-duration');
var durSec=durEl?parseDuration(durEl.value):0;
if(durSec<=0){alert('Duration is required (e.g. "1h 30m")');return}

var body={};
var extras={};
fields.forEach(function(f){
var el=document.getElementById('f-'+f.name);
if(!el)return;
var val;
if(f.type==='checkbox')val=el.checked?1:0;
else if(f.type==='duration')val=parseDuration(el.value);
else if(f.type==='datetime-local'){
val=el.value?new Date(el.value).toISOString():'';
}
else if(f.type==='number'||f.type==='integer')val=parseFloat(el.value)||0;
else val=el.value.trim();
if(f.isCustom)extras[f.name]=val;
else body[f.name]=val;
});

var savedId=editId;
try{
if(editId){
var r1=await fetch(A+'/'+RESOURCE+'/'+editId,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
if(!r1.ok){var e1=await r1.json().catch(function(){return{}});alert(e1.error||'Save failed');return}
}else{
var r2=await fetch(A+'/'+RESOURCE,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
if(!r2.ok){var e2=await r2.json().catch(function(){return{}});alert(e2.error||'Save failed');return}
var created=await r2.json();
savedId=created.id;
}
if(savedId&&Object.keys(extras).length){
await fetch(A+'/extras/'+RESOURCE+'/'+savedId,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(extras)}).catch(function(){});
}
}catch(e){
alert('Network error: '+e.message);
return;
}

closeModal();
load();
}

async function del(id){
if(!confirm('Delete this time entry?'))return;
await fetch(A+'/'+RESOURCE+'/'+id,{method:'DELETE'});
load();
}

function esc(s){
if(s===undefined||s===null)return'';
var d=document.createElement('div');
d.textContent=String(s);
return d.innerHTML;
}

document.addEventListener('keydown',function(e){if(e.key==='Escape')closeModal()});

// ─── Personalization ──────────────────────────────────────────────

(function loadPersonalization(){
fetch('/api/config').then(function(r){return r.json()}).then(function(cfg){
if(!cfg||typeof cfg!=='object')return;

if(cfg.dashboard_title){
var h1=document.getElementById('dash-title');
if(h1)h1.innerHTML='<span>&#9670;</span> '+esc(cfg.dashboard_title);
document.title=cfg.dashboard_title;
}

if(cfg.empty_state_message)window._emptyMsg=cfg.empty_state_message;
if(cfg.placeholder_name)window._placeholderName=cfg.placeholder_name;
if(cfg.primary_label)window._customSectionLabel=cfg.primary_label+' Details';

if(Array.isArray(cfg.custom_fields)){
cfg.custom_fields.forEach(function(cf){
if(!cf||!cf.name||!cf.label)return;
if(fieldByName(cf.name))return;
fields.push({
name:cf.name,
label:cf.label,
type:cf.type||'text',
options:cf.options||[],
isCustom:true
});
});
}
}).catch(function(){
}).finally(function(){
load();
});
})();
</script>
</body>
</html>`
