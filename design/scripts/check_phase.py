#!/usr/bin/env python3
"""
check_phase.py — 디자인 하네스 단계별 산출물 결정론적 검사 스크립트.

LLM이 "다 채웠다"고 말해도, design/ 폴더의 실제 파일(마크다운 표·헤더, HTML 시안)을
정규식/표 파싱으로 검사해 게이트 역할을 한다. 외부 라이브러리 없이 표준 라이브러리만 사용한다.

사용법:
    python3 scripts/check_phase.py --phase {structure|flow|taste|rules|probes|all}
                                    [--design-dir design] [--json]

종료 코드: 0 통과 / 1 실패 / 2 파일 없음·파싱 불가.
"""
from __future__ import annotations

import argparse
import glob
import json
import os
import re
import sys
from dataclasses import dataclass, field
from typing import Dict, List, Optional, Tuple

# ---------------------------------------------------------------------------
# 공통 유틸
# ---------------------------------------------------------------------------

CIRCLED = "①②③④⑤⑥⑦⑧⑨⑩⑪⑫⑬⑭⑮⑯⑰⑱⑲⑳"
CIRCLED_RE = re.compile("[" + CIRCLED + "]")
CIRCLED_CLASS = "[" + CIRCLED + "]"


@dataclass
class Result:
    """검사 항목 하나의 결과. detail은 '위치 — 이유' 형태로 합쳐서 담는다."""

    name: str
    ok: bool
    file: str = ""
    detail: str = ""

    def to_dict(self) -> Dict:
        return {"name": self.name, "ok": self.ok, "file": self.file, "detail": self.detail}


class Reporter:
    """한 단계(phase) 검사 동안 결과를 모으는 헬퍼."""

    def __init__(self, phase: str):
        self.phase = phase
        self.results: List[Result] = []

    def ok(self, name: str, file: str = "", detail: str = "") -> None:
        self.results.append(Result(f"{self.phase}:{name}", True, file, detail))

    def fail(self, name: str, file: str, detail: str) -> None:
        self.results.append(Result(f"{self.phase}:{name}", False, file, detail))

    def ok_if_no_fail_since(self, mark: int, name: str, file: str = "") -> None:
        """mark 이후 새 fail이 하나도 없으면 OK 하나를 추가한다(행 단위 루프용)."""
        if not any(not r.ok for r in self.results[mark:]):
            self.ok(name, file)


def read_file(path: str) -> Optional[str]:
    if not os.path.isfile(path):
        return None
    try:
        with open(path, "r", encoding="utf-8") as f:
            return f.read()
    except OSError:
        return None


# ---------------------------------------------------------------------------
# 마크다운 파싱
# ---------------------------------------------------------------------------

_HEADING_RE = re.compile(r"^(#{1,6})\s+(.*)$")
_SEP_RE = re.compile(r"^\s*\|?[\s:|-]+\|?\s*$")


def split_sections(text: str) -> List[Dict]:
    """'# 제목' 헤딩 기준으로 문서를 섹션(제목/레벨/본문)으로 나눈다."""
    lines = text.splitlines()
    headings = []
    for i, line in enumerate(lines):
        m = _HEADING_RE.match(line)
        if m:
            headings.append((i, len(m.group(1)), m.group(2).strip()))
    sections = []
    for idx, (line_i, level, title) in enumerate(headings):
        end = len(lines)
        for j in range(idx + 1, len(headings)):
            if headings[j][1] <= level:
                end = headings[j][0]
                break
        sections.append(
            {
                "title": title,
                "level": level,
                "start": line_i + 1,
                "end": end,
                "text": "\n".join(lines[line_i + 1 : end]),
            }
        )
    return sections


def find_section(sections: List[Dict], title_regex: str) -> Optional[Dict]:
    pat = re.compile(title_regex)
    for sec in sections:
        if pat.search(sec["title"]):
            return sec
    return None


def split_row(line: str) -> List[str]:
    s = line.strip()
    if s.startswith("|"):
        s = s[1:]
    if s.endswith("|"):
        s = s[:-1]
    return [c.strip() for c in s.split("|")]


