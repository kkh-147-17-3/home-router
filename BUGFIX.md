# DHCP 서버 버그 수정 기록 (2026-03-19)

## 1. DHCP ACK 미응답

**증상**: 클라이언트가 DISCOVER → OFFER 후 REQUEST를 보내도 IP를 사용하지 않음

**원인**: 서버가 메시지 타입에 관계없이 항상 `MessageTypeOffer`만 응답. REQUEST에 대해 ACK를 보내지 않아 클라이언트가 IP를 설정하지 못함.

**수정**: `server.go`에서 메시지 타입별 분기 처리

```go
case dhcpv4.MessageTypeDiscover:
    replyType = dhcpv4.MessageTypeOffer
case dhcpv4.MessageTypeRequest:
    replyType = dhcpv4.MessageTypeAck
```

## 2. Gateway 옵션 오류

**증상**: IP를 받아도 인터넷 연결 불가

**원인**: `WithGatewayIP()`는 DHCP relay agent용 `giaddr` 필드를 설정함. 클라이언트의 기본 게이트웨이는 Router 옵션(Option 3)으로 전달해야 함.

**수정**: `server.go`에서 `WithGatewayIP()` → `WithRouter()` 변경, `WithServerIP()` 및 `OptServerIdentifier()` 추가

```go
dhcpv4.WithServerIP(serverIP),
dhcpv4.WithOption(dhcpv4.OptServerIdentifier(serverIP)),
dhcpv4.WithRouter(gatewayIP),
```

## 3. net.IP 슬라이스 mutation

**증상**: 두 번째 이후 DHCP 요청에서 IP 풀이 소진됨

**원인**: `GetNextIP()`가 `ip.To4()`로 원본 슬라이스를 직접 수정. Go의 슬라이스는 참조 타입이라 `Pool.RangeStart`와 기존 lease의 IP가 전부 오염됨.

**수정**: `pool.go`에서 `GetNextIP()`와 lease 저장 시 `make` + `copy`로 복사본 생성

```go
func GetNextIP(ip net.IP) net.IP {
    src := ip.To4()
    next := make(net.IP, 4)
    copy(next, src)
    // ...
}
```

## 4. net.IP 16바이트/4바이트 불일치

**증상**: 첫 번째 IP 할당 후 풀 소진

**원인**: `net.ParseIP()`는 16바이트 IP를 반환하지만 `GetNextIP()`는 `To4()`로 4바이트 IP를 반환. `bytes.Compare()`로 4바이트와 16바이트를 비교하면 루프가 즉시 종료됨.

**수정**: `pool.go`의 `NewPool()`에서 4바이트로 정규화

```go
func NewPool(rangeStart, rangeEnd net.IP) *Pool {
    return &Pool{
        RangeStart: rangeStart.To4(),
        RangeEnd:   rangeEnd.To4(),
        // ...
    }
}
```

## 5. DHCP DECLINE 미처리

**증상**: 클라이언트가 IP 충돌을 감지해 DECLINE을 보내도 같은 IP를 계속 재할당 → 무한루프

**원인**: DECLINE 메시지를 무시하고 기존 lease를 유지

**수정**: `pool.go`에 `DeclinedIPs` 맵과 `handleDecline()` 추가. `server.go`에서 DECLINE 메시지 처리

```go
case dhcpv4.MessageTypeDecline:
    pool.handleDecline(macAddress.String())
    return
```

## 6. FORWARD 체인 DROP (Docker 간섭)

**증상**: LAN 클라이언트가 인터넷에 접속 불가

**원인**: Docker가 iptables FORWARD 체인 기본 정책을 DROP으로 설정. LAN↔WAN 포워딩 룰이 없어서 모든 트래픽 차단.

**수정**: `nat/forward.go`의 `Enable()`에 FORWARD 룰 추가

```go
iptables -I FORWARD -i <wan> -m state --state RELATED,ESTABLISHED -j ACCEPT
iptables -I FORWARD -o <wan> -j ACCEPT
```

