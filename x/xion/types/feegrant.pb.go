// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: xion/v1/feegrant.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	types "github.com/cosmos/cosmos-sdk/codec/types"
	_ "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	_ "google.golang.org/protobuf/types/known/durationpb"
	_ "google.golang.org/protobuf/types/known/timestamppb"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// AuthzAllowance creates allowance only authz message for a specific grantee
type AuthzAllowance struct {
	// allowance can be any of basic and periodic fee allowance.
	Allowance    *types.Any `protobuf:"bytes,1,opt,name=allowance,proto3" json:"allowance,omitempty"`
	AuthzGrantee string     `protobuf:"bytes,2,opt,name=authz_grantee,json=authzGrantee,proto3" json:"authz_grantee,omitempty"`
}

func (m *AuthzAllowance) Reset()         { *m = AuthzAllowance{} }
func (m *AuthzAllowance) String() string { return proto.CompactTextString(m) }
func (*AuthzAllowance) ProtoMessage()    {}
func (*AuthzAllowance) Descriptor() ([]byte, []int) {
	return fileDescriptor_38e1987a87c7c3e9, []int{0}
}
func (m *AuthzAllowance) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *AuthzAllowance) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_AuthzAllowance.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *AuthzAllowance) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AuthzAllowance.Merge(m, src)
}
func (m *AuthzAllowance) XXX_Size() int {
	return m.Size()
}
func (m *AuthzAllowance) XXX_DiscardUnknown() {
	xxx_messageInfo_AuthzAllowance.DiscardUnknown(m)
}

var xxx_messageInfo_AuthzAllowance proto.InternalMessageInfo

// ContractsAllowance creates allowance only for specific contracts
type ContractsAllowance struct {
	// allowance can be any allowance interface type.
	Allowance         *types.Any `protobuf:"bytes,1,opt,name=allowance,proto3" json:"allowance,omitempty"`
	ContractAddresses []string   `protobuf:"bytes,2,rep,name=contract_addresses,json=contractAddresses,proto3" json:"contract_addresses,omitempty"`
}

func (m *ContractsAllowance) Reset()         { *m = ContractsAllowance{} }
func (m *ContractsAllowance) String() string { return proto.CompactTextString(m) }
func (*ContractsAllowance) ProtoMessage()    {}
func (*ContractsAllowance) Descriptor() ([]byte, []int) {
	return fileDescriptor_38e1987a87c7c3e9, []int{1}
}
func (m *ContractsAllowance) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ContractsAllowance) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ContractsAllowance.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ContractsAllowance) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ContractsAllowance.Merge(m, src)
}
func (m *ContractsAllowance) XXX_Size() int {
	return m.Size()
}
func (m *ContractsAllowance) XXX_DiscardUnknown() {
	xxx_messageInfo_ContractsAllowance.DiscardUnknown(m)
}

var xxx_messageInfo_ContractsAllowance proto.InternalMessageInfo

// MultiAnyAllowance creates an allowance that pays if any of the internal allowances are met
type MultiAnyAllowance struct {
	// allowance can be any allowance interface type.
	Allowances []*types.Any `protobuf:"bytes,1,rep,name=allowances,proto3" json:"allowances,omitempty"`
}

func (m *MultiAnyAllowance) Reset()         { *m = MultiAnyAllowance{} }
func (m *MultiAnyAllowance) String() string { return proto.CompactTextString(m) }
func (*MultiAnyAllowance) ProtoMessage()    {}
func (*MultiAnyAllowance) Descriptor() ([]byte, []int) {
	return fileDescriptor_38e1987a87c7c3e9, []int{2}
}
func (m *MultiAnyAllowance) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MultiAnyAllowance) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MultiAnyAllowance.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MultiAnyAllowance) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MultiAnyAllowance.Merge(m, src)
}
func (m *MultiAnyAllowance) XXX_Size() int {
	return m.Size()
}
func (m *MultiAnyAllowance) XXX_DiscardUnknown() {
	xxx_messageInfo_MultiAnyAllowance.DiscardUnknown(m)
}

var xxx_messageInfo_MultiAnyAllowance proto.InternalMessageInfo

func init() {
	proto.RegisterType((*AuthzAllowance)(nil), "xion.v1.AuthzAllowance")
	proto.RegisterType((*ContractsAllowance)(nil), "xion.v1.ContractsAllowance")
	proto.RegisterType((*MultiAnyAllowance)(nil), "xion.v1.MultiAnyAllowance")
}

func init() { proto.RegisterFile("xion/v1/feegrant.proto", fileDescriptor_38e1987a87c7c3e9) }

