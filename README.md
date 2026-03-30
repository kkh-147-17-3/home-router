# home-router

Go로 작성한 리눅스 기반 소프트웨어 라우터. 일반 PC에 NIC 2개(WAN/LAN)를 꽂아 가정용 공유기로 사용한다.

## 주요 기능

- **NAT/포트포워딩** — iptables 기반 MASQUERADE, DNAT 포트포워딩
- **DHCP 서버** — IP 풀 관리, 고정 임대, ARP 프로브 충돌 방지, 임대 영속화
- **DHCP 클라이언트** — WAN 인터페이스 IP 자동 획득 및 갱신
- **DNS 서버** — 업스트림 포워딩, 캐시, 광고 차단(blocklist)
- **DDNS** — Cloudflare 등 외부 DNS 레코드 자동 갱신
- **네트워크 모니터링** — conntrack 기반 트래픽 로깅, GeoIP 조직 정보
- **Web UI** — React + Tailwind 관리 대시보드 (REST API)

## 요구 사항

- Linux (netlink, iptables 사용)
- Go 1.26+
- Node.js 18+ (Web UI 빌드)
- root 권한

## 빌드 및 실행

```bash
# 전체 빌드 (Web UI + Go 바이너리)
make build

# 실행
sudo ./home-router

# systemd 배포
make deploy
```

## 설정

`config.yml`에서 WAN/LAN MAC 주소, DHCP 범위, DNS 업스트림, 포트포워딩 등을 설정한다.

```yaml
network:
  wan:
    mac_address: "xx:xx:xx:xx:xx:xx"
  lan:
    mac_address: "xx:xx:xx:xx:xx:xx"
    subnet: "192.168.1.1/24"

dhcp:
  server:
    range_start: "192.168.1.3"
    range_end: "192.168.1.254"

dns:
  enabled: true
  upstream: ["1.1.1.1", "8.8.8.8"]
  blocklists:
    - "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts"
```

## 개발

```bash
# Web UI 개발 서버
make dev-web

# Go API 개발 실행
make dev-api
```

## 프로젝트 구조

```
main.go        — 엔트리포인트, 서비스 부트스트랩
config.yml     — 라우터 설정 파일
nat/           — NAT, 포트포워딩 (iptables)
dhcp/          — DHCP 서버/클라이언트
dns/           — DNS 포워더, 캐시, 광고 차단
ddns/          — DDNS 클라이언트
monitor/       — 트래픽 모니터링, GeoIP
network/       — 인터페이스 IP/라우트 설정
api/           — REST API 서버
web/           — React 프론트엔드 (Vite + Tailwind)
internal/      — 내부 패키지 (config, iface)
```
