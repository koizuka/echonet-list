package props

import "fmt"

type Illuminance uint8 // 0-100%

func DecodeIlluminance(EDT []byte) *Illuminance {
	if len(EDT) < 1 {
		return nil
	}
	illuminance := Illuminance(EDT[0])
	return &illuminance
}

func (i *Illuminance) String() string {
	if i == nil {
		return "nil"
	}
	return fmt.Sprintf("%d%%", *i)
}

func (i *Illuminance) EDT() []byte {
	return []byte{byte(*i)}
}
