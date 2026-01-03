package regelverk

import (
	"bytes"
	"encoding/json"
	"errors"
)

type Z2MDevices []Z2MDevice

func UnmarshalZ2MDevices(data []byte) (Z2MDevices, error) {
	var r Z2MDevices
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *Z2MDevices) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type Z2MDevice struct {
	Disabled           bool                        `json:"disabled"`
	Endpoints          map[string]Z2MEndpointValue `json:"endpoints"`
	FriendlyName       string                      `json:"friendly_name"`
	IEEEAddress        string                      `json:"ieee_address"`
	InterviewCompleted bool                        `json:"interview_completed"`
	InterviewState     Z2MInterviewState           `json:"interview_state"`
	Interviewing       bool                        `json:"interviewing"`
	NetworkAddress     int64                       `json:"network_address"`
	Supported          bool                        `json:"supported"`
	Type               Z2MDeviceType               `json:"type"`
	DateCode           *string                     `json:"date_code,omitempty"`
	Definition         *Z2MDefinition              `json:"definition,omitempty"`
	Manufacturer       *Z2MManufacturer            `json:"manufacturer,omitempty"`
	ModelID            *string                     `json:"model_id,omitempty"`
	PowerSource        *Z2MPowerSource             `json:"power_source,omitempty"`
	SoftwareBuildID    *string                     `json:"software_build_id,omitempty"`
}

type Z2MDefinition struct {
	Description string      `json:"description"`
	Exposes     []Z2MExpose `json:"exposes"`
	Model       string      `json:"model"`
	Options     []Z2MOption `json:"options"`
	Source      Z2MSource   `json:"source"`
	SupportsOta bool        `json:"supports_ota"`
	Vendor      string      `json:"vendor"`
}

type Z2MExpose struct {
	Features    []Z2MFeature   `json:"features,omitempty"`
	Type        Z2MFeatureType `json:"type"`
	Access      *int64         `json:"access,omitempty"`
	Description *string        `json:"description,omitempty"`
	Label       *string        `json:"label,omitempty"`
	Name        *string        `json:"name,omitempty"`
	Property    *string        `json:"property,omitempty"`
	Unit        *string        `json:"unit,omitempty"`
	ValueMax    *int64         `json:"value_max,omitempty"`
	ValueMin    *int64         `json:"value_min,omitempty"`
	ValueStep   *int64         `json:"value_step,omitempty"`
	Values      []string       `json:"values,omitempty"`
	ValueOff    *ValueO        `json:"value_off"`
	ValueOn     *ValueO        `json:"value_on"`
	Category    *Z2MCategory   `json:"category,omitempty"`
}

type Z2MFeature struct {
	Access      int64          `json:"access"`
	Description *string        `json:"description,omitempty"`
	Label       string         `json:"label"`
	Name        string         `json:"name"`
	Property    string         `json:"property"`
	Type        Z2MFeatureType `json:"type"`
	ValueOff    *string        `json:"value_off,omitempty"`
	ValueOn     *string        `json:"value_on,omitempty"`
	ValueToggle *string        `json:"value_toggle,omitempty"`
	Values      []string       `json:"values,omitempty"`
	Unit        *string        `json:"unit,omitempty"`
	ValueMax    *int64         `json:"value_max,omitempty"`
	ValueMin    *int64         `json:"value_min,omitempty"`
}

type Z2MOption struct {
	Access      int64         `json:"access"`
	Description string        `json:"description"`
	Label       string        `json:"label"`
	Name        string        `json:"name"`
	Property    string        `json:"property"`
	Type        Z2MOptionType `json:"type"`
	ValueStep   *float64      `json:"value_step,omitempty"`
	ValueMax    *int64        `json:"value_max,omitempty"`
	ValueMin    *int64        `json:"value_min,omitempty"`
	ValueOff    *bool         `json:"value_off,omitempty"`
	ValueOn     *bool         `json:"value_on,omitempty"`
	ItemType    *ItemType     `json:"item_type,omitempty"`
}

type ItemType struct {
	Access int64          `json:"access"`
	Label  string         `json:"label"`
	Name   string         `json:"name"`
	Type   Z2MFeatureType `json:"type"`
}

type Z2MEndpointValue struct {
	Bindings             []Z2MBinding             `json:"bindings"`
	Clusters             Z2MClusters              `json:"clusters"`
	ConfiguredReportings []Z2MConfiguredReporting `json:"configured_reportings"`
	Scenes               []interface{}            `json:"scenes"`
	Name                 *string                  `json:"name,omitempty"`
}

type Z2MBinding struct {
	Cluster string    `json:"cluster"`
	Target  Z2MTarget `json:"target"`
}

type Z2MTarget struct {
	Endpoint    *int64        `json:"endpoint,omitempty"`
	IEEEAddress *IEEEAddress  `json:"ieee_address,omitempty"`
	Type        Z2MTargetType `json:"type"`
	ID          *int64        `json:"id,omitempty"`
}

type Z2MClusters struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

type Z2MConfiguredReporting struct {
	Attribute             string            `json:"attribute"`
	Cluster               string            `json:"cluster"`
	MaximumReportInterval int64             `json:"maximum_report_interval"`
	MinimumReportInterval int64             `json:"minimum_report_interval"`
	ReportableChange      *ReportableChange `json:"reportable_change"`
}

type Z2MCategory string

const (
	Z2MConfig     Z2MCategory = "config"
	Diagnostic Z2MCategory = "diagnostic"
)

