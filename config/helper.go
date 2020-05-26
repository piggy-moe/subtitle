package config

import (
	"encoding/json"
	"fmt"
	"image/color"
)

func diff(x, y uint8) int {
	if x >= y {
		return int(x - y)
	}
	return int(y - x)
}

type RelativeValue struct {
	Ratio  float64
	Offset int
}

func (rv RelativeValue) Calculate(base int) int {
	return int(float64(base)*rv.Ratio) + rv.Offset
}

func (rv RelativeValue) Equal(ratio float64, offset int) bool {
	return rv.Ratio == ratio && rv.Offset == offset
}

func MustNewRelativeValue(s string) RelativeValue {
	rv := new(RelativeValue)
	if err := rv.Assign(s); err != nil {
		panic(err)
	}
	return *rv
}

func (rv *RelativeValue) Assign(s string) error {
	if _, err := fmt.Sscanf(s, "%f%%%d", &rv.Ratio, &rv.Offset); err != nil {
		return fmt.Errorf("unable to assign %s to relative value: %w", s, err)
	}
	rv.Ratio /= 100.0
	return nil
}

func (rv *RelativeValue) String() string {
	return fmt.Sprintf("%g%%%+d", rv.Ratio*100.0, rv.Offset)
}

func (rv *RelativeValue) UnmarshalJSON(b []byte) error {
	var s string
	json.Unmarshal(b, &s)
	return rv.Assign(s)
}

func (rv RelativeValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(rv.String())
}

type ColorGroup struct {
	R, G, B uint8
	Error   int
	Color   color.RGBA
}

func (cg *ColorGroup) Contains(c color.RGBA) bool {
	return (diff(cg.R, c.R) + diff(cg.G, c.G) + diff(cg.B, c.B)) <= cg.Error*3
}

func MustNewColorGroup(s string) ColorGroup {
	cg := new(ColorGroup)
	if err := cg.Assign(s); err != nil {
		panic(err)
	}
	return *cg
}

func (cg *ColorGroup) Assign(s string) error {
	var err error
	if len(s) == len("#rrggbb") {
		_, err = fmt.Sscanf(s, "#%02x%02x%02x", &cg.R, &cg.G, &cg.B)
	} else {
		_, err = fmt.Sscanf(s, "#%02x%02x%02x/%d", &cg.R, &cg.G, &cg.B, &cg.Error)
	}
	if err != nil {
		return fmt.Errorf("unable to assign %s to color group: %w", s, err)
	}
	cg.Color = color.RGBA{uint8(cg.R), uint8(cg.G), uint8(cg.B), 0}
	return nil
}

func (cg *ColorGroup) String() string {
	if cg.Error == 0 {
		return fmt.Sprintf("#%02x%02x%02x", cg.R, cg.G, cg.B)
	}
	return fmt.Sprintf("#%02x%02x%02x/%d", cg.R, cg.G, cg.B, cg.Error)
}

func (cg ColorGroup) MarshalJSON() ([]byte, error) {
	return json.Marshal(cg.String())
}

func (cg *ColorGroup) UnmarshalJSON(b []byte) error {
	var s string
	json.Unmarshal(b, &s)
	return cg.Assign(s)
}

type Range struct {
	Min RelativeValue `json:"min"`
	Max RelativeValue `json:"max"`
}

func MustNewRange(min, max string) Range {
	return Range{MustNewRelativeValue(min), MustNewRelativeValue(max)}
}

type Area struct {
	Left   RelativeValue `json:"left"`
	Right  RelativeValue `json:"right"`
	Top    RelativeValue `json:"top"`
	Bottom RelativeValue `json:"bottom"`
}
