#!/usr/bin/env python3
"""od_audit.py — OpenDesign A단계(구조적 사실) 검사기.

`design/od/` 디렉토리(또는 OpenDesign 프로젝트 resolvedDir)의
HTML/CSS artifact를 읽어 `design/design-rules.md` 기준값으로 검사하고
결함 목록을 낸다. 스냅샷 단계 없이 파일이 곧 진실이다.

    python3 scripts/od_audit.py \
        --root     design/od \
        --rules    design/design-rules.md \
        [--brief   design/brief.md] \
        [--icons   design/icons.md] \
        [--screens design/screens.md] \
        [--json] [--fix-list design/fix-list.md]

파일 규약 (od-builder가 지키는 형식)
    <root>/
      tokens.css            :root CSS 변수 전부 (색·간격·radius·크기·z·그림자·타이포 역할 .t-*)
      components.css        컴포넌트 클래스 전부 (.button .card .app-bar .tab-bar …)
      icons.svg             lucide <symbol id="i-이름"> 스프라이트
      index.html            화면 갤러리 (artifact entry, 선택)
      screens/<slug>-<state>.html   화면. default/empty/loading/error/long-title/many-items/text-120(+keyboard)
      screens/<slug>-default.html 은 ../tokens.css ../components.css 를 <link>로만 쓰고 <style>·style= 금지

화면 HTML에서 아이콘은 `<svg class="icon" data-icon="home" width="20" height="20"><use href="../icons.svg#i-home"/></svg>`.

종료 코드
    0  결함 없음
    1  결함 있음
    2  실행 오류 (디렉토리 없음, 형식 불일치 등)

출력 한 줄 형식
    [FAIL] <파일> <선택자/요소> <규칙 키>: 현재 → 기대

Python 3.11 표준 라이브러리만 사용한다.
"""

import argparse
import json
import re
import sys
from html.parser import HTMLParser
from pathlib import Path

# ── 규칙 파싱 실패 시 쓰는 기본값 ─────────────────────────────────────────
# 출처: references/design-rules.md
DEFAULT_RULES = {
    "space.scale": "4 / 8 / 12 / 16 / 24 / 32 / 48 (4 배수만 허용)",
    "icon.sizes": "16 (인라인·캡션) / 20 (버튼·목록) / 24 (앱바·탭바). 세 값 외 금지",
    "tap.min": "44×44. 인접 탭 영역 간격 최소 8",
    "device.frame": "390×844 기준. 검증 폭 360 / 390 / 430",
    "safe-area": "상단 상태바 44(노치 기기 47) · 하단 홈 인디케이터 34",
    "z.scale": ("base 0 · sticky 100 · app-bar 200 · tab-bar 200 · overlay 300 · "
                "sheet 400 · dialog 500 · snackbar 600"),
    "button.sizes": "sm 36h / px12 / text14 · md 44h / px16 / text15 · lg 52h / px20 / text16",
    "button.states": "default · pressed · selected · disabled · loading",
    "icon.set": "lucide 단일. 다른 세트 혼용 금지",
}

REQUIRED_STATES = ["empty", "loading", "error", "long-title", "many-items", "text-120"]
STATE_SELECTORS = {                       # button.states → components.css에 있어야 할 선택자
    "pressed": [".button:active", ".is-pressed", "[aria-pressed"],
    "selected": [".is-selected", "[aria-selected", ".selected"],
    "disabled": [":disabled", ".is-disabled", "[disabled]"],
    "loading": [".is-loading", "[aria-busy"],
}
VARIANT_NAMES = ["primary", "secondary", "ghost", "danger"]
SIZE_NAMES = ["sm", "md", "lg"]

ICON_CLASS = "icon"
ICON_PREFIX = "i-"


# ── 결함 ──────────────────────────────────────────────────────────────────
class Finding:
    """검사 실패 1건."""

    __slots__ = ("page", "frame", "node_name", "node_id", "key", "actual", "expected")

    def __init__(self, page, frame, node_name, node_id, key, actual, expected):
        self.page = page or "-"
        self.frame = frame or "-"
        self.node_name = node_name or "-"
        self.node_id = node_id or "-"
        self.key = key
        self.actual = str(actual)
        self.expected = str(expected)

    def line(self):
        return "[FAIL] {}/{} {}({}) {}: {} → {}".format(
            self.page, self.frame, self.node_name, self.node_id,
            self.key, self.actual, self.expected)

    def as_dict(self):
        return {
            "file": self.page, "screen": self.frame,
            "where": self.node_name,
            "rule": self.key, "actual": self.actual, "expected": self.expected,
        }

    def sort_key(self):
        return (self.key, self.page, self.frame, self.node_name)


