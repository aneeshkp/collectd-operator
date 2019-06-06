package configs

import (
	collectdv1alpha1 "github.com/aneeshkp/collectd-operator/pkg/apis/collectdmon/v1alpha1"
)

const (
	collectdConfigPath = "/opt/collectd/etc/collectd.conf"
)

//ConfigForCollectd  ....
func ConfigForCollectd(m *collectdv1alpha1.Collectd) string {
	config := `
        FQDNLookup false
        LoadPlugin syslog
        <Plugin syslog>
        LogLevel info
        </Plugin>

        LoadPlugin cpu

        LoadPlugin memory
    
        <Plugin "cpu">
        Interval 5
        ReportByState false
        ReportByCpu false
        </Plugin>

        <Plugin "memory">
        Interval 30
        ValuesAbsolute false
        ValuesPercentage true
        </Plugin>

        LoadPlugin processes
        <Plugin " processes">
        Process "docker"
        # Add any other processes you wish to monitor...
        </Plugin>
    
        #Last line (collectd requires ‘\n’ at the last line)`
	return config

	//var buff bytes.Buffer
	//collectdconfig := template.Must(template.New("collectdconfig").Parse(config))
	//collectdconfig.Execute(&buff, m.Spec)
	//return buff.String()
}