def parse_tables(text: str) -> List[Dict]:
    """텍스트 안의 모든 마크다운 표를 등장 순서대로 파싱한다."""
    lines = text.splitlines()
    tables = []
    i = 0
    n = len(lines)
    while i < n:
        line = lines[i]
        if line.strip().startswith("|") and i + 1 < n and _SEP_RE.match(lines[i + 1]) and "-" in lines[i + 1]:
            header = split_row(line)
            rows = []
            j = i + 2
            while j < n and lines[j].strip().startswith("|"):
                rows.append(split_row(lines[j]))
                j += 1
            tables.append({"header": header, "rows": rows})
            i = j
            continue
        i += 1
    return tables


def safe_index(header: List[str], name: str) -> Optional[int]:
    for i, h in enumerate(header):
        if h.strip() == name:
            return i
    return None


def cell(row: List[str], idx: Optional[int]) -> str:
    if idx is None or idx >= len(row):
        return ""
    return row[idx].strip()


def find_line_value(text: str, label_regex: str) -> Optional[str]:
    """'라벨: 값' 형태 줄에서 값을 찾는다. 못 찾으면 None."""
    pat = re.compile(label_regex)
    for line in text.splitlines():
        m = pat.match(line.strip())
        if m:
            return m.group(1).strip()
    return None


def find_all_blocks(text: str, header_regex: str) -> List[str]:
    """header_regex에 매치하는 줄들을 구분자로 텍스트를 블록으로 쪼갠다."""
    lines = text.splitlines()
    pat = re.compile(header_regex)
    idxs = [i for i, l in enumerate(lines) if pat.match(l.strip())]
    blocks = []
    for k, i in enumerate(idxs):
        end = idxs[k + 1] if k + 1 < len(idxs) else len(lines)
        blocks.append("\n".join(lines[i:end]))
    return blocks


# ---------------------------------------------------------------------------
# brief.md 공용 헬퍼 (구조/플로우/취향 단계에서 재사용)
# ---------------------------------------------------------------------------

def get_screen_rows(brief_text: str) -> Tuple[List[str], List[List[str]]]:
    sections = split_sections(brief_text)
    sec = find_section(sections, r"^1\.\s*화면 목록")
    if sec is None:
        return [], []
    tables = parse_tables(sec["text"])
    if not tables:
        return [], []
    return tables[0]["header"], tables[0]["rows"]


def get_assumption_rows(brief_text: str) -> Tuple[List[str], List[List[str]]]:
    sections = split_sections(brief_text)
    sec = find_section(sections, r"^5\.\s*가정 로그")
    if sec is None:
        return [], []
    tables = parse_tables(sec["text"])
    if not tables:
        return [], []
    return tables[0]["header"], tables[0]["rows"]


# ---------------------------------------------------------------------------
# 1. structure — design/brief.md
# ---------------------------------------------------------------------------

REQUIRED_STATES = ["초기", "빈", "로딩", "성공", "실패", "비활성"]


