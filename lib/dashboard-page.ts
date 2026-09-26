// lib/dashboard-page.ts — SATU file = halaman HTML dashboard.
// Di-import bin/tor-dashboard (Bun serve string ini di GET /).
// Edit UI di sini tanpa menyentuh logika server/SSE.
export const PAGE = `<!doctype html><html lang="id"><head><meta charset="utf-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>torstack dashboard · EVENT-DRIVEN</title>
<style>
body{font-family:system-ui,sans-serif;background:#0f1115;color:#e6e6e6;margin:0;padding:16px;max-width:940px}
.card{background:#181c24;border:1px solid #2a3040;border-radius:12px;padding:14px;margin-bottom:12px}
.row{display:flex;gap:10px;flex-wrap:wrap;align-items:center}
button{background:#2f6feb;border:0;color:#fff;padding:8px 14px;border-radius:8px;cursor:pointer}
button.ghost{background:#2a3040}button.danger{background:#b3261e}button:disabled{opacity:.5}
input,textarea{background:#0f1115;color:#e6e6e6;border:1px solid #2a3040;border-radius:8px;padding:8px;width:100%;box-sizing:border-box;font-family:monospace}
.badge{padding:2px 10px;border-radius:99px;font-size:12px}.on{background:#1a5c2e}.off{background:#5c1a1a}
pre{background:#0a0c10;padding:10px;border-radius:8px;overflow:auto;max-height:240px;font-size:12px}
small{color:#9aa3b2}
.dot{display:inline-block;width:9px;height:9px;border-radius:99px;background:#5c1a1a;margin-right:5px;vertical-align:middle}
.dot.live{background:#2ea043;animation:pulse 1.4s infinite}
@keyframes pulse{0%{box-shadow:0 0 0 0 rgba(46,160,67,.6)}70%{box-shadow:0 0 0 8px rgba(46,160,67,0)}100%{box-shadow:0 0 0 0 rgba(46,160,67,0)}}
#ip.flash{animation:flash 1s}@keyframes flash{0%{background:#2f6feb;color:#fff}100%{background:transparent}}
.chip{background:#2a3040;border-radius:99px;padding:2px 10px;font-size:12px;font-family:monospace}
#toast{position:fixed;bottom:16px;right:16px;background:#1a5c2e;padding:10px 16px;border-radius:10px;display:none}
#next{font-variant-numeric:tabular-nums;font-weight:bold}
#feed div{border-bottom:1px solid #222a38;padding:2px 0;font-size:12px;font-family:monospace}
.t-ev{color:#9aa3b2}.t-ip{color:#2f6feb}.t-bw{color:#2ea043}.t-err{color:#f85149}
</style></head><body>
<h2>🧅 torstack <small><span id="livedot" class="dot"></span><span id="livetxt">connecting…</span> · event-driven</small></h2>
<div class="card"><div class="row">
<div>tor: <span id="tor" class="badge">…</span></div>
<div>privoxy: <span id="priv" class="badge">…</span></div>
<div>autorotate: <span id="auto" class="badge">…</span></div>
<div>events: <span id="evrun" class="badge">…</span></div>
<div>IP: <b id="ip">…</b> <small id="ipat"></small></div>
<div>⏳ next: <span id="next">-</span></div>
</div><div class="row" style="margin-top:6px">
<small>boot:</small><b id="boot">-</b>
<small>bw:</small><b id="bw">-</b>
<small>circ:</small><b id="circ">-</b>
</div><div class="row" style="margin-top:6px"><small>riwayat IP:</small><span id="hist" class="row"></span></div>
<div class="row" style="margin-top:8px">
<button id="btnRotate" onclick="doRotate()">🔁 Rotate sekarang</button>
<label style="font-size:13px"><input type="checkbox" id="live" checked style="width:auto"/> live log</label>
<span><small id="clock"></small></span>
</div><pre id="msg"></pre></div>
<div class="card"><h3>⚡ Event realtime <small id="evcount"></small></h3><div id="feed"><small>tunggu event…</small></div></div>
<div class="card"><h3>⏱ Auto-rotate + JS Hook</h3>
<div class="row"><label>Interval (detik): <input id="interval" type="number" min="10" value="60" style="width:120px"/></label>
<button onclick="autoStart()">▶ Start</button>
<button class="danger" onclick="autoStop()">⏹ Stop</button>
<button class="ghost" onclick="loadConfig()">Muat hook</button>
<button onclick="saveConfig()">💾 Simpan</button></div>
<small>Hook = <code>hooks/post-rotate.mjs</code>. Env: OLD_IP, NEW_IP, TIMESTAMP.</small>
<textarea id="hook" rows="14"></textarea>
</div>
<div class="card"><h3>📜 Logs <small id="logmeta"></small></h3><div class="row">
<button class="ghost" onclick="setLog('tor')">tor.log</button>
<button class="ghost" onclick="setLog('auto')">autorotate.log</button>
<label style="font-size:13px"><input type="checkbox" id="follow" checked style="width:auto"/> follow</label></div>
<pre id="logs">…</pre></div>
<div id="toast"></div>
<script>
let S={nextIn:-1,interval:60,ip:'',lastRotate:0},logWhich='auto',lastLogs='',evN=0;
async function api(p,m='GET',b){const r=await fetch(p,{method:m,headers:{'content-type':'application/json'},body:b?JSON.stringify(b):undefined});return r.json();}
function toast(t,err){const e=document.getElementById('toast');e.textContent=t;e.style.background=err?'#5c1a1a':'#1a5c2e';e.style.display='block';setTimeout(()=>e.style.display='none',3500);}
function badge(id,v){const e=document.getElementById(id);e.textContent=v?'RUN':'STOP';e.className='badge '+(v?'on':'off');}
function renderSnap(s){S={...S,...s};badge('tor',s.tor);badge('priv',s.priv);badge('auto',s.auto);badge('evrun',s.evRun);
 document.getElementById('interval').value=s.interval;
 if(s.ip){const ipEl=document.getElementById('ip');ipEl.textContent=s.ip;}
 document.getElementById('ipat').textContent=s.ipAt?('· '+Math.round((Date.now()-s.ipAt)/1000)+'s lalu'):'';
 document.getElementById('hist').innerHTML=(s.history||[]).map(h=>'<span class="chip">'+h.ip+'</span>').join('')||'<small>-</small>';
 if(s.bootstrap&&s.bootstrap.progress>=0)document.getElementById('boot').textContent=s.bootstrap.progress+'%';
 if(s.bw&&s.bw.at)document.getElementById('bw').textContent='↓'+s.bw.down+' ↑'+s.bw.up;
 if(s.circ)document.getElementById('circ').textContent='built:'+s.circ.built+' fail:'+s.circ.failed;
 S._t=Date.now();}
function feed(ev){evN++;document.getElementById('evcount').textContent='· '+evN+' event';
 const f=document.getElementById('feed');if(evN===1)f.innerHTML='';
 const cls=ev.type.includes('fail')||ev.type.includes('error')?'t-err':ev.type.includes('ip_changed')||ev.type.includes('rotate')?'t-ip':ev.type.includes('bw')?'t-bw':'t-ev';
 const d=document.createElement('div');d.className=cls;
 const time=new Date(ev.ts).toLocaleTimeString();
 d.textContent=time+' '+ev.type+' '+JSON.stringify(ev.data||{}).slice(0,160);
 f.prepend(d);while(f.children.length>40)f.lastChild.remove();}
function applyEv(ev){
 feed(ev);
 if(ev.type==='net.ip_changed'&&ev.data.ip){const ipEl=document.getElementById('ip');ipEl.textContent=ev.data.ip;ipEl.classList.remove('flash');void ipEl.offsetWidth;ipEl.classList.add('flash');toast('IP baru: '+ev.data.ip);
  S.history=[{ip:ev.data.ip},...(S.history||[])].slice(0,10);
  document.getElementById('hist').innerHTML=S.history.map(h=>'<span class="chip">'+h.ip+'</span>').join('');}
 else if(ev.type==='rotate.ok'){S.lastRotate=ev.ts;toast('Rotate OK');}
 else if(ev.type==='rotate.pre'){document.getElementById('msg').textContent='rotating… '+JSON.stringify(ev.data);}
 else if(ev.type==='tor.bootstrap'){document.getElementById('boot').textContent=ev.data.progress+'%';}
 else if(ev.type==='net.bw'){document.getElementById('bw').textContent='↓'+ev.data.down+' ↑'+ev.data.up;}
 else if(ev.type==='tor.circ'){document.getElementById('circ').textContent=(ev.data.raw||'').slice(0,80);}
 else if(ev.type==='proc.status'){badge('tor',ev.data.tor);badge('priv',ev.data.priv);badge('auto',ev.data.auto);if('evRun' in ev.data)badge('evrun',ev.data.evRun);}
 else if((ev.type||'').includes('fail')){toast(ev.type+' '+JSON.stringify(ev.data||{}),true);document.getElementById('msg').textContent=ev.type+' '+JSON.stringify(ev.data||{});}
}
// init Sekali (bukan polling): snapshot + replay history, lalu murni SSE
(async()=>{try{const s=await api('/api/status');renderSnap(s);
 const h=await api('/api/history?tail=30');(h.events||[]).forEach(feed);}catch{}})();
const es=new EventSource('/api/events');
es.onopen=()=>{document.getElementById('livedot').className='dot live';document.getElementById('livetxt').textContent='LIVE';};
es.onerror=()=>{document.getElementById('livedot').className='dot';document.getElementById('livetxt').textContent='reconnecting…';};
es.onmessage=e=>{try{const m=JSON.parse(e.data);
 if(m.kind==='snapshot')renderSnap(m.snapshot);
 else if(m.kind==='event')applyEv(m.event);}catch{}};
setInterval(()=>{const n=document.getElementById('next');
 if(S.auto&&S.nextIn>=0&&S.lastRotate){const v=Math.max(0,S.nextIn-Math.round((Date.now()-S._t||0)/1000));n.textContent=v+'s';}
 else if(S.auto)n.textContent=S.interval+'s';else n.textContent='-';
 document.getElementById('clock').textContent=new Date().toLocaleTimeString();},1000);
async function pollLogs(){if(!document.getElementById('live').checked)return;try{const l=await api('/api/logs?which='+logWhich);if(l.logs!==lastLogs){lastLogs=l.logs;const el=document.getElementById('logs');el.textContent=l.logs;document.getElementById('logmeta').textContent='· '+logWhich+' · '+new Date().toLocaleTimeString();if(document.getElementById('follow').checked)el.scrollTop=el.scrollHeight;}}catch{}}
function setLog(w){logWhich=w;lastLogs='';pollLogs();}
async function doRotate(){const b=document.getElementById('btnRotate');b.disabled=true;b.textContent='⏳ rotating…';try{const r=await api('/api/rotate','POST');document.getElementById('msg').textContent=(r.note||JSON.stringify(r)).slice(0,500);if(!r.ok)toast('Rotate gagal',true);}finally{b.disabled=false;b.textContent='🔁 Rotate sekarang';}}
async function loadConfig(){const c=await api('/api/config');document.getElementById('interval').value=c.interval;document.getElementById('hook').value=c.hookCode;}
async function saveConfig(){const r=await api('/api/config','POST',{interval:Number(document.getElementById('interval').value||60),hookCode:document.getElementById('hook').value});document.getElementById('msg').textContent=JSON.stringify(r);toast(r.ok?'Hook tersimpan':'Gagal simpan',!r.ok);}
async function autoStart(){const r=await api('/api/autorotate/start','POST',{interval:Number(document.getElementById('interval').value||60)});document.getElementById('msg').textContent=JSON.stringify(r);}
async function autoStop(){const r=await api('/api/autorotate/stop','POST');document.getElementById('msg').textContent=JSON.stringify(r);}
loadConfig();pollLogs();setInterval(pollLogs,3000);
</script></body></html>`;
