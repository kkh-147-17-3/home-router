package monitor

import "fmt"

var wellKnownPorts = map[int]string{
	20:    "FTP-Data",
	21:    "FTP",
	22:    "SSH",
	23:    "Telnet",
	25:    "SMTP",
	53:    "DNS",
	80:    "HTTP",
	110:   "POP3",
	119:   "NNTP",
	123:   "NTP",
	135:   "MS-RPC",
	137:   "NetBIOS-NS",
	138:   "NetBIOS-DGM",
	139:   "NetBIOS-SSN",
	143:   "IMAP",
	161:   "SNMP",
	162:   "SNMP-Trap",
	389:   "LDAP",
	443:   "HTTPS",
	445:   "SMB",
	465:   "SMTPS",
	514:   "Syslog",
	587:   "SMTP-Submit",
	636:   "LDAPS",
	993:   "IMAPS",
	995:   "POP3S",
	1433:  "MSSQL",
	1434:  "MSSQL-Browser",
	1521:  "Oracle",
	1723:  "PPTP",
	1883:  "MQTT",
	2222:  "SSH-Alt",
	3306:  "MySQL",
	3389:  "RDP",
	5060:  "SIP",
	5432:  "PostgreSQL",
	5900:  "VNC",
	6379:  "Redis",
	6667:  "IRC",
	8080:  "HTTP-Alt",
	8443:  "HTTPS-Alt",
	8888:  "HTTP-Alt",
	9090:  "WebUI",
	9200:  "Elasticsearch",
	27017: "MongoDB",
}

func wellKnownPort(port int, protocol string) string {
	if name, ok := wellKnownPorts[port]; ok {
		return name
	}
	if port > 0 && port < 1024 {
		return fmt.Sprintf("System/%d", port)
	}
	return ""
}
