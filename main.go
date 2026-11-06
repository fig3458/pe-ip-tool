package main

import (
	"fmt"
	"net"
	"strings"

	"pe-ip-tool/internal/hive"
	"pe-ip-tool/internal/peip"
	"pe-ip-tool/internal/ui"
)

type state struct {
	app      *ui.App
	installs []hive.WinInstall
	adapters []hive.Adapter
	loaded   bool
}

func main() {
	st := &state{}
	app := &ui.App{}
	st.app = app

	app.OnRefresh = st.refresh
	app.OnSysChange = st.onSysChange
	app.OnIfChange = st.onIfChange
	app.OnApply = st.onApply

	_ = app.Run()
	_ = hive.Unload()
}

func (st *state) refresh() {
	_ = hive.Unload()
	st.loaded = false
	st.adapters = nil
	st.installs = hive.FindWindowsInstalls()

	st.app.ComboReset(ui.ID_SYSCOMBO)
	st.app.ComboReset(ui.ID_IFCOMBO)
	st.app.SetCurIP("-")
	st.app.SetText(ui.ID_IP, "")
	st.app.SetText(ui.ID_GW, "")
	st.app.SetText(ui.ID_DNS, "")
	st.app.SetText(ui.ID_MASK, "255.255.255.0")

	if len(st.installs) == 0 {
		st.app.SetStatus("未找到目标 Windows（需存在 \\Windows\\System32\\config\\SYSTEM）")
		return
	}
	for _, in := range st.installs {
		st.app.ComboAdd(ui.ID_SYSCOMBO, in.Label)
	}
	st.app.ComboSetSel(ui.ID_SYSCOMBO, 0)
	st.app.SetStatus(fmt.Sprintf("找到 %d 个系统，正在加载…", len(st.installs)))
	st.onSysChange(0)
}

func (st *state) onSysChange(index int) {
	_ = hive.Unload()
	st.loaded = false
	st.adapters = nil
	st.app.ComboReset(ui.ID_IFCOMBO)
	st.app.SetCurIP("-")

	if index < 0 || index >= len(st.installs) {
		return
	}
	in := st.installs[index]
	if err := hive.Load(in.HivePath); err != nil {
		st.app.SetStatus("加载注册表失败")
		ui.Error("加载失败", err.Error())
		return
	}
	st.loaded = true

	ads, err := hive.ListAdapters()
	if err != nil {
		st.app.SetStatus("读取网卡失败")
		ui.Error("读取失败", err.Error())
		return
	}
	st.adapters = ads
	if len(ads) == 0 {
		st.app.SetStatus("已加载 " + in.Root + "，但未发现可用网卡项")
		return
	}
	for _, ad := range ads {
		line := fmt.Sprintf("%s  |  %s  |  %s", ad.Name, ad.DisplayIP(), short(ad.GUID))
		st.app.ComboAdd(ui.ID_IFCOMBO, line)
	}
	st.app.ComboSetSel(ui.ID_IFCOMBO, 0)
	st.app.SetStatus("已加载 " + in.Root + "，请选择网卡并填写新 IP")
	st.onIfChange(0)
}

func short(g string) string {
	g = strings.Trim(g, "{}")
	if len(g) > 8 {
		return g[:8] + "…"
	}
	return g
}

func (st *state) onIfChange(index int) {
	if index < 0 || index >= len(st.adapters) {
		return
	}
	ad := st.adapters[index]
	st.app.SetCurIP(ad.DisplayIP())
	if ad.IPAddress != "" {
		st.app.SetText(ui.ID_IP, ad.IPAddress)
	} else if ad.DhcpIPAddress != "" {
		st.app.SetText(ui.ID_IP, ad.DhcpIPAddress)
	}
	if ad.SubnetMask != "" {
		st.app.SetText(ui.ID_MASK, ad.SubnetMask)
	} else {
		st.app.SetText(ui.ID_MASK, "255.255.255.0")
	}
	st.app.SetText(ui.ID_GW, ad.DefaultGateway)
	st.app.SetText(ui.ID_DNS, ad.NameServer)
}

