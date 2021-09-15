package system

import (
	"fmt"
	"math"
	"strings"

	"idrac-exporter/config"
	"idrac-exporter/redfishstruct"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish/redfish"
)

type SystemCollector struct{}

func (collector SystemCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- config.S_bios
	ch <- config.S_network_adapter_status
	ch <- config.S_storage_drive
	ch <- config.S_storage_drive_predicted_media_life_left_percent
	ch <- config.S_ethernetinterface
	ch <- config.S_health
	ch <- config.S_memory
	ch <- config.S_networkport
	ch <- config.S_processor
	ch <- config.S_storage
	ch <- config.S_storage_volume
	ch <- config.S_idrac_status
}

func (collector SystemCollector) Collect(ch chan<- prometheus.Metric) {
	metric := config.GOFISH.Service
	systems, sysErr := metric.Systems() //Systems get the system instances from the service

	if nil != sysErr {
		panic(sysErr)
	}
	//10 * 7
	//systems[0]
	//systems[1].fan
	for _, system := range systems {
		collector.collectSystemHealth(ch, system)
		collector.collectBios(ch, system)
		collector.collectEthernetInterfaces(ch, system)
		collector.collectMemories(ch, system)
		collector.collectStorage(ch, system)
		collector.collectProcessors(ch, system)
		collector.collectorNetworks(ch, system)
	}
	collector.collectIDRACPortStatus(ch, &redfishstruct.IDRACPort{})
}

func (collector SystemCollector) collectSystemHealth(ch chan<- prometheus.Metric, v *redfish.ComputerSystem) {
	status := config.State_dict[string(v.Status.Health)]
	ch <- prometheus.MustNewConstMetric(config.S_health, prometheus.GaugeValue, status,
		v.BIOSVersion,
		v.Description,
		v.HostName,
		v.HostedServices,
		v.Manufacturer,
		v.Model,
		v.Name,
		v.PartNumber,
		fmt.Sprintf("%v", v.PowerRestorePolicy),
		fmt.Sprintf("%v", v.PowerState),
		v.SKU,
		v.SerialNumber,
		v.SubModel,
		fmt.Sprintf("%v", v.SystemType),
		v.UUID,
	)
}

func (collector SystemCollector) collectBios(ch chan<- prometheus.Metric, system *redfish.ComputerSystem) {
	val, biosErr := system.Bios()

	if nil != biosErr {
		panic(biosErr)
	}

	ch <- prometheus.MustNewConstMetric(config.S_bios,
		prometheus.GaugeValue,
		float64(0),
		val.AttributeRegistry,
		val.Description,
	)
}

func (collector SystemCollector) collectMemories(ch chan<- prometheus.Metric, system *redfish.ComputerSystem) {
	memories, err := system.Memory()

	if nil != err {
		panic(err)
	}

	if nil == err {
		for _, memory := range memories {
			status := config.State_dict[string(memory.Status.Health)]
			ch <- prometheus.MustNewConstMetric(config.S_memory, prometheus.GaugeValue, float64(status),
				fmt.Sprintf("%v", memory.AllocationAlignmentMiB),
				fmt.Sprintf("%v", memory.AllocationIncrementMiB),
				fmt.Sprintf("%v", memory.BaseModuleType),
				fmt.Sprintf("%v", memory.BusWidthBits),
				fmt.Sprintf("%v", memory.CacheSizeMiB),
				fmt.Sprintf("%v", memory.CapacityMiB),
				fmt.Sprintf("%v", memory.ConfigurationLocked),
				fmt.Sprintf("%v", memory.DataWidthBits),
				memory.Description,
				memory.DeviceLocator,
				fmt.Sprintf("%v", memory.ErrorCorrection),
				memory.FirmwareAPIVersion,
				memory.FirmwareRevision,
				fmt.Sprintf("%v", memory.IsRankSpareEnabled),
				fmt.Sprintf("%v", memory.IsSpareDeviceEnabled),
				fmt.Sprintf("%v", memory.LogicalSizeMiB),
				memory.Manufacturer,
				fmt.Sprintf("%v", memory.MemoryDeviceType),
				fmt.Sprintf("%v", memory.MemoryType),
				fmt.Sprintf("%v", memory.OperatingSpeedMhz),
				memory.PartNumber,
				fmt.Sprintf("%v", memory.RankCount),
				memory.SerialNumber,
			)
		}
	}
}

func (collector SystemCollector) collectStorage(ch chan<- prometheus.Metric, system *redfish.ComputerSystem) {
	storages, storageErr := system.Storage()

	if nil != storageErr {
		panic(storageErr)
	}

	if 0 != len(storages) {
		for _, storage := range storages {
			status := config.State_dict[string(storage.Status.Health)]
			ch <- prometheus.MustNewConstMetric(config.S_storage,
				prometheus.GaugeValue,
				float64(status),
				storage.Description,
				fmt.Sprintf("%v", storage.DrivesCount),
				fmt.Sprintf("%v", storage.RedundancyCount),
				fmt.Sprintf("%v", storage.EnclosuresCount),
			)

			collector.collectDrives(ch, storage)
		}
	}
}

