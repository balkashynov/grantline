const $ = id => document.getElementById(id);
const notes = {
  'lost-access':'The replacement is missing one project role and half the deployment network.',
  'equivalent':'A textual diff sees three changed lines. The network sets are identical.',
  'empty-list':'When the organization does not require an access list, an empty list means unrestricted addresses.',
  'unknown-policy':'Without the organization setting, an empty list can mean different things. Grantline refuses an equality verdict.',
  'role-change':'ORG_MEMBER became ORG_OWNER. This is a changed assignment; the tool does not expand the roles into effective permissions.',
  'ipv6-gap':'Three of four IPv6 addresses remain. Exact arithmetic identifies the fourth. This is a set-engine example, not a claim of Atlas IPv6 acceptance.'
};
let examples, worker, engineReady=false, nextID=0, pendingID=null, lastReport=null, timer;
const node=(tag,text,cls)=>{const e=document.createElement(tag);if(text!==undefined)e.textContent=text;if(cls)e.className=cls;return e;};
function download(value,name){const url=URL.createObjectURL(new Blob([JSON.stringify(value,null,2)+'\n'],{type:'application/json'}));const a=node('a');a.href=url;a.download=name;a.click();setTimeout(()=>URL.revokeObjectURL(url),1000);}
function error(message){$('result').replaceChildren(node('div',message,'error'));lastReport=null;$('download-report').hidden=true;}
function dirty(){pendingID=null;clearTimeout(timer);lastReport=null;$('download-report').hidden=true;$('result').replaceChildren(node('div','Input changed. Compare again to refresh the result.','empty-state'));$('compare').disabled=!engineReady;}
function render(r){
 lastReport=r;const result=$('result');result.replaceChildren();const changed=r.status==='differences_found',incomplete=r.status==='incomplete';
 const head=node('div',undefined,'result-head');head.append(node('h3',incomplete?'The policy is missing.':changed?'Here’s what changed.':'Same configuration in scope.'));head.append(node('span',incomplete?'Incomplete':changed?'Review changes':'Match',`status-tag ${incomplete?'incomplete':changed?'changed':''}`));result.append(head);
 result.append(node('p',`${r.before} → ${r.after}`,'result-summary'));
 const metrics=node('div',undefined,'metrics');const pairs=[[r.removedGrants.length+r.addedGrants.length,'role assignments changed'],[r.network?r.network.removed.length+r.network.added.length:'—','network ranges changed']];for(const [n,label] of pairs){const m=node('div',undefined,'metric');m.append(node('strong',String(n)),node('span',label));metrics.append(m);}result.append(metrics);
 for(const issue of r.issues){const box=node('div',undefined,'finding warning');box.append(node('h4','Needs information'),node('p',issue));result.append(box);}
 for(const [grants,kind] of [[r.removedGrants,'Removed'],[r.addedGrants,'Added']]){
  for(const g of grants.slice(0,12)){const box=node('div',undefined,`finding ${kind==='Added'?'added':''}`);box.append(node('h4',`${kind} role assignment`),node('code',g.role),node('div',g.scope,'scope'));result.append(box);}if(grants.length>12)result.append(node('p',`${grants.length-12} more ${kind.toLowerCase()} assignments are in the full report.`,'small'));
 }
 if(r.network)for(const [ranges,kind]of [[r.network.removed,'Removed'],[r.network.added,'Added']]){
  for(const d of ranges.slice(0,12)){const box=node('div',undefined,`finding ${kind==='Added'?'added':''}`);box.append(node('h4',`${kind} from the allowed address set`),node('code',d.cidr),node('p',`${BigInt(d.addresses).toLocaleString()} ${d.family} address${d.addresses==='1'?'':'es'} · witness ${d.witness}`));result.append(box);}if(ranges.length>12)result.append(node('p',`${ranges.length-12} more ${kind.toLowerCase()} ranges are in the full report.`,'small'));
 }
 if(!changed&&!incomplete){const box=node('div',undefined,'match');box.append(node('strong','No assignment or address-set differences.'),node('p','CIDR splits, overlaps and duplicates normalize to the same set. Continue with application-level verification.'));result.append(box);}
 result.append(node('p','Snapshot scope only. Exact role labels are compared; effective permissions, unmanaged access and authentication are not validated.','result-limit'));$('download-report').hidden=false;
}
function compare(){
 if(!engineReady)return;const input=$('input').value;if(new TextEncoder().encode(input).length>16*1024*1024){error('Input exceeds 16 MiB.');return;}
 pendingID=++nextID;lastReport=null;$('download-report').hidden=true;$('compare').disabled=true;$('result').replaceChildren(node('div','Comparing with the Go engine…','empty-state'));worker.postMessage({id:pendingID,input});clearTimeout(timer);timer=setTimeout(()=>{worker.terminate();engineReady=false;pendingID=null;$('engine-status').textContent='Engine timed out';error('The comparison took too long. Reload or use the local CLI.');},30000);
}
function select(name){$('input').value=JSON.stringify(examples[name],null,2);$('scenario-note').textContent=notes[name];document.querySelectorAll('[data-example]').forEach(b=>b.setAttribute('aria-pressed',String(b.dataset.example===name)));dirty();if(engineReady)compare();}
$('compare').addEventListener('click',compare);$('input').addEventListener('input',()=>{document.querySelectorAll('[data-example]').forEach(b=>b.setAttribute('aria-pressed','false'));$('scenario-note').textContent='Custom comparison. Changes run locally when you select Compare configurations.';dirty();});
$('download-report').addEventListener('click',()=>{if(lastReport)download(lastReport,'grantline-report.json');});
$('download-input').addEventListener('click',()=>{try{download(JSON.parse($('input').value),'grantline-comparison.json');}catch{error('Fix the input JSON before saving.');}});
$('load').addEventListener('click',()=>$('file').click());$('file').addEventListener('change',async()=>{const f=$('file').files[0];if(!f)return;if(f.size>16*1024*1024){error('Input exceeds 16 MiB.');return;}try{$('input').value=await f.text();$('input').dispatchEvent(new Event('input'));compare();}catch{error('Could not read the file.');}finally{$('file').value='';}});
(async()=>{
 try{const response=await fetch('examples.json');if(!response.ok)throw new Error();examples=await response.json();select('lost-access');document.querySelectorAll('[data-example]').forEach(b=>b.addEventListener('click',()=>select(b.dataset.example)));
 worker=new Worker('worker.js');worker.onmessage=({data})=>{if(data.fatal){clearTimeout(timer);engineReady=false;$('compare').disabled=true;$('engine-status').textContent='Engine unavailable';error(data.fatal);return;}if(data.ready){engineReady=true;$('engine-status').textContent='Go / WASM · local execution';$('compare').disabled=false;compare();return;}if(data.id!==pendingID)return;clearTimeout(timer);pendingID=null;$('compare').disabled=false;if(data.error)error(data.error);else render(data.report);};worker.onerror=()=>{clearTimeout(timer);engineReady=false;$('compare').disabled=true;$('engine-status').textContent='Engine unavailable';error('Could not run the browser engine. Use the Go CLI or reload.');};
 }catch{$('engine-status').textContent='Examples unavailable';error('Could not load the examples. Reload or use the Go CLI.');}
})();
