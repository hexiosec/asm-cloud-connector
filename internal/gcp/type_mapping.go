package gcp

import "fmt"

type ValidationErr struct {
	Err error
}

func (e *ValidationErr) Error() string {
	return fmt.Sprintf("gcp: failed to decode data, %s", e.Err.Error())
}

func (e *ValidationErr) Unwrap() error {
	return e.Err
}

/*-----------------------------------------------------------
These are minimal type definitions for each asset/resource type.

The Cloud Asset Inventory API returns assets as untyped, generic
structures, and the fields do not line up cleanly with the types
exposed by the dedicated service APIs.
https://docs.cloud.google.com/asset-inventory/docs/reference/rest/v1/Asset#resource

To avoid relying on loosely typed maps and to give us predictable
field shapes, we define the specific structs we need here.
-----------------------------------------------------------*/

type instance struct {
	NetworkInterfaces []*struct {
		AccessConfigs []*struct {
			NatIP *string `mapstructure:"natIP"`
		} `mapstructure:"accessConfigs"`
	} `mapstructure:"networkInterfaces"`
}

type resourceRecordSet struct {
	Name    *string   `mapstructure:"name"`
	Type    *string   `mapstructure:"type"`
	RRDatas []*string `mapstructure:"rrdatas"`
}

type address struct {
	Address *string `mapstructure:"address"`
	Type    *string `mapstructure:"type"`
}

type function struct {
	HTTPSTrigger *struct {
		URL *string `mapstructure:"url"`
	} `mapstructure:"httpsTrigger"`
}

type forwardingRule struct {
	IPAddress           *string `mapstructure:"IPAddress"`
	LoadBalancingScheme *string `mapstructure:"loadBalancingScheme"`
}

type service struct {
	Status *struct {
		URL *string `mapstructure:"url"`
	} `mapstructure:"status"`
}

type domainMapping struct {
	Metadata *struct {
		Name *string `mapstructure:"name"`
	} `mapstructure:"metadata"`
}

type sqlInstance struct {
	IPAddresses []*struct {
		IPAddress *string `mapstructure:"ipAddress"`
		Type      *string `mapstructure:"type"`
	} `mapstructure:"ipAddresses"`
}

type urlMap struct {
	HostRules []*struct {
		Hosts []*string `mapstructure:"hosts"`
	} `mapstructure:"hostRules"`
}

type cluster struct {
	Endpoint             *string `mapstructure:"endpoint"`
	PrivateClusterConfig *struct {
		EnablePrivateEndpoint *bool `mapstructure:"enablePrivateEndpoint"`
	} `mapstructure:"privateClusterConfig"`
}
