// Code generated by capnpc-go. DO NOT EDIT.

package rpctwoparty

import (
	capnp "capnproto.org/go/capnp/v3"
	text "capnproto.org/go/capnp/v3/encoding/text"
	schemas "capnproto.org/go/capnp/v3/schemas"
)

type Side uint16

// Side_TypeID is the unique identifier for the type Side.
const Side_TypeID = 0x9fd69ebc87b9719c

// Values of Side.
const (
	Side_server Side = 0
	Side_client Side = 1
)

// String returns the enum's constant name.
func (c Side) String() string {
	switch c {
	case Side_server:
		return "server"
	case Side_client:
		return "client"

	default:
		return ""
	}
}

// SideFromString returns the enum value with a name,
// or the zero value if there's no such value.
func SideFromString(c string) Side {
	switch c {
	case "server":
		return Side_server
	case "client":
		return Side_client

	default:
		return 0
	}
}

type Side_List = capnp.EnumList[Side]

func NewSide_List(s *capnp.Segment, sz int32) (Side_List, error) {
	return capnp.NewEnumList[Side](s, sz)
}

type VatId capnp.Struct

// VatId_TypeID is the unique identifier for the type VatId.
const VatId_TypeID = 0xd20b909fee733a8e

func NewVatId(s *capnp.Segment) (VatId, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0})
	return VatId(st), err
}

func NewRootVatId(s *capnp.Segment) (VatId, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0})
	return VatId(st), err
}

func ReadRootVatId(msg *capnp.Message) (VatId, error) {
	root, err := msg.Root()
	return VatId(root.Struct()), err
}

func (s VatId) String() string {
	str, _ := text.Marshal(0xd20b909fee733a8e, capnp.Struct(s))
	return str
}

func (s VatId) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (VatId) DecodeFromPtr(p capnp.Ptr) VatId {
	return VatId(capnp.Struct{}.DecodeFromPtr(p))
}

func (s VatId) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s VatId) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s VatId) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s VatId) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s VatId) Side() Side {
	return Side(capnp.Struct(s).Uint16(0))
}

func (s VatId) SetSide(v Side) {
	capnp.Struct(s).SetUint16(0, uint16(v))
}

// VatId_List is a list of VatId.
type VatId_List = capnp.StructList[VatId]

// NewVatId creates a new list of VatId.
func NewVatId_List(s *capnp.Segment, sz int32) (VatId_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0}, sz)
	return capnp.StructList[VatId](l), err
}

// VatId_Future is a wrapper for a VatId promised by a client call.
type VatId_Future struct{ *capnp.Future }

func (f VatId_Future) Struct() (VatId, error) {
	p, err := f.Future.Ptr()
	return VatId(p.Struct()), err
}

type ProvisionId capnp.Struct

// ProvisionId_TypeID is the unique identifier for the type ProvisionId.
const ProvisionId_TypeID = 0xb88d09a9c5f39817

func NewProvisionId(s *capnp.Segment) (ProvisionId, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0})
	return ProvisionId(st), err
}

func NewRootProvisionId(s *capnp.Segment) (ProvisionId, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0})
	return ProvisionId(st), err
}

func ReadRootProvisionId(msg *capnp.Message) (ProvisionId, error) {
	root, err := msg.Root()
	return ProvisionId(root.Struct()), err
}

func (s ProvisionId) String() string {
	str, _ := text.Marshal(0xb88d09a9c5f39817, capnp.Struct(s))
	return str
}

func (s ProvisionId) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (ProvisionId) DecodeFromPtr(p capnp.Ptr) ProvisionId {
	return ProvisionId(capnp.Struct{}.DecodeFromPtr(p))
}

func (s ProvisionId) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s ProvisionId) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s ProvisionId) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s ProvisionId) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s ProvisionId) JoinId() uint32 {
	return capnp.Struct(s).Uint32(0)
}

func (s ProvisionId) SetJoinId(v uint32) {
	capnp.Struct(s).SetUint32(0, v)
}

// ProvisionId_List is a list of ProvisionId.
type ProvisionId_List = capnp.StructList[ProvisionId]

// NewProvisionId creates a new list of ProvisionId.
func NewProvisionId_List(s *capnp.Segment, sz int32) (ProvisionId_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0}, sz)
	return capnp.StructList[ProvisionId](l), err
}

// ProvisionId_Future is a wrapper for a ProvisionId promised by a client call.
type ProvisionId_Future struct{ *capnp.Future }

