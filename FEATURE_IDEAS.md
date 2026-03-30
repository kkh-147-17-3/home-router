# Home Router 추가 기능 제안

Go + React 기반 홈 라우터 프로젝트에 추가할 만한 기능을 조사한 결과.
OpenWrt, pfSense, Pi-hole, AdGuard Home 등을 참고하여 실용적인 기능을 선별함.

---

## 우선순위 High — 바로 구현할 만한 기능

### 1. 블록리스트 자동 갱신 (Small)
- 현재: 수동 reload 또는 재시작 시에만 갱신
- 추가: `time.Ticker` 기반 주기적 자동 갱신 (기본 24시간)
- `Config.Dns`에 `blocklist_refresh_interval` 추가
- `BlockerStats`에 마지막/다음 갱신 시각 노출
- **~30줄 수준의 작은 변경**

### 2. Custom DNS 레코드 (Local DNS Rewrite) (Small)
- `nas.home → 192.168.1.9` 같은 로컬 DNS 레코드 정의
- `dns/server.go` `handleQuery()`에서 블록리스트 체크 전에 커스텀 레코드 먼저 매칭
- API: `GET/POST/DELETE /api/dns/rewrites`
- 기존 whitelist 패턴과 동일한 구조로 구현 가능

### 3. 디바이스 온라인/오프라인 상태 (Small)
- 주기적 ARP 프로브로 DHCP 클라이언트 접속 상태 확인
- 기존 `mdlayher/arp` 의존성 재활용
- `LeaseInfo`에 `online`, `lastSeen` 필드 추가
- DHCP 페이지에 녹색/회색 표시

### 4. Wake-on-LAN (Small)
- Web UI에서 매직 패킷 전송으로 디바이스 원격 부팅
- 순수 Go 구현 (~50줄): 0xFF×6 + MAC×16을 UDP 브로드캐스트
- API: `POST /api/tools/wol {mac}`
- DHCP 테이블에 "Wake" 버튼 추가

### 5. 설정 백업/복원 (Small)
- `GET /api/system/backup` → `config.yml` + `leases.json`을 tar.gz로 다운로드
- `POST /api/system/restore` → 업로드된 tar.gz로 복원
- Go `archive/tar` + `compress/gzip` 사용
- System 페이지에 버튼 추가

### 6. 시스템 리소스 모니터링 (Small)
- CPU, 메모리, 온도를 `/proc/stat`, `/proc/meminfo`, `/sys/class/thermal/` 에서 읽기
- API: `GET /api/system/resources`
- 대시보드에 게이지/카드 위젯 추가

### 7. 대역폭 사용 히스토리 (Medium)
- 현재 conntrack은 실시간 스냅샷만 제공, 연결 종료 시 데이터 소실
- 5분 주기로 호스트별 트래픽 스냅샷 저장 (bbolt 또는 SQLite)
- API: `GET /api/monitor/traffic/history?period=24h&host=...`
- Security 페이지에 시계열 차트 추가

---

## 우선순위 Medium — 핵심 인프라 개선

### 8. DNS-over-HTTPS (DoH) / DNS-over-TLS (DoT) (Medium)
- ISP의 DNS 스누핑 방지
- `miekg/dns`가 TLS 이미 지원: `dns.Client{Net: "tcp-tls"}`
- DoH는 `net/http`로 `application/dns-message` POST
- `Config.Dns`에 `upstream_mode: plain|dot|doh` 추가

### 9. MAC 벤더 조회 (OUI 데이터베이스) (Small)
- MAC 앞 3옥텟으로 제조사 식별 (예: `90:09:d0` → Synology)
- IEEE OUI 데이터 임베드 (~1MB) 또는 `klauspost/oui` 패키지
- DHCP, 트래픽 응답에 `vendor` 필드 추가

### 10. 디바이스 별명 시스템 (Small)
- MAC 기반 사용자 정의 이름 (DHCP hostname 대체)
- `DeviceResolver`: 별명 > 고정임대명 > DHCP hostname > MAC 벤더
- 모든 API 응답에 적용

### 11. WAN 지연시간 모니터링 (Medium)
- 외부 타겟(1.1.1.1)에 주기적 ICMP ping → 지연/손실 기록
- `golang.org/x/net/icmp` 사용
- 대시보드에 현재 지연시간 + 스파크라인 차트

### 12. 네트워크 진단 도구 (Small)
- Web UI에서 ping, traceroute, nslookup 실행
- API: `POST /api/tools/{ping,traceroute,nslookup}`
- SSE로 실시간 결과 스트리밍
- **입력값 검증 필수** (커맨드 인젝션 방지)

### 13. 인터넷 접속 스케줄링 (Medium)
- 디바이스별 시간대 기반 인터넷 차단 (자녀 보호)
- `iptables -m mac --mac-source <mac> -j DROP` 규칙 동적 추가/제거
- 매 분 스케줄러 고루틴 평가
- 캘린더/타임피커 UI

### 14. HTTPS Web UI (Medium)
- 현재 HTTP로 비밀번호 전송 → 보안 취약
- 자체 서명: `crypto/tls` + `crypto/x509` 자동 생성
- Let's Encrypt: `golang.org/x/crypto/acme/autocert`
- `Config.Web`에 `tls` 설정 추가

### 15. 서비스 재시작 (Small)
- Web UI에서 DNS/DHCP/모니터 또는 전체 프로세스 재시작
- 각 서비스에 `Restart()` 메서드 추가
- API: `POST /api/system/restart/{service}`

---

## 우선순위 Low — 대규모 신규 서브시스템

### 16. WAN 방화벽 규칙 관리 (Large)
- 포트포워딩 외에 명시적 인바운드 허용/거부 규칙
- iptables HOME-ROUTER 체인에 규칙 삽입
- 순서(우선순위) 지원 필요

### 17. GeoIP 국가 차단 (Medium)
- MaxMind GeoLite2 오프라인 DB로 국가별 차단
- ipset + iptables 연동
- 현재 ip-api.com 속도 제한 문제도 해결

### 18. 디바이스별 QoS (Large)
- Linux `tc` (HTB qdisc)로 디바이스별 대역폭 제한
- 설정/관리 복잡도 높음

### 19. Per-Client DNS 필터링 프로필 (Medium)
- 디바이스별 다른 블록리스트/화이트리스트 정책
- AdGuard Home 스타일 클라이언트 설정

### 20. 포트 스캔 탐지 (Medium)
- 슬라이딩 윈도우로 스캔 패턴 감지 → 자동 차단
- 기존 access log 데이터 활용

---

## 권장 구현 순서

| Phase | 기능 | 난이도 |
|-------|------|--------|
| **Phase 1** | 블록리스트 자동갱신, WoL, Custom DNS, 온라인 상태, 백업/복원, 시스템 리소스 | Small ×6 |
| **Phase 2** | DoH/DoT, 대역폭 히스토리, MAC 벤더, 디바이스 별명 | Small~Medium ×4 |
| **Phase 3** | WAN 지연 모니터링, 진단 도구, HTTPS, Per-Client DNS, 접속 스케줄링, 서비스 재시작 | Small~Medium ×6 |
| **Phase 4** | 방화벽 규칙, GeoIP 차단, QoS, 포트스캔 탐지 | Medium~Large ×4 |

## 주요 수정 파일
- `dns/server.go` — DNS 기능 확장 (DoH/DoT, 커스텀 레코드, 프로필)
- `internal/config/config.go` — 모든 신규 설정 필드
- `api/server.go` — 신규 API 엔드포인트 등록
- `monitor/traffic.go` — 대역폭 히스토리 확장
- `main.go` — 신규 서브시스템 초기화