## 7. MASQUERADE 룰 중복

**증상**: 라우터 재시작마다 동일한 MASQUERADE 룰이 누적

**원인**: `nat.Enable()`이 기존 룰 확인 없이 `-A`(append)만 수행

**수정**: `nat/forward.go`에서 `-D`(delete)로 기존 룰 제거 후 `-A` 실행

## 8. loopback에 LAN IP 잔류 → ARP 오동작

**증상**: 세탁기(5c:86:c1:2e:ec:dd)가 모든 IP를 DECLINE

**원인**: `lo` 인터페이스에 `192.168.1.1/24`가 남아있어 커널이 192.168.1.0/24 전체를 로컬 주소로 인식. 세탁기가 ARP probe를 보내면 라우터 커널이 "그 IP는 내 거"라고 응답.

```
# 문제 확인
$ ip route get 192.168.1.26
local 192.168.1.26 dev lo src 192.168.1.1   ← 모든 IP가 local!

$ ip route show table local | grep 192.168.1
local 192.168.1.0/24 dev lo   ← 이 엔트리가 원인

$ ip addr show lo | grep 192.168
inet 192.168.1.1/24 scope global lo   ← lo에 /24가 할당됨
```

**수정**: `network/subnet.go`의 `SetIP()`에서 대상 인터페이스 외의 다른 인터페이스에 같은 IP가 있으면 제거

```go
allLinks, _ := netlink.LinkList()
for _, l := range allLinks {
    if l.Attrs().Index == link.Attrs().Index {
        continue
    }
    addrs, _ := netlink.AddrList(l, netlink.FAMILY_V4)
    for _, a := range addrs {
        if a.IPNet.String() == ip.IPNet.String() {
            netlink.AddrDel(l, &a)
        }
    }
}
```

## 9. SSH 접속 끊김 (INPUT 체인 누락)

**증상**: 192.168.1.1로 SSH 접속하면 자꾸 끊김

**원인**: `nat.Enable()`이 FORWARD 체인 룰만 관리하고 INPUT 체인 룰을 설정하지 않음. Docker 등이 iptables 기본 정책을 변경하면 LAN에서 라우터 자체로의 트래픽(SSH, DNS 등)이 INPUT 체인에서 DROP될 수 있음.

```
# 문제 확인
$ iptables -L INPUT -v -n
Chain INPUT (policy DROP)   ← DROP이면 LAN→라우터 트래픽 차단
```

FORWARD 체인은 라우터를 *경유*하는 트래픽(LAN↔WAN)만 처리. 라우터 *자체*로 향하는 트래픽(SSH 등)은 INPUT 체인을 거치므로, INPUT에 LAN 인터페이스 허용 룰이 필요.

**수정**: `nat/forward.go`의 `Enable()`에 `lanIface` 파라미터 추가, INPUT 룰 설정

```go
func Enable(wanIface, lanIface string) error {
    // ... 기존 FORWARD/MASQUERADE 룰 ...

    // INPUT 룰 추가: LAN에서 라우터로의 트래픽 허용 (SSH 등)
    exec.Command("iptables", "-D", "INPUT", "-i", lanIface, "-j", "ACCEPT").Run()
    err = exec.Command("iptables", "-I", "INPUT", "-i", lanIface, "-j", "ACCEPT").Run()
    if err != nil {
        return err
    }
    return nil
}

func Disable(wanIface, lanIface string) error {
    // ... 기존 룰 제거 ...
    exec.Command("iptables", "-D", "INPUT", "-i", lanIface, "-j", "ACCEPT").Run()
    return nil
}
```

`main.go` 호출부도 LAN 인터페이스 이름 전달하도록 변경:

```go
nat.Enable(wanIface.Attrs().Name, lanIface.Attrs().Name)
nat.Disable(wanIface.Attrs().Name, lanIface.Attrs().Name)
```