var fileDescriptor_38e1987a87c7c3e9 = []byte{
	// 445 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xbc, 0x53, 0xbf, 0x0a, 0xd3, 0x40,
	0x1c, 0xce, 0xb5, 0xa0, 0xf4, 0xfc, 0x03, 0x8d, 0x45, 0x63, 0x87, 0x34, 0x14, 0xc4, 0x2a, 0x34,
	0x47, 0x74, 0x2b, 0x38, 0xa4, 0xa2, 0xb5, 0x83, 0x4b, 0xdd, 0x04, 0x09, 0x97, 0xf4, 0x9a, 0x06,
	0x92, 0xbb, 0x92, 0xbb, 0xd4, 0xc6, 0x17, 0x50, 0x9c, 0x7c, 0x04, 0x1f, 0xc1, 0xa1, 0xb3, 0xb3,
	0x38, 0x15, 0x27, 0x47, 0x69, 0x07, 0x9f, 0xc0, 0x5d, 0x72, 0x97, 0xb4, 0xda, 0x82, 0x58, 0x10,
	0x97, 0xe4, 0x7e, 0xff, 0xbe, 0xdf, 0xf7, 0x7d, 0xc7, 0xc1, 0xeb, 0xab, 0x88, 0x51, 0xb4, 0x74,
	0xd0, 0x8c, 0x90, 0x30, 0xc5, 0x54, 0xd8, 0x8b, 0x94, 0x09, 0xa6, 0x5f, 0x2c, 0xf2, 0xf6, 0xd2,
	0x69, 0xb7, 0x42, 0x16, 0x32, 0x99, 0x43, 0xc5, 0x49, 0x95, 0xdb, 0x37, 0x43, 0xc6, 0xc2, 0x98,
	0x20, 0x19, 0xf9, 0xd9, 0x0c, 0x61, 0x9a, 0x57, 0xa5, 0x80, 0xf1, 0x84, 0x71, 0x4f, 0xcd, 0xa8,
	0xa0, 0x2c, 0x99, 0x2a, 0x42, 0x3e, 0xe6, 0x04, 0x2d, 0x1d, 0x9f, 0x08, 0xec, 0xa0, 0x80, 0x45,
	0xb4, 0xac, 0x37, 0x71, 0x12, 0x51, 0x86, 0xe4, 0xb7, 0x4c, 0x75, 0x8e, 0x17, 0x89, 0x28, 0x21,
	0x5c, 0xe0, 0x64, 0x51, 0x61, 0x1e, 0x37, 0x4c, 0xb3, 0x14, 0x8b, 0x82, 0xbc, 0xcc, 0x74, 0x7f,
	0x00, 0x78, 0xd5, 0xcd, 0xc4, 0xfc, 0x95, 0x1b, 0xc7, 0xec, 0x25, 0xa6, 0x01, 0xd1, 0x5f, 0xc0,
	0x06, 0xae, 0x02, 0x03, 0x58, 0xa0, 0x77, 0xe9, 0x5e, 0xcb, 0x56, 0x30, 0x76, 0x05, 0x63, 0xbb,
	0x34, 0x1f, 0xde, 0xf9, 0xbc, 0xee, 0xdf, 0x2a, 0x15, 0xec, 0xfd, 0x29, 0x79, 0xdb, 0x8f, 0x09,
	0xd9, 0x43, 0x8e, 0x27, 0x07, 0x44, 0xfd, 0x01, 0xbc, 0x82, 0x8b, 0x85, 0x9e, 0xec, 0x27, 0xc4,
	0xa8, 0x59, 0xa0, 0xd7, 0x18, 0x1a, 0x5f, 0xd6, 0xfd, 0x56, 0x09, 0xe6, 0x4e, 0xa7, 0x29, 0xe1,
	0xfc, 0x99, 0x48, 0x23, 0x1a, 0x4e, 0x2e, 0xcb, 0xf6, 0x91, 0xea, 0x1e, 0x3c, 0x7a, 0xf3, 0xbe,
	0xa3, 0xfd, 0xf5, 0xe2, 0xb7, 0xdf, 0x3f, 0xdc, 0xbd, 0x26, 0xef, 0xf0, 0x77, 0x91, 0xdd, 0xd7,
	0x35, 0xa8, 0x3f, 0x64, 0x54, 0xa4, 0x38, 0x10, 0xfc, 0xbf, 0x69, 0x1f, 0x41, 0x3d, 0x28, 0x97,
	0x7a, 0x58, 0x89, 0x24, 0xdc, 0xa8, 0x59, 0xf5, 0x3f, 0x1a, 0xd0, 0xac, 0x66, 0xdc, 0x6a, 0x64,
	0x30, 0x3e, 0xdb, 0x85, 0x1b, 0xd2, 0x85, 0x53, 0xc9, 0xdd, 0x8f, 0x00, 0x36, 0x9f, 0x66, 0xb1,
	0x88, 0x5c, 0x9a, 0x1f, 0x8c, 0xf0, 0x20, 0xdc, 0xd3, 0xe6, 0x06, 0xb0, 0xea, 0xff, 0xc2, 0x89,
	0x5f, 0x20, 0x07, 0x4f, 0xce, 0x56, 0xa0, 0xde, 0xe2, 0x09, 0xd5, 0xa1, 0xfb, 0x69, 0x6b, 0x82,
	0xcd, 0xd6, 0x04, 0xdf, 0xb6, 0x26, 0x78, 0xb7, 0x33, 0xb5, 0xcd, 0xce, 0xd4, 0xbe, 0xee, 0x4c,
	0xed, 0xf9, 0xed, 0x30, 0x12, 0xf3, 0xcc, 0xb7, 0x03, 0x96, 0x20, 0x3f, 0x4b, 0xa9, 0xe8, 0xc7,
	0xd8, 0xe7, 0x48, 0xe2, 0xac, 0xd4, 0x4f, 0xe4, 0x0b, 0xc2, 0xfd, 0x0b, 0x52, 0xd1, 0xfd, 0x9f,
	0x01, 0x00, 0x00, 0xff, 0xff, 0xcf, 0x43, 0xb0, 0x30, 0xef, 0x03, 0x00, 0x00,
}