# ── design-rules.md 파싱 ──────────────────────────────────────────────────
_ROW_RE = re.compile(r"^\|(.+)\|\s*$")
_KEY_RE = re.compile(r"^[a-z][a-z0-9]*(?:[.\-][a-z0-9]+)*$")


def parse_rules_file(path):
    """마크다운 표에서 `키 → 값` 을 읽는다. (rules, warnings) 반환."""
    warnings = []
    rules = {}
    try:
        text = Path(path).read_text(encoding="utf-8")
    except OSError as exc:
        warnings.append("design-rules.md를 읽지 못했습니다 ({}). 기본값을 사용합니다.".format(exc))
        return {}, warnings

    for raw in text.splitlines():
        m = _ROW_RE.match(raw.strip())
        if not m:
            continue
        cells = [c.strip() for c in m.group(1).split("|")]
        if len(cells) < 2:
            continue
        key, value = cells[0], cells[1]
        if not _KEY_RE.match(key):
            continue
        if set(value) <= set("- :"):      # 표 구분선
            continue
        if not value:
            continue
        rules[key] = value
    if not rules:
        warnings.append("design-rules.md에서 표를 찾지 못했습니다. 전부 기본값을 사용합니다.")
    return rules, warnings


def _ints(text, lo=0, hi=10000):
    return [int(n) for n in re.findall(r"\d+", text or "") if lo <= int(n) <= hi]


class Thresholds:
    """design-rules.md 값에서 뽑아낸 검사 기준값."""

    def __init__(self, rules, warnings):
        self.warnings = warnings
        self.raw = {}

        def get(key):
            value = rules.get(key)
            if not value:
                self.warnings.append(
                    "규칙 '{}' 파싱 실패 — references/design-rules.md 기본값으로 대체합니다.".format(key))
                value = DEFAULT_RULES.get(key, "")
            self.raw[key] = value
            return value

        scale = _ints(get("space.scale"), 1, 512)
        self.space_step = min(scale) if scale else 4
        self.space_scale = sorted(set(scale)) or [4, 8, 12, 16, 24, 32, 48]

        sizes = _ints(get("icon.sizes"), 8, 96)
        self.icon_sizes = sorted(set(sizes)) or [16, 20, 24]

        tap = get("tap.min")
        tap_nums = _ints(tap, 1, 200)
        self.tap_min = tap_nums[0] if tap_nums else 44
        gap_m = re.search(r"간격[^\d]*(\d+)", tap)
        self.tap_gap = int(gap_m.group(1)) if gap_m else (tap_nums[-1] if len(tap_nums) > 1 else 8)

        dev = get("device.frame")
        pair = re.search(r"(\d+)\s*[×xX]\s*(\d+)", dev)
        self.frame_w, self.frame_h = (int(pair.group(1)), int(pair.group(2))) if pair else (390, 844)
        widths = set(_ints(dev, 200, 1200)) - {self.frame_h}
        widths.add(self.frame_w)
        self.alt_widths = sorted(widths)

        sa = get("safe-area")
        top = re.search(r"상단[^\d]*(\d+)", sa)
        bottom = re.search(r"하단[^\d]*(\d+)", sa)
        sa_nums = _ints(sa, 1, 200)
        self.safe_top = int(top.group(1)) if top else (sa_nums[0] if sa_nums else 44)
        self.safe_bottom = int(bottom.group(1)) if bottom else (sa_nums[-1] if sa_nums else 34)

        self.z_scale = {k: int(v) for k, v in re.findall(r"([a-z][a-z\-]*)\s+(\d+)", get("z.scale"))}

        bs = get("button.sizes")
        heights = [int(n) for n in re.findall(r"(\d+)\s*h", bs)]
        self.button_heights = sorted(set(heights)) or [36, 44, 52]

        states = re.findall(r"\b(default|pressed|selected|disabled|loading)\b", get("button.states"))
        seen, ordered = set(), []
        for s in states:
            if s not in seen:
                seen.add(s)
                ordered.append(s)
        self.button_states = ordered or ["default", "pressed", "selected", "disabled", "loading"]


