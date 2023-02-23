package main_test

import (
	"encoding/json"
	"log"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/KonradKuznicki/must"

	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/share/query"
)

var serializedClient = `{"id":"76c2a67347e0daab20074cab6359452c","session_id":"c1724b6d-714c-4a61-89f8-adc7e156450c","name":"pop-os","os":"Linux pop-os 6.0.12-76060006-generic #202212290932~1674139725~22.04~ca93ccf SMP PREEMPT_DYNAMIC Thu J x86_64 x86_64 x86_64 GNU/Linux","os_arch":"amd64","os_family":"debian","os_kernel":"linux","os_full_name":"Debian bookworm/sid","os_version":"bookworm/sid","os_virtualization_system":"KVM","os_virtualization_role":"guest","cpu_family":"15","cpu_model":"6","cpu_model_name":"Common KVM processor","cpu_vendor":"AuthenticAMD","num_cpus":12,"mem_total":33655844864,"timezone":"CET (UTC+01:00)","hostname":"pop-os","ipv4":["192.168.7.64","172.17.0.1"],"ipv6":["fe80::74d:6cb8:91b2:1c8d"],"tags":["task-vm","vm"],"labels":{"city":"Cologne","country":"Germany","datacenter":"NetCologne GmbH"},"version":"0.0.0-src","address":"127.0.0.1","tunnels":[],"disconnected_at":null,"last_heartbeat_at":null,"client_auth_id":"clientAuth1","allowed_user_groups":null,"updates_status":{"refreshed":"2023-02-17T16:04:54.612411037+01:00","updates_available":0,"security_updates_available":0,"update_summaries":null,"reboot_pending":false,"error":"sudo: a password is required\n"},"client_configuration":{"client":{"server":"ws://0.0.0.0:8080","fallback_servers":[],"server_switchback_interval":120000000000,"fingerprint":"","auth":"clientAuth1:1234","proxy":"","id":"","use_system_id":true,"name":"","use_hostname":true,"tags":["task-vm","vm"],"labels":{"city":"Cologne","country":"Germany","datacenter":"NetCologne GmbH"},"remotes":null,"tunnel_allowed":null,"allow_root":false,"updates_interval":14400000000000,"data_dir":"./rc-dev-resources","bind_interface":"","proxy_url":null,"tunnels":null,"auth_user":"clientAuth1","auth_pass":"1234"},"connection":{"keep_alive":180000000000,"keep_alive_timeout":30000000000,"max_retry_count":-1,"max_retry_interval":300000000000,"headers":[],"hostname":"","watchdog_integration":false,"http_headers":{"User-Agent":["rport 0.0.0-src"]}},"logging":{"log_file":{"File":{}},"log_level":1},"remote_commands":{"enabled":true,"send_back_limit":4194304,"allow":["^/usr/bin/.*","^/usr/local/bin/.*","^C:\\\\Windows\\\\System32\\\\.*"],"deny":["(\\||\u003c|\u003e|;|,|\\n|\u0026)"],"order":["allow","deny"],"allow_regexp":[{},{},{}],"deny_regexp":[{}]},"remote_scripts":{"enabled":false},"monitoring":{"enabled":true,"interval":60000000000,"fs_type_include":["ext3","ext4","xfs","jfs","ntfs","btrfs","hfs","apfs","exfat","smbfs","nfs"],"fs_path_exclude":[],"fs_path_exclude_recurse":false,"fs_identify_mountpoints_by_device":true,"pm_enabled":true,"pm_kerneltasks_enabled":true,"pm_max_number_processes":500,"net_lan":[],"net_wan":[],"lan_card":null,"wan_card":null},"file_reception":{"protected":["/bin","/sbin","/boot","/usr/bin","/usr/sbin","/dev","/lib*","/run"],"enabled":true},"interpreter_aliases":{},"interpreter_aliases_encodings":{}},"groups":[],"connection_state":"connected"}`

func times[T any](calls int, f func(count int) T) []T {
	tmp := make([]T, calls)
	for i := 0; i < calls; i++ {
		tmp[i] = f(i)
	}
	return tmp
}