## 10. WAN DHCP 갱신 시 불필요한 NAT 재구성 → 연결 끊김

**증상**: 외부에서 SSH 접속(port 22222) 시 약 1시간마다 연결이 끊김

**원인**: ISP DHCP 리스 타임이 ~2시간이라, 절반인 1시간마다 갱신 발생. IP가 변하지 않았는데도(`14.40.83.188/24`) 매번 `nat.Disable()` → `nat.Enable()` + 포트포워딩 전체 재설정을 수행. 이 과정에서 iptables 룰이 순간적으로 삭제되어 기존 TCP 연결(SSH 포함)이 끊어짐.

```
05:01:08  WAN IP 수신: 14.40.83.188/24   ← 최초 획득
06:01:08  WAN IP 수신: 14.40.83.188/24   ← 1시간 뒤 갱신 (IP 동일)
06:01:08  NAT 갱신 완료                   ← 불필요한 NAT 재구성
```

**수정**: `main.go`에서 이전 WAN IP를 기억하고, IP가 실제로 변경된 경우에만 NAT를 재설정

```go
var currentWanIP string
for lease := range client {
    cidr := fmt.Sprintf("%s/%d", assignedIP, prefixLength)
    if cidr == currentWanIP {
        log.Printf("WAN IP 갱신 완료 (변경 없음: %s)", cidr)
        continue
    }
    // IP 변경 시에만 NAT 재설정
    nat.Disable(...)
    nat.Enable(...)
    currentWanIP = cidr
}
```

## 11. DHCP 클라이언트 10회 실패 시 영구 종료

**증상**: WAN이 일시적으로 불안정하면 라우터 전체가 조용히 죽음

**원인**: `client.go`에서 DHCP 요청 실패 시 최대 10번만 재시도하고 채널을 닫음. 채널이 닫히면 `main.go`의 `for lease := range client` 루프도 종료되어 프로그램이 아무 로그 없이 종료.

```go
// 기존 코드
count := 1
maxCount := 10
if count < maxCount {
    count++
} else {
    close(ch)  // ← 10회 실패 시 영구 종료
    return
}
```

**수정**: `client.go`에서 재시도 횟수 제한 제거, 지수 백오프(3초→60초) 적용

```go
retryDelay := 3 * time.Second
maxDelay := 60 * time.Second
for {
    lease, err := doDHCP(ctx, iface)
    if err != nil {
        select {
        case <-time.After(retryDelay):
            retryDelay = retryDelay * 2
            if retryDelay > maxDelay {
                retryDelay = maxDelay
            }
        case <-ctx.Done():
            return
        }
        continue
    }
    retryDelay = 3 * time.Second  // 성공 시 초기화
    // ...
}
```

## 12. DHCP 갱신 시 매번 전체 DORA 수행

**증상**: WAN DHCP 갱신 시 불필요한 지연 및 일시적 IP 부재

**원인**: `client.go`의 `doDHCP()`가 매번 `client.Request()`로 전체 DORA(Discover→Offer→Request→Ack) 사이클을 수행. 기존 리스를 유지한 채 Renew만 하면 되는 상황에서 불필요하게 새 IP를 요청.

**수정**: `client.go`에 `doRenew()` 추가. 기존 리스가 있으면 `dhcpv4.NewRenewFromAck()`으로 유니캐스트 갱신을 시도하고, 실패 시 전체 DORA로 폴백

```go
if prevLease != nil {
    lease, err = doRenew(ctx, iface, prevLease)
    if err != nil {
        lease, err = doDHCP(ctx, iface)  // 폴백
    }
} else {
    lease, err = doDHCP(ctx, iface)
}
```

## 13. WAN 기본 라우트(default route) 미설정

**증상**: WAN IP를 받아도 라우터 자체의 인터넷이 불안정하거나, IP 갱신 후 default route가 사라짐