# ── brief.md 화면 목록 파싱 ───────────────────────────────────────────────
def parse_brief_screens(path):
    """brief.md §1 화면 목록 표의 첫 열을 화면 이름으로 읽는다."""
    try:
        text = Path(path).read_text(encoding="utf-8")
    except OSError as exc:
        return None, ["brief.md를 읽지 못했습니다 ({}). 파일 목록에서 화면을 추론합니다.".format(exc)]

    lines = text.splitlines()
    start = None
    for i, line in enumerate(lines):
        if re.match(r"^#{1,6}\s*1\.?\s*화면 목록", line.strip()):
            start = i + 1
            break
    if start is None:
        return None, ["brief.md에서 '1. 화면 목록' 절을 찾지 못했습니다. 파일 목록에서 추론합니다."]

    screens, header_seen = [], False
    for line in lines[start:]:
        stripped = line.strip()
        if stripped.startswith("#"):
            break
        m = _ROW_RE.match(stripped)
        if not m:
            if screens:
                break
            continue
        cells = [c.strip() for c in m.group(1).split("|")]
        first = cells[0] if cells else ""
        if not header_seen:
            header_seen = True
            continue                                   # 표 머리행
        if not first or set(first) <= set("- :"):
            continue                                   # 구분선 / 빈 템플릿 행
        if first.startswith("<") or first == "화면":
            continue
        screens.append(first)
    if not screens:
        return None, ["brief.md 화면 목록 표가 비어 있습니다. 파일 목록에서 추론합니다."]
    return screens, []


# ── icons.md 허용 목록 파싱 ───────────────────────────────────────────────
def parse_icon_allowlist(path):
    """icons.md 표의 'lucide 이름' 열을 허용 목록으로 읽는다."""
    try:
        text = Path(path).read_text(encoding="utf-8")
    except OSError as exc:
        return None, ["icons.md를 읽지 못했습니다 ({}). icon.allowlist 검사를 건너뜁니다.".format(exc)]

    allow, col = set(), None
    for raw in text.splitlines():
        stripped = raw.strip()
        if stripped.startswith("#") and col is not None:
            break                                      # 허용 목록 표가 끝났다
        m = _ROW_RE.match(stripped)
        if not m:
            continue
        cells = [c.strip() for c in m.group(1).split("|")]
        if col is None:
            for i, c in enumerate(cells):
                if "lucide" in c.lower():
                    col = i
                    break
            continue
        if col >= len(cells):
            continue
        value = cells[col]
        if not value or set(value) <= set("- :"):
            continue
        for name in re.split(r"[,/]", value):
            name = name.strip().strip("`")
            if name and _KEY_RE.match(name.lower()):
                allow.add(name.lower())
    if col is None:
        return None, ["icons.md에서 'lucide 이름' 열을 찾지 못했습니다. icon.allowlist 검사를 건너뜁니다."]
    if not allow:
        return None, ["icons.md의 'lucide 이름' 열이 비어 있습니다. icon.allowlist 검사를 건너뜁니다."]
    return allow, []


# ── screens.md 구성표 파싱 ────────────────────────────────────────────────
def parse_screens_manifest(path):
    """design/screens.md 의 구성표를 읽는다. {화면명 또는 slug(소문자): [컴포넌트명, ...]}"""
    text = Path(path).read_text(encoding="utf-8")
    manifest, warnings, states_map = {}, [], {}
    header = None
    for line in text.splitlines():
        line = line.strip()
        if not line.startswith("|"):
            continue
        cells = [c.strip() for c in line.strip("|").split("|")]
        if header is None:
            if "구성" in "".join(cells):
                header = cells
            continue
        if set(line) <= set("|-: "):
            continue
        row = dict(zip(header, cells))
        comp_cell = next((v for k, v in row.items() if "구성" in k), "")
        screen = row.get("화면") or ""
        slug = row.get("slug") or ""
        if not comp_cell or not (screen or slug):
            continue
        comps = []
        clean = re.sub(r"\([^()]*\)", "", comp_cell)
        clean = re.sub(r"\([^()]*\)", "", clean)
        for part in re.split(r"[·,]", clean):
            part = re.sub(r"^[\s①-⑳⓪-⓿]+", "", part).strip()
            name = re.split(r"[\s(×xX*]", part)[0].strip()
            if name and name.lower() not in ("icon",):
                comps.append(name)
        if not comps:
            continue
        for key in (screen, slug):
            if key:
                manifest[key.lower()] = comps
        state_cell = next((v for k, v in row.items() if "상태" in k), "")
        states = [m.group(0) for m in
                  (re.match(r"[a-z][a-z\-0-9]*", p.strip()) for p in re.split(r"[·,]", state_cell))
                  if m]
        if slug and states:
            states_map[slug] = states
    if not manifest:
        warnings.append("screens.md에서 구성표를 읽지 못했습니다 ('구성' 열이 있는 표 필요). component.manifest 검사를 건너뜁니다.")
    return manifest, states_map, warnings


MANIFEST_IGNORE = {"DeviceFrame", "Icon", "Skeleton", "Sidebar", "Card", "CardGrid", "Canvas", "Center", "Button", "IconButton"}