var clientsP = times(5000, desP)
var clientsS = times(5000, des)

func desP(count int) *clients.Client {
	tmp := &clients.Client{}
	must.Must0(json.Unmarshal([]byte(serializedClient), tmp))
	return tmp
}

func des(count int) clients.Client {
	tmp := clients.Client{}
	must.Must0(json.Unmarshal([]byte(serializedClient), &tmp))
	return tmp
}

var filterOptions = query.ParseFilterOptions(map[string][]string{
	"filter[name]": {"fail"},
})

func BenchmarkMatcher(b *testing.B) {
	f := time.Now()
	count := 0
	for _, c := range clientsP {
		filters := must.Must(query.MatchesFilters(c, filterOptions))
		count += len(c.Tunnels)
		if filters {
			count++
		}
	}
	log.Println(count, len(clientsP), time.Since(f)/time.Millisecond)
}

func BenchmarkSerializationDeserialization(b *testing.B) {
	f := time.Now()
	count := 0
	for _, c := range clientsP {
		m := must.Must(json.Marshal(c))
		l := &clients.Client{}
		must.Must0(json.Unmarshal(m, l))
		count += len(l.Tunnels)
	}
	log.Println(count, len(clientsP), time.Since(f)/time.Millisecond)
}

//
//func BenchmarkDes(b *testing.B) {
//	f := time.Now()
//	count := 0
//	clientsPz := times(5000, des)
//
//	log.Println(count, len(clientsPz), time.Since(f)/time.Millisecond)
//}

func BenchmarkReflectionP(b *testing.B) {
	f := time.Now()
	count := 0
	v := reflect.ValueOf(*clientsP[0])
	m := map[string]string{"name": "dupa"}
	tt := buildTranslationTable(v)
	q := translateQuery(tt, m)

	for _, c := range clientsP {

		count += len(c.Tunnels)
		if filterRefP(c, q) {
			count++
		}
	}
	log.Println(count, len(clientsP), time.Since(f)/time.Millisecond)
}

func BenchmarkReflectionS(b *testing.B) {
	f := time.Now()
	count := 0

	v := reflect.ValueOf(clientsS[0])
	m := map[string]string{"name": "pop-os"}
	tt := buildTranslationTable(v)
	q := translateQuery(tt, m)

	for _, c := range clientsS {

		count += len(c.Tunnels)
		if filterRefS(c, q) {
			count++
		}
	}
	log.Println(count, len(clientsP), time.Since(f)/time.Millisecond)
}

func TestFilterRef(t *testing.T) {
	assert.True(t, filterRefP(clientsP[0], map[string]string{"name": "pop-os"}))

	assert.False(t, filterRefP(clientsP[0], map[string]string{"name": "fail"}))
}

func buildTranslationTable(v reflect.Value) map[string]string {
	tmp := map[string]string{}
	typeOf := v.Type()

	for i := 0; i < v.NumField(); i++ {
		p := typeOf.Field(i)
		jsonTag, _ := p.Tag.Lookup("json")
		jsonParts := strings.Split(jsonTag, ",")

		t := jsonTag
		if len(jsonParts) != 0 {
			t = jsonParts[0]
		}

		if len(t) > 0 && t != "-" {
			tmp[t] = p.Name
		} else if len(t) == 0 {
			tmp[p.Name] = p.Name
		}

	}

	return tmp
}

func translateQuery(translationTable map[string]string, query map[string]string) map[string]string {
	tmp := map[string]string{}
	for k, v := range query {
		if t, ok := translationTable[k]; ok {
			tmp[t] = v
		}
	}
	return tmp
}

func filterRefP(c *clients.Client, m map[string]string) bool {
	v := reflect.ValueOf(*c)

	for k, va := range m {
		if v.FieldByName(k).String() != va {
			return false
		}
	}

	return true
}

func filterRefS(c clients.Client, m map[string]string) bool {
	v := reflect.ValueOf(c)

	for k, va := range m {
		if v.FieldByName(k).String() != va {
			return false
		}
	}

	return true
}