def check_structure(design_dir: str) -> Optional[List[Result]]:
    path = os.path.join(design_dir, "brief.md")
    text = read_file(path)
    if text is None:
        return None
    rep = Reporter("structure")

    # §1 화면 목록
    header, rows = get_screen_rows(text)
    if not header:
        rep.fail("screens-table", path, "§1 화면 목록 표를 찾을 수 없음")
    else:
        if len(rows) < 1:
            rep.fail("screens-rows", path, "§1 화면 목록에 데이터 행이 0개")
        else:
            idx_state = safe_index(header, "정의된 상태")
            idx_primary = safe_index(header, "primary 액션")
            mark = len(rep.results)
            for i, row in enumerate(rows):
                screen_name = cell(row, 0) or f"행{i+1}"
                # 모든 셀 비어있지 않음
                for ci, h in enumerate(header):
                    if cell(row, ci) == "":
                        rep.fail(
                            "screens-cell-empty",
                            path,
                            f"§1 화면 목록[{screen_name}] '{h}' 셀이 비어있음",
                        )
                # 정의된 상태 6개 전부
                state_cell = cell(row, idx_state)
                missing = [s for s in REQUIRED_STATES if s not in state_cell]
                if missing:
                    rep.fail(
                        "screens-states",
                        path,
                        f"§1 화면 목록[{screen_name}] '정의된 상태'에 {missing} 누락",
                    )
                # primary 액션 정확히 1개
                primary_cell = cell(row, idx_primary)
                if primary_cell and ("," in primary_cell or "/" in primary_cell):
                    rep.fail(
                        "screens-primary-single",
                        path,
                        f"§1 화면 목록[{screen_name}] 'primary 액션'에 여러 값(쉼표/슬래시) 있음: {primary_cell!r}",
                    )
            rep.ok_if_no_fail_since(mark, "screens-rows-valid", path)

    # 플랫폼 줄
    platform_val = find_line_value(text, r"^-\s*플랫폼\s*:\s*(.*)$")
    if platform_val is None:
        rep.fail("platform", path, "'- 플랫폼:' 줄을 찾을 수 없음")
    elif platform_val.strip() == "iOS / Android / 둘 다":
        rep.fail("platform", path, "플랫폼이 템플릿 기본값 그대로(미선택)")
    elif platform_val.strip() == "":
        rep.fail("platform", path, "플랫폼 값이 비어있음")
    else:
        rep.ok("platform", path)

    # §5 가정 로그 표 존재
    a_header, _ = get_assumption_rows(text)
    if not a_header:
        rep.fail("assumption-table", path, "§5 가정 로그 표를 찾을 수 없음")
    else:
        rep.ok("assumption-table", path)

    return rep.results


# ---------------------------------------------------------------------------
# 2. flow — design/brief.md §2 + design/probes/flow-*.html
# ---------------------------------------------------------------------------

_STEP_LINE_RE = re.compile(r"^\s*\d+\.\s+(.*)$")
_STEP_END_LABEL_RE = re.compile(r"\((" + CIRCLED_CLASS + r")\)\s*$")


def check_flow(design_dir: str) -> Optional[List[Result]]:
    brief_path = os.path.join(design_dir, "brief.md")
    text = read_file(brief_path)
    if text is None:
        return None
    rep = Reporter("flow")

    sections = split_sections(text)
    sec2 = find_section(sections, r"^2\.\s*플로우")
    all_scenario_numbers: set = set()

    if sec2 is None:
        rep.fail("section", brief_path, "§2 플로우 섹션을 찾을 수 없음")
    else:
        scenario_blocks = find_all_blocks(sec2["text"], r"^###\s+시나리오")
        if not scenario_blocks:
            rep.fail("scenario-count", brief_path, "시나리오가 1개도 없음")
        else:
            mark = len(rep.results)
            for block in scenario_blocks:
                title_line = block.splitlines()[0].strip()
                steps = [m.group(1) for l in block.splitlines() for m in [_STEP_LINE_RE.match(l)] if m]
                if not (5 <= len(steps) <= 9):
                    rep.fail(
                        "scenario-steps-count",
                        brief_path,
                        f"'{title_line}' 단계 수 {len(steps)}개 (5~9개 필요)",
                    )
                for si, step in enumerate(steps, start=1):
                    m = _STEP_END_LABEL_RE.search(step)
                    if not m:
                        rep.fail(
                            "scenario-step-label",
                            brief_path,
                            f"'{title_line}' {si}번째 단계 끝에 원문자 번호 없음: {step!r}",
                        )
                    else:
                        all_scenario_numbers.add(m.group(1))
            rep.ok_if_no_fail_since(mark, "scenarios-valid", brief_path)

        # 화면별 primary CTA 줄
        cta_line = find_line_value(sec2["text"], r"^-\s*화면별 primary CTA\s*:\s*(.*)$")
        screen_header, screen_rows = get_screen_rows(text)
        screen_names = [cell(r, 0) for r in screen_rows if cell(r, 0)]
        if cta_line is None or cta_line.strip() == "":
            rep.fail("cta-line", brief_path, "'- 화면별 primary CTA:' 줄이 없거나 비어있음")
        else:
            parts = [p.strip() for p in re.split(r"[,·]", cta_line) if p.strip()]
            if not parts:
                rep.fail("cta-line", brief_path, "화면별 primary CTA 항목이 비어있음")
            else:
                mark = len(rep.results)
                seg_re = re.compile(r".+→.*" + CIRCLED_CLASS)
                for p in parts:
                    if not seg_re.match(p):
                        rep.fail(
                            "cta-line-format",
                            brief_path,
                            f"'화면 → 번호' 형식이 아님: {p!r}",
                        )
                if screen_names and len(parts) != len(screen_names):
                    rep.fail(
                        "cta-line-count",
                        brief_path,
                        f"화면별 primary CTA 항목 수({len(parts)})가 §1 화면 수({len(screen_names)})와 다름",
                    )
                rep.ok_if_no_fail_since(mark, "cta-line-valid", brief_path)

    # 로우파이 HTML
    flow_files = sorted(glob.glob(os.path.join(design_dir, "probes", "flow-*.html")))
    if not flow_files:
        rep.fail("lofi-html", os.path.join(design_dir, "probes"), "flow-*.html 로우파이 파일이 없음")
    else:
        html_all = ""
        for f in flow_files:
            c = read_file(f)
            if c:
                html_all += c
        html_labels = set(CIRCLED_RE.findall(html_all))
        missing = sorted(all_scenario_numbers - html_labels)
        if missing:
            rep.fail(
                "lofi-labels-cover",
                ", ".join(flow_files),
                f"시나리오 원문자 번호 {missing}가 로우파이 HTML 라벨에 없음",
            )
        else:
            rep.ok("lofi-labels-cover", ", ".join(flow_files))

    return rep.results


# ---------------------------------------------------------------------------
# 3. taste — design/decisions.md
# ---------------------------------------------------------------------------

_AXIS_TITLES = {
    1: "밝기·색온도",
    2: "정보 밀도",
    3: "형태",
    4: "강조색 성격",
    5: "타이포 성격",
}


def check_taste(design_dir: str) -> Optional[List[Result]]:
    dec_path = os.path.join(design_dir, "decisions.md")
    text = read_file(dec_path)
    if text is None:
        return None
    rep = Reporter("taste")

    brief_path = os.path.join(design_dir, "brief.md")
    brief_text = read_file(brief_path)
    _, assumption_rows = get_assumption_rows(brief_text) if brief_text else ([], [])

    for n in range(1, 6):
        blocks = find_all_blocks(text, rf"^###\s+축\s*{n}\b")
        if not blocks:
            rep.fail("axis-section", dec_path, f"'### 축 {n}' 섹션이 없음")
            continue
        block = blocks[0]

        selected_val = find_line_value(block, r"^-\s*선택값\s*:\s*(.*)$")
        if selected_val is None or selected_val.strip() == "":
            rep.fail("axis-selected-value", dec_path, f"축 {n} '선택값:' 이 비어있음")
        else:
            rep.ok("axis-selected-value", dec_path)

        best_val = find_line_value(block, r"^-\s*가장 좋은 것\s*:\s*(.*)$")
        if best_val is None:
            rep.fail("axis-best", dec_path, f"축 {n} '가장 좋은 것:' 줄이 없음")
            continue
        best_val = best_val.strip()
        if best_val == "A / B / C / 모르겠어요":
            rep.fail("axis-best", dec_path, f"축 {n} '가장 좋은 것'이 템플릿 기본값 그대로(미확정)")
            continue
        if best_val not in ("A", "B", "C", "모르겠어요"):
            rep.fail("axis-best", dec_path, f"축 {n} '가장 좋은 것' 값이 잘못됨: {best_val!r}")
            continue
        rep.ok("axis-best", dec_path)

        if best_val == "모르겠어요":
            axis_marker = f"축 {n}"
            found = any(
                any(axis_marker in c or _AXIS_TITLES.get(n, "") in c for c in row)
                for row in assumption_rows
            )
            if not found:
                rep.fail(
                    "axis-unknown-logged",
                    brief_path,
                    f"축 {n}을(를) '모르겠어요'로 선택했는데 §5 가정 로그에 관련 행이 없음",
                )
            else:
                rep.ok("axis-unknown-logged", brief_path)

    return rep.results


