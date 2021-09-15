package redfishstruct

import (
	"encoding/json"
	"io/ioutil"

	"idrac-exporter/config"
)

type Ipv4Address struct {
	Address       string `json:"address"`
	AddressOrigin string `json:"addressorigin"`
	Gateway       string `json:"gateway"`
	SubnetMask    string `json:"subnetmask"`
}

type VLAN struct {
	VLANEnable bool `json:"vlanenable"`
	VLANId     int  `json:"vlanid"`
}

type Status struct {
	Health string `json:"health"`
	State  string `json:"state"`
}

type IDRACPort struct {
	ODataID       string        `json:"@odata.id"`
	ODataType     string        `json:"@odata.type"`
	AutoNeg       bool          `json:"autoneg"`
	Description   string        `json:"description"`
	ID            string        `json:"id"`
	FullDuplex    bool          `json:"fullduplex"`
	HostName      string        `json:"hostname"`
	IPv4Addresses []Ipv4Address `json:"ipv4addresses"`
	LinkStatus    string        `json:"linkstatus"`
	MTUSize       int           `json:"mtusize"`
	MACAddress    string        `json:"macaddress"`
	Name          string        `json:"name"`
	SpeedMbps     int           `json:"speedmbps"`
	Status        Status        `json:"status"`
	VLAN          VLAN          `json:"vlan"`
}

func (idracport *IDRACPort) UnmarshalJson(str string) (error, *IDRACPort) {
	t, _ := config.GOFISH.Get(str)
	bodyBytes, _ := ioutil.ReadAll(t.Body)

	var temp IDRACPort

	err := json.Unmarshal(bodyBytes, &temp)
	if err != nil {
		panic(err)
	}
	return nil, &temp
}
