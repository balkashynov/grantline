const fs=require('node:fs');
const assert=require('node:assert/strict');
const {execFileSync}=require('node:child_process');
if(!globalThis.crypto)globalThis.crypto=require('node:crypto').webcrypto;
require('../web/wasm_exec.js');
(async()=>{
 const go=new Go();const {instance}=await WebAssembly.instantiate(fs.readFileSync('web/grantline.wasm'),go.importObject);go.run(instance);
 const names=fs.readdirSync('examples').filter(n=>n.endsWith('.json')&&!n.includes('terraform'));
 for(const name of names){const input=fs.readFileSync('examples/'+name,'utf8');const wasm=JSON.parse(globalThis.grantlineAnalyze(input));assert.ok(wasm.report);let cli;try{cli=execFileSync('./bin/grantline',['compare','-input','examples/'+name],{encoding:'utf8'});}catch(e){assert.ok([2,3].includes(e.status));cli=e.stdout;}assert.deepEqual(wasm.report,JSON.parse(cli));console.log(name+': CLI = Go/WASM');}
 assert.ok(JSON.parse(globalThis.grantlineAnalyze('{"private_key":"sentinel"}')).error);
 console.log('All browser-engine parity checks passed. This does not exercise the DOM.');process.exit(0);
})().catch(e=>{console.error(e);process.exit(1);});