# ---------------------------------------------------------------------------
# 4. rules — design/design-rules.md
# ---------------------------------------------------------------------------


def _nums(v: str) -> List[int]:
    return [int(x) for x in re.findall(r"\d+", v)]


def validate_color_value(v: str) -> bool:
    v = v.strip()
    if re.fullmatch(r"#[0-9A-Fa-f]{6}", v):
        return True
    if re.search(r"rgba\([^)]*\)", v):
        return True
    if "accent" in v:
        return True
    return False


def validate_space_scale(v: str) -> bool:
    nums = _nums(v)
    if not nums:
        return False
    return all(n % 4 == 0 for n in nums)


def validate_icon_sizes(v: str) -> bool:
    nums = set(_nums(v))
    if not nums:
        return False
    return nums.issubset({16, 20, 24})


def validate_tap_min(v: str) -> bool:
    nums = _nums(v)
    if not nums:
        return False
    return nums[0] >= 44


def validate_device_frame(v: str) -> bool:
    return "390" in v


def validate_zscale_ascending(v: str) -> bool:
    nums = _nums(v)
    if len(nums) < 2:
        return False
    return all(nums[i] <= nums[i + 1] for i in range(len(nums) - 1))


def validate_button_sizes(v: str) -> bool:
    heights = re.findall(r"(sm|md|lg)\s+(\d+)h", v)
    if not heights:
        return False
    for size, h in heights:
        h = int(h)
        if h >= 44:
            continue
        if size == "sm" and h == 36:
            continue
        return False
    return True


A_VALIDATORS = {
    "space.scale": validate_space_scale,
    "tap.min": validate_tap_min,
    "device.frame": validate_device_frame,
    "z.scale": validate_zscale_ascending,
}

B_VALIDATORS = {
    "icon.sizes": validate_icon_sizes,
    "button.sizes": validate_button_sizes,
}

_STAGE_MARK_RE = re.compile(r"축\s*\d|[123]단계|레퍼런스")
_NO_STAGE_NEEDED = {"기본값", "고정", "자동", ""}