def kebab(name):
    """'AppBar' → 'app-bar', 'BottomCTA' → 'bottom-cta'."""
    s = re.sub(r"([a-z0-9])([A-Z])", r"\1-\2", name)
    return re.sub(r"[\s_]+", "-", s).lower()


# ── 파일 트리 모델 ────────────────────────────────────────────────────────
# 상태 이름에 '-'가 들어가므로 greedy 정규식 대신 알려진 상태 suffix로 분리한다.
KNOWN_STATES = ["long-title", "many-items", "text-120",
                "default", "empty", "loading", "error", "keyboard"]


def split_screen_name(stem):
    """'home-long-title' → ('home', 'long-title'). 상태 없으면 default."""
    for st in KNOWN_STATES:
        if stem.endswith("-" + st):
            slug = stem[: -(len(st) + 1)]
            if slug:
                return slug, st
    if "-" in stem:
        slug, st = stem.rsplit("-", 1)
        if slug and re.fullmatch(r"[a-z0-9]+", st):
            return slug, st
    return stem, "default"


class OdTree:
    """design/od 디렉토리."""

    def __init__(self, root):
        self.root = Path(root)
        if not self.root.is_dir():
            raise ValueError("artifact 디렉토리가 없습니다: {}".format(root))
        self.tokens_css = self._read("tokens.css")
        self.components_css = self._read("components.css")
        self.icons_svg = self._read("icons.svg")
        self.screens_dir = self.root / "screens"
        self.screen_files = {}                       # {slug: {state: Path}}
        if self.screens_dir.is_dir():
            for p in sorted(self.screens_dir.glob("*.html")):
                slug, state = split_screen_name(p.stem)
                if not re.fullmatch(r"[a-z0-9][a-z0-9\-]*", slug):
                    continue
                self.screen_files.setdefault(slug, {})[state] = p

    def _read(self, rel):
        p = self.root / rel
        try:
            return p.read_text(encoding="utf-8")
        except OSError:
            return ""

    def screen_html(self, slug, state):
        p = self.screen_files.get(slug, {}).get(state)
        return p.read_text(encoding="utf-8") if p else ""

    def slugs(self):
        return sorted(self.screen_files)


# ── HTML 파서 ─────────────────────────────────────────────────────────────
class ScreenDoc(HTMLParser):
    """화면 HTML에서 검사에 필요한 것만 모은다."""

    def __init__(self):
        super().__init__()
        self.links = []                # link href 목록
        self.style_blocks = 0          # <style> 태그 개수
        self.inline_styles = []        # (tag, style 내용)
        self.svgs = []                 # {data-icon, width, height, has_use, inline_path}
        self.classes = set()           # 등장한 class 값 전부
        self.elements = []             # (tag, attrs dict) 순서대로
        self._svg_depth = 0
        self.screen_root = False       # .screen 루트 존재 여부
        self.has_input = False
        self.primary_count = 0
        self.button_rows = []          # [(컨테이너 번호, [버튼 size 클래스])]
        self._row_stack = []           # [(row_id, [sizes])]

    def handle_starttag(self, tag, attrs):
        a = dict(attrs)
        self.elements.append((tag, a))
        cls = a.get("class", "")
        for c in cls.split():
            self.classes.add(c)
        if "screen" in cls.split() and tag in ("main", "div", "body", "section"):
            self.screen_root = True
        if tag == "link" and a.get("href"):
            self.links.append(a["href"])
        if tag == "style":
            self.style_blocks += 1
        if a.get("style"):
            self.inline_styles.append((tag, a["style"]))
        if tag in ("input", "textarea", "select"):
            self.has_input = True
        if tag == "svg":
            self._svg_depth += 1
            self.svgs.append({
                "data-icon": a.get("data-icon"),
                "width": a.get("width"),
                "height": a.get("height"),
                "cls": cls,
                "inline_path": False,
            })
        elif self._svg_depth and tag == "path" and self.svgs:
            self.svgs[-1]["inline_path"] = True
        if "button-primary" in cls.split():
            self.primary_count += 1
        # 버튼 행: .actions / .row / .cta-row 안의 .btn size 클래스 수집
        in_row = bool(self._row_stack)
        cls_set = set(cls.split())
        if cls_set & {"actions", "row", "cta-row", "button-row"}:
            self._row_stack.append([])
            in_row = True
        if cls_set & {"button", "button-primary", "button-secondary", "button-ghost", "button-danger"} \
                or (cls_set & {"button-sm", "button-md", "button-lg"}) and "icon-button" not in cls_set:
            if self._row_stack:
                size = next((s for s in ("button-sm", "button-md", "button-lg") if s in cls_set), "button-md")
                self._row_stack[-1].append(size)

    def handle_endtag(self, tag):
        if tag == "svg" and self._svg_depth:
            self._svg_depth -= 1
        if tag in ("div", "footer", "section", "nav") and self._row_stack:
            self.button_rows.append(self._row_stack.pop())

    def result_rows(self):
        out = [r for r in self.button_rows if len(r) > 1]
        if self._row_stack:
            out += [r for r in self._row_stack if len(r) > 1]
        return out