func (m *AuthzAllowance) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *AuthzAllowance) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *AuthzAllowance) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.AuthzGrantee) > 0 {
		i -= len(m.AuthzGrantee)
		copy(dAtA[i:], m.AuthzGrantee)
		i = encodeVarintFeegrant(dAtA, i, uint64(len(m.AuthzGrantee)))
		i--
		dAtA[i] = 0x12
	}
	if m.Allowance != nil {
		{
			size, err := m.Allowance.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintFeegrant(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ContractsAllowance) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ContractsAllowance) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ContractsAllowance) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.ContractAddresses) > 0 {
		for iNdEx := len(m.ContractAddresses) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.ContractAddresses[iNdEx])
			copy(dAtA[i:], m.ContractAddresses[iNdEx])
			i = encodeVarintFeegrant(dAtA, i, uint64(len(m.ContractAddresses[iNdEx])))
			i--
			dAtA[i] = 0x12
		}
	}
	if m.Allowance != nil {
		{
			size, err := m.Allowance.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintFeegrant(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MultiAnyAllowance) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MultiAnyAllowance) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MultiAnyAllowance) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Allowances) > 0 {
		for iNdEx := len(m.Allowances) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Allowances[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintFeegrant(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func encodeVarintFeegrant(dAtA []byte, offset int, v uint64) int {
	offset -= sovFeegrant(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *AuthzAllowance) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Allowance != nil {
		l = m.Allowance.Size()
		n += 1 + l + sovFeegrant(uint64(l))
	}
	l = len(m.AuthzGrantee)
	if l > 0 {
		n += 1 + l + sovFeegrant(uint64(l))
	}
	return n
}

func (m *ContractsAllowance) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Allowance != nil {
		l = m.Allowance.Size()
		n += 1 + l + sovFeegrant(uint64(l))
	}
	if len(m.ContractAddresses) > 0 {
		for _, s := range m.ContractAddresses {
			l = len(s)
			n += 1 + l + sovFeegrant(uint64(l))
		}
	}
	return n
}

func (m *MultiAnyAllowance) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Allowances) > 0 {
		for _, e := range m.Allowances {
			l = e.Size()
			n += 1 + l + sovFeegrant(uint64(l))
		}
	}
	return n
}

func sovFeegrant(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozFeegrant(x uint64) (n int) {
	return sovFeegrant(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *AuthzAllowance) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowFeegrant
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: AuthzAllowance: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: AuthzAllowance: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Allowance", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFeegrant
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthFeegrant
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthFeegrant
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Allowance == nil {
				m.Allowance = &types.Any{}
			}
			if err := m.Allowance.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AuthzGrantee", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFeegrant
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthFeegrant
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFeegrant
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.AuthzGrantee = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipFeegrant(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthFeegrant
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ContractsAllowance) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowFeegrant
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ContractsAllowance: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ContractsAllowance: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Allowance", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFeegrant
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthFeegrant
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthFeegrant
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Allowance == nil {
				m.Allowance = &types.Any{}
			}
			if err := m.Allowance.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ContractAddresses", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFeegrant
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthFeegrant
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFeegrant
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ContractAddresses = append(m.ContractAddresses, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipFeegrant(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthFeegrant
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MultiAnyAllowance) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowFeegrant
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MultiAnyAllowance: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MultiAnyAllowance: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Allowances", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFeegrant
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthFeegrant
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthFeegrant
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Allowances = append(m.Allowances, &types.Any{})
			if err := m.Allowances[len(m.Allowances)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipFeegrant(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthFeegrant
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipFeegrant(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowFeegrant
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowFeegrant
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowFeegrant
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthFeegrant
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupFeegrant
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthFeegrant
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthFeegrant        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowFeegrant          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupFeegrant = fmt.Errorf("proto: unexpected end of group")
)