# Push

근거 수집부터 문서 작성과 지원 추적까지 관리하는 macOS 우선 데스크톱 작업공간.

| 경로 | 역할 |
| --- | --- |
| [push-fe](https://github.com/push-dot/push-fe) | Tauri v2 · React · ky · Zustand · FSD · TipTap · SQLite |
| [push-be](https://github.com/push-dot/push-be) | Go · Echo · PostgreSQL API |
| [docs/plan.md](docs/plan.md) | 제품 계획 및 확정 UI 결정 |
| [docs/api.md](docs/api.md) | 요청·응답·상태·승인·오류 계약 |
| [docs/delivery-status.md](docs/delivery-status.md) | 구현과 실제 검증 상태 |
| [docs/release.md](docs/release.md) | VPS 배포·서명·업데이트·베타 절차 |

## 체크아웃

```sh
git clone --recurse-submodules https://github.com/push-dot/push-workspace.git
cd push-workspace
git switch develop
git submodule update --init --recursive
```

모든 저장소는 비공개다. 세 저장소 모두 읽기 권한이 필요하다. 서브모듈은 특정 커밋으로 고정되며 앱과 API 변경 시 검증한 두 커밋을 함께 갱신한다.

## 개발

Node.js 22+, Go, PostgreSQL, Rust stable, macOS Xcode를 준비한다. 구체적인 실행과 테스트 명령은 각 서브모듈 README에 있다. API 기본 포트는 8080이다. 서버 개발 인증은 `APP_ENV=development`와 직접 설정한 `DEV_AUTH_TOKEN`으로만 활성화된다.

프런트엔드는 확정된 [v6 프로토타입](docs/no-right-sidebar-prototype-v6.png)을 따른다. 왼쪽은 기능·채팅·프로젝트, 문서 기능명은 `내 서류`이며 오른쪽 컨텍스트 패널은 없다.

운영 서버는 `.env.example`과 서버 환경변수 예시를 사용해 `.env`를 설정한 뒤 실행한다.

```sh
docker compose config --quiet
docker compose up --build -d
```

외부 OAuth·결제·Google·AI·서명·베타 설정은 실제 계정 준비 후 검증한다. 아직 검증하지 않은 기능을 출시 완료로 간주하지 않는다.

## Git 작업 규칙

`develop`에서 `feat|fix|refactor|chore|docs|test/<short-kebab-summary>` 브랜치를 만들고 작업 단위로 scoped Conventional Commit을 남긴다. PR은 `develop`을 대상으로 하며 검사 통과 후 squash merge한다. workspace PR은 하위 저장소의 검증된 커밋과 관련 문서 변경을 포함한다.
