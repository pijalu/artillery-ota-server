package main

import (
	"embed"
)

//go:embed "config.json" "artillery-m1-debs/Yuntu_m1-Yuntu_m1_ALGO_APP-403.deb" "artillery-m1-debs/Yuntu_m1-Yuntu_m1_client_deb-129.deb" "artillery-m1-debs/Yuntu_m1-Yuntu_m1_s1_client_deb-200.deb" "artillery-m1-debs/Yuntu_m1-Yuntu_m1_s1_test-201.deb" "artillery-m1-debs/Yuntu_m1-Yuntu_m1_test-129.deb"
var embeddedFiles embed.FS