type Z2MFeatureType string

const (
	Cover         Z2MFeatureType = "cover"
	Enum          Z2MFeatureType = "enum"
	PurpleBinary  Z2MFeatureType = "binary"
	PurpleNumeric Z2MFeatureType = "numeric"
	Switch        Z2MFeatureType = "switch"
)

type Z2MOptionType string

const (
	FluffyBinary  Z2MOptionType = "binary"
	FluffyNumeric Z2MOptionType = "numeric"
	List          Z2MOptionType = "list"
)

type Z2MSource string

const (
	Native Z2MSource = "native"
)

type IEEEAddress string

const (
	The0X00124B0029E8B7F6 IEEEAddress = "0x00124b0029e8b7f6"
)

type Z2MTargetType string

const (
	Endpoint Z2MTargetType = "endpoint"
	Group    Z2MTargetType = "group"
)

type Z2MInterviewState string

const (
	Successful Z2MInterviewState = "SUCCESSFUL"
)

type Z2MManufacturer string

const (
	IKEAOfSweden   Z2MManufacturer = "IKEA of Sweden"
	Lumi           Z2MManufacturer = "LUMI"
	TZ3000Gjnozsaz Z2MManufacturer = "_TZ3000_gjnozsaz"
)

type Z2MPowerSource string

const (
	Battery          Z2MPowerSource = "Battery"
	MainsSinglePhase Z2MPowerSource = "Mains (single phase)"
)

type Z2MDeviceType string

const (
	Coordinator Z2MDeviceType = "Coordinator"
	EndDevice   Z2MDeviceType = "EndDevice"
	Router      Z2MDeviceType = "Router"
)

type ValueO struct {
	Bool   *bool
	String *string
}

func (x *ValueO) UnmarshalJSON(data []byte) error {
	object, err := unmarshalUnion(data, nil, nil, &x.Bool, &x.String, false, nil, false, nil, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
	}
	return nil
}

func (x *ValueO) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, x.Bool, x.String, false, nil, false, nil, false, nil, false, nil, false)
}

type ReportableChange struct {
	Integer      *int64
	IntegerArray []int64
}

func (x *ReportableChange) UnmarshalJSON(data []byte) error {
	x.IntegerArray = nil
	object, err := unmarshalUnion(data, &x.Integer, nil, nil, nil, true, &x.IntegerArray, false, nil, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
	}
	return nil
}

func (x *ReportableChange) MarshalJSON() ([]byte, error) {
	return marshalUnion(x.Integer, nil, nil, nil, x.IntegerArray != nil, x.IntegerArray, false, nil, false, nil, false, nil, false)
}

func unmarshalUnion(data []byte, pi **int64, pf **float64, pb **bool, ps **string, haveArray bool, pa interface{}, haveObject bool, pc interface{}, haveMap bool, pm interface{}, haveEnum bool, pe interface{}, nullable bool) (bool, error) {
	if pi != nil {
		*pi = nil
	}
	if pf != nil {
		*pf = nil
	}
	if pb != nil {
		*pb = nil
	}
	if ps != nil {
		*ps = nil
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return false, err
	}

	switch v := tok.(type) {
	case json.Number:
		if pi != nil {
			i, err := v.Int64()
			if err == nil {
				*pi = &i
				return false, nil
			}
		}
		if pf != nil {
			f, err := v.Float64()
			if err == nil {
				*pf = &f
				return false, nil
			}
			return false, errors.New("Unparsable number")
		}
		return false, errors.New("Union does not contain number")
	case float64:
		return false, errors.New("Decoder should not return float64")
	case bool:
		if pb != nil {
			*pb = &v
			return false, nil
		}
		return false, errors.New("Union does not contain bool")
	case string:
		if haveEnum {
			return false, json.Unmarshal(data, pe)
		}
		if ps != nil {
			*ps = &v
			return false, nil
		}
		return false, errors.New("Union does not contain string")
	case nil:
		if nullable {
			return false, nil
		}
		return false, errors.New("Union does not contain null")
	case json.Delim:
		if v == '{' {
			if haveObject {
				return true, json.Unmarshal(data, pc)
			}
			if haveMap {
				return false, json.Unmarshal(data, pm)
			}
			return false, errors.New("Union does not contain object")
		}
		if v == '[' {
			if haveArray {
				return false, json.Unmarshal(data, pa)
			}
			return false, errors.New("Union does not contain array")
		}
		return false, errors.New("Cannot handle delimiter")
	}
	return false, errors.New("Cannot unmarshal union")

}

func marshalUnion(pi *int64, pf *float64, pb *bool, ps *string, haveArray bool, pa interface{}, haveObject bool, pc interface{}, haveMap bool, pm interface{}, haveEnum bool, pe interface{}, nullable bool) ([]byte, error) {
	if pi != nil {
		return json.Marshal(*pi)
	}
	if pf != nil {
		return json.Marshal(*pf)
	}
	if pb != nil {
		return json.Marshal(*pb)
	}
	if ps != nil {
		return json.Marshal(*ps)
	}
	if haveArray {
		return json.Marshal(pa)
	}
	if haveObject {
		return json.Marshal(pc)
	}
	if haveMap {
		return json.Marshal(pm)
	}
	if haveEnum {
		return json.Marshal(pe)
	}
	if nullable {
		return json.Marshal(nil)
	}
	return nil, errors.New("Union must not be null")
}
