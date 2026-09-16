#!/usr/bin/env python3
"""
build_hub.py — design/probes/*.html 시안 페이지들을 탭 하나씩으로 묶어
design/probes/hub.html(허브) 한 장을 만든다. 링크 하나만 사용자에게 준다.

- 탭 순서·제목은 design/probes/hub.json 매니페스트가 정한다.
- 각 시안 페이지는 <iframe srcdoc>로 그대로 들어간다(수정 없음). 단, 자식 페이지가
  부모의 window.claude(db 저장)를 쓰도록 짧은 스크립트를 <head> 맨 앞에 주입한다.
- 허브 헤더에 탭별 "저장 n건 / 아직 없음" 배지와 "안 본 탭" 표시를 둔다.

사용법:
    python3 scripts/build_hub.py [--probes design/probes] [--out design/probes/hub.html]

hub.json 형식:
{
  "title": "<프로젝트명> — 디자인 인터뷰",
  "tabs": [
    {"file": "structure.html", "title": "구조",       "prefix": "structure-", "stage": "1단계 구조"},
    {"file": "flow-tour.html", "title": "따라가 보기", "prefix": "scene-", "stage": "2단계 플로우"},
    {"file": "reference.html", "title": "레퍼런스",   "prefix": "app-",   "stage": "3단계 레퍼런스"},
    {"file": "taste-1-2.html", "title": "취향 1·2",   "prefix": "axis-",  "stage": "3단계 취향"}
  ]
}
prefix(또는 prefixes 배열) = 그 탭의 의견 문서 id 접두어 (feedback/<prefix>…). 배지 계산에 쓴다. 예: 취향 "색상" 탭은 "prefixes": ["axis-1","axis-4"].
"""
from __future__ import annotations

import argparse
import html
import json
import os
import re
import sys

BRIDGE = (
    "<script>(function(){"
    "if(window.parent===window)return;"
    "var seq=0,pend={};"
    "window.addEventListener('message',function(e){var m=e.data;if(!m||m.__hubdb!==1||!pend[m.id])return;"
    "var p=pend[m.id];delete pend[m.id];m.error?p.reject(m.error):p.resolve(m.result);});"
    "function rpc(op,path,data){return new Promise(function(resolve,reject){var id=++seq;pend[id]={resolve:resolve,reject:reject};"
    "window.parent.postMessage({__hubdb:1,id:id,op:op,path:path,data:data},'*');"
    "setTimeout(function(){if(pend[id]){delete pend[id];reject({code:'unavailable',message:'timeout'});}},15000);});}"
    "function snap(r){return {id:r.id,exists:r.exists,data:function(){return r.data;},metadata:{fromCache:false,hasPendingWrites:false}};}"
    "var db={doc:function(p){return {id:p.split('/').pop(),path:p,"
    "get:function(){return rpc('get',p).then(snap);},"
    "set:function(d){return rpc('set',p,d);},"
    "update:function(d){return rpc('update',p,d);},"
    "delete:function(){return rpc('delete',p);}};},"
    "collection:function(p){return {path:p,"
    "get:function(){return rpc('list',p).then(function(rs){return {docs:rs.map(snap),size:rs.length,empty:!rs.length};});},"
    "doc:function(id){return db.doc(p+'/'+id);}};}};"
    "var avail=null;"
    "window.claude={use:function(n){return (async function(){if(n!=='db')return null;"
    "if(avail===null){try{avail=await rpc('avail');}catch(e){avail=false;}}return avail?db:null;})();}};"
    "})();</script>"
)


SHARE_BRIDGE = (
    "<script>window.claude={use:function(){return Promise.resolve(null);}};</script>"
)


def inject_bridge_share(doc: str) -> str:
    """공유본: db 없음. 자식 페이지는 localStorage 폴백으로 동작(보는 사람 브라우저에만 저장)."""
    m = re.search(r"<head[^>]*>", doc, re.I)
    if m:
        return doc[: m.end()] + SHARE_BRIDGE + doc[m.end() :]
    return SHARE_BRIDGE + doc


def inject_bridge(doc: str) -> str:
    """자식 문서 <head> 시작 직후(없으면 문서 맨 앞)에 부모 claude 브리지를 넣는다."""
    m = re.search(r"<head[^>]*>", doc, re.I)
    if m:
        return doc[: m.end()] + BRIDGE + doc[m.end() :]
    return BRIDGE + doc