func (collector SystemCollector) associatedDriveIds(volume *redfish.Volume) []string {
	drives, _ := volume.Drives()
	driveId := make([]string, 0)

	if 0 != len(drives) {
		for _, drive := range drives {
			words := strings.Split(drive.Description, " ")
			driveId = append(driveId, words[len(words)-1])
		}
	}

	return driveId
}

func (collector SystemCollector) collectDrives(ch chan<- prometheus.Metric, storage *redfish.Storage) {
	drives, driveErr := storage.Drives()

	if nil != driveErr {
		panic(driveErr)
	}

	for _, drive := range drives {
		status := config.State_dict[string(drive.Status.Health)]
		ch <- prometheus.MustNewConstMetric(config.S_storage_drive,
			prometheus.GaugeValue,
			float64(status),
			fmt.Sprintf("%v", drive.BlockSizeBytes),
			fmt.Sprintf("%v", drive.CapableSpeedGbs),
			collector.convertCapacity(float64(drive.CapacityBytes)),
			drive.Description,
			fmt.Sprintf("%v", drive.IndicatorLED),
			drive.Manufacturer,
			fmt.Sprintf("%v", drive.MediaType),
			drive.Model,
			drive.PartNumber,
			fmt.Sprintf("%v", drive.Protocol),
			drive.Revision,
			drive.SerialNumber,
		)

		if "SSD" == fmt.Sprintf("%v", drive.MediaType) {
			collector.collectSSDDrives(ch, drive)
		}
	}
}

func (collector SystemCollector) collectSSDDrives(ch chan<- prometheus.Metric, drive *redfish.Drive) {
	ch <- prometheus.MustNewConstMetric(config.S_storage_drive_predicted_media_life_left_percent,
		prometheus.GaugeValue,
		float64(drive.PredictedMediaLifeLeftPercent),
		fmt.Sprintf("%v", drive.BlockSizeBytes),
		fmt.Sprintf("%v", drive.CapableSpeedGbs),
		collector.convertCapacity(float64(drive.CapacityBytes)),
		drive.Description,
		drive.Manufacturer,
		fmt.Sprintf("%v", drive.MediaType),
		drive.Model,
		drive.PartNumber,
		fmt.Sprintf("%v", drive.Protocol),
		drive.Revision,
		drive.SerialNumber,
	)
}

func (collector SystemCollector) convertCapacity(num float64) string {
	units := []string{"TB", "GB", "MB", "KB", "B"}
	idx := len(units) - 1

	for idx > -1 && num >= 1000 {
		idx -= 1
		num = num / 1000
	}

	return fmt.Sprintf("%v", math.RoundToEven(num)) + units[idx]
}

func (collector SystemCollector) collectEthernetInterfaces(ch chan<- prometheus.Metric, system *redfish.ComputerSystem) {
	ethernetInterfaces, ethernetErr := system.EthernetInterfaces()
	if nil != ethernetErr {
		panic(ethernetErr)
	}

	if 0 != len(ethernetInterfaces) {
		for _, ethernetInterface := range ethernetInterfaces {
			status := config.State_dict[string(ethernetInterface.Status.Health)]
			ch <- prometheus.MustNewConstMetric(config.S_ethernetinterface,
				prometheus.GaugeValue,
				float64(status),
				fmt.Sprintf("%v", ethernetInterface.AutoNeg),
				ethernetInterface.Description,
				fmt.Sprintf("%v", ethernetInterface.EthernetInterfaceType),
				ethernetInterface.FQDN,
				fmt.Sprintf("%v", ethernetInterface.FullDuplex),
				ethernetInterface.HostName,
				ethernetInterface.MACAddress,
				fmt.Sprintf("%v", ethernetInterface.MTUSize),
				fmt.Sprintf("%v", ethernetInterface.SpeedMbps),
				//
				fmt.Sprintf("%v", ethernetInterface.DHCPv4.UseGateway),
			)
		}
	}
}

func (collector SystemCollector) collectProcessors(ch chan<- prometheus.Metric, system *redfish.ComputerSystem) {
	processors, proErr := system.Processors()

	if nil != proErr {
		panic(proErr)
	}

	for _, processor := range processors {
		status := config.State_dict[string(processor.Status.Health)]
		ch <- prometheus.MustNewConstMetric(config.S_processor,
			prometheus.GaugeValue,
			float64(status),
			processor.Actions,
			processor.Description,
			processor.Manufacturer,
			fmt.Sprintf("%v", processor.MaxSpeedMHz),
			fmt.Sprintf("%v", processor.MaxTDPWatts),
			processor.Model,
			fmt.Sprintf("%v", processor.ProcessorType),
			processor.Socket,
			processor.SubProcessors,
			fmt.Sprintf("%v", processor.TDPWatts),
			fmt.Sprintf("%v", processor.TotalCores),
			fmt.Sprintf("%v", processor.TotalEnabledCores),
			fmt.Sprintf("%v", processor.TotalThreads),
			processor.UUID,
		)
	}
}