**원인**: `main.go`에서 WAN IP를 인터페이스에 설정하지만, DHCP ACK에 포함된 게이트웨이를 기본 라우트로 등록하는 코드가 없음. 기존에 NetworkManager 등이 설정한 라우트에 의존하지만, IP 갱신 시 깨질 수 있음.

**수정**: `network/subnet.go`에 `SetDefaultRoute()` 추가. `main.go`에서 WAN IP 설정 후 ACK의 Router 옵션(Option 3)에서 게이트웨이를 읽어 default route 설정

```go
routers := lease.ACK.Router()
if len(routers) > 0 {
    network.SetDefaultRoute(cfg.Network.Wan.MacAddress, routers[0])
}
```

## 14. DHCP 풀 범위가 고정 IP와 겹침 → DECLINE 연쇄

**증상**: 새 기기(MAC=28:d0:43:ad:c0:64, 호스트=kkh)가 IP를 3~4번 DECLINE한 뒤에야 사용 가능한 IP를 받음

**원인**: DHCP 풀이 `192.168.1.3`부터 시작하는데, 시놀로지 NAS가 `192.168.1.3`을 고정 IP로 사용 중. 새 기기에 `.3`이 할당되면 ARP probe에서 시놀로지가 응답 → 충돌 감지 → DECLINE. `.4`, `.5`도 다른 기기와 충돌하여 `.6`에서야 수락됨.

```
05:34:47  .3 할당 → 05:34:51 DECLINE
05:35:07  .4 할당 → 05:35:10 DECLINE
05:35:30  .5 할당 → 05:35:34 DECLINE
05:35:54  .6 할당 → (수락)
```

**수정**: `config.yml`에서 `range_start`를 고정 IP 대역 이후로 변경

```yaml
dhcp:
  server:
    range_start: "192.168.1.100"  # 기존: "192.168.1.3"
    range_end: "192.168.1.254"
```

## 15. Docker FORWARD 체인 간섭 (커스텀 체인으로 해결)

**증상**: 핸드폰 WiFi 연결 시 패킷 수신이 간헐적으로 지연되거나 끊김

**원인**: Docker가 `iptables -P FORWARD DROP`으로 기본 정책을 설정하고, 자체 룰을 지속적으로 관리함. 기존 코드가 `iptables -I FORWARD`로 룰을 삽입해도 Docker가 체인을 재관리할 때마다 home-router의 룰이 밀려나 LAN→WAN 트래픽이 간헐적으로 DROP됨.

**수정**: `nat/forward.go`에서 `HOME-ROUTER` 커스텀 iptables 체인을 생성하고, FORWARD 체인 맨 앞에서 점프하도록 변경. Docker가 FORWARD 체인을 조작해도 커스텀 체인 내부 룰은 영향받지 않음.

```go
const chainName = "HOME-ROUTER"

// Enable()
iptables -N HOME-ROUTER
iptables -F HOME-ROUTER
iptables -I FORWARD 1 -j HOME-ROUTER
iptables -A HOME-ROUTER -i lanIface -o wanIface -j ACCEPT
iptables -A HOME-ROUTER -i wanIface -o lanIface -m state --state RELATED,ESTABLISHED -j ACCEPT

// Disable()
iptables -D FORWARD -j HOME-ROUTER
iptables -F HOME-ROUTER
iptables -X HOME-ROUTER
```

`AddPortForward()`, `RemovePortForward()`도 FORWARD 체인 대신 `HOME-ROUTER` 체인에 룰을 추가/제거하도록 변경.

## 16. TCP MSS Clamping 누락 → 대형 패킷 드랍

**증상**: 웹 페이지 로딩이 중간에 멈추거나 타임아웃 발생

**원인**: NAT 환경에서 WAN 쪽 MTU가 1500보다 작을 경우(PPPoE 등), TCP SYN 패킷의 MSS 값이 실제 경로 MTU보다 크면 대형 패킷이 조용히 드랍됨. MSS clamping이 없어 path MTU discovery에만 의존.