func (f ProvisionId_Future) Struct() (ProvisionId, error) {
	p, err := f.Future.Ptr()
	return ProvisionId(p.Struct()), err
}

type RecipientId capnp.Struct

// RecipientId_TypeID is the unique identifier for the type RecipientId.
const RecipientId_TypeID = 0x89f389b6fd4082c1

func NewRecipientId(s *capnp.Segment) (RecipientId, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return RecipientId(st), err
}

func NewRootRecipientId(s *capnp.Segment) (RecipientId, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return RecipientId(st), err
}

func ReadRootRecipientId(msg *capnp.Message) (RecipientId, error) {
	root, err := msg.Root()
	return RecipientId(root.Struct()), err
}

func (s RecipientId) String() string {
	str, _ := text.Marshal(0x89f389b6fd4082c1, capnp.Struct(s))
	return str
}

func (s RecipientId) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (RecipientId) DecodeFromPtr(p capnp.Ptr) RecipientId {
	return RecipientId(capnp.Struct{}.DecodeFromPtr(p))
}

func (s RecipientId) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s RecipientId) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s RecipientId) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s RecipientId) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}

// RecipientId_List is a list of RecipientId.
type RecipientId_List = capnp.StructList[RecipientId]

// NewRecipientId creates a new list of RecipientId.
func NewRecipientId_List(s *capnp.Segment, sz int32) (RecipientId_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0}, sz)
	return capnp.StructList[RecipientId](l), err
}

// RecipientId_Future is a wrapper for a RecipientId promised by a client call.
type RecipientId_Future struct{ *capnp.Future }

func (f RecipientId_Future) Struct() (RecipientId, error) {
	p, err := f.Future.Ptr()
	return RecipientId(p.Struct()), err
}

type ThirdPartyCapId capnp.Struct

// ThirdPartyCapId_TypeID is the unique identifier for the type ThirdPartyCapId.
const ThirdPartyCapId_TypeID = 0xb47f4979672cb59d

func NewThirdPartyCapId(s *capnp.Segment) (ThirdPartyCapId, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return ThirdPartyCapId(st), err
}

func NewRootThirdPartyCapId(s *capnp.Segment) (ThirdPartyCapId, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return ThirdPartyCapId(st), err
}

func ReadRootThirdPartyCapId(msg *capnp.Message) (ThirdPartyCapId, error) {
	root, err := msg.Root()
	return ThirdPartyCapId(root.Struct()), err
}

func (s ThirdPartyCapId) String() string {
	str, _ := text.Marshal(0xb47f4979672cb59d, capnp.Struct(s))
	return str
}

func (s ThirdPartyCapId) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (ThirdPartyCapId) DecodeFromPtr(p capnp.Ptr) ThirdPartyCapId {
	return ThirdPartyCapId(capnp.Struct{}.DecodeFromPtr(p))
}

func (s ThirdPartyCapId) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s ThirdPartyCapId) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s ThirdPartyCapId) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s ThirdPartyCapId) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}

// ThirdPartyCapId_List is a list of ThirdPartyCapId.
type ThirdPartyCapId_List = capnp.StructList[ThirdPartyCapId]

// NewThirdPartyCapId creates a new list of ThirdPartyCapId.
func NewThirdPartyCapId_List(s *capnp.Segment, sz int32) (ThirdPartyCapId_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0}, sz)
	return capnp.StructList[ThirdPartyCapId](l), err
}

// ThirdPartyCapId_Future is a wrapper for a ThirdPartyCapId promised by a client call.
type ThirdPartyCapId_Future struct{ *capnp.Future }

func (f ThirdPartyCapId_Future) Struct() (ThirdPartyCapId, error) {
	p, err := f.Future.Ptr()
	return ThirdPartyCapId(p.Struct()), err
}

type JoinKeyPart capnp.Struct

// JoinKeyPart_TypeID is the unique identifier for the type JoinKeyPart.
const JoinKeyPart_TypeID = 0x95b29059097fca83

func NewJoinKeyPart(s *capnp.Segment) (JoinKeyPart, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0})
	return JoinKeyPart(st), err
}

func NewRootJoinKeyPart(s *capnp.Segment) (JoinKeyPart, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0})
	return JoinKeyPart(st), err
}

func ReadRootJoinKeyPart(msg *capnp.Message) (JoinKeyPart, error) {
	root, err := msg.Root()
	return JoinKeyPart(root.Struct()), err
}