def check_rules(design_dir: str) -> Optional[List[Result]]:
    path = os.path.join(design_dir, "design-rules.md")
    text = read_file(path)
    if text is None:
        return None
    rep = Reporter("rules")

    status = find_line_value(text, r"^status\s*:\s*(.*)$")
    if status is None or status.strip() != "confirmed":
        rep.fail("status", path, f"status가 confirmed가 아님 (현재: {status!r})")
    else:
        rep.ok("status", path)

    confirmed_at = find_line_value(text, r"^confirmed_at\s*:\s*(.*)$")
    if not confirmed_at or not confirmed_at.strip():
        rep.fail("confirmed_at", path, "confirmed_at 값이 비어있음")
    elif not re.search(r"\d{4}-\d{2}-\d{2}", confirmed_at):
        rep.fail("confirmed_at", path, f"confirmed_at에 날짜 형식(YYYY-MM-DD)이 없음: {confirmed_at!r}")
    else:
        rep.ok("confirmed_at", path)

    sections = split_sections(text)
    source_rows: List[Tuple[str, str, str, str]] = []  # (key, source, value, table)

    # §A 토큰
    sec_a = find_section(sections, r"^A\.\s*토큰")
    if sec_a is None:
        rep.fail("section-A", path, "'A. 토큰' 섹션이 없음")
    else:
        tables = parse_tables(sec_a["text"])
        if not tables:
            rep.fail("section-A-table", path, "'A. 토큰' 표가 없음")
        else:
            header = tables[0]["header"]
            idx_key, idx_val, idx_src = (
                safe_index(header, "키"),
                safe_index(header, "값"),
                safe_index(header, "출처"),
            )
            mark = len(rep.results)
            for row in tables[0]["rows"]:
                key = cell(row, idx_key)
                if not key:
                    continue
                val = cell(row, idx_val)
                src = cell(row, idx_src)
                if not val:
                    rep.fail("A-value-empty", path, f"A.토큰[{key}] 값이 비어있음")
                    continue
                if key.startswith("color."):
                    if not validate_color_value(val):
                        rep.fail(
                            "A-value-format",
                            path,
                            f"A.토큰[{key}] color 형식이 아님(#RRGGBB/rgba/accent 파생 아님): {val!r}",
                        )
                elif key in A_VALIDATORS and not A_VALIDATORS[key](val):
                    rep.fail("A-value-format", path, f"A.토큰[{key}] 값 규칙 위반: {val!r}")
                source_rows.append((key, src, val, "A"))
            rep.ok_if_no_fail_since(mark, "A-values-valid", path)

    # §B 컴포넌트 규칙
    sec_b = find_section(sections, r"^B\.\s*컴포넌트 규칙")
    if sec_b is None:
        rep.fail("section-B", path, "'B. 컴포넌트 규칙' 섹션이 없음")
    else:
        tables = parse_tables(sec_b["text"])
        if not tables:
            rep.fail("section-B-table", path, "'B. 컴포넌트 규칙' 표가 없음")
        else:
            header = tables[0]["header"]
            idx_key, idx_val, idx_src, idx_use = (
                safe_index(header, "키"),
                safe_index(header, "값"),
                safe_index(header, "출처"),
                safe_index(header, "사용 여부"),
            )
            mark = len(rep.results)
            for row in tables[0]["rows"]:
                key = cell(row, idx_key)
                if not key:
                    continue
                val = cell(row, idx_val)
                src = cell(row, idx_src)
                use = cell(row, idx_use)
                unused = "미사용" in use
                if not val and not unused:
                    rep.fail(
                        "B-value-empty",
                        path,
                        f"B.컴포넌트[{key}] 값이 비어있고 사용 여부도 (미사용)이 아님",
                    )
                    continue
                if val and not unused and key in B_VALIDATORS and not B_VALIDATORS[key](val):
                    rep.fail("B-value-format", path, f"B.컴포넌트[{key}] 값 규칙 위반: {val!r}")
                if val:
                    source_rows.append((key, src, val, "B"))
            rep.ok_if_no_fail_since(mark, "B-values-valid", path)

    # §C 프로젝트 전용 규칙 (있으면 출처 검사에만 포함)
    sec_c = find_section(sections, r"^C\.\s*프로젝트 전용 규칙")
    if sec_c is not None:
        tables = parse_tables(sec_c["text"])
        if tables:
            header = tables[0]["header"]
            idx_key, idx_val, idx_src = (
                safe_index(header, "키"),
                safe_index(header, "값"),
                safe_index(header, "출처"),
            )
            for row in tables[0]["rows"]:
                key = cell(row, idx_key)
                if not key:
                    continue
                val = cell(row, idx_val)
                src = cell(row, idx_src)
                if val:
                    source_rows.append((key, src, val, "C"))

    # 출처 단계 표기 검사 (A/B/C 공통)
    if source_rows:
        mark = len(rep.results)
        for key, src, val, table_name in source_rows:
            src_norm = src.strip()
            if src_norm in _NO_STAGE_NEEDED:
                continue
            if not _STAGE_MARK_RE.search(src_norm):
                rep.fail(
                    "source-stage-mark",
                    path,
                    f"{table_name}.[{key}] 출처에 단계 표기(축 n/1~3단계/레퍼런스) 없음: {src_norm!r}",
                )
        rep.ok_if_no_fail_since(mark, "source-stage-valid", path)

    return rep.results


# ---------------------------------------------------------------------------
# 5. probes — design/probes/*.html
# ---------------------------------------------------------------------------

_SCRIPT_SRC_RE = re.compile(r"<script[^>]*\bsrc\s*=\s*[\"']([^\"']+)[\"']", re.I)
_LINK_HREF_RE = re.compile(r"<link[^>]*\bhref\s*=\s*[\"']([^\"']+)[\"']", re.I)
_DATA_AXIS_RE = re.compile(r"<section[^>]*\bdata-axis\s*=\s*[\"']([^\"']+)[\"'][^>]*>", re.I)
_DATA_V_RE = re.compile(r"data-v\s*=\s*[\"'](A|B|C)[\"']", re.I)
_CLASS_LABEL_RE = re.compile(r'class\s*=\s*"[^"]*\blabel\b[^"]*"')
# 데스크톱 예외: --frame-w:<N>px 변수면 기기 폭 선언으로 인정 (모바일 390 / 데스크톱 1440 등)
_FRAME_390_RE = re.compile(r"(width\s*:\s*390px|--frame-w\s*:\s*\d+px)")
_DB_USE_RE = re.compile(r"claude\.use\(\s*[\"']db[\"']\s*\)")
_FLOW_STATES = ["로딩", "실패", "비어있음"]


