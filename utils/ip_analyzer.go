package utils

import (
	"context"
	"net"
)

type IPRiskLevel int

const (
	IPRiskLow IPRiskLevel = iota
	IPRiskMedium
	IPRiskHigh
	IPRiskVeryHigh
)

type IPAnalyzer struct {
	highRiskCountries map[string]bool
	proxyRanges       []string
	datacenterRanges  []string
}

func CreateIPAnalyzer() *IPAnalyzer {
	return &IPAnalyzer{
		highRiskCountries: map[string]bool{
			"CN": true, "RU": true, "KP": true, "IR": true,
		},
		proxyRanges: []string{
			"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		},
		datacenterRanges: []string{
			"54.0.0.0/8", "52.0.0.0/8", "34.0.0.0/8",
		},
	}
}

func (ia *IPAnalyzer) AnalyzeIP(ctx context.Context, ip string) IPRiskLevel {
	if ip == "" {
		return IPRiskHigh
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return IPRiskHigh
	}

	if ia.isPrivateIP(parsedIP) {
		return IPRiskMedium
	}

	if ia.isDatacenterIP(parsedIP) {
		return IPRiskHigh
	}

	if ia.isProxyIP(parsedIP) {
		return IPRiskVeryHigh
	}

	country := ia.getCountryFromIP(parsedIP)
	if ia.highRiskCountries[country] {
		return IPRiskHigh
	}

	return IPRiskLow
}

func (ia *IPAnalyzer) isPrivateIP(ip net.IP) bool {
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
	}

	for _, block := range privateBlocks {
		_, network, _ := net.ParseCIDR(block)
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func (ia *IPAnalyzer) isDatacenterIP(ip net.IP) bool {
	datacenterBlocks := []string{
		"54.0.0.0/8",
		"52.0.0.0/8",
		"34.0.0.0/8",
		"35.0.0.0/8",
		"104.0.0.0/8",
		"108.0.0.0/8",
	}

	for _, block := range datacenterBlocks {
		_, network, _ := net.ParseCIDR(block)
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func (ia *IPAnalyzer) isProxyIP(ip net.IP) bool {
	for _, block := range ia.proxyRanges {
		_, network, _ := net.ParseCIDR(block)
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func (ia *IPAnalyzer) getCountryFromIP(ip net.IP) string {
	countryMap := map[string]string{
		"8.8.8.8":        "US",
		"1.1.1.1":        "US",
		"208.67.222.222": "US",
	}

	if country, exists := countryMap[ip.String()]; exists {
		return country
	}

	return "US"
}

func (ia *IPAnalyzer) GetRiskScore(riskLevel IPRiskLevel) int {
	switch riskLevel {
	case IPRiskLow:
		return 0
	case IPRiskMedium:
		return 25
	case IPRiskHigh:
		return 50
	case IPRiskVeryHigh:
		return 75
	default:
		return 50
	}
}

func (ia *IPAnalyzer) GetRiskDescription(riskLevel IPRiskLevel) string {
	switch riskLevel {
	case IPRiskLow:
		return "Low risk IP address"
	case IPRiskMedium:
		return "Medium risk IP address (private network)"
	case IPRiskHigh:
		return "High risk IP address (datacenter or high-risk country)"
	case IPRiskVeryHigh:
		return "Very high risk IP address (proxy or VPN)"
	default:
		return "Unknown risk level"
	}
}