func (collector SystemCollector) collectorNetworks(ch chan<- prometheus.Metric, system *redfish.ComputerSystem) {
	interfaces, err := system.NetworkInterfaces()

	if nil != err {
		panic(err)
	}

	if 0 != len(interfaces) {
		collector.makeNetworkPortMetricFromNetworkInterfaces(ch, interfaces)
	}
}

func (collector SystemCollector) makeNetworkPortMetricFromNetworkInterfaces(ch chan<- prometheus.Metric,
	interfaces []*redfish.NetworkInterface) {
	for _, netInterface := range interfaces {
		adapter, err := netInterface.NetworkAdapter()

		if nil != err {
			panic(err)
		}

		if nil != adapter {
			collector.collectNetworkPortMetricFromNetworkAdapter(ch, adapter)
			collector.collectNetworkAdapterStatus(ch, adapter)
		}
	}
}

func (collector SystemCollector) collectNetworkPortMetricFromNetworkAdapter(ch chan<- prometheus.Metric,
	adapter *redfish.NetworkAdapter) {
	networkPorts, err := adapter.NetworkPorts()
	netState := map[string]float64{"Up": 0.0, "Down": 1.0}

	if nil != err {
		panic(err)
	}

	for _, networkPort := range networkPorts {
		stateString := fmt.Sprintf("%v", networkPort.LinkStatus)
		status := netState[stateString]
		ch <- prometheus.MustNewConstMetric(config.S_networkport,
			prometheus.GaugeValue,
			status,
			adapter.Manufacturer,
			fmt.Sprintf("%v", networkPort.LinkStatus),
			fmt.Sprintf("%v", networkPort.CurrentLinkSpeedMbps),
			networkPort.Description,
			fmt.Sprintf("%v", networkPort.MaxFrameSize),
			fmt.Sprintf("%v", networkPort.NumberDiscoveredRemotePorts),
			networkPort.PhysicalPortNumber,
			fmt.Sprintf("%v", networkPort.PortMaximumMTU),
		)
	}
}

func (collector SystemCollector) collectNetworkAdapterStatus(ch chan<- prometheus.Metric,
	adapter *redfish.NetworkAdapter) {
	controllers := adapter.Controllers

	if 0 != len(controllers) {
		for _, control := range controllers {
			ch <- prometheus.MustNewConstMetric(config.S_network_adapter_status,
				prometheus.GaugeValue,
				float64(0),
				adapter.Manufacturer,
				control.FirmwarePackageVersion,
				fmt.Sprintf("%v", control.NetworkDeviceFunctionsCount),
				fmt.Sprintf("%v", control.NetworkPortsCount),
			)
		}
	}
}

func (collector SystemCollector) collectIDRACPortStatus(ch chan<- prometheus.Metric, pd *redfishstruct.IDRACPort) {
	var pds redfishstruct.IDRACPort
	err, idracports := pds.UnmarshalJson("/redfish/v1/Managers/1/EthernetInterfaces/1/")

	if err != nil {
		panic(err)
	}
	idrac_temp := string(idracports.Status.Health)
	idrac_temp1 := 0.0
	if idrac_temp == "OK" {
		idrac_temp1 = 0
	} else if idrac_temp == "WARNING" {
		idrac_temp1 = 1
	} else {
		idrac_temp1 = 2
	}
	//status := config.State_dict[string(iloports.Status.Health)]
	ch <- prometheus.MustNewConstMetric(config.S_idrac_status,
		prometheus.GaugeValue,
		float64(idrac_temp1),
		fmt.Sprintf("%v", idracports.ID),
		fmt.Sprintf("%v", idracports.AutoNeg),
		fmt.Sprintf("%v", idracports.FullDuplex),
		fmt.Sprintf("%v", idracports.HostName),
		fmt.Sprintf("%v", idracports.IPv4Addresses[0].Address),
		fmt.Sprintf("%v", idracports.IPv4Addresses[0].AddressOrigin),
		fmt.Sprintf("%v", idracports.IPv4Addresses[0].Gateway),
		fmt.Sprintf("%v", idracports.IPv4Addresses[0].SubnetMask),
		fmt.Sprintf("%v", idracports.MTUSize),
		fmt.Sprintf("%v", idracports.SpeedMbps),
		fmt.Sprintf("%v", idracports.Status.Health),
		fmt.Sprintf("%v", idracports.Status.State),
		fmt.Sprintf("%v", idracports.VLAN.VLANEnable),
		fmt.Sprintf("%v", idracports.VLAN.VLANId),
	)

}