def _is_external(url: str) -> bool:
    return bool(re.match(r"^(https?:)?//", url.strip(), re.I))


def _split_data_axis_sections(html: str) -> List[Tuple[str, str]]:
    matches = list(_DATA_AXIS_RE.finditer(html))
    blocks = []
    for i, m in enumerate(matches):
        start = m.end()
        end = matches[i + 1].start() if i + 1 < len(matches) else len(html)
        blocks.append((m.group(1), html[start:end]))
    return blocks


def check_probes(design_dir: str) -> Optional[List[Result]]:
    files = sorted(glob.glob(os.path.join(design_dir, "probes", "*.html")))
    if not files:
        return None
    rep = Reporter("probes")

    for f in files:
        content = read_file(f)
        if content is None:
            rep.fail("readable", f, "파일을 읽을 수 없음")
            continue
        base = os.path.basename(f)

        # hub.html은 생성된 셸이라 내용 검사 면제
        if base == "hub.html":
            rep.ok("hub-shell", f, "생성된 허브 — 면제")
            continue

        # 외부 스크립트 금지
        scripts = _SCRIPT_SRC_RE.findall(content)
        ext_scripts = [s for s in scripts if _is_external(s)]
        if ext_scripts:
            rep.fail("no-external-script", f, f"외부 <script src>가 있음: {ext_scripts}")
        else:
            rep.ok("no-external-script", f)

        # 외부 스타일시트: fonts.googleapis.com만 허용
        links = _LINK_HREF_RE.findall(content)
        bad_links = [h for h in links if _is_external(h) and "fonts.googleapis.com" not in h]
        if bad_links:
            rep.fail("no-bad-external-link", f, f"허용되지 않은 외부 <link>: {bad_links}")
        else:
            rep.ok("no-bad-external-link", f)

        # 번호 라벨 5개 이상
        circled_count = len(CIRCLED_RE.findall(content))
        label_count = circled_count if circled_count > 0 else len(_CLASS_LABEL_RE.findall(content))
        if base.startswith("structure"):
            rep.ok("label-count", f, "structure 설문은 면제")
        elif label_count < 5:
            rep.fail("label-count", f, f"원문자/class=label 라벨이 {label_count}개 (5개 이상 필요)")
        else:
            rep.ok("label-count", f)

        # 취향 시안: data-axis 1~2개, 각 섹션에 A/B/C
        if base.startswith("taste"):
            axis_blocks = _split_data_axis_sections(content)
            if not (1 <= len(axis_blocks) <= 2):
                rep.fail("axis-section-count", f, f"data-axis 섹션이 {len(axis_blocks)}개 (1~2개 필요)")
            else:
                rep.ok("axis-section-count", f)
            mark = len(rep.results)
            for axis_name, block in axis_blocks:
                variants = set(_DATA_V_RE.findall(block))
                missing = sorted({"A", "B", "C"} - variants)
                if missing:
                    rep.fail(
                        "axis-variants-abc",
                        f,
                        f"data-axis={axis_name!r} 섹션에 data-v {missing} 누락",
                    )
            rep.ok_if_no_fail_since(mark, "axis-variants-valid", f)

        # 390 프레임 폭 (structure 설문 탭은 폰 프레임이 없으므로 면제)
        if base.startswith("structure"):
            rep.ok("frame-390", f, "structure 설문은 면제")
        elif not _FRAME_390_RE.search(content):
            rep.fail("frame-390", f, "width:390px 또는 --frame-w:390px 를 찾을 수 없음")
        else:
            rep.ok("frame-390", f)

        # lorem ipsum 금지
        if re.search(r"lorem ipsum", content, re.I):
            rep.fail("no-lorem-ipsum", f, "lorem ipsum 문자열이 있음")
        else:
            rep.ok("no-lorem-ipsum", f)

        # 의견 패널: 페이지 안에서 고르고 저장 (Artifact db)
        if not _DB_USE_RE.search(content):
            rep.fail("feedback-db", f, 'claude.use("db") 호출이 없음 — 의견 패널 저장 코드 필요')
        elif "feedback/" not in content:
            rep.fail("feedback-path", f, "db 문서 경로 'feedback/…'가 없음")
        else:
            rep.ok("feedback-panel", f)

        # 투어형 플로우: 장면 구조 + 상태 세그먼트
        if base.startswith("flow"):
            if "장면" not in content:
                rep.fail("flow-scenes", f, "'장면' 구조가 없음 (투어형이어야 함: 한 번에 화면 하나)")
            else:
                rep.ok("flow-scenes", f)
            missing_states = [s for s in _FLOW_STATES if s not in content]
            if missing_states:
                rep.fail("flow-states", f, f"상태 세그먼트 누락: {missing_states}")
            else:
                rep.ok("flow-states", f)

    return rep.results


