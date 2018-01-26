// Code generated by protoc-gen-go.
// source: phonemetadata.proto
// DO NOT EDIT!

/*
Package i18n_phonenumbers is a generated protocol buffer package.

It is generated from these files:
	phonemetadata.proto
	phonenumber.proto

It has these top-level messages:
	NumberFormat
	PhoneNumberDesc
	PhoneMetadata
	PhoneMetadataCollection
	PhoneNumber
*/
package phonenumbers

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type NumberFormat struct {
	// pattern is a regex that is used to match the national (significant)
	// number. For example, the pattern "(20)(\d{4})(\d{4})" will match number
	// "2070313000", which is the national (significant) number for Google London.
	// Note the presence of the parentheses, which are capturing groups what
	// specifies the grouping of numbers.
	Pattern *string `protobuf:"bytes,1,req,name=pattern" json:"pattern,omitempty"`
	// format specifies how the national (significant) number matched by
	// pattern should be formatted.
	// Using the same example as above, format could contain "$1 $2 $3",
	// meaning that the number should be formatted as "20 7031 3000".
	// Each $x are replaced by the numbers captured by group x in the
	// regex specified by pattern.
	Format *string `protobuf:"bytes,2,req,name=format" json:"format,omitempty"`
	// This field is a regex that is used to match a certain number of digits
	// at the beginning of the national (significant) number. When the match is
	// successful, the accompanying pattern and format should be used to format
	// this number. For example, if leading_digits="[1-3]|44", then all the
	// national numbers starting with 1, 2, 3 or 44 should be formatted using the
	// accompanying pattern and format.
	//
	// The first leadingDigitsPattern matches up to the first three digits of the
	// national (significant) number; the next one matches the first four digits,
	// then the first five and so on, until the leadingDigitsPattern can uniquely
	// identify one pattern and format to be used to format the number.
	//
	// In the case when only one formatting pattern exists, no
	// leading_digits_pattern is needed.
	LeadingDigitsPattern []string `protobuf:"bytes,3,rep,name=leading_digits_pattern,json=leadingDigitsPattern" json:"leading_digits_pattern,omitempty"`
	// This field specifies how the national prefix ($NP) together with the first
	// group ($FG) in the national significant number should be formatted in
	// the NATIONAL format when a national prefix exists for a certain country.
	// For example, when this field contains "($NP$FG)", a number from Beijing,
	// China (whose $NP = 0), which would by default be formatted without
	// national prefix as 10 1234 5678 in NATIONAL format, will instead be
	// formatted as (010) 1234 5678; to format it as (0)10 1234 5678, the field
	// would contain "($NP)$FG". Note $FG should always be present in this field,
	// but $NP can be omitted. For example, having "$FG" could indicate the
	// number should be formatted in NATIONAL format without the national prefix.
	// This is commonly used to override the rule specified for the territory in
	// the XML file.
	//
	// When this field is missing, a number will be formatted without national
	// prefix in NATIONAL format. This field does not affect how a number
	// is formatted in other formats, such as INTERNATIONAL.
	NationalPrefixFormattingRule *string `protobuf:"bytes,4,opt,name=national_prefix_formatting_rule,json=nationalPrefixFormattingRule" json:"national_prefix_formatting_rule,omitempty"`
	// This field specifies whether the $NP can be omitted when formatting a
	// number in national format, even though it usually wouldn't be. For example,
	// a UK number would be formatted by our library as 020 XXXX XXXX. If we have
	// commonly seen this number written by people without the leading 0, for
	// example as (20) XXXX XXXX, this field would be set to true. This will be
	// inherited from the value set for the territory in the XML file, unless a
	// national_prefix_optional_when_formatting is defined specifically for this
	// NumberFormat.
	NationalPrefixOptionalWhenFormatting *bool `protobuf:"varint,6,opt,name=national_prefix_optional_when_formatting,json=nationalPrefixOptionalWhenFormatting,def=0" json:"national_prefix_optional_when_formatting,omitempty"`
	// This field specifies how any carrier code ($CC) together with the first
	// group ($FG) in the national significant number should be formatted
	// when formatWithCarrierCode is called, if carrier codes are used for a
	// certain country.
	DomesticCarrierCodeFormattingRule *string `protobuf:"bytes,5,opt,name=domestic_carrier_code_formatting_rule,json=domesticCarrierCodeFormattingRule" json:"domestic_carrier_code_formatting_rule,omitempty"`
	XXX_unrecognized                  []byte  `json:"-"`
}

func (m *NumberFormat) Reset()                    { *m = NumberFormat{} }
func (m *NumberFormat) String() string            { return proto.CompactTextString(m) }
func (*NumberFormat) ProtoMessage()               {}
func (*NumberFormat) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

const Default_NumberFormat_NationalPrefixOptionalWhenFormatting bool = false

func (m *NumberFormat) GetPattern() string {
	if m != nil && m.Pattern != nil {
		return *m.Pattern
	}
	return ""
}

func (m *NumberFormat) GetFormat() string {
	if m != nil && m.Format != nil {
		return *m.Format
	}
	return ""
}

func (m *NumberFormat) GetLeadingDigitsPattern() []string {
	if m != nil {
		return m.LeadingDigitsPattern
	}
	return nil
}

func (m *NumberFormat) GetNationalPrefixFormattingRule() string {
	if m != nil && m.NationalPrefixFormattingRule != nil {
		return *m.NationalPrefixFormattingRule
	}
	return ""
}