func (s JoinKeyPart) String() string {
	str, _ := text.Marshal(0x95b29059097fca83, capnp.Struct(s))
	return str
}

func (s JoinKeyPart) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (JoinKeyPart) DecodeFromPtr(p capnp.Ptr) JoinKeyPart {
	return JoinKeyPart(capnp.Struct{}.DecodeFromPtr(p))
}

func (s JoinKeyPart) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s JoinKeyPart) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s JoinKeyPart) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s JoinKeyPart) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s JoinKeyPart) JoinId() uint32 {
	return capnp.Struct(s).Uint32(0)
}

func (s JoinKeyPart) SetJoinId(v uint32) {
	capnp.Struct(s).SetUint32(0, v)
}

func (s JoinKeyPart) PartCount() uint16 {
	return capnp.Struct(s).Uint16(4)
}

func (s JoinKeyPart) SetPartCount(v uint16) {
	capnp.Struct(s).SetUint16(4, v)
}

func (s JoinKeyPart) PartNum() uint16 {
	return capnp.Struct(s).Uint16(6)
}

func (s JoinKeyPart) SetPartNum(v uint16) {
	capnp.Struct(s).SetUint16(6, v)
}

// JoinKeyPart_List is a list of JoinKeyPart.
type JoinKeyPart_List = capnp.StructList[JoinKeyPart]

// NewJoinKeyPart creates a new list of JoinKeyPart.
func NewJoinKeyPart_List(s *capnp.Segment, sz int32) (JoinKeyPart_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0}, sz)
	return capnp.StructList[JoinKeyPart](l), err
}

// JoinKeyPart_Future is a wrapper for a JoinKeyPart promised by a client call.
type JoinKeyPart_Future struct{ *capnp.Future }

func (f JoinKeyPart_Future) Struct() (JoinKeyPart, error) {
	p, err := f.Future.Ptr()
	return JoinKeyPart(p.Struct()), err
}

type JoinResult capnp.Struct

// JoinResult_TypeID is the unique identifier for the type JoinResult.
const JoinResult_TypeID = 0x9d263a3630b7ebee

func NewJoinResult(s *capnp.Segment) (JoinResult, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1})
	return JoinResult(st), err
}

func NewRootJoinResult(s *capnp.Segment) (JoinResult, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1})
	return JoinResult(st), err
}

func ReadRootJoinResult(msg *capnp.Message) (JoinResult, error) {
	root, err := msg.Root()
	return JoinResult(root.Struct()), err
}

func (s JoinResult) String() string {
	str, _ := text.Marshal(0x9d263a3630b7ebee, capnp.Struct(s))
	return str
}

func (s JoinResult) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (JoinResult) DecodeFromPtr(p capnp.Ptr) JoinResult {
	return JoinResult(capnp.Struct{}.DecodeFromPtr(p))
}

func (s JoinResult) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s JoinResult) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s JoinResult) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s JoinResult) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s JoinResult) JoinId() uint32 {
	return capnp.Struct(s).Uint32(0)
}

func (s JoinResult) SetJoinId(v uint32) {
	capnp.Struct(s).SetUint32(0, v)
}

func (s JoinResult) Succeeded() bool {
	return capnp.Struct(s).Bit(32)
}

func (s JoinResult) SetSucceeded(v bool) {
	capnp.Struct(s).SetBit(32, v)
}

func (s JoinResult) Cap() (capnp.Ptr, error) {
	return capnp.Struct(s).Ptr(0)
}

func (s JoinResult) HasCap() bool {
	return capnp.Struct(s).HasPtr(0)
}

func (s JoinResult) SetCap(v capnp.Ptr) error {
	return capnp.Struct(s).SetPtr(0, v)
}

// JoinResult_List is a list of JoinResult.
type JoinResult_List = capnp.StructList[JoinResult]

// NewJoinResult creates a new list of JoinResult.
func NewJoinResult_List(s *capnp.Segment, sz int32) (JoinResult_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1}, sz)
	return capnp.StructList[JoinResult](l), err
}

// JoinResult_Future is a wrapper for a JoinResult promised by a client call.
type JoinResult_Future struct{ *capnp.Future }

func (f JoinResult_Future) Struct() (JoinResult, error) {
	p, err := f.Future.Ptr()
	return JoinResult(p.Struct()), err
}
func (p JoinResult_Future) Cap() *capnp.Future {
	return p.Future.Field(0, nil)
}

