package jwt

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"
)

// TimePrecision sets the precision of times and dates within this library.
// This has an influence on the precision of times when comparing expiry or
// other related time fields. Furthermore, it is also the precision of times
// when serializing.
//
// For backwards compatibility the default precision is set to seconds, so that
// no fractional timestamps are generated.
var TimePrecision = time.Second

// MarshalSingleStringAsArray modifies the behaviour of the ClaimStrings type, especially
// its MarshalJSON function.
//
// If it is set to true (the default), it will always serialize the type as an
// array of strings, even if it just contains one element, defaulting to the behaviour
// of the underlying []string. If it is set to false, it will serialize to a single
// string, if it contains one element. Otherwise, it will serialize to an array of strings.
var MarshalSingleStringAsArray = true

// NumericDate represents a JSON numeric date value, as referenced at
// https://datatracker.ietf.org/doc/html/rfc7519#section-2.
type NumericDate struct {
	time.Time
}

// NewNumericDate constructs a new *NumericDate from a standard library time.Time struct.
// It will truncate the timestamp according to the precision specified in TimePrecision.
func NewNumericDate(t time.Time) *NumericDate {
	return &NumericDate{t.Truncate(TimePrecision)}
}

// newNumericDateFromSeconds creates a new *NumericDate out of a float64 representing a
// UNIX epoch with the float fraction representing non-integer seconds.
func newNumericDateFromSeconds(f float64) *NumericDate {
	round, frac := math.Modf(f)
	return NewNumericDate(time.Unix(int64(round), int64(frac*1e9)))
}

// MarshalJSON is an implementation of the json.RawMessage interface and serializes the UNIX epoch
// represented in NumericDate to a byte array, using the precision specified in TimePrecision.
func (date NumericDate) MarshalJSON() (b []byte, err error) {
	prec := -1
	if TimePrecision == time.Second {
		prec = 0
	}
	f := float64(date.Truncate(TimePrecision).UnixNano()) / float64(time.Second)

	return []byte(strconv.FormatFloat(f, 'f', prec, 64)), nil
}

// UnmarshalJSON is an implementation of the json.RawMessage interface and deserializses a
// NumericDate from a JSON representation, i.e. a json.Number. This number represents an UNIX epoch
// with either integer or non-integer seconds.
func (date *NumericDate) UnmarshalJSON(b []byte) (err error) {
	var (
		number json.Number
		f      float64
	)

	if err = json.Unmarshal(b, &number); err != nil {
		return fmt.Errorf("could not parse NumericData: %w", err)
	}

	if f, err = number.Float64(); err != nil {
		return fmt.Errorf("could not convert json number value to float: %w", err)
	}

	n := newNumericDateFromSeconds(f)
	*date = *n

	return nil
}

// ClaimStrings is basically just a slice of strings, but it can be either serialized from a string array or just a string.
// This type is necessary, since the "aud" claim can either be a single string or an array.
type ClaimStrings []string

func (s *ClaimStrings) UnmarshalJSON(data []byte) (err error) {
	var value interface{}

	if err = json.Unmarshal(data, &value); err != nil {
		return err
	}

	var aud []string

	switch v := value.(type) {
	case string:
		aud = append(aud, v)
	case []string:
		aud = ClaimStrings(v)
	case []interface{}:
		for _, vv := range v {
			vs, ok := vv.(string)
			if !ok {
				return &json.UnsupportedTypeError{Type: reflect.TypeOf(vv)}
			}
			aud = append(aud, vs)
		}
	case nil:
		return nil
	default:
		return &json.UnsupportedTypeError{Type: reflect.TypeOf(v)}
	}

	*s = aud

	return
}

func (s ClaimStrings) MarshalJSON() (b []byte, err error) {
	// This handles a special case in the JWT RFC. If the string array, e.g. used by the "aud" field,
	// only contains one element, it MAY be serialized as a single string. This may or may not be
	// desired based on the ecosystem of other JWT library used, so we make it configurable by the
	// variable MarshalSingleStringAsArray.
	if len(s) == 1 && !MarshalSingleStringAsArray {
		return json.Marshal(s[0])
	}

	return json.Marshal([]string(s))
}
