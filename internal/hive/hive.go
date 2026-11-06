package hive

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	modadvapi32      = windows.NewLazySystemDLL("advapi32.dll")
	procRegLoadKeyW  = modadvapi32.NewProc("RegLoadKeyW")
	procRegUnLoadKey = modadvapi32.NewProc("RegUnLoadKeyW")
)

const offlineRoot = `OFFLINE_PEIP`

// WinInstall describes a detected Windows installation.
type WinInstall struct {
	Root     string // e.g. D:\
	System32 string // e.g. D:\Windows\System32
	HivePath string // SYSTEM hive path
	Label    string // display text
}

// Adapter holds offline TCP/IP settings for one interface GUID.
type Adapter struct {
	GUID           string
	Name           string
	EnableDHCP     uint32
	IPAddress      string
	SubnetMask     string
	DefaultGateway string
	NameServer     string
	DhcpIPAddress  string
	ControlSets    []string // e.g. ControlSet001
}

// FindWindowsInstalls scans fixed/removable drive letters for Windows.
func FindWindowsInstalls() []WinInstall {
	var out []WinInstall
	for c := 'C'; c <= 'Z'; c++ {
		root := string(c) + `:\`
		hive := filepath.Join(root, `Windows`, `System32`, `config`, `SYSTEM`)
		if !fileExists(hive) {
			continue
		}
		// Skip the running PE itself when possible: PE usually has X:\
		label := fmt.Sprintf("%s  (Windows)", root)
		out = append(out, WinInstall{
			Root:     root,
			System32: filepath.Join(root, `Windows`, `System32`),
			HivePath: hive,
			Label:    label,
		})
	}
	return out
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func enablePrivilege(name string) error {
	var token windows.Token
	proc, err := windows.GetCurrentProcess()
	if err != nil {
		return err
	}
	err = windows.OpenProcessToken(proc, windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &token)
	if err != nil {
		return err
	}
	defer token.Close()

	var luid windows.LUID
	err = windows.LookupPrivilegeValue(nil, windows.StringToUTF16Ptr(name), &luid)
	if err != nil {
		return err
	}
	tp := windows.Tokenprivileges{
		PrivilegeCount: 1,
		Privileges: [1]windows.LUIDAndAttributes{
			{Luid: luid, Attributes: windows.SE_PRIVILEGE_ENABLED},
		},
	}
	return windows.AdjustTokenPrivileges(token, false, &tp, 0, nil, nil)
}

// Load mounts the offline SYSTEM hive under HKLM\OFFLINE_PEIP.
func Load(hivePath string) error {
	_ = Unload() // ignore
	if err := enablePrivilege("SeBackupPrivilege"); err != nil {
		return fmt.Errorf("SeBackupPrivilege: %w", err)
	}
	if err := enablePrivilege("SeRestorePrivilege"); err != nil {
		return fmt.Errorf("SeRestorePrivilege: %w", err)
	}
	pathPtr, err := windows.UTF16PtrFromString(hivePath)
	if err != nil {
		return err
	}
	subKey, err := windows.UTF16PtrFromString(offlineRoot)
	if err != nil {
		return err
	}
	r, _, _ := procRegLoadKeyW.Call(uintptr(windows.HKEY_LOCAL_MACHINE), uintptr(unsafe.Pointer(subKey)), uintptr(unsafe.Pointer(pathPtr)))
	if r != 0 {
		return fmt.Errorf("RegLoadKey: %w (请确认以管理员运行，且 SYSTEM 未被占用)", syscall.Errno(r))
	}
	return nil
}

// Unload unmounts the offline hive.
func Unload() error {
	subKey, err := windows.UTF16PtrFromString(offlineRoot)
	if err != nil {
		return err
	}
	r, _, _ := procRegUnLoadKey.Call(uintptr(windows.HKEY_LOCAL_MACHINE), uintptr(unsafe.Pointer(subKey)))
	if r != 0 {
		// ERROR_FILE_NOT_FOUND / not loaded is fine
		if syscall.Errno(r) == 2 {
			return nil
		}
		return syscall.Errno(r)
	}
	return nil
}

func activeControlSets() ([]string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, offlineRoot+`\Select`, registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}
	defer k.Close()
	cur, _, err := k.GetIntegerValue("Current")
	if err != nil {
		return nil, err
	}
	primary := fmt.Sprintf("ControlSet%03d", cur)
	sets := []string{primary}
	// Also include the other common set if present, for safety.
	other := "ControlSet001"
	if primary == "ControlSet001" {
		other = "ControlSet002"
	}
	if keyExists(offlineRoot + `\` + other + `\Services\Tcpip\Parameters\Interfaces`) {
		sets = append(sets, other)
	}
	return sets, nil
}

func keyExists(path string) bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	k.Close()
	return true
}

// ListAdapters reads TCP/IP interfaces from the loaded hive.
func ListAdapters() ([]Adapter, error) {
	sets, err := activeControlSets()
	if err != nil {
		return nil, err
	}
	primary := sets[0]
	base := offlineRoot + `\` + primary + `\Services\Tcpip\Parameters\Interfaces`
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, base, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil, err
	}
	defer k.Close()
	names, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return nil, err
	}

	var out []Adapter
	for _, guid := range names {
		ad, ok := readAdapter(primary, sets, guid)
		if !ok {
			continue
		}
		out = append(out, ad)
	}
	return out, nil
}

func readAdapter(primary string, sets []string, guid string) (Adapter, bool) {
	path := offlineRoot + `\` + primary + `\Services\Tcpip\Parameters\Interfaces\` + guid
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		return Adapter{}, false
	}
	defer k.Close()

	ad := Adapter{GUID: guid, ControlSets: sets}
	if v, _, err := k.GetIntegerValue("EnableDHCP"); err == nil {
		ad.EnableDHCP = uint32(v)
	}
	ad.IPAddress = firstMulti(k, "IPAddress")
	ad.SubnetMask = firstMulti(k, "SubnetMask")
	ad.DefaultGateway = firstMulti(k, "DefaultGateway")
	ad.NameServer, _, _ = k.GetStringValue("NameServer")
	ad.DhcpIPAddress, _, _ = k.GetStringValue("DhcpIPAddress")
	ad.Name = lookupConnName(primary, guid)
	if ad.Name == "" {
		ad.Name = shortGUID(guid)
	}

	// Skip completely empty tunnel/placeholder adapters with no address info.
	if ad.IPAddress == "" && ad.DhcpIPAddress == "" && ad.SubnetMask == "" {
		// keep if EnableDHCP is set (real NIC often has DHCP)
		if ad.EnableDHCP == 0 && ad.NameServer == "" {
			return Adapter{}, false
		}
	}
	return ad, true
}

func firstMulti(k registry.Key, name string) string {
	vals, _, err := k.GetStringsValue(name)
	if err != nil || len(vals) == 0 {
		return ""
	}
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v != "" && v != "0.0.0.0" {
			return v
		}
	}
	return strings.TrimSpace(vals[0])
}

func shortGUID(g string) string {
	g = strings.Trim(g, "{}")
	if len(g) > 8 {
		return g[:8]
	}
	return g
}

func lookupConnName(controlSet, guid string) string {
	// Control\Network\{4D36E972-E325-11CE-BFC1-08002BE10318}\{guid}\Connection
	path := fmt.Sprintf(`%s\%s\Control\Network\{4D36E972-E325-11CE-BFC1-08002BE10318}\%s\Connection`,
		offlineRoot, controlSet, guid)
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	name, _, err := k.GetStringValue("Name")
	if err != nil {
		return ""
	}
	return name
}

// ApplyStatic writes static IPv4 settings into all relevant control sets.
func ApplyStatic(ad Adapter, ip, mask, gw, dns string) error {
	if len(ad.ControlSets) == 0 {
		sets, err := activeControlSets()
		if err != nil {
			return err
		}
		ad.ControlSets = sets
	}
	dns = normalizeDNS(dns)
	for _, cs := range ad.ControlSets {
		path := offlineRoot + `\` + cs + `\Services\Tcpip\Parameters\Interfaces\` + ad.GUID
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.SET_VALUE)
		if err != nil {
			return fmt.Errorf("打开 %s: %w", path, err)
		}
		if err := k.SetDWordValue("EnableDHCP", 0); err != nil {
			k.Close()
			return err
		}
		if err := k.SetStringsValue("IPAddress", []string{ip}); err != nil {
			k.Close()
			return err
		}
		if err := k.SetStringsValue("SubnetMask", []string{mask}); err != nil {
			k.Close()
			return err
		}
		if gw == "" {
			_ = k.SetStringsValue("DefaultGateway", []string{})
		} else {
			if err := k.SetStringsValue("DefaultGateway", []string{gw}); err != nil {
				k.Close()
				return err
			}
		}
		if err := k.SetStringValue("NameServer", dns); err != nil {
			k.Close()
			return err
		}
		// Common companion values for static config.
		_ = k.SetDWordValue("DisableDhcpOnConnect", 1)
		k.Close()
	}
	return nil
}

func normalizeDNS(dns string) string {
	dns = strings.TrimSpace(dns)
	dns = strings.ReplaceAll(dns, "，", ",")
	dns = strings.ReplaceAll(dns, " ", ",")
	parts := strings.Split(dns, ",")
	var clean []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			clean = append(clean, p)
		}
	}
	return strings.Join(clean, ",")
}

// DisplayIP returns the best IP string for list UI.
func (a Adapter) DisplayIP() string {
	if a.IPAddress != "" && a.EnableDHCP == 0 {
		return a.IPAddress + " (静态)"
	}
	if a.DhcpIPAddress != "" {
		return a.DhcpIPAddress + " (DHCP)"
	}
	if a.IPAddress != "" {
		return a.IPAddress
	}
	if a.EnableDHCP != 0 {
		return "(DHCP)"
	}
	return "(无地址)"
}
