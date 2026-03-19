# 홈라우터 버그 수정 (2026-03-19)

서비스 로그 분석을 통해 발견된 버그 및 성능 문제를 수정했습니다.

---

## 1. DHCP 임대 만료시간 미갱신 (심각)

**파일:** `dhcp/pool.go` (`handleClientRequest`)

**문제:** 클라이언트가 REQUEST로 임대를 갱신할 때, 서버가 ACK를 보내면서도 내부 임대 만료시간(`ExpiredAt`)을 업데이트하지 않았습니다. 최초 할당 시점의 만료시간이 그대로 유지되어, 장시간 연결된 클라이언트의 임대가 실제로는 만료 상태임에도 계속 사용되는 상황이 발생했습니다.

**증상:**
- Mac 장치의 임대 만료시간이 항상 `2026-03-20T09:13:47Z`로 고정
- Mac이 약 15분마다 REQUEST를 3회 연속 전송 (만료 임박한 임대를 받아 재시도)

**수정:** 기존 임대 반환 시 `ExpiredAt`을 현재시간 + LeaseTime으로 갱신하고, 변경된 임대를 파일에 저장하도록 수정했습니다.

```go
// 수정 전
lease, ok := p.Leases[mac]
if ok {
    return lease.Address  // 만료시간 갱신 없이 반환
}

// 수정 후
lease, ok := p.Leases[mac]
if ok {
    lease.ExpiredAt = time.Now().Add(time.Duration(cfg.Dhcp.Server.LeaseTime) * time.Second)
    p.Leases[mac] = lease
    p.saveLeases()
    return lease.Address
}
```

---

## 2. DHCP RELEASE 메시지 미처리

**파일:** `dhcp/server.go`, `dhcp/pool.go`

**문제:** 클라이언트가 RELEASE 메시지를 보내면 서버가 `default` 분기로 빠져 "무시하는 메시지 타입: RELEASE"를 출력하고 아무 처리도 하지 않았습니다. IP가 즉시 풀로 반환되지 않아, 풀 고갈 위험이 있었습니다.

**증상:**
- Samsung-Washer가 RELEASE 후 재접속 시 매번 DISCOVER부터 시작
- 해제된 IP가 풀에 남아 불필요하게 점유

**수정:**
- `pool.go`에 `handleRelease` 메서드 추가 (임대 삭제 + IP-MAC 매핑 해제 + 파일 저장)
- `server.go`의 switch문에 `MessageTypeRelease` 케이스 추가

---

## 3. WAN DHCP Client 유니캐스트 갱신 실패

**파일:** `dhcp/client.go`

**문제:** WAN IP 갱신 시 이전 ACK의 `ServerIdentifier`로 유니캐스트 REQUEST를 보냈으나, ISP가 DHCP Relay Agent 구조를 사용하여 중앙 DHCP 서버에 직접 도달할 수 없었습니다. 매시간 유니캐스트 실패 → 브로드캐스트 폴백이 반복되었습니다.

**증상:**
- 매시간 `갱신 실패: no matching response packet received, 전체 요청으로 전환` 로그 출력
- 폴백으로 정상 동작하지만 불필요한 지연과 에러 로그 발생

**수정:** 유니캐스트 갱신(`doRenew`) 로직을 제거하고, 갱신 시에도 브로드캐스트 방식(`doDHCP`)을 사용하도록 단순화했습니다. 미사용 코드(`doRenew`, `prevLease`, 관련 import)도 함께 제거했습니다.

---

## 4. DHCP IP 충돌 — ARP 프로브 미수행 (심각)

**파일:** `dhcp/pool.go` (`handleClientRequest`, `isIPInUse`, `newARPClient`)

**문제:** DHCP 풀이 새 IP를 할당할 때 자체 임대 맵(`IPToMAC`)만 확인하고, 네트워크에서 실제로 해당 IP를 사용 중인 기기가 있는지 확인하지 않았습니다. 이전 세션의 잔여 IP를 가진 기기가 있으면 충돌이 발생했습니다.

**증상:**
- `SM-L705N`(2a:4f:49:e0:57:49)이 DHCP 임대 IP(.7)와 이전 세션의 잔여 IP(.11)를 동시 보유
- `kkh` 핸드폰(28:d0:43:ad:c0:64)이 .11을 할당받은 후 ARP 충돌 감지 → DECLINE → 재할당까지 ~10초 네트워크 불통
- `5c:86:c1:2e:ec:dd` 기기도 .8에서 동일한 충돌 발생

**ARP 테이블 증거:**
```
192.168.1.7  → 2a:4f:49:e0:57:49 (SM-L705N)  ← DHCP 임대
192.168.1.11 → 2a:4f:49:e0:57:49 (SM-L705N)  ← 이전 세션 잔여 IP (DHCP에 없음)
```

**수정:** IP 할당 전 ARP 프로브(RFC 5227)를 수행하여 네트워크에서 실제 사용 여부를 확인합니다. 응답이 오면 해당 IP를 건너뛰고 다음 IP를 시도합니다.

- `newARPClient()`: LAN 인터페이스에 ARP 클라이언트 생성
- `isIPInUse()`: 500ms 타임아웃으로 ARP Resolve 수행
- 할당 루프에서 `isIPInUse()` 통과한 IP만 할당
- ARP 클라이언트 생성 실패 시 기존 방식으로 폴백 (fail-open)

```go
arpClient := p.newARPClient()
if arpClient != nil {
    defer arpClient.Close()
}
for curr := p.RangeStart; ...; curr = GetNextIP(curr) {
    // 내부 맵 확인
    if _, ok := p.IPToMAC[curr.String()]; ok { continue }
    if p.DeclinedIPs[curr.String()] { continue }
    // ARP 프로브로 실제 네트워크 확인
    if arpClient != nil && p.isIPInUse(arpClient, curr) { continue }
    // 할당 진행
}
```

**의존성 추가:** `github.com/mdlayher/arp`

---

## 5. DNS 캐시 lock 비효율

**파일:** `dns/cache.go` (`Get`, `Stats`)

**문제:** `Cache.Get()` 호출마다 RWMutex를 2회 획득했습니다 (RLock 1회 + WLock 1회). `hits`/`misses` 카운터를 1 증가시키기 위해 매번 WLock을 잡아, 동시 DNS 쿼리 시 캐시 읽기까지 블로킹되었습니다.

```
// 수정 전: 모든 경로에서 lock 2회
RLock → 캐시 조회 → RUnlock → Lock → 카운터++ → Unlock

// 수정 후: 캐시 히트 시 lock 1회
RLock → 캐시 조회 → RUnlock → atomic 카운터++ (lock 없음)
```

**수정:** `hits`/`misses`를 `sync/atomic.Uint64`로 변경하여 WLock 없이 카운터를 증가시킵니다. `Stats()`도 `atomic.Load()`로 읽도록 변경했습니다.

---

## 6. config.yml DNS 설정 불일치

**파일:** `config.yml`

**문제:** DHCP가 클라이언트에게 전달하는 DNS가 `1.1.1.1`(외부)로 설정되어 있었습니다. 실제로는 `main.go`에서 DNS 서버 활성화 시 gateway(`192.168.1.1`)로 override하고 있어 런타임에는 문제없었으나, 설정 파일만 보면 로컬 DNS 서버를 우회하는 것처럼 보이는 불일치가 있었습니다.

**수정:** `dns: "1.1.1.1"` → `dns: "192.168.1.1"` (설정 파일과 런타임 동작 일치)