# ---------------------------------------------------------------------------
# 실행/출력
# ---------------------------------------------------------------------------

PHASE_FUNCS = {
    "structure": check_structure,
    "flow": check_flow,
    "taste": check_taste,
    "rules": check_rules,
    "probes": check_probes,
}

PHASE_HINT_FILE = {
    "structure": "brief.md",
    "flow": "brief.md / probes/flow-*.html",
    "taste": "decisions.md",
    "rules": "design-rules.md",
    "probes": "probes/*.html",
}

PHASE_ORDER = ["structure", "flow", "taste", "rules", "probes"]


def run_all(design_dir: str) -> Tuple[List[Result], List[str]]:
    all_results: List[Result] = []
    skipped: List[str] = []
    for name in PHASE_ORDER:
        res = PHASE_FUNCS[name](design_dir)
        if res is None:
            skipped.append(name)
            continue
        all_results.extend(res)
    return all_results, skipped


def print_text(phase: str, results: List[Result], skipped: List[str]) -> None:
    for name in skipped:
        print(f"[SKIP] {name} — 파일 없음: {PHASE_HINT_FILE[name]}")
    for r in results:
        if r.ok:
            print(f"[OK] {r.name}")
        else:
            print(f"[FAIL] {r.file} {r.name} {r.detail}")


def main() -> int:
    parser = argparse.ArgumentParser(description="디자인 하네스 단계별 산출물 검사")
    parser.add_argument(
        "--phase",
        required=True,
        choices=["structure", "flow", "taste", "rules", "probes", "all"],
    )
    parser.add_argument("--design-dir", default="design")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    design_dir = args.design_dir

    if args.phase == "all":
        results, skipped = run_all(design_dir)
        passed = bool(results) and all(r.ok for r in results)
        if args.json:
            print(
                json.dumps(
                    {
                        "phase": "all",
                        "passed": passed,
                        "results": [r.to_dict() for r in results],
                        "skipped": skipped,
                    },
                    ensure_ascii=False,
                    indent=2,
                )
            )
        else:
            print_text("all", results, skipped)
        if not results:
            return 2
        return 0 if passed else 1

    func = PHASE_FUNCS[args.phase]
    results = func(design_dir)
    if results is None:
        hint = PHASE_HINT_FILE[args.phase]
        if args.json:
            print(
                json.dumps(
                    {
                        "phase": args.phase,
                        "passed": False,
                        "results": [],
                        "error": f"필요한 파일이 없음: {hint}",
                    },
                    ensure_ascii=False,
                    indent=2,
                )
            )
        else:
            print(f"[SKIP] {args.phase} — 파일 없음: {hint}")
        return 2

    passed = all(r.ok for r in results)
    if args.json:
        print(
            json.dumps(
                {"phase": args.phase, "passed": passed, "results": [r.to_dict() for r in results]},
                ensure_ascii=False,
                indent=2,
            )
        )
    else:
        print_text(args.phase, results, [])
    return 0 if passed else 1


if __name__ == "__main__":
    sys.exit(main())