def parse_screen(html):
    doc = ScreenDoc()
    doc.feed(html)
    return doc


# ── 개별 검사 ─────────────────────────────────────────────────────────────
def check_files_exist(tree, findings):
    if not tree.tokens_css:
        findings.append(Finding("-", "-", "tokens.css", "-", "file.missing", "없음", "tokens.css 생성"))
    if not tree.components_css:
        findings.append(Finding("-", "-", "components.css", "-", "file.missing", "없음", "components.css 생성"))
    if not tree.screen_files:
        findings.append(Finding("-", "-", "screens/", "-", "file.missing", "화면 html 0개", "screens/<slug>-<state>.html 생성"))


COLOR_LITERAL_RE = re.compile(
    r"#[0-9a-fA-F]{3,8}\b|rgba?\s*\(|hsla?\s*\(", re.I)


def check_palette(tree, findings):
    """색 리터럴은 tokens.css에만. 화면·컴포넌트는 var(--*) 참조만."""
    for slug, states in tree.screen_files.items():
        for state, path in states.items():
            html = path.read_text(encoding="utf-8")
            for i, line in enumerate(html.splitlines(), 1):
                for m in COLOR_LITERAL_RE.finditer(line):
                    findings.append(Finding(
                        path.name, state, "줄 {}".format(i), slug, "palette.bound",
                        "색 리터럴 {}".format(m.group(0)), "var(--*) 토큰 참조"))
    for i, line in enumerate(tree.components_css.splitlines(), 1):
        if "var(" in line and COLOR_LITERAL_RE.search(line):
            continue                                  # fallback: var(--x, #fff) 허용
        if COLOR_LITERAL_RE.search(line):
            findings.append(Finding(
                "components.css", "-", "줄 {}".format(i), "-", "palette.bound",
                "색 리터럴", "var(--*) 토큰 참조"))


def check_typography(tree, findings):
    """font-size/family/weight 선언은 tokens.css(.t-*)에만."""
    for fname, css in (("components.css", tree.components_css),):
        for i, line in enumerate(css.splitlines(), 1):
            if re.search(r"font-(size|family|weight)\s*:", line) and "var(" not in line:
                findings.append(Finding(
                    fname, "-", "줄 {}".format(i), "-", "typo.style",
                    line.strip()[:60], "var(--*) 타이포 토큰"))
    for slug, states in tree.screen_files.items():
        for state, path in states.items():
            html = path.read_text(encoding="utf-8")
            for i, line in enumerate(html.splitlines(), 1):
                if re.search(r"font-(size|family|weight)\s*:", line):
                    findings.append(Finding(
                        path.name, state, "줄 {}".format(i), slug, "typo.style",
                        "인라인 폰트 선언", ".t-* 역할 클래스"))


SPACE_PROP_RE = re.compile(
    r"(?<![\w\-])(padding|margin|gap|top|left|right|bottom|inset)[\w\-]*\s*:\s*([\d.]+)px", re.I)


def check_spacing(tree, th, findings):
    """components.css의 간격 px 값이 그리드 배수여야 한다."""
    for i, line in enumerate(tree.components_css.splitlines(), 1):
        for m in SPACE_PROP_RE.finditer(line):
            v = float(m.group(2))
            if abs(v - round(v)) > 1e-6 or int(round(v)) % th.space_step != 0:
                findings.append(Finding(
                    "components.css", "-", "줄 {}".format(i), "-", "space.grid",
                    "{} {}px".format(m.group(1), m.group(2)), "{} 배수".format(th.space_step)))