const schema_a184c7885cdaf2a1 = "x\xda|\x92\xcfk\xd4@\x14\xc7\xdfw&u\xb7\xe8" +
	"\xb2\xa4\xb3\"zQ\x04=\x88J\x8b\xa2\x90K\x16\x17" +
	"\xc1h\x91\x9d\xb5\x15\x0b\xf6\x10\x92\xa0\x916I\x93l" +
	"e\x0f\xb2\xa0\x82\xf6\xa0\xf4\"x\xb0\x96\x1e=\x89\xe2" +
	"\xaf\x82\x1e\x14D\xe8\xd1\x83\x07\xff\x05\xa1\x87\xf6\xb6 " +
	"\x91\x09\xd8\x95\xfd\xe1m\xe6\xf1y_>\xef\xcd\x8c\x9b" +
	"\xa8j\x13\xa5\xb6FL\x9e\x1e\xd9\x95}\xbe[\xfd\xfd" +
	"ni{\x89t\x81lm\xeb\xe7\xf5\x87\xdf\xee\xaf\x91" +
	"V \x12GXGL\xb0\x02\xf1\xec\xdeF{tf" +
	"\xf9\xf5\x13\x92\x02\xbdT\x89u\xc4~\xa6N{\xd9K" +
	"B\xb6\xf9\xeb\xfd\xf8\x19\xe3\xe8J\x0f;\x02\x85\xbcb" +
	"[\xe2S\x0e\xaf\xe7\xf0\xb3\x85\xf5\x07\x1f\x9f\xffX%" +
	"]\xb0.K\x10\xd3\xfc\x8b\x98\xe5\x0a\x9c\xe1g\x09\xd9" +
	"\xca\xdb\xe37ZV\xfbM\xbf\xe6\xa9Y~\x00b\x9e" +
	"+\xcf}O\xb7\xbf\xbe\x18}\xf4a\x90\xa7\xc5;b" +
	":O\x94\xdc$d\x8f\x8ddsuy\xf7\xf7A\xec" +
	"\x02\xdf\x10wr\xb6\xc5M\xaaeq\xe4\x9cHo\x87" +
	"\x11\xb3\xe3\xb4u\xd2\xb1\xa3 2\x1a\x9e\xe3G\xa6\xef" +
	"\x05\xa9\xe5\xd6\x81\x81\xcc\xc5\xd0\x0f.\x99^\xabn\xc7" +
	"i\x1d\x90{\xb8F\xa4\x81H?o\x10\xc9*\x87\x9c" +
	"d\xd0\xc1*PE\xabA$/p\xc8)\x06\x9d\xf1" +
	"\x0a\x18\x91.\xcf\x11\xc9I\x0ey\x8d\xc1\xbc\x15\xfa\x81" +
	"\xe5\xa2H\x0cEB\x16\xd9qZ\x0b\x9b\x01!E\x81" +
	"\x18\x0a\x84\xb6\xaa]n\xce\xff\xbd\x0f\xf5j\x1c\xf4\x92" +
	"\xe6\xdc\x7f\xb5\x0e\xf5k)\xd5\xdc\xea\xf0p\xab\xa4\xe9" +
	"8\x9e\xe7z\x04\x17 \x06\x10\x0a\x8e\x1da\x8c\x18\xc6" +
	"\x86\x18]\xf1]\x8f\x94K1\x8f\xd7\x0d\"@\x1f5" +
	"\x88\xcc\xc4\x8b\x17\xbd\xd8t\xe6\xd4\xaew\x9a\xf9?\xcd" +
	"S7\xfd\xd8U;n\xd5\xec\x88\x0f\x7f\x8ez\x1c." +
	"\xfaf\xe2\x87A\xceHmg\xee\x92\x9a\xbb\xc8!+" +
	"\xfd\xe3\x0cJ\xbaj\xa7\x96K\xd4\x13r\xac\x1bRN" +
	"|\xd7C\xb9\xfb\xc1\x09(\x13\xfe\x04\x00\x00\xff\xff[" +
	"\xb7\xf3\xb7"

func init() {
	schemas.Register(schema_a184c7885cdaf2a1,
		0x89f389b6fd4082c1,
		0x95b29059097fca83,
		0x9d263a3630b7ebee,
		0x9fd69ebc87b9719c,
		0xb47f4979672cb59d,
		0xb88d09a9c5f39817,
		0xd20b909fee733a8e)
}
