// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    devices, err := UnmarshalDevices(bytes)
//    bytes, err = devices.Marshal()

package z2m

import (
	"bytes"
	"encoding/json"
	"errors"
)

type Devices []Device

func UnmarshalDevices(data []byte) (Devices, error) {
	var r Devices
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *Devices) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type Device struct {
	Disabled           bool                     `json:"disabled"`
	Endpoints          map[string]EndpointValue `json:"endpoints"`
	FriendlyName       string                   `json:"friendly_name"`
	IEEEAddress        string                   `json:"ieee_address"`
	InterviewCompleted bool                     `json:"interview_completed"`
	InterviewState     InterviewState           `json:"interview_state"`
	Interviewing       bool                     `json:"interviewing"`
	NetworkAddress     int64                    `json:"network_address"`
	Supported          bool                     `json:"supported"`
	Type               DeviceType               `json:"type"`
	DateCode           *string                  `json:"date_code,omitempty"`
	Definition         *Definition              `json:"definition,omitempty"`
	Manufacturer       *Manufacturer            `json:"manufacturer,omitempty"`
	ModelID            *string                  `json:"model_id,omitempty"`
	PowerSource        *PowerSource             `json:"power_source,omitempty"`
	SoftwareBuildID    *string                  `json:"software_build_id,omitempty"`
}

type Definition struct {
	Description string   `json:"description"`
	Exposes     []Expose `json:"exposes"`
	Model       string   `json:"model"`
	Options     []Option `json:"options"`
	Source      Source   `json:"source"`
	SupportsOta bool     `json:"supports_ota"`
	Vendor      string   `json:"vendor"`
}

type Expose struct {
	Features    []Feature   `json:"features,omitempty"`
	Type        FeatureType `json:"type"`
	Access      *int64      `json:"access,omitempty"`
	Description *string     `json:"description,omitempty"`
	Label       *string     `json:"label,omitempty"`
	Name        *string     `json:"name,omitempty"`
	Property    *string     `json:"property,omitempty"`
	Unit        *string     `json:"unit,omitempty"`
	ValueMax    *int64      `json:"value_max,omitempty"`
	ValueMin    *int64      `json:"value_min,omitempty"`
	ValueStep   *int64      `json:"value_step,omitempty"`
	Values      []string    `json:"values,omitempty"`
	ValueOff    *ValueO     `json:"value_off"`
	ValueOn     *ValueO     `json:"value_on"`
	Category    *Category   `json:"category,omitempty"`
}

type Feature struct {
	Access      int64       `json:"access"`
	Description *string     `json:"description,omitempty"`
	Label       string      `json:"label"`
	Name        string      `json:"name"`
	Property    string      `json:"property"`
	Type        FeatureType `json:"type"`
	ValueOff    *string     `json:"value_off,omitempty"`
	ValueOn     *string     `json:"value_on,omitempty"`
	ValueToggle *string     `json:"value_toggle,omitempty"`
	Values      []string    `json:"values,omitempty"`
	Unit        *string     `json:"unit,omitempty"`
	ValueMax    *int64      `json:"value_max,omitempty"`
	ValueMin    *int64      `json:"value_min,omitempty"`
}

type Option struct {
	Access      int64      `json:"access"`
	Description string     `json:"description"`
	Label       string     `json:"label"`
	Name        string     `json:"name"`
	Property    string     `json:"property"`
	Type        OptionType `json:"type"`
	ValueStep   *float64   `json:"value_step,omitempty"`
	ValueMax    *int64     `json:"value_max,omitempty"`
	ValueMin    *int64     `json:"value_min,omitempty"`
	ValueOff    *bool      `json:"value_off,omitempty"`
	ValueOn     *bool      `json:"value_on,omitempty"`
	ItemType    *ItemType  `json:"item_type,omitempty"`
}

type ItemType struct {
	Access int64       `json:"access"`
	Label  string      `json:"label"`
	Name   string      `json:"name"`
	Type   FeatureType `json:"type"`
}

type EndpointValue struct {
	Bindings             []Binding             `json:"bindings"`
	Clusters             Clusters              `json:"clusters"`
	ConfiguredReportings []ConfiguredReporting `json:"configured_reportings"`
	Scenes               []interface{}         `json:"scenes"`
	Name                 *string               `json:"name,omitempty"`
}

type Binding struct {
	Cluster string `json:"cluster"`
	Target  Target `json:"target"`
}

type Target struct {
	Endpoint    *int64       `json:"endpoint,omitempty"`
	IEEEAddress *IEEEAddress `json:"ieee_address,omitempty"`
	Type        TargetType   `json:"type"`
	ID          *int64       `json:"id,omitempty"`
}

type Clusters struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

type ConfiguredReporting struct {
	Attribute             string            `json:"attribute"`
	Cluster               string            `json:"cluster"`
	MaximumReportInterval int64             `json:"maximum_report_interval"`
	MinimumReportInterval int64             `json:"minimum_report_interval"`
	ReportableChange      *ReportableChange `json:"reportable_change"`
}

type Category string

const (
	Config     Category = "config"
	Diagnostic Category = "diagnostic"
)

type FeatureType string

const (
	Cover         FeatureType = "cover"
	Enum          FeatureType = "enum"
	PurpleBinary  FeatureType = "binary"
	PurpleNumeric FeatureType = "numeric"
	Switch        FeatureType = "switch"
)

type OptionType string

const (
	FluffyBinary  OptionType = "binary"
	FluffyNumeric OptionType = "numeric"
	List          OptionType = "list"
)

type Source string

const (
	Native Source = "native"
)

type IEEEAddress string

const (
	The0X00124B0029E8B7F6 IEEEAddress = "0x00124b0029e8b7f6"
)

type TargetType string

const (
	Endpoint TargetType = "endpoint"
	Group    TargetType = "group"
)

type InterviewState string

const (
	Successful InterviewState = "SUCCESSFUL"
)

type Manufacturer string

const (
	IKEAOfSweden   Manufacturer = "IKEA of Sweden"
	Lumi           Manufacturer = "LUMI"
	TZ3000Gjnozsaz Manufacturer = "_TZ3000_gjnozsaz"
)

type PowerSource string

const (
	Battery          PowerSource = "Battery"
	MainsSinglePhase PowerSource = "Mains (single phase)"
)

type DeviceType string

const (
	Coordinator DeviceType = "Coordinator"
	EndDevice   DeviceType = "EndDevice"
	Router      DeviceType = "Router"
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