def check_screen_markup(tree, th, findings):
    """화면 html: 링크·style 금지·.screen 루트·폭 규칙·primary 1개·버튼 행."""
    for slug, states in tree.screen_files.items():
        for state, path in states.items():
            doc = parse_screen(path.read_text(encoding="utf-8"))
            name = path.name
            if state != "default":
                continue                               # 구조 검사는 default만
            if not any("tokens.css" in h for h in doc.links):
                findings.append(Finding(name, state, "head", slug, "file.link",
                                        "tokens.css 링크 없음", '<link href="../tokens.css">'))
            if not any("components.css" in h for h in doc.links):
                findings.append(Finding(name, state, "head", slug, "file.link",
                                        "components.css 링크 없음", '<link href="../components.css">'))
            if doc.style_blocks:
                findings.append(Finding(name, state, "<style>", slug, "markup.style",
                                        "{}개".format(doc.style_blocks), "0개 (스타일은 css 파일에)"))
            if doc.inline_styles:
                findings.append(Finding(name, state, "style= 속성", slug, "markup.style",
                                        "{}개".format(len(doc.inline_styles)), "0개 (클래스만)"))
            if not doc.screen_root:
                findings.append(Finding(name, state, "body", slug, "device.frame",
                                        ".screen 루트 없음", '<main class="screen">'))
            if doc.primary_count != 1:
                findings.append(Finding(name, state, "button-primary", slug, "primary.count",
                                        "{}개".format(doc.primary_count), "정확히 1개"))
            for row in doc.result_rows():
                if len(set(row)) > 1:
                    findings.append(Finding(name, state, ".actions 행", slug, "button.row",
                                            "size 혼합 {}".format("/".join(sorted(set(row)))),
                                            "같은 행은 같은 size"))


def check_device_frame(tree, th, findings):
    """.screen 크기가 규칙의 device.frame과 같아야 한다."""
    m = re.search(r"\.screen\b[^}]*}", tree.components_css, re.S)
    if not m:
        findings.append(Finding("components.css", "-", ".screen", "-", "device.frame",
                                "규칙 없음", ".screen {{width:{}px;height:{}px}}".format(th.frame_w, th.frame_h)))
        return
    block = m.group(0)
    w = re.search(r"width\s*:\s*(\d+)px", block)
    h = re.search(r"height\s*:\s*(\d+)px", block)
    if "var(--frame-w)" in block and "var(--frame-h)" in block:
        return
    if not w or not h:
        findings.append(Finding("components.css", "-", ".screen", "-", "device.frame",
                                "width/height 선언 없음", "{}x{}px".format(th.frame_w, th.frame_h)))
    elif int(w.group(1)) != th.frame_w or int(h.group(1)) != th.frame_h:
        findings.append(Finding("components.css", "-", ".screen", "-", "device.frame",
                                "{}x{}".format(w.group(1), h.group(1)),
                                "{}x{}".format(th.frame_w, th.frame_h)))


def check_safe_area(tree, th, findings):
    """하단 고정 바·탭바가 safe-area 하단을 확보해야 한다. (데스크톱 프레임이면 생략)"""
    if th.frame_w >= 1024:
        return
    if "safe-area-inset-bottom" not in tree.components_css \
            and not re.search(r"padding-bottom\s*:\s*(\d+)", tree.components_css or ""):
        findings.append(Finding("components.css", "-", "tab-bar/bottom-cta", "-", "safe-area",
                                "safe-area 처리 없음",
                                "env(safe-area-inset-bottom) 또는 하단 패딩 {}px".format(th.safe_bottom)))


def check_variants(tree, th, findings):
    """components.css에 버튼 variant × size × state 선택자가 있어야 한다."""
    css = tree.components_css
    for v in VARIANT_NAMES:
        if "button-{}".format(v) not in css:
            findings.append(Finding("components.css", "-", "button-{}".format(v), "-",
                                    "variant.coverage", "없음", "variant 클래스"))
    for s in SIZE_NAMES:
        if "button-{}".format(s) not in css:
            findings.append(Finding("components.css", "-", "button-{}".format(s), "-",
                                    "variant.coverage", "없음", "size 클래스"))
    for state in th.button_states:
        if state == "default":
            continue
        if not any(sel in css for sel in STATE_SELECTORS.get(state, [])):
            findings.append(Finding("components.css", "-", "state:{}".format(state), "-",
                                    "variant.coverage", "없음",
                                    "상태 선택자 {}".format(" 또는 ".join(STATE_SELECTORS.get(state, [])))))
    for m in re.finditer(r"\.(button(?:-(?:sm|md|lg))?)\s*\{([^}]*)\}", css):
        h = re.search(r"height\s*:\s*(\d+)px", m.group(2))
        if h and int(h.group(1)) not in th.button_heights:
            findings.append(Finding("components.css", "-", ".{}".format(m.group(1)), "-",
                                    "button.sizes", "height {}px".format(h.group(1)),
                                    "허용 높이 {}".format(th.button_heights)))


