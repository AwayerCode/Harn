// Run: node --test apps/swe-cli/internal/systemdesign/html_interaction_test.cjs
const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const path=require('node:path');
const source=fs.readFileSync(path.join(__dirname,'architecture.html.tmpl'),'utf8');
const script=source.match(/<script type="application\/javascript">([\s\S]*?)<\/script>/)[1];
function element(dataset,classes=[]) {
 const names=new Set(classes), events={};
 return {dataset,classList:{contains:n=>names.has(n),toggle(n,on){if(on)names.add(n);else names.delete(n);}},
 addEventListener(n,fn){events[n]=fn;},fire(n){events[n]();}};
}
function fixture(){
 const nodes=['up','selected','down','indirect','isolated'].map(module=>element({module}));
 const edges=[['up','selected','module'],['selected','down','module'],['down','indirect','module'],['selected','down','layer']].map(([from,to,kind])=>element({from,to},['edge-'+kind]));
 const svg=element({}),status={textContent:'idle'};
 svg.querySelectorAll=s=>s==='.node-link'?nodes:edges;
 vm.runInNewContext(script,{document:{querySelector:()=>svg,getElementById:()=>status}});
 return {nodes,edges,svg,status};
}
test('hover highlights direct relationships only, then restores',()=>{
 const {nodes,edges,svg,status}=fixture();
 nodes[1].fire('pointerenter');
 assert.equal(svg.classList.contains('tracing'),true);
 assert.deepEqual(nodes.map(n=>n.classList.contains('related')),[true,true,true,false,false]);
 assert.deepEqual(edges.map(n=>n.classList.contains('related')),[true,true,false,false]);
 assert.equal(nodes[1].classList.contains('selected'),true);
 assert.match(status.textContent,/1 个上游.*1 个下游/);
 nodes[1].fire('pointerleave');
 assert.equal(svg.classList.contains('tracing'),false);
 assert.equal(status.textContent,'idle');
 assert.ok([...nodes,...edges].every(n=>!n.classList.contains('related')));
});
test('hover overrides focus and falls back to focused module',()=>{
 const {nodes,svg}=fixture();
 nodes[1].fire('focus');nodes[2].fire('pointerenter');
 assert.equal(nodes[2].classList.contains('selected'),true);
 nodes[2].fire('pointerleave');
 assert.equal(nodes[1].classList.contains('selected'),true);
 nodes[1].fire('blur');assert.equal(svg.classList.contains('tracing'),false);
});
test('isolated module is visible without unrelated edges',()=>{
 const {nodes,edges,status}=fixture();nodes[4].fire('pointerenter');
 assert.deepEqual(nodes.map(n=>n.classList.contains('related')),[false,false,false,false,true]);
 assert.ok(edges.every(n=>!n.classList.contains('related')));
 assert.match(status.textContent,/0 个上游.*0 个下游/);
});