**수정**: `nat/forward.go`의 `Enable()`에 mangle 테이블 MSS clamping 룰 추가, `Disable()`에 정리 로직 추가

```
iptables -t mangle -A FORWARD -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu
```

## 17. DNS 지연 (Google DNS → Cloudflare)

**증상**: 첫 요청 시 DNS 조회 지연으로 체감 속도 저하

**원인**: DHCP에서 `dns: "8.8.8.8"` (Google DNS)을 할당. 한국에서 Google DNS는 상대적으로 응답이 느림.

**수정**: `config.yml`에서 DNS를 Cloudflare `1.1.1.1`로 변경 (대안: KT DNS `168.126.63.1`)

```yaml
dhcp:
  server:
    dns: "1.1.1.1"  # 기존: "8.8.8.8"
```

## 18. wlp4s0 이중 default route → 간헐적 NAT 우회

**증상**: 간헐적으로 응답 패킷이 돌아오지 않아 연결 끊김

**원인**: 라우팅 테이블에 default route가 2개 존재

```
default via 14.40.83.254 dev enp2s0              ← WAN (metric 0, 최우선)
default via 10.206.185.89 dev wlp4s0 metric 600  ← WiFi 클라이언트 (백업)
```

`enp2s0` 경로가 잠깐이라도 불안정하면 트래픽이 `wlp4s0`으로 빠지는데, 이 인터페이스에는 MASQUERADE가 없어서 응답 패킷이 돌아오지 못함. 또한 기존 `SetDefaultRoute()`가 `RouteAdd`를 사용하여 경로가 이미 존재하면 `file exists` 에러로 실패 (로그: `기본 라우트 설정 실패: file exists`).

**수정**: `network/subnet.go`의 `SetDefaultRoute()`를 변경
1. `RouteAdd` → `RouteReplace` (EEXIST 에러 방지)
2. WAN default route 설정 후, WAN이 아닌 인터페이스의 default route를 자동 제거

```go
func SetDefaultRoute(mac string, gateway net.IP) error {
    // WAN default route 설정 (이미 있으면 교체)
    netlink.RouteReplace(&netlink.Route{
        LinkIndex: link.Attrs().Index,
        Gw:        gateway,
    })

    // 다른 인터페이스의 default route 제거 (wlp4s0 등 경쟁 방지)
    routes, _ := netlink.RouteList(nil, netlink.FAMILY_V4)
    for _, r := range routes {
        if r.Dst == nil && r.LinkIndex != link.Attrs().Index {
            netlink.RouteDel(&r)
        }
    }
    return nil
}
```

## 19. USB LAN 어댑터 장애 시 DHCP 서버 복구 불가

**증상**: USB 이더넷 어댑터(LAN)가 일시적으로 분리되면 DHCP 서버가 종료되고 복구되지 않음

**원인**: `main.go`에서 DHCP 서버를 단일 goroutine으로 실행. 서버가 에러로 종료되면 `log.Fatalf`로 프로세스 전체가 종료됨. USB 어댑터 재연결 후에도 자동 복구 불가.

**수정**: `main.go`에서 DHCP 서버를 재시작 루프로 감싸고, `internal/iface/finder.go`에 `WaitForInterface()` 추가. 인터페이스가 사라지면 MAC 주소로 재연결을 대기한 뒤 IP 재설정 및 서버 재시작.

```go
go func() {
    lanName := lanIface.Attrs().Name
    for {
        err := dhcp.RunServer(ctx, lanName, pool, cfg)
        if ctx.Err() != nil {
            return
        }
        // 인터페이스 복구 대기
        newLan, _ := iface.WaitForInterface(cfg.Network.Lan.MacAddress, ctx)
        lanName = newLan.Attrs().Name
        network.SetIP(cfg.Network.Lan.MacAddress, cfg.Network.Lan.Subnet)
    }
}()
```