def check_z_scale(tree, th, findings):
    """z-index 값은 z.scale 값만 허용."""
    allowed = set(th.z_scale.values()) | {0}
    for i, line in enumerate(tree.components_css.splitlines(), 1):
        for m in re.finditer(r"z-index\s*:\s*(-?\d+)", line):
            if int(m.group(1)) not in allowed:
                findings.append(Finding("components.css", "-", "줄 {}".format(i), "-", "z.scale",
                                        "z-index {}".format(m.group(1)),
                                        "허용값 {}".format(sorted(allowed))))


def check_icons(tree, th, allow, findings):
    """svg 아이콘: data-icon 허용 목록 + 허용 크기. 스프라이트 use 참조만."""
    sprite_ids = set(re.findall(r'<symbol[^>]*id="([^"]+)"', tree.icons_svg))
    for slug, states in tree.screen_files.items():
        for state, path in states.items():
            if state != "default":
                continue
            doc = parse_screen(path.read_text(encoding="utf-8"))
            for svg in doc.svgs:
                name = (svg["data-icon"] or "").lower()
                if svg["inline_path"]:
                    findings.append(Finding(path.name, state, "icon:{}".format(name or "?"), slug,
                                            "icon.set", "직접 그린 <path>", "icons.svg <use> 스프라이트"))
                if not name:
                    findings.append(Finding(path.name, state, "<svg>", slug, "icon.raw",
                                            "data-icon 없음", 'data-icon="이름" + <use> 스프라이트'))
                    continue
                if allow is not None and name not in allow:
                    findings.append(Finding(path.name, state, "icon:{}".format(name), slug,
                                            "icon.allowlist", "목록 밖", "icons.md 허용 목록에 추가 후 사용자 확인"))
                if sprite_ids and ICON_PREFIX + name not in sprite_ids:
                    findings.append(Finding(path.name, state, "icon:{}".format(name), slug,
                                            "icon.sprite", "icons.svg에 symbol 없음",
                                            '<symbol id="{}{}">'.format(ICON_PREFIX, name)))
                for dim in ("width", "height"):
                    v = svg[dim]
                    if v is not None and v.isdigit() and int(v) not in th.icon_sizes:
                        findings.append(Finding(path.name, state, "icon:{}".format(name), slug,
                                                "icon.size", "{}={}".format(dim, v),
                                                "{} 중 하나".format(th.icon_sizes)))


def component_classes(css):
    """components.css 선택자에 등장하는 클래스 이름 집합 (주석·url() 제외)."""
    css = re.sub(r"/\*.*?\*/", "", css, flags=re.S)
    css = re.sub(r"url\([^)]*\)", "", css)
    return set(re.findall(r"\.([a-zA-Z][\w\-]*)", css))


def check_component_manifest(tree, manifest, findings):
    """`<slug>-default.html`의 클래스가 screens.md 구성표와 일치해야 한다."""
    if not manifest:
        return
    css_component_classes = component_classes(tree.components_css)
    for slug, states in tree.screen_files.items():
        path = states.get("default")
        if not path:
            continue
        doc = parse_screen(path.read_text(encoding="utf-8"))
        expected = manifest.get(slug) or manifest.get(slug.replace("-", " "))
        if expected is None:
            findings.append(Finding(path.name, "default", path.name, slug,
                                    "component.manifest", "screens.md에 없는 화면",
                                    "screens.md 구성표에 행 추가"))
            continue
        found_kinds = set()
        for comp in expected:
            cls = kebab(comp)
            if cls in doc.classes:
                found_kinds.add(comp)
        for missing in sorted(set(expected) - found_kinds):
            findings.append(Finding(path.name, "default", path.name, slug,
                                    "component.manifest", "{} 없음".format(missing),
                                    "screens.md 구성대로 .{} 사용".format(kebab(missing))))
        expected_classes = {kebab(c) for c in expected} | {kebab(c) for c in MANIFEST_IGNORE}
        implicit = {"screen", "icon", "icon-button", "skeleton",
                    "actions", "row", "cta-row", "button-row"}
        for cls in sorted(doc.classes):
            if cls not in css_component_classes or cls in implicit or cls.startswith("is-"):
                continue                    # 유틸리티·레이아웃·상태 클래스는 검사 대상 아님
            if cls in expected_classes or any(cls.startswith(e + "-") for e in expected_classes):
                continue                    # manifest 컴포넌트 본체 또는 variant 클래스
            findings.append(Finding(path.name, "default", ".{}".format(cls), slug,
                                    "component.manifest", "{} (구성표 밖)".format(cls),
                                    "제거하거나 screens.md에 추가 후 사용자 확인"))