func (st *state) onApply() {
	if !st.loaded {
		ui.Error("无法写入", "请先选择并成功加载目标系统")
		return
	}
	idx := st.app.ComboSel(ui.ID_IFCOMBO)
	if idx < 0 || idx >= len(st.adapters) {
		ui.Error("无法写入", "请选择网卡接口")
		return
	}
	ip := strings.TrimSpace(st.app.GetText(ui.ID_IP))
	mask := strings.TrimSpace(st.app.GetText(ui.ID_MASK))
	gw := strings.TrimSpace(st.app.GetText(ui.ID_GW))
	dns := strings.TrimSpace(st.app.GetText(ui.ID_DNS))

	if err := validateIPv4(ip); err != nil {
		ui.Error("IP 无效", err.Error())
		return
	}
	if err := validateIPv4(mask); err != nil {
		ui.Error("掩码无效", err.Error())
		return
	}
	if gw != "" {
		if err := validateIPv4(gw); err != nil {
			ui.Error("网关无效", err.Error())
			return
		}
	}
	for _, d := range splitDNS(dns) {
		if err := validateIPv4(d); err != nil {
			ui.Error("DNS 无效", err.Error())
			return
		}
	}

	ad := st.adapters[idx]
	msg := fmt.Sprintf("将把目标系统网卡「%s」改为静态：\n\nIP: %s\n掩码: %s\n网关: %s\nDNS: %s\n\n确认写入离线注册表？",
		ad.Name, ip, mask, gw, dns)
	if !ui.WarnYesNo("确认写入", msg) {
		return
	}

	// Respect checkbox: if unchecked, keep only primary control set.
	if !st.app.Checked(ui.ID_BOTHSETS) && len(ad.ControlSets) > 0 {
		ad.ControlSets = ad.ControlSets[:1]
	}

	if err := hive.ApplyStatic(ad, ip, mask, gw, dns); err != nil {
		ui.Error("写入失败", err.Error())
		st.app.SetStatus("写入失败")
		return
	}

	extra := ""
	if st.app.Checked(ui.ID_PEIP) {
		iface := strings.TrimSpace(st.app.GetText(ui.ID_PEIFACE))
		if err := peip.SetStatic(iface, ip, mask, gw, dns); err != nil {
			extra = "\n\n注意：目标系统已写入成功，但 PE 临时 IP 设置失败：\n" + err.Error()
		} else {
			extra = "\n\nPE 本机临时 IP 已同步设置。"
		}
	}

	// Reload list to show new values while hive still loaded.
	if ads, err := hive.ListAdapters(); err == nil {
		st.adapters = ads
		st.app.ComboReset(ui.ID_IFCOMBO)
		for _, a := range ads {
			line := fmt.Sprintf("%s  |  %s  |  %s", a.Name, a.DisplayIP(), short(a.GUID))
			st.app.ComboAdd(ui.ID_IFCOMBO, line)
		}
		if idx >= len(ads) {
			idx = 0
		}
		st.app.ComboSetSel(ui.ID_IFCOMBO, idx)
		st.onIfChange(idx)
	}

	st.app.SetStatus("写入成功。请卸载后重启进入目标系统验证。")
	ui.Info("完成", "离线静态 IP 已写入。"+extra+"\n\n建议：退出本工具后重启进入目标系统检查。")
}

func validateIPv4(s string) error {
	ip := net.ParseIP(s)
	if ip == nil || ip.To4() == nil {
		return fmt.Errorf("不是合法 IPv4: %s", s)
	}
	return nil
}

func splitDNS(dns string) []string {
	dns = strings.ReplaceAll(dns, "，", ",")
	var out []string
	for _, p := range strings.FieldsFunc(dns, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	}) {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