def build(probes_dir: str, out_path: str, share: bool = False) -> int:
    manifest_path = os.path.join(probes_dir, "hub.json")
    if not os.path.isfile(manifest_path):
        print(f"[FAIL] {manifest_path} 없음", file=sys.stderr)
        return 2
    with open(manifest_path, encoding="utf-8") as f:
        manifest = json.load(f)

    tabs = manifest.get("tabs", [])
    if not tabs:
        print("[FAIL] hub.json tabs가 비어있음", file=sys.stderr)
        return 1

    frames = []
    nav = []
    meta = []
    total = 0
    for i, t in enumerate(tabs):
        path = os.path.join(probes_dir, t["file"])
        if not os.path.isfile(path):
            print(f"[WARN] 탭 파일 없음, 건너뜀: {path}", file=sys.stderr)
            continue
        with open(path, encoding="utf-8") as f:
            doc = f.read()
        doc = inject_bridge(doc) if not share else inject_bridge_share(doc)
        total += len(doc)
        key = re.sub(r"[^a-z0-9]+", "-", t["file"].lower().rsplit(".", 1)[0])
        nav.append(
            f'<button class="tab" data-key="{key}" role="tab">'
            f'<span class="t">{html.escape(t["title"])}</span>'
            f'<span class="badge" id="badge-{key}">…</span></button>'
        )
        frames.append(
            f'<iframe class="pane" id="pane-{key}" data-key="{key}" title="{html.escape(t["title"])}" '
            f'srcdoc="{html.escape(doc, quote=True)}" hidden></iframe>'
        )
        prefixes = t.get("prefixes") or ([t["prefix"]] if t.get("prefix") else [])
        meta.append({"key": key, "title": t["title"], "prefixes": prefixes, "stage": t.get("stage", "")})

    if total > 15_000_000:
        print(f"[FAIL] 합친 크기 {total/1e6:.1f}MB — 16MB 한도 초과 위험", file=sys.stderr)
        return 1

    title = manifest.get("title", "디자인 인터뷰")
    page = f"""<title>{html.escape(title)}</title>
<style>
:root{{--bg:#F7F8FA;--card:#fff;--line:#E5E7EB;--ink:#1F2937;--sub:#6B7280;--acc:#2F6BFF}}
html,body{{height:100%}}
body{{margin:0;background:var(--bg);color:var(--ink);font-family:"Pretendard",-apple-system,"Apple SD Gothic Neo",system-ui,sans-serif;display:flex;flex-direction:column}}
header{{flex:none;background:var(--card);border-bottom:1px solid var(--line);padding:10px 20px 0}}
.top{{display:flex;align-items:baseline;gap:14px;flex-wrap:wrap}}
h1{{font-size:17px;margin:0 0 6px}}
.todo{{font-size:13px;color:var(--sub)}}
.todo b{{color:var(--acc)}}
.tabs{{display:flex;gap:6px;flex-wrap:wrap}}
.tab{{font:inherit;font-size:14px;font-weight:600;border:1px solid var(--line);border-bottom:0;background:#F3F4F6;color:var(--sub);padding:9px 14px;border-radius:10px 10px 0 0;cursor:pointer;display:flex;align-items:center;gap:8px}}
.tab.on{{background:var(--card);color:var(--ink);box-shadow:0 1px 0 var(--card)}}
.tab.unseen .t::after{{content:"NEW";font-size:10px;font-weight:700;color:#fff;background:var(--acc);border-radius:6px;padding:1px 5px;margin-left:6px}}
.badge{{font-size:11px;font-weight:600;color:var(--sub);background:#E5E7EB;border-radius:999px;padding:2px 8px}}
.badge.ok{{color:#065F46;background:#D1FAE5}}
main{{flex:1;min-height:0;position:relative}}
.pane{{position:absolute;inset:0;width:100%;height:100%;border:0;background:var(--bg)}}
</style>
<header>
  <div class="top"><h1>{html.escape(title)}</h1><div class="todo" id="todo">{'공유용 보기 전용 — 여기서 저장한 것은 본인 브라우저에만 남습니다' if share else '불러오는 중…'}</div></div>
  <nav class="tabs" role="tablist">{''.join(nav)}</nav>
</header>
<main>{''.join(frames)}</main>
<script>
const META={json.dumps(meta, ensure_ascii=False)};
const tabs=[...document.querySelectorAll('.tab')],panes=[...document.querySelectorAll('.pane')];
let seen={{}};try{{seen=JSON.parse(localStorage.getItem('hub-seen')||'{{}}')}}catch(e){{}}
function show(key){{
  tabs.forEach(t=>t.classList.toggle('on',t.dataset.key===key));
  panes.forEach(p=>p.hidden=p.dataset.key!==key);
  seen[key]=1;try{{localStorage.setItem('hub-seen',JSON.stringify(seen))}}catch(e){{}}
  try{{location.hash=key}}catch(e){{}}
  paint();
}}
tabs.forEach(t=>t.addEventListener('click',()=>show(t.dataset.key)));
let counts={{}};
function paint(){{
  let unseen=[],empty=[];
  META.forEach(m=>{{
    const t=tabs.find(x=>x.dataset.key===m.key);
    t.classList.toggle('unseen',!seen[m.key]);
    const n=m.prefixes.reduce((a,p)=>a+(counts[p]||0),0);const b=document.getElementById('badge-'+m.key);
    b.textContent=n?`저장 ${{n}}건`:'아직 없음';b.classList.toggle('ok',n>0);
    if(!seen[m.key])unseen.push(m.title);else if(!n)empty.push(m.title);
  }});
  const el=document.getElementById('todo');
  if(!unseen.length&&!empty.length)el.innerHTML='모든 탭을 보고 저장했어요 ✓';
  else el.innerHTML=(unseen.length?`아직 안 본 탭: <b>${{unseen.join(', ')}}</b>`:'')+(unseen.length&&empty.length?' · ':'')+(empty.length?`저장 없는 탭: <b>${{empty.join(', ')}}</b>`:'');
}}
let realDb=null;
const dbReady=(async()=>{{try{{realDb=window.claude&&typeof window.claude.use==='function'?await window.claude.use('db'):null;}}catch(e){{realDb=null;}}}})();
window.addEventListener('message',async e=>{{
  const m=e.data;if(!m||m.__hubdb!==1||!e.source)return;
  if(!panes.some(p=>p.contentWindow===e.source))return;
  const reply=(result,error)=>{{try{{e.source.postMessage({{__hubdb:1,id:m.id,result,error}},'*');}}catch(x){{}}}};
  try{{
    await dbReady;
    if(m.op==='avail')return reply(!!realDb);
    if(!realDb)throw {{code:'not_granted',message:'db unavailable'}};
    if(m.op==='get'){{const s=await realDb.doc(m.path).get();return reply({{id:s.id,exists:s.exists,data:s.data()}});}}
    if(m.op==='set'){{await realDb.doc(m.path).set(m.data);loadCounts();return reply(true);}}
    if(m.op==='update'){{await realDb.doc(m.path).update(m.data);loadCounts();return reply(true);}}
    if(m.op==='delete'){{await realDb.doc(m.path).delete();loadCounts();return reply(true);}}
    if(m.op==='list'){{const q=await realDb.collection(m.path).get();return reply(q.docs.map(d=>({{id:d.id,exists:d.exists,data:d.data()}})));}}
    throw {{code:'invalid_argument',message:'unknown op'}};
  }}catch(err){{reply(null,{{code:(err&&err.code)||'unavailable',message:(err&&err.message)||String(err)}});}}
}});
async function loadCounts(){{
  try{{
    await dbReady;const db=realDb;
    if(!db)return;
    const snap=await db.collection('feedback').get();
    counts={{}};
    snap.docs.forEach(d=>{{const id=d.id;const dd=d.data()||{{}};const has=(dd.reaction||dd.best||dd.worst||(dd.marks&&dd.marks.length)||(dd.want&&dd.want.length)||(dd.dislike&&dd.dislike.length)||dd.answers||dd.choices||(dd.text&&dd.text.trim()));if(!has)return;META.forEach(x=>x.prefixes.forEach(p=>{{if(id.startsWith(p)&&(id===p||!/\d/.test(p.slice(-1))||id.length===p.length))counts[p]=(counts[p]||0)+1;}}));}});
    paint();
  }}catch(e){{}}
}}
const first=(location.hash||'').slice(1);
show(META.some(m=>m.key===first)?first:META[0].key);
{'' if share else 'loadCounts();setInterval(loadCounts,20000);'}
</script>
"""
    with open(out_path, "w", encoding="utf-8") as f:
        f.write(page)
    print(f"[OK] {out_path} ({total/1e6:.1f}MB, 탭 {len(meta)}개: {', '.join(m['title'] for m in meta)})")
    return 0


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--probes", default="design/probes")
    ap.add_argument("--out", default=None)
    ap.add_argument("--share", action="store_true", help="db 없는 공유본(hub-share.html)을 만든다")
    a = ap.parse_args()
    out = a.out or os.path.join(a.probes, "hub-share.html" if a.share else "hub.html")
    return build(a.probes, out, share=a.share)


if __name__ == "__main__":
    sys.exit(main())