def check_screen_states(tree, brief_screens, findings, screen_states=None):
    """화면마다 상태 파일 세트가 있어야 한다. screens.md states_map 우선."""
    if screen_states:
        targets = screen_states.items()
    elif brief_screens:
        targets = [(kebab(n) if re.search(r"[A-Z]", n) else re.sub(r"[\s_]+", "-", n.lower()),
                    ["default"] + REQUIRED_STATES) for n in brief_screens]
    else:
        return
    for slug, required in targets:
        states = tree.screen_files.get(slug)
        if states is None:
            findings.append(Finding("screens/", "-", slug, slug, "screen.missing",
                                    "화면 파일 없음", "screens/{}-default.html".format(slug)))
            continue
        if "default" not in states:
            findings.append(Finding("screens/", "-", slug, slug, "screen.missing",
                                    "default 없음", "{}-default.html".format(slug)))
            continue
        for st in required:
            if st in ("default",):
                continue
            if st not in states:
                findings.append(Finding("screens/", st, slug, slug, "screen.states",
                                        "{} 없음".format(st), "{}-{}.html".format(slug, st)))
        doc = parse_screen(tree.screen_html(slug, "default"))
        if doc.has_input and "keyboard" in required and "keyboard" not in states:
            findings.append(Finding("screens/", "keyboard", slug, slug, "screen.states",
                                    "keyboard 없음", "{}-keyboard.html (입력 화면)".format(slug)))


# ── 메인 ──────────────────────────────────────────────────────────────────
def run_audit(root, rules_path, brief_path=None, icons_path=None, screens_path=None):
    """(findings, warnings, stats) 반환."""
    tree = OdTree(root)
    rules, warnings = parse_rules_file(rules_path)
    th = Thresholds(rules, warnings)

    brief_screens = None
    if brief_path:
        brief_screens, w = parse_brief_screens(brief_path)
        warnings += w

    allow = None
    if icons_path:
        allow, w = parse_icon_allowlist(icons_path)
        warnings += w

    manifest, screen_states = None, None
    if screens_path:
        manifest, screen_states, w = parse_screens_manifest(screens_path)
        warnings += w

    findings = []
    check_files_exist(tree, findings)
    check_palette(tree, findings)
    check_typography(tree, findings)
    check_spacing(tree, th, findings)
    check_screen_markup(tree, th, findings)
    check_device_frame(tree, th, findings)
    check_safe_area(tree, th, findings)
    check_variants(tree, th, findings)
    check_z_scale(tree, th, findings)
    check_icons(tree, th, allow, findings)
    check_component_manifest(tree, manifest, findings)
    check_screen_states(tree, brief_screens, findings, screen_states)

    stats = {
        "screens": len(tree.screen_files),
        "manifestScreens": len(manifest or {}),
        "findings": len(findings),
    }
    findings.sort(key=lambda f: f.sort_key())
    return findings, warnings, stats


def write_fix_list(findings, path):
    lines = ["# fix-list", "",
             "| 파일 | 화면/상태 | 위치 | 규칙 키 | 현재 | 기대 |",
             "|---|---|---|---|---|---|"]
    for f in findings:
        lines.append("| {} | {} | {} | {} | {} | {} |".format(
            f.page, f.frame, f.node_name, f.key, f.actual, f.expected))
    Path(path).write_text("\n".join(lines) + "\n", encoding="utf-8")


def main(argv=None):
    ap = argparse.ArgumentParser(description="OpenDesign artifact A단계 검사")
    ap.add_argument("--root", required=True, help="artifact 루트 (design/od 또는 OD 프로젝트 resolvedDir)")
    ap.add_argument("--rules", required=True)
    ap.add_argument("--brief")
    ap.add_argument("--icons")
    ap.add_argument("--screens")
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--fix-list", dest="fix_list")
    args = ap.parse_args(argv)

    try:
        findings, warnings, stats = run_audit(
            args.root, args.rules, args.brief, args.icons, args.screens)
    except (ValueError, OSError) as exc:
        print("[ERROR] {}".format(exc), file=sys.stderr)
        return 2

    if args.fix_list:
        write_fix_list(findings, args.fix_list)

    if args.json:
        print(json.dumps({
            "passed": not findings,
            "findings": [f.as_dict() for f in findings],
            "warnings": warnings,
            "stats": stats,
        }, ensure_ascii=False, indent=2))
    else:
        for w in warnings:
            print("[WARN] {}".format(w))
        for f in findings:
            print(f.line())
        if not findings:
            print("[PASS] 결함 없음 (화면 {}개)".format(stats["screens"]))
    return 1 if findings else 0


if __name__ == "__main__":
    sys.exit(main())
