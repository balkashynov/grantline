/* The same Go engine as the CLI runs in this worker. No input is sent over the network. */
importScripts('wasm_exec.js');
let ready = false;
self.onmessage = ({data}) => {
  if (!ready) { self.postMessage({id:data.id,error:'The engine is still loading.'}); return; }
  try { self.postMessage({id:data.id,...JSON.parse(self.grantlineAnalyze(data.input))}); }
  catch { self.postMessage({id:data.id,error:'The engine could not process this input.'}); }
};
(async () => {
  const go = new Go();
  const response = await fetch('grantline.wasm');
  if (!response.ok) throw new Error('Engine download failed');
  const {instance} = await WebAssembly.instantiate(await response.arrayBuffer(), go.importObject);
  go.run(instance).catch(() => self.postMessage({fatal:'The engine stopped. Reload to retry.'}));
  ready = typeof self.grantlineAnalyze === 'function';
  if (!ready) throw new Error('Engine did not initialize');
  self.postMessage({ready:true});
})().catch(() => self.postMessage({fatal:'Could not load the Go engine. Reload or use the CLI.'}));
