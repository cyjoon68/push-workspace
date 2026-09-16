#!/usr/bin/env python3
"""hub.html에 localStorage 기반 window.claude 심 + '의견 복사' 버튼을 주입한다.
build_hub.py 재실행 후마다 다시 실행한다."""
import re, sys

PATH = "design/probes/hub.html"
MARKER = "push-hubdb"

SHIM = """<script>
(function(){
  var KEY='push-hubdb';
  function load(){try{return JSON.parse(localStorage.getItem(KEY)||'{}')}catch(e){return{}}}
  function saveAll(o){try{localStorage.setItem(KEY,JSON.stringify(o))}catch(e){}}
  var db={
    doc:function(p){var id=p.split('/').pop();return{id:id,path:p,
      get:function(){var a=load(),d=a[p];return Promise.resolve({id:id,exists:!!d,data:function(){return d}})},
      set:function(d){var a=load();a[p]=d;saveAll(a);return Promise.resolve()},
      update:function(d){var a=load();a[p]=Object.assign({},a[p],d);saveAll(a);return Promise.resolve()},
      delete:function(){var a=load();delete a[p];saveAll(a);return Promise.resolve()}}},
    collection:function(c){return{path:c,get:function(){var a=load();var docs=Object.keys(a).filter(function(k){return k.indexOf(c+'/')===0}).map(function(k){return{id:k.split('/').pop(),data:function(){return a[k]}}});return Promise.resolve({docs:docs,size:docs.length,empty:!docs.length})}}}
  };
  window.claude={use:function(n){return Promise.resolve(n==='db'?db:null)}};
  window.addEventListener('DOMContentLoaded',function(){
    var b=document.createElement('button');
    b.textContent='의견 복사';
    b.style.cssText='margin-left:auto;font:inherit;font-size:13px;font-weight:600;padding:6px 12px;border:1px solid #E5E7EB;border-radius:8px;background:#fff;cursor:pointer;color:#1F2937';
    b.onclick=function(){
      var a=load(),out={};Object.keys(a).forEach(function(k){if(k.indexOf('feedback/')===0)out[k]=a[k]});
      var s=JSON.stringify(out,null,2);
      if(navigator.clipboard&&navigator.clipboard.writeText){navigator.clipboard.writeText(s).then(function(){b.textContent='복사됨 — 터미널에 붙여넣기'},function(){prompt('의견 JSON:',s)})}else{prompt('의견 JSON:',s)}
    };
    var top=document.querySelector('.top');if(top)top.appendChild(b);
  });
})();
</script>
"""

with open(PATH, encoding="utf-8") as f:
    doc = f.read()

changed = False
if "meta charset" not in doc[:500]:
    doc = '<meta charset="utf-8">' + doc
    changed = True

if MARKER not in doc:
    m = re.search(r"<title>", doc)
    if not m:
        print("[FAIL] <title> 없음", file=sys.stderr)
        sys.exit(1)
    doc = doc[: m.start()] + SHIM + doc[m.start() :]
    changed = True

if not changed:
    print("[OK] 이미 주입됨")
    sys.exit(0)
with open(PATH, "w", encoding="utf-8") as f:
    f.write(doc)
print("[OK] hub.html에 db 심 + 의견 복사 버튼 주입")
