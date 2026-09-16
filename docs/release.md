# 배포와 비공개 베타

## 저장소

```sh
git clone https://github.com/push-dot/push-workspace.git
cd push-workspace
git switch develop
```

저장소는 비공개다. 앱과 API 코드는 `push-fe`, `push-be`에 모노레포로 함께 들어 있다.

## Hostinger VPS

Docker Engine과 Compose, 도메인 A/AAAA 레코드, 80/443 인바운드 포트를 준비한다. PostgreSQL은 호스트 포트를 열지 않는다.

1. 저장소를 배포 서버에 복제하고 릴리스 커밋으로 체크아웃한다.
2. `.env.example`을 `.env`로 복사하고 `chmod 600 .env`를 적용한다.
3. `POSTGRES_PASSWORD`는 URL 예약 문자가 없는 충분히 긴 무작위 값으로 설정한다. `AES_KEY`는 32바이트 키의 base64 값이다. `push-be/.env.example`의 OAuth·AI·결제 설정을 추가한다.
4. `docker compose config --quiet`로 구성을 검증하고 `docker compose up --build -d`를 실행한다.
5. 서버 README의 health endpoint와 인증 거부, OAuth 왕복, 실제 DB 보존을 확인한 후 사용자 접근을 연다.
6. 배포 전 DB 백업을 만들고 복원 테스트를 별도 DB에서 실행한다. 복구 시 workspace를 같은 릴리스 커밋으로 되돌린다. 하위 호환이 아닌 스키마 변경은 DB 복구 계획까지 있어야 한다.

Caddy는 도메인 인증서를 발급하고 TLS를 종료한다. API 로그에는 요청 본문·Authorization·메일·이력서·키를 남기지 않는다. 배포·서버 측 `.env`와 DB 백업은 git에 포함하지 않는다.

## 데스크톱 서명과 업데이트

Tauri v2의 [macOS 서명 안내](https://v2.tauri.app/distribute/sign/macos/)와 [업데이터 안내](https://v2.tauri.app/plugin/updater/)를 따른다. Rust와 Xcode가 필요하며 Universal 빌드는 `aarch64-apple-darwin`, `x86_64-apple-darwin` 두 타깃을 사용한다.

Apple Developer ID 인증서와 공증 자격, Tauri 서명 키를 CI secrets에 설정한다. 업데이터 공개 키와 HTTPS 업데이트 endpoint를 릴리스 설정에 넣는다. 저장소가 비공개이므로 사용자의 앱에 GitHub PAT를 넣지 않는다. 서명된 번들과 업데이트 JSON을 별도의 접근 정책이 있는 HTTPS 다운로드 호스트에서 제공한다.

실제 인증서가 없는 빌드는 로컬 테스트용이다. 서명·공증 성공, 다운로드 설치, 기존 버전에서의 업데이트 및 서명 위변조 거부를 확인하기 전에는 배포 완료로 기록하지 않는다. Windows 설치·업데이트는 Windows 실행 환경에서 별도로 검증한다.

## 20명 베타

참가자 연락·초대는 담당자가 진행한다. 동의한 사용자에게만 빌드를 제공하고 실제 개인정보는 테스트 픽스처로 수집하지 않는다.

기록은 익명 참가자 ID, 플랫폼, 빌드 커밋, 공고 등록 여부, 근거 연결 여부, 문서 생성/출력 여부, 지원 추적 완료 여부, 실패 단계만 포함한다. 20명 중 16명 이상이 실제 공고 1건에서 전체 흐름을 완료해야 계획의 베타 기준을 충족한다.

현재 참가자 0명, 성공 기록 0건이며 미완료다. 테스트 자동화 통과를 베타 성공률로 대체하지 않는다.
