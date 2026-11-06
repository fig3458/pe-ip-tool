package peip

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// SetStatic configures the running PE (or current OS) NIC via netsh.
func SetStatic(iface, ip, mask, gw, dns string) error {
	iface = strings.TrimSpace(iface)
	if iface == "" {
		iface = "以太网"
	}
	args := []string{"interface", "ip", "set", "address",
		"name=" + iface, "static", ip, mask}
	if strings.TrimSpace(gw) != "" {
		args = append(args, gw)
	}
	if err := runNetsh(args...); err != nil {
		// fallback English interface name
		if iface == "以太网" {
			args[4] = "name=Ethernet"
			if err2 := runNetsh(args...); err2 == nil {
				iface = "Ethernet"
			} else {
				return err
			}
		} else {
			return err
		}
	}
	dns = strings.TrimSpace(dns)
	if dns == "" {
		return nil
	}
	dns = strings.ReplaceAll(dns, "，", ",")
	parts := strings.FieldsFunc(dns, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	})
	for i, d := range parts {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if i == 0 {
			_ = runNetsh("interface", "ip", "set", "dns", "name="+iface, "static", d, "validate=no")
		} else {
			_ = runNetsh("interface", "ip", "add", "dns", "name="+iface, d, fmt.Sprintf("index=%d", i+1), "validate=no")
		}
	}
	return nil
}

// ListInterfaces returns netsh interface names.
func ListInterfaces() ([]string, error) {
	cmd := exec.Command("netsh", "interface", "ip", "show", "interfaces")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var names []string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// netsh table: Idx Met MTU State Name — take last field-ish
		if line == "" || strings.HasPrefix(line, "Idx") || strings.Contains(line, "---") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		name := strings.Join(fields[4:], " ")
		if name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

func runNetsh(args ...string) error {
	cmd := exec.Command("netsh", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
