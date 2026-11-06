package main

import (
	"fmt"
	"os"

	"pe-ip-tool/internal/hive"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: smoke <SYSTEM hive path>")
		os.Exit(2)
	}
	if err := hive.Load(os.Args[1]); err != nil {
		fmt.Println("LOAD_FAIL", err)
		os.Exit(1)
	}
	defer hive.Unload()
	ads, err := hive.ListAdapters()
	if err != nil {
		fmt.Println("LIST_FAIL", err)
		os.Exit(1)
	}
	fmt.Println("ADAPTERS", len(ads))
	for i, a := range ads {
		if i >= 8 {
			break
		}
		fmt.Printf("%d %s | dhcp=%d ip=%s mask=%s gw=%s dns=%s\n",
			i, a.Name, a.EnableDHCP, a.IPAddress, a.SubnetMask, a.DefaultGateway, a.NameServer)
	}
	fmt.Println("SMOKE_OK")
}