func (m *NumberFormat) GetNationalPrefixOptionalWhenFormatting() bool {
	if m != nil && m.NationalPrefixOptionalWhenFormatting != nil {
		return *m.NationalPrefixOptionalWhenFormatting
	}
	return Default_NumberFormat_NationalPrefixOptionalWhenFormatting
}

func (m *NumberFormat) GetDomesticCarrierCodeFormattingRule() string {
	if m != nil && m.DomesticCarrierCodeFormattingRule != nil {
		return *m.DomesticCarrierCodeFormattingRule
	}
	return ""
}

// If you add, remove, or rename fields, or change their semantics, check if you
// should change the excludable field sets or the behavior in MetadataFilter.
type PhoneNumberDesc struct {
	// The national_number_pattern is the pattern that a valid national
	// significant number would match. This specifies information such as its
	// total length and leading digits.
	NationalNumberPattern *string `protobuf:"bytes,2,opt,name=national_number_pattern,json=nationalNumberPattern" json:"national_number_pattern,omitempty"`
	// The possible_number_pattern represents what a potentially valid phone
	// number for this region may be written as. This is a superset of the
	// national_number_pattern above and includes numbers that have the area code
	// omitted. Typically the only restrictions here are in the number of digits.
	// This could be used to highlight tokens in a text that may be a phone
	// number, or to quickly prune numbers that could not possibly be a phone
	// number for this locale.
	PossibleNumberPattern *string `protobuf:"bytes,3,opt,name=possible_number_pattern,json=possibleNumberPattern" json:"possible_number_pattern,omitempty"`
	// These represent the lengths a phone number from this region can be. They
	// will be sorted from smallest to biggest. Note that these lengths are for
	// the full number, without country calling code or national prefix. For
	// example, for the Swiss number +41789270000, in local format 0789270000,
	// this would be 9.
	// This could be used to highlight tokens in a text that may be a phone
	// number, or to quickly prune numbers that could not possibly be a phone
	// number for this locale.
	PossibleLength []int32 `protobuf:"varint,9,rep,name=possible_length,json=possibleLength" json:"possible_length,omitempty"`
	// These represent the lengths that only local phone numbers (without an area
	// code) from this region can be. They will be sorted from smallest to
	// biggest. For example, since the American number 456-1234 may be locally
	// diallable, although not diallable from outside the area, 7 could be a
	// possible value.
	// This could be used to highlight tokens in a text that may be a phone
	// number.
	// To our knowledge, area codes are usually only relevant for some fixed-line
	// and mobile numbers, so this field should only be set for those types of
	// numbers (and the general description) - however there are exceptions for
	// NANPA countries.
	// This data is used to calculate whether a number could be a possible number
	// for a particular type.
	PossibleLengthLocalOnly []int32 `protobuf:"varint,10,rep,name=possible_length_local_only,json=possibleLengthLocalOnly" json:"possible_length_local_only,omitempty"`
	// An example national significant number for the specific type. It should
	// not contain any formatting information.
	ExampleNumber    *string `protobuf:"bytes,6,opt,name=example_number,json=exampleNumber" json:"example_number,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *PhoneNumberDesc) Reset()                    { *m = PhoneNumberDesc{} }
func (m *PhoneNumberDesc) String() string            { return proto.CompactTextString(m) }
func (*PhoneNumberDesc) ProtoMessage()               {}
func (*PhoneNumberDesc) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *PhoneNumberDesc) GetNationalNumberPattern() string {
	if m != nil && m.NationalNumberPattern != nil {
		return *m.NationalNumberPattern
	}
	return ""
}

func (m *PhoneNumberDesc) GetPossibleNumberPattern() string {
	if m != nil && m.PossibleNumberPattern != nil {
		return *m.PossibleNumberPattern
	}
	return ""
}

func (m *PhoneNumberDesc) GetPossibleLength() []int32 {
	if m != nil {
		return m.PossibleLength
	}
	return nil
}

func (m *PhoneNumberDesc) GetPossibleLengthLocalOnly() []int32 {
	if m != nil {
		return m.PossibleLengthLocalOnly
	}
	return nil
}

func (m *PhoneNumberDesc) GetExampleNumber() string {
	if m != nil && m.ExampleNumber != nil {
		return *m.ExampleNumber
	}
	return ""
}

// If you add, remove, or rename fields, or change their semantics, check if you
// should change the excludable field sets or the behavior in MetadataFilter.
type PhoneMetadata struct {
	// The general_desc contains information which is a superset of descriptions
	// for all types of phone numbers. If any element is missing in the
	// description of a specific type in the XML file, the element will inherit
	// from its counterpart in the general_desc. Every locale is assumed to have
	// fixed line and mobile numbers - if these types are missing in the
	// PhoneNumberMetadata XML file, they will inherit all fields from the
	// general_desc. For all other types that are generally relevant to normal
	// phone numbers, if the whole type is missing in the PhoneNumberMetadata XML
	// file, it will be given a national_number_pattern of "NA" and a
	// possible_number_pattern of "NA".
	GeneralDesc     *PhoneNumberDesc `protobuf:"bytes,1,opt,name=general_desc,json=generalDesc" json:"general_desc,omitempty"`
	FixedLine       *PhoneNumberDesc `protobuf:"bytes,2,opt,name=fixed_line,json=fixedLine" json:"fixed_line,omitempty"`
	Mobile          *PhoneNumberDesc `protobuf:"bytes,3,opt,name=mobile" json:"mobile,omitempty"`
	TollFree        *PhoneNumberDesc `protobuf:"bytes,4,opt,name=toll_free,json=tollFree" json:"toll_free,omitempty"`
	PremiumRate     *PhoneNumberDesc `protobuf:"bytes,5,opt,name=premium_rate,json=premiumRate" json:"premium_rate,omitempty"`
	SharedCost      *PhoneNumberDesc `protobuf:"bytes,6,opt,name=shared_cost,json=sharedCost" json:"shared_cost,omitempty"`
	PersonalNumber  *PhoneNumberDesc `protobuf:"bytes,7,opt,name=personal_number,json=personalNumber" json:"personal_number,omitempty"`
	Voip            *PhoneNumberDesc `protobuf:"bytes,8,opt,name=voip" json:"voip,omitempty"`
	Pager           *PhoneNumberDesc `protobuf:"bytes,21,opt,name=pager" json:"pager,omitempty"`
	Uan             *PhoneNumberDesc `protobuf:"bytes,25,opt,name=uan" json:"uan,omitempty"`
	Emergency       *PhoneNumberDesc `protobuf:"bytes,27,opt,name=emergency" json:"emergency,omitempty"`
	Voicemail       *PhoneNumberDesc `protobuf:"bytes,28,opt,name=voicemail" json:"voicemail,omitempty"`
	ShortCode       *PhoneNumberDesc `protobuf:"bytes,29,opt,name=short_code,json=shortCode" json:"short_code,omitempty"`
	StandardRate    *PhoneNumberDesc `protobuf:"bytes,30,opt,name=standard_rate,json=standardRate" json:"standard_rate,omitempty"`
	CarrierSpecific *PhoneNumberDesc `protobuf:"bytes,31,opt,name=carrier_specific,json=carrierSpecific" json:"carrier_specific,omitempty"`
	// The rules here distinguish the numbers that are only able to be dialled
	// nationally.
	NoInternationalDialling *PhoneNumberDesc `protobuf:"bytes,24,opt,name=no_international_dialling,json=noInternationalDialling" json:"no_international_dialling,omitempty"`
	// The CLDR 2-letter representation of a country/region, with the exception of
	// "country calling codes" used for non-geographical entities, such as
	// Universal International Toll Free Number (+800). These are all given the ID
	// "001", since this is the numeric region code for the world according to UN
	// M.49: http://en.wikipedia.org/wiki/UN_M.49
	Id *string `protobuf:"bytes,9,req,name=id" json:"id,omitempty"`
	// The country calling code that one would dial from overseas when trying to
	// dial a phone number in this country. For example, this would be "64" for
	// New Zealand.
	CountryCode *int32 `protobuf:"varint,10,opt,name=country_code,json=countryCode" json:"country_code,omitempty"`
	// The international_prefix of country A is the number that needs to be
	// dialled from country A to another country (country B). This is followed
	// by the country code for country B. Note that some countries may have more
	// than one international prefix, and for those cases, a regular expression
	// matching the international prefixes will be stored in this field.
	InternationalPrefix *string `protobuf:"bytes,11,opt,name=international_prefix,json=internationalPrefix" json:"international_prefix,omitempty"`
	// If more than one international prefix is present, a preferred prefix can
	// be specified here for out-of-country formatting purposes. If this field is
	// not present, and multiple international prefixes are present, then "+"
	// will be used instead.
	PreferredInternationalPrefix *string `protobuf:"bytes,17,opt,name=preferred_international_prefix,json=preferredInternationalPrefix" json:"preferred_international_prefix,omitempty"`
	// The national prefix of country A is the number that needs to be dialled
	// before the national significant number when dialling internally. This
	// would not be dialled when dialling internationally. For example, in New
	// Zealand, the number that would be locally dialled as 09 345 3456 would be
	// dialled from overseas as +64 9 345 3456. In this case, 0 is the national
	// prefix.
	NationalPrefix *string `protobuf:"bytes,12,opt,name=national_prefix,json=nationalPrefix" json:"national_prefix,omitempty"`
	// The preferred prefix when specifying an extension in this country. This is
	// used for formatting only, and if this is not specified, a suitable default
	// should be used instead. For example, if you wanted extensions to be
	// formatted in the following way:
	// 1 (365) 345 445 ext. 2345
	// " ext. "  should be the preferred extension prefix.
	PreferredExtnPrefix *string `protobuf:"bytes,13,opt,name=preferred_extn_prefix,json=preferredExtnPrefix" json:"preferred_extn_prefix,omitempty"`
	// This field is used for cases where the national prefix of a country
	// contains a carrier selection code, and is written in the form of a
	// regular expression. For example, to dial the number 2222-2222 in
	// Fortaleza, Brazil (area code 85) using the long distance carrier Oi
	// (selection code 31), one would dial 0 31 85 2222 2222. Assuming the
	// only other possible carrier selection code is 32, the field will
	// contain "03[12]".
	//
	// When it is missing from the XML file, this field inherits the value of
	// national_prefix, if that is present.
	NationalPrefixForParsing *string `protobuf:"bytes,15,opt,name=national_prefix_for_parsing,json=nationalPrefixForParsing" json:"national_prefix_for_parsing,omitempty"`
	// This field is only populated and used under very rare situations.
	// For example, mobile numbers in Argentina are written in two completely
	// different ways when dialed in-country and out-of-country
	// (e.g. 0343 15 555 1212 is exactly the same number as +54 9 343 555 1212).
	// This field is used together with national_prefix_for_parsing to transform
	// the number into a particular representation for storing in the phonenumber
	// proto buffer in those rare cases.
	NationalPrefixTransformRule *string `protobuf:"bytes,16,opt,name=national_prefix_transform_rule,json=nationalPrefixTransformRule" json:"national_prefix_transform_rule,omitempty"`
	// Specifies whether the mobile and fixed-line patterns are the same or not.
	// This is used to speed up determining phone number type in countries where
	// these two types of phone numbers can never be distinguished.
	SameMobileAndFixedLinePattern *bool `protobuf:"varint,18,opt,name=same_mobile_and_fixed_line_pattern,json=sameMobileAndFixedLinePattern,def=0" json:"same_mobile_and_fixed_line_pattern,omitempty"`
	// Note that the number format here is used for formatting only, not parsing.
	// Hence all the varied ways a user *may* write a number need not be recorded
	// - just the ideal way we would like to format it for them. When this element
	// is absent, the national significant number will be formatted as a whole
	// without any formatting applied.
	NumberFormat []*NumberFormat `protobuf:"bytes,19,rep,name=number_format,json=numberFormat" json:"number_format,omitempty"`
	// This field is populated only when the national significant number is
	// formatted differently when it forms part of the INTERNATIONAL format
	// and NATIONAL format. A case in point is mobile numbers in Argentina:
	// The number, which would be written in INTERNATIONAL format as
	// +54 9 343 555 1212, will be written as 0343 15 555 1212 for NATIONAL
	// format. In this case, the prefix 9 is inserted when dialling from
	// overseas, but otherwise the prefix 0 and the carrier selection code
	// 15 (inserted after the area code of 343) is used.
	// Note: this field is populated by setting a value for <intlFormat> inside
	// the <numberFormat> tag in the XML file. If <intlFormat> is not set then it
	// defaults to the same value as the <format> tag.
	//
	// Examples:
	//   To set the <intlFormat> to a different value than the <format>:
	//     <numberFormat pattern=....>
	//       <format>$1 $2 $3</format>
	//       <intlFormat>$1-$2-$3</intlFormat>
	//     </numberFormat>
	//
	//   To have a format only used for national formatting, set <intlFormat> to
	//   "NA":
	//     <numberFormat pattern=....>
	//       <format>$1 $2 $3</format>
	//       <intlFormat>NA</intlFormat>
	//     </numberFormat>
	IntlNumberFormat []*NumberFormat `protobuf:"bytes,20,rep,name=intl_number_format,json=intlNumberFormat" json:"intl_number_format,omitempty"`
	// This field is set when this country is considered to be the main country
	// for a calling code. It may not be set by more than one country with the
	// same calling code, and it should not be set by countries with a unique
	// calling code. This can be used to indicate that "GB" is the main country
	// for the calling code "44" for example, rather than Jersey or the Isle of
	// Man.
	MainCountryForCode *bool `protobuf:"varint,22,opt,name=main_country_for_code,json=mainCountryForCode,def=0" json:"main_country_for_code,omitempty"`
	// This field is populated only for countries or regions that share a country
	// calling code. If a number matches this pattern, it could belong to this
	// region. This is not intended as a replacement for IsValidForRegion since a
	// matching prefix is insufficient for a number to be valid. Furthermore, it
	// does not contain all the prefixes valid for a region - for example, 800
	// numbers are valid for all NANPA countries and are hence not listed here.
	// This field should be a regular expression of the expected prefix match.
	// It is used merely as a short-cut for working out which region a number
	// comes from in the case that there is only one, so leading_digit prefixes
	// should not overlap.
	LeadingDigits *string `protobuf:"bytes,23,opt,name=leading_digits,json=leadingDigits" json:"leading_digits,omitempty"`
	// The leading zero in a phone number is meaningful in some countries (e.g.
	// Italy). This means they cannot be dropped from the national number when
	// converting into international format. If leading zeros are possible for
	// valid international numbers for this region/country then set this to true.
	// This only needs to be set for the region that is the main_country_for_code
	// and all regions associated with that calling code will use the same
	// setting.
	LeadingZeroPossible *bool `protobuf:"varint,26,opt,name=leading_zero_possible,json=leadingZeroPossible,def=0" json:"leading_zero_possible,omitempty"`
	// This field is set when this country has implemented mobile number
	// portability. This means that transferring mobile numbers between carriers
	// is allowed. A consequence of this is that phone prefix to carrier mapping
	// is less reliable.
	MobileNumberPortableRegion *bool  `protobuf:"varint,32,opt,name=mobile_number_portable_region,json=mobileNumberPortableRegion,def=0" json:"mobile_number_portable_region,omitempty"`
	XXX_unrecognized           []byte `json:"-"`
}

func (m *PhoneMetadata) Reset()                    { *m = PhoneMetadata{} }
func (m *PhoneMetadata) String() string            { return proto.CompactTextString(m) }
func (*PhoneMetadata) ProtoMessage()               {}
func (*PhoneMetadata) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

const Default_PhoneMetadata_SameMobileAndFixedLinePattern bool = false
const Default_PhoneMetadata_MainCountryForCode bool = false
const Default_PhoneMetadata_LeadingZeroPossible bool = false
const Default_PhoneMetadata_MobileNumberPortableRegion bool = false

func (m *PhoneMetadata) GetGeneralDesc() *PhoneNumberDesc {
	if m != nil {
		return m.GeneralDesc
	}
	return nil
}

func (m *PhoneMetadata) GetFixedLine() *PhoneNumberDesc {
	if m != nil {
		return m.FixedLine
	}
	return nil
}

func (m *PhoneMetadata) GetMobile() *PhoneNumberDesc {
	if m != nil {
		return m.Mobile
	}
	return nil
}

func (m *PhoneMetadata) GetTollFree() *PhoneNumberDesc {
	if m != nil {
		return m.TollFree
	}
	return nil
}

func (m *PhoneMetadata) GetPremiumRate() *PhoneNumberDesc {
	if m != nil {
		return m.PremiumRate
	}
	return nil
}

func (m *PhoneMetadata) GetSharedCost() *PhoneNumberDesc {
	if m != nil {
		return m.SharedCost
	}
	return nil
}

func (m *PhoneMetadata) GetPersonalNumber() *PhoneNumberDesc {
	if m != nil {
		return m.PersonalNumber
	}
	return nil
}

func (m *PhoneMetadata) GetVoip() *PhoneNumberDesc {
	if m != nil {
		return m.Voip
	}
	return nil
}

func (m *PhoneMetadata) GetPager() *PhoneNumberDesc {
	if m != nil {
		return m.Pager
	}
	return nil
}

func (m *PhoneMetadata) GetUan() *PhoneNumberDesc {
	if m != nil {
		return m.Uan
	}
	return nil
}

func (m *PhoneMetadata) GetEmergency() *PhoneNumberDesc {
	if m != nil {
		return m.Emergency
	}
	return nil
}

func (m *PhoneMetadata) GetVoicemail() *PhoneNumberDesc {
	if m != nil {
		return m.Voicemail
	}
	return nil
}

func (m *PhoneMetadata) GetShortCode() *PhoneNumberDesc {
	if m != nil {
		return m.ShortCode
	}
	return nil
}

func (m *PhoneMetadata) GetStandardRate() *PhoneNumberDesc {
	if m != nil {
		return m.StandardRate
	}
	return nil
}

func (m *PhoneMetadata) GetCarrierSpecific() *PhoneNumberDesc {
	if m != nil {
		return m.CarrierSpecific
	}
	return nil
}

func (m *PhoneMetadata) GetNoInternationalDialling() *PhoneNumberDesc {
	if m != nil {
		return m.NoInternationalDialling
	}
	return nil
}

func (m *PhoneMetadata) GetId() string {
	if m != nil && m.Id != nil {
		return *m.Id
	}
	return ""
}

func (m *PhoneMetadata) GetCountryCode() int32 {
	if m != nil && m.CountryCode != nil {
		return *m.CountryCode
	}
	return 0
}

func (m *PhoneMetadata) GetInternationalPrefix() string {
	if m != nil && m.InternationalPrefix != nil {
		return *m.InternationalPrefix
	}
	return ""
}

func (m *PhoneMetadata) GetPreferredInternationalPrefix() string {
	if m != nil && m.PreferredInternationalPrefix != nil {
		return *m.PreferredInternationalPrefix
	}
	return ""
}

func (m *PhoneMetadata) GetNationalPrefix() string {
	if m != nil && m.NationalPrefix != nil {
		return *m.NationalPrefix
	}
	return ""
}

func (m *PhoneMetadata) GetPreferredExtnPrefix() string {
	if m != nil && m.PreferredExtnPrefix != nil {
		return *m.PreferredExtnPrefix
	}
	return ""
}

func (m *PhoneMetadata) GetNationalPrefixForParsing() string {
	if m != nil && m.NationalPrefixForParsing != nil {
		return *m.NationalPrefixForParsing
	}
	return ""
}

func (m *PhoneMetadata) GetNationalPrefixTransformRule() string {
	if m != nil && m.NationalPrefixTransformRule != nil {
		return *m.NationalPrefixTransformRule
	}
	return ""
}

func (m *PhoneMetadata) GetSameMobileAndFixedLinePattern() bool {
	if m != nil && m.SameMobileAndFixedLinePattern != nil {
		return *m.SameMobileAndFixedLinePattern
	}
	return Default_PhoneMetadata_SameMobileAndFixedLinePattern
}

func (m *PhoneMetadata) GetNumberFormat() []*NumberFormat {
	if m != nil {
		return m.NumberFormat
	}
	return nil
}

func (m *PhoneMetadata) GetIntlNumberFormat() []*NumberFormat {
	if m != nil {
		return m.IntlNumberFormat
	}
	return nil
}

func (m *PhoneMetadata) GetMainCountryForCode() bool {
	if m != nil && m.MainCountryForCode != nil {
		return *m.MainCountryForCode
	}
	return Default_PhoneMetadata_MainCountryForCode
}

func (m *PhoneMetadata) GetLeadingDigits() string {
	if m != nil && m.LeadingDigits != nil {
		return *m.LeadingDigits
	}
	return ""
}

func (m *PhoneMetadata) GetLeadingZeroPossible() bool {
	if m != nil && m.LeadingZeroPossible != nil {
		return *m.LeadingZeroPossible
	}
	return Default_PhoneMetadata_LeadingZeroPossible
}

func (m *PhoneMetadata) GetMobileNumberPortableRegion() bool {
	if m != nil && m.MobileNumberPortableRegion != nil {
		return *m.MobileNumberPortableRegion
	}
	return Default_PhoneMetadata_MobileNumberPortableRegion
}

type PhoneMetadataCollection struct {
	Metadata         []*PhoneMetadata `protobuf:"bytes,1,rep,name=metadata" json:"metadata,omitempty"`
	XXX_unrecognized []byte           `json:"-"`
}

func (m *PhoneMetadataCollection) Reset()                    { *m = PhoneMetadataCollection{} }
func (m *PhoneMetadataCollection) String() string            { return proto.CompactTextString(m) }
func (*PhoneMetadataCollection) ProtoMessage()               {}
func (*PhoneMetadataCollection) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *PhoneMetadataCollection) GetMetadata() []*PhoneMetadata {
	if m != nil {
		return m.Metadata
	}
	return nil
}

func init() {
	proto.RegisterType((*NumberFormat)(nil), "i18n.phonenumbers.NumberFormat")
	proto.RegisterType((*PhoneNumberDesc)(nil), "i18n.phonenumbers.PhoneNumberDesc")
	proto.RegisterType((*PhoneMetadata)(nil), "i18n.phonenumbers.PhoneMetadata")
	proto.RegisterType((*PhoneMetadataCollection)(nil), "i18n.phonenumbers.PhoneMetadataCollection")
}

func init() { proto.RegisterFile("phonemetadata.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 1033 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x96, 0xeb, 0x6e, 0xdb, 0x46,
	0x13, 0x86, 0x21, 0x29, 0x4e, 0xac, 0x91, 0x6c, 0x39, 0xeb, 0x83, 0x36, 0x3e, 0x2a, 0xc2, 0x17,
	0x44, 0xbf, 0x0c, 0xc4, 0x08, 0x0c, 0x7f, 0x69, 0x8b, 0x36, 0x95, 0xed, 0x26, 0xa8, 0xdd, 0x08,
	0x6c, 0x81, 0x00, 0x05, 0x5a, 0x62, 0x4d, 0x8e, 0xa4, 0x05, 0xc8, 0x5d, 0x62, 0xb9, 0x4a, 0xed,
	0x5e, 0x44, 0xaf, 0xa5, 0x57, 0xd6, 0x6b, 0x28, 0xb8, 0x07, 0x5a, 0x92, 0x5d, 0x80, 0xff, 0x4c,
	0xce, 0xfb, 0xbc, 0x5c, 0xcf, 0x8e, 0x66, 0x06, 0x36, 0xb3, 0xa9, 0x14, 0x98, 0xa2, 0x66, 0x31,
	0xd3, 0xec, 0x38, 0x53, 0x52, 0x4b, 0xf2, 0x9c, 0xbf, 0x39, 0x13, 0xc7, 0x26, 0x22, 0x66, 0xe9,
	0x0d, 0xaa, 0xbc, 0xff, 0x4f, 0x1d, 0xda, 0x3f, 0x99, 0xbf, 0x2f, 0xa5, 0x4a, 0x99, 0x26, 0x14,
	0x9e, 0x65, 0x4c, 0x6b, 0x54, 0x82, 0xd6, 0x7a, 0xf5, 0x41, 0x33, 0xf0, 0x8f, 0x64, 0x07, 0x9e,
	0x8e, 0x8d, 0x86, 0xd6, 0x4d, 0xc0, 0x3d, 0x91, 0xb7, 0xb0, 0x93, 0x20, 0x8b, 0xb9, 0x98, 0x84,
	0x31, 0x9f, 0x70, 0x9d, 0x87, 0xde, 0xa0, 0xd1, 0x6b, 0x0c, 0x9a, 0xc1, 0x96, 0x8b, 0x9e, 0x9b,
	0xe0, 0xc8, 0xb9, 0x5d, 0xc0, 0x91, 0x60, 0x9a, 0x4b, 0xc1, 0x92, 0x30, 0x53, 0x38, 0xe6, 0xb7,
	0xa1, 0xf5, 0xd3, 0x85, 0x91, 0x9a, 0x25, 0x48, 0x9f, 0xf4, 0x6a, 0x83, 0x66, 0xb0, 0xef, 0x65,
	0x23, 0xa3, 0xba, 0x2c, 0x45, 0xc1, 0x2c, 0x41, 0xf2, 0x1b, 0x0c, 0x96, 0x6d, 0x64, 0xe6, 0x9e,
	0xff, 0x98, 0xa2, 0x98, 0x33, 0xa5, 0x4f, 0x7b, 0xb5, 0xc1, 0xea, 0xbb, 0x95, 0x31, 0x4b, 0x72,
	0x0c, 0xfe, 0xb7, 0x68, 0xfb, 0xc9, 0x41, 0x9f, 0xa7, 0x28, 0xee, 0x3f, 0x41, 0x46, 0xf0, 0x2a,
	0x96, 0x29, 0xe6, 0x9a, 0x47, 0x61, 0xc4, 0x94, 0xe2, 0xa8, 0xc2, 0x48, 0xc6, 0xf8, 0xe0, 0xac,
	0x2b, 0xe6, 0xac, 0x2f, 0xbd, 0x78, 0x68, 0xb5, 0x43, 0x19, 0xe3, 0xe2, 0x81, 0xfb, 0x7f, 0xd5,
	0xa1, 0x33, 0x2a, 0x6e, 0xc0, 0x66, 0xfd, 0x1c, 0xf3, 0x88, 0x9c, 0x42, 0xb7, 0xfc, 0x27, 0xec,
	0xc5, 0x94, 0x29, 0xac, 0x1b, 0xdf, 0x6d, 0x1f, 0xb6, 0x90, 0xcf, 0xe1, 0x29, 0x74, 0x33, 0x99,
	0xe7, 0xfc, 0x26, 0xc1, 0x65, 0xae, 0x61, 0x39, 0x1f, 0x5e, 0xe4, 0x5e, 0x43, 0xa7, 0xe4, 0x12,
	0x14, 0x13, 0x3d, 0xa5, 0xcd, 0x5e, 0x63, 0xb0, 0x12, 0xac, 0xfb, 0xd7, 0x57, 0xe6, 0x2d, 0xf9,
	0x0a, 0x76, 0x97, 0x84, 0x61, 0x22, 0x23, 0x96, 0x84, 0x52, 0x24, 0x77, 0x14, 0x0c, 0xd3, 0x5d,
	0x64, 0xae, 0x8a, 0xf8, 0x27, 0x91, 0xdc, 0x91, 0x57, 0xb0, 0x8e, 0xb7, 0x2c, 0xcd, 0xca, 0xc3,
	0x99, 0x0b, 0x68, 0x06, 0x6b, 0xee, 0xad, 0x3d, 0x53, 0xff, 0xef, 0x0e, 0xac, 0x99, 0x84, 0x5c,
	0xbb, 0x62, 0x25, 0x17, 0xd0, 0x9e, 0xa0, 0x40, 0xc5, 0x92, 0x30, 0xc6, 0x3c, 0xa2, 0xb5, 0x5e,
	0x6d, 0xd0, 0x3a, 0xe9, 0x1f, 0x3f, 0xa8, 0xde, 0xe3, 0xa5, 0x44, 0x06, 0x2d, 0xc7, 0x99, 0xac,
	0xbe, 0x07, 0x18, 0xf3, 0x5b, 0x8c, 0xc3, 0x84, 0x0b, 0x34, 0x89, 0xac, 0x66, 0xd2, 0x34, 0xd4,
	0x15, 0x17, 0x48, 0xde, 0xc1, 0xd3, 0x54, 0xde, 0xf0, 0x04, 0x4d, 0x3e, 0xab, 0xe1, 0x8e, 0x20,
	0xdf, 0x42, 0x53, 0xcb, 0x24, 0x09, 0xc7, 0x0a, 0x6d, 0x29, 0x57, 0xc3, 0x57, 0x0b, 0xe8, 0x52,
	0x21, 0x16, 0x69, 0xc8, 0x14, 0xa6, 0x7c, 0x96, 0x86, 0x8a, 0x69, 0x5b, 0x62, 0x15, 0xd3, 0xe0,
	0xb8, 0x80, 0x69, 0x24, 0x43, 0x68, 0xe5, 0x53, 0xa6, 0x30, 0x0e, 0x23, 0x99, 0x6b, 0x73, 0x07,
	0xd5, 0x5c, 0xc0, 0x62, 0x43, 0x99, 0x6b, 0xf2, 0x23, 0x74, 0x32, 0x54, 0xf9, 0x5c, 0x85, 0xd2,
	0x67, 0x95, 0x8d, 0xd6, 0x3d, 0x6a, 0xdf, 0x91, 0x53, 0x78, 0xf2, 0x45, 0xf2, 0x8c, 0xae, 0x56,
	0x76, 0x30, 0x7a, 0x72, 0x06, 0x2b, 0x19, 0x9b, 0xa0, 0xa2, 0xdb, 0x95, 0x41, 0x0b, 0x90, 0xb7,
	0xd0, 0x98, 0x31, 0x41, 0x5f, 0x54, 0xe6, 0x0a, 0x39, 0xf9, 0x0e, 0x9a, 0x98, 0xa2, 0x9a, 0xa0,
	0x88, 0xee, 0xe8, 0x5e, 0xf5, 0xfa, 0x29, 0xa1, 0xc2, 0xe1, 0x8b, 0xe4, 0x11, 0xa6, 0x8c, 0x27,
	0x74, 0xbf, 0xba, 0x43, 0x09, 0x15, 0x45, 0x9c, 0x4f, 0xa5, 0xd2, 0xa6, 0xeb, 0xd0, 0x83, 0xea,
	0x16, 0x86, 0x2a, 0xfa, 0x0f, 0xf9, 0x01, 0xd6, 0x72, 0xcd, 0x44, 0xcc, 0x54, 0x6c, 0x0b, 0xe9,
	0xb0, 0xb2, 0x4b, 0xdb, 0x83, 0xa6, 0x92, 0xae, 0x61, 0xc3, 0xf7, 0xc0, 0x3c, 0xc3, 0x88, 0x8f,
	0x79, 0x44, 0x8f, 0x2a, 0x7b, 0x75, 0x1c, 0xfb, 0xb3, 0x43, 0xc9, 0xef, 0xf0, 0x42, 0xc8, 0x90,
	0x8b, 0xa2, 0x25, 0xf9, 0xee, 0x17, 0x73, 0x96, 0x24, 0x45, 0xaf, 0xa6, 0x95, 0x7d, 0xbb, 0x42,
	0x7e, 0x9c, 0xf7, 0x38, 0x77, 0x16, 0x64, 0x1d, 0xea, 0x3c, 0xa6, 0x4d, 0x33, 0xab, 0xea, 0x3c,
	0x26, 0x2f, 0xa1, 0x1d, 0xc9, 0x99, 0xd0, 0xea, 0xce, 0x26, 0x13, 0x7a, 0xb5, 0xc1, 0x4a, 0xd0,
	0x72, 0xef, 0x4c, 0xaa, 0xde, 0xc0, 0xd6, 0xe2, 0x79, 0xec, 0x48, 0xa1, 0x2d, 0xd3, 0xb8, 0x36,
	0x17, 0x62, 0x76, 0x6e, 0x90, 0x73, 0x38, 0x2c, 0x44, 0xa8, 0x8a, 0x5f, 0xd8, 0xa3, 0xf0, 0x73,
	0x3b, 0xc6, 0x4a, 0xd5, 0xc7, 0x47, 0x5c, 0x5e, 0x43, 0x67, 0x19, 0x6b, 0x1b, 0x6c, 0x7d, 0x49,
	0x78, 0x02, 0xdb, 0xf7, 0x9f, 0xc3, 0x5b, 0x2d, 0xbc, 0x7c, 0xcd, 0x1e, 0xb1, 0x0c, 0x5e, 0xdc,
	0x6a, 0xe1, 0x98, 0x6f, 0x60, 0xef, 0x91, 0x51, 0x1b, 0x66, 0x4c, 0xe5, 0x45, 0xaa, 0x3b, 0x86,
	0xa4, 0x0f, 0xc6, 0xec, 0xc8, 0xc6, 0xc9, 0x10, 0x0e, 0x97, 0x71, 0xad, 0x98, 0xc8, 0x8b, 0x11,
	0x68, 0x87, 0xdf, 0x86, 0x71, 0xd8, 0x5b, 0x74, 0xf8, 0xc5, 0x6b, 0xcc, 0x9c, 0x1e, 0x41, 0x3f,
	0x67, 0x29, 0x86, 0xb6, 0x39, 0x86, 0x4c, 0xc4, 0xe1, 0x7d, 0x73, 0x2e, 0xa7, 0x16, 0x99, 0x9f,
	0xd0, 0x07, 0x05, 0x70, 0x6d, 0xf4, 0xef, 0x45, 0x7c, 0xe9, 0x9b, 0xb2, 0x1f, 0x62, 0xe7, 0xb0,
	0xe6, 0x66, 0x9e, 0xdb, 0x4a, 0x36, 0x7b, 0x8d, 0x41, 0xeb, 0xe4, 0xe8, 0x91, 0x92, 0x99, 0x5f,
	0x70, 0x82, 0xb6, 0x98, 0x5f, 0x77, 0xae, 0x81, 0x70, 0xa1, 0xcb, 0xb1, 0xeb, 0xac, 0xb6, 0xaa,
	0x59, 0x6d, 0x14, 0xe8, 0xc2, 0xf6, 0x74, 0x06, 0xdb, 0x29, 0xe3, 0x22, 0xf4, 0x85, 0x56, 0xe4,
	0xd9, 0x14, 0xdb, 0xce, 0xfc, 0x7f, 0x46, 0x0a, 0xcd, 0xd0, 0x4a, 0x2e, 0xa5, 0xd9, 0x12, 0x8a,
	0x69, 0xb9, 0xb8, 0x45, 0xd1, 0xae, 0x9d, 0x96, 0x0b, 0xdb, 0x13, 0xf9, 0x3f, 0x6c, 0x7b, 0xd9,
	0x9f, 0xa8, 0x64, 0xe8, 0x87, 0x2f, 0xdd, 0x9d, 0xff, 0xc0, 0xa6, 0xd3, 0xfc, 0x8a, 0x4a, 0x8e,
	0x9c, 0x82, 0x7c, 0x80, 0x03, 0x97, 0x7d, 0xbf, 0x2b, 0x48, 0xa5, 0x59, 0x31, 0xda, 0x15, 0x4e,
	0xb8, 0x14, 0xb4, 0x37, 0x6f, 0xb1, 0x6b, 0xb5, 0x6e, 0x71, 0x70, 0xca, 0xc0, 0x08, 0xfb, 0x9f,
	0xa1, 0xbb, 0x30, 0xb1, 0x87, 0x32, 0x49, 0x30, 0x2a, 0x2e, 0x9f, 0x7c, 0x0d, 0xab, 0x7e, 0xe9,
	0xa4, 0x35, 0x93, 0xc5, 0xde, 0x7f, 0xfd, 0x86, 0x3d, 0x1d, 0x94, 0xc4, 0xf7, 0x3d, 0xd8, 0x8f,
	0x64, 0x7a, 0x3c, 0x91, 0x72, 0x92, 0xe0, 0x43, 0xee, 0x43, 0xe3, 0xdf, 0x00, 0x00, 0x00, 0xff,
	0xff, 0xe7, 0x95, 0xf6, 0xa7, 0xd8, 0x0a, 0x00, 0x00,
}