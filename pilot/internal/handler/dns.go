package handler

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

// DNSRecord represents a DNS check result.
type DNSRecord struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Expected string `json:"expected"`
	Current  string `json:"current"`
	OK       bool   `json:"ok"`
}

// APIDNSCheck handles POST /api/v1/dns/check.
func (h *Handler) APIDNSCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	domain := strings.TrimSpace(req.Domain)
	if domain == "" {
		settings, _ := h.DB.GetAllSettings()
		if settings != nil {
			domain = settings["postfix_myorigin"]
		}
	}
	if domain == "" {
		h.jsonError(w, "Please provide a domain to check", http.StatusBadRequest)
		return
	}

	hostname, _ := h.DB.GetSetting("postfix_hostname")
	if hostname == "" {
		hostname = "mail." + domain
	}

	results := checkDNS(domain, hostname)

	h.jsonOK(w, map[string]any{
		"results": results,
		"domain":  domain,
	})
}

func checkDNS(domain, hostname string) []DNSRecord {
	var records []DNSRecord

	// A record for mail hostname
	records = append(records, checkA(hostname))

	// MX record
	records = append(records, checkMX(domain, hostname))

	// SPF
	records = append(records, checkSPF(domain))

	// DKIM
	records = append(records, checkDKIM(domain))

	// DMARC
	records = append(records, checkDMARC(domain))

	// PTR
	records = append(records, checkPTR(hostname))

	return records
}

func checkA(hostname string) DNSRecord {
	ips, err := net.LookupHost(hostname)
	rec := DNSRecord{Type: "A", Name: hostname, Expected: "<your-server-ip>"}
	if err != nil || len(ips) == 0 {
		rec.Current = "not found"
		return rec
	}
	rec.Current = strings.Join(ips, ", ")
	rec.OK = true
	return rec
}

func checkMX(domain, hostname string) DNSRecord {
	mxs, err := net.LookupMX(domain)
	rec := DNSRecord{Type: "MX", Name: domain, Expected: hostname + " (priority 10)"}
	if err != nil || len(mxs) == 0 {
		rec.Current = "not found"
		return rec
	}
	var found []string
	for _, mx := range mxs {
		found = append(found, strings.TrimSuffix(mx.Host, "."))
	}
	rec.Current = strings.Join(found, ", ")
	rec.OK = containsIgnoreCase(found, strings.TrimSuffix(hostname, "."))
	return rec
}

func checkSPF(domain string) DNSRecord {
	txts, err := net.LookupTXT(domain)
	rec := DNSRecord{Type: "TXT (SPF)", Name: domain, Expected: "v=spf1 ip4:<ip> -all"}
	if err != nil {
		rec.Current = "not found"
		return rec
	}
	for _, txt := range txts {
		if strings.HasPrefix(txt, "v=spf1") {
			rec.Current = txt
			rec.OK = true
			return rec
		}
	}
	rec.Current = "not found"
	return rec
}

func checkDKIM(domain string) DNSRecord {
	dkimHost := "default._domainkey." + domain
	txts, err := net.LookupTXT(dkimHost)
	rec := DNSRecord{Type: "TXT (DKIM)", Name: dkimHost, Expected: "v=DKIM1; k=rsa; p=..."}
	if err != nil {
		rec.Current = "not found"
		return rec
	}
	for _, txt := range txts {
		if strings.Contains(txt, "DKIM1") {
			rec.Current = txt[:min(80, len(txt))] + "..."
			rec.OK = true
			return rec
		}
	}
	rec.Current = "not found"
	return rec
}

func checkDMARC(domain string) DNSRecord {
	dmarcHost := "_dmarc." + domain
	txts, err := net.LookupTXT(dmarcHost)
	rec := DNSRecord{Type: "TXT (DMARC)", Name: dmarcHost, Expected: "v=DMARC1; p=reject; ..."}
	if err != nil {
		rec.Current = "not found"
		return rec
	}
	for _, txt := range txts {
		if strings.HasPrefix(txt, "v=DMARC1") {
			rec.Current = txt
			rec.OK = strings.Contains(txt, "p=reject")
			return rec
		}
	}
	rec.Current = "not found"
	return rec
}

func checkPTR(hostname string) DNSRecord {
	ips, _ := net.LookupHost(hostname)
	rec := DNSRecord{Type: "PTR", Name: "<server-ip>", Expected: hostname}
	if len(ips) == 0 {
		rec.Current = "no A record found"
		return rec
	}
	names, err := net.LookupAddr(ips[0])
	rec.Name = ips[0]
	if err != nil || len(names) == 0 {
		rec.Current = "not configured"
		return rec
	}
	rec.Current = strings.TrimSuffix(names[0], ".")
	rec.OK = strings.EqualFold(rec.Current, hostname)
	return rec
}

func containsIgnoreCase(list []string, target string) bool {
	for _, s := range list {
		if strings.EqualFold(s, target) {
			return true
		}
	}
	return false
}
