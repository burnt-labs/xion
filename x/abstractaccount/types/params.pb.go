// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: abstractaccount/v1/params.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
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

type Params struct {
	// AllowAllCodeIDs determines whether a Wasm code ID can be used to register
	// AbstractAccounts:
	// - if set to true, any code ID can be used;
	// - if set to false, only code IDs whitelisted in the AllowedCodeIDs list can
	// be used.
	AllowAllCodeIDs bool `protobuf:"varint,1,opt,name=allow_all_code_ids,json=allowAllCodeIds,proto3" json:"allow_all_code_ids,omitempty"`
	// AllowedCodeIDs is the whitelist of Wasm code IDs that can be used to
	// regiseter AbstractAccounts.
	AllowedCodeIDs []uint64 `protobuf:"varint,2,rep,packed,name=allowed_code_ids,json=allowedCodeIds,proto3" json:"allowed_code_ids,omitempty"`
	// MaxGasBefore is the maximum amount of gas that can be consumed by the
	// contract call in the before_tx decorator.
	//
	// Must be greater than zero.
	MaxGasBefore uint64 `protobuf:"varint,3,opt,name=max_gas_before,json=maxGasBefore,proto3" json:"max_gas_before,omitempty"`
	// MaxGasAfter is the maximum amount of gas that can be consumed by the
	// contract call in the after_tx decorator.
	//
	// Must be greater than zero.
	MaxGasAfter uint64 `protobuf:"varint,4,opt,name=max_gas_after,json=maxGasAfter,proto3" json:"max_gas_after,omitempty"`
}

func (m *Params) Reset()         { *m = Params{} }
func (m *Params) String() string { return proto.CompactTextString(m) }
func (*Params) ProtoMessage()    {}
func (*Params) Descriptor() ([]byte, []int) {
	return fileDescriptor_9649df9baf604574, []int{0}
}
func (m *Params) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Params) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Params.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Params) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Params.Merge(m, src)
}
func (m *Params) XXX_Size() int {
	return m.Size()
}
func (m *Params) XXX_DiscardUnknown() {
	xxx_messageInfo_Params.DiscardUnknown(m)
}

var xxx_messageInfo_Params proto.InternalMessageInfo

func (m *Params) GetAllowAllCodeIDs() bool {
	if m != nil {
		return m.AllowAllCodeIDs
	}
	return false
}

func (m *Params) GetAllowedCodeIDs() []uint64 {
	if m != nil {
		return m.AllowedCodeIDs
	}
	return nil
}

func (m *Params) GetMaxGasBefore() uint64 {
	if m != nil {
		return m.MaxGasBefore
	}
	return 0
}

func (m *Params) GetMaxGasAfter() uint64 {
	if m != nil {
		return m.MaxGasAfter
	}
	return 0
}

func init() {
	proto.RegisterType((*Params)(nil), "abstractaccount.v1.Params")
}

func init() { proto.RegisterFile("abstractaccount/v1/params.proto", fileDescriptor_9649df9baf604574) }

var fileDescriptor_9649df9baf604574 = []byte{
	// 292 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x5c, 0x90, 0x31, 0x4b, 0xfb, 0x40,
	0x18, 0xc6, 0x7b, 0xff, 0x96, 0xf2, 0xe7, 0xd4, 0x56, 0x4e, 0x87, 0xe2, 0x70, 0x0d, 0xc5, 0x21,
	0x8b, 0x89, 0xc5, 0x51, 0x07, 0x13, 0x05, 0x71, 0x93, 0x82, 0x8b, 0x4b, 0x78, 0x93, 0xbb, 0xc6,
	0xc2, 0xc5, 0x0b, 0x77, 0xd7, 0x9a, 0x7e, 0x0b, 0x3f, 0x96, 0x63, 0x71, 0x72, 0x2a, 0x92, 0x7c,
	0x11, 0xc9, 0x85, 0xd0, 0xe2, 0xf6, 0xbe, 0xbf, 0xe7, 0xf7, 0x2c, 0x0f, 0x1e, 0x43, 0xac, 0x8d,
	0x82, 0xc4, 0x40, 0x92, 0xc8, 0xe5, 0x9b, 0xf1, 0x57, 0x53, 0x3f, 0x07, 0x05, 0x99, 0xf6, 0x72,
	0x25, 0x8d, 0x24, 0xe4, 0x8f, 0xe0, 0xad, 0xa6, 0x67, 0xa7, 0xa9, 0x4c, 0xa5, 0x8d, 0xfd, 0xfa,
	0x6a, 0xcc, 0xc9, 0x17, 0xc2, 0xfd, 0x27, 0x5b, 0x25, 0xb7, 0x98, 0x80, 0x10, 0xf2, 0x3d, 0x02,
	0x21, 0xa2, 0x44, 0x32, 0x1e, 0x2d, 0x98, 0x1e, 0x21, 0x07, 0xb9, 0xff, 0xc3, 0x93, 0x72, 0x3b,
	0x1e, 0x06, 0x75, 0x1a, 0x08, 0x71, 0x27, 0x19, 0x7f, 0xbc, 0xd7, 0xb3, 0x21, 0xec, 0x03, 0xa6,
	0xc9, 0x0d, 0x3e, 0xb6, 0x88, 0xb3, 0x5d, 0xff, 0x9f, 0xd3, 0x75, 0x7b, 0x21, 0x29, 0xb7, 0xe3,
	0x41, 0xd0, 0x64, 0x6d, 0x7d, 0x00, 0x7b, 0x3f, 0xd3, 0xe4, 0x1c, 0x0f, 0x32, 0x28, 0xa2, 0x14,
	0x74, 0x14, 0xf3, 0xb9, 0x54, 0x7c, 0xd4, 0x75, 0x90, 0xdb, 0x9b, 0x1d, 0x66, 0x50, 0x3c, 0x80,
	0x0e, 0x2d, 0x23, 0x13, 0x7c, 0xd4, 0x5a, 0x30, 0x37, 0x5c, 0x8d, 0x7a, 0x56, 0x3a, 0x68, 0xa4,
	0xa0, 0x46, 0xe1, 0xf3, 0x67, 0x49, 0xd1, 0xa6, 0xa4, 0xe8, 0xa7, 0xa4, 0xe8, 0xa3, 0xa2, 0x9d,
	0x4d, 0x45, 0x3b, 0xdf, 0x15, 0xed, 0xbc, 0x5c, 0xa7, 0x0b, 0xf3, 0xba, 0x8c, 0xbd, 0x44, 0x66,
	0xbe, 0x00, 0xa5, 0xd6, 0x97, 0x85, 0xdf, 0x6e, 0x75, 0xd1, 0xae, 0xb9, 0x43, 0x2d, 0x31, 0xeb,
	0x9c, 0xeb, 0xb8, 0x6f, 0x27, 0xbb, 0xfa, 0x0d, 0x00, 0x00, 0xff, 0xff, 0x2d, 0xd6, 0xf8, 0xa4,
	0x7f, 0x01, 0x00, 0x00,
}

func (m *Params) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Params) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Params) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.MaxGasAfter != 0 {
		i = encodeVarintParams(dAtA, i, uint64(m.MaxGasAfter))
		i--
		dAtA[i] = 0x20
	}
	if m.MaxGasBefore != 0 {
		i = encodeVarintParams(dAtA, i, uint64(m.MaxGasBefore))
		i--
		dAtA[i] = 0x18
	}
	if len(m.AllowedCodeIDs) > 0 {
		dAtA2 := make([]byte, len(m.AllowedCodeIDs)*10)
		var j1 int
		for _, num := range m.AllowedCodeIDs {
			for num >= 1<<7 {
				dAtA2[j1] = uint8(uint64(num)&0x7f | 0x80)
				num >>= 7
				j1++
			}
			dAtA2[j1] = uint8(num)
			j1++
		}
		i -= j1
		copy(dAtA[i:], dAtA2[:j1])
		i = encodeVarintParams(dAtA, i, uint64(j1))
		i--
		dAtA[i] = 0x12
	}
	if m.AllowAllCodeIDs {
		i--
		if m.AllowAllCodeIDs {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintParams(dAtA []byte, offset int, v uint64) int {
	offset -= sovParams(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Params) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.AllowAllCodeIDs {
		n += 2
	}
	if len(m.AllowedCodeIDs) > 0 {
		l = 0
		for _, e := range m.AllowedCodeIDs {
			l += sovParams(uint64(e))
		}
		n += 1 + sovParams(uint64(l)) + l
	}
	if m.MaxGasBefore != 0 {
		n += 1 + sovParams(uint64(m.MaxGasBefore))
	}
	if m.MaxGasAfter != 0 {
		n += 1 + sovParams(uint64(m.MaxGasAfter))
	}
	return n
}

func sovParams(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozParams(x uint64) (n int) {
	return sovParams(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Params) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowParams
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
			return fmt.Errorf("proto: Params: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Params: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field AllowAllCodeIDs", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowParams
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.AllowAllCodeIDs = bool(v != 0)
		case 2:
			if wireType == 0 {
				var v uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowParams
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					v |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				m.AllowedCodeIDs = append(m.AllowedCodeIDs, v)
			} else if wireType == 2 {
				var packedLen int
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowParams
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					packedLen |= int(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				if packedLen < 0 {
					return ErrInvalidLengthParams
				}
				postIndex := iNdEx + packedLen
				if postIndex < 0 {
					return ErrInvalidLengthParams
				}
				if postIndex > l {
					return io.ErrUnexpectedEOF
				}
				var elementCount int
				var count int
				for _, integer := range dAtA[iNdEx:postIndex] {
					if integer < 128 {
						count++
					}
				}
				elementCount = count
				if elementCount != 0 && len(m.AllowedCodeIDs) == 0 {
					m.AllowedCodeIDs = make([]uint64, 0, elementCount)
				}
				for iNdEx < postIndex {
					var v uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowParams
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						v |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					m.AllowedCodeIDs = append(m.AllowedCodeIDs, v)
				}
			} else {
				return fmt.Errorf("proto: wrong wireType = %d for field AllowedCodeIDs", wireType)
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaxGasBefore", wireType)
			}
			m.MaxGasBefore = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowParams
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MaxGasBefore |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaxGasAfter", wireType)
			}
			m.MaxGasAfter = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowParams
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MaxGasAfter |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipParams(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthParams
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
func skipParams(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowParams
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
					return 0, ErrIntOverflowParams
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
					return 0, ErrIntOverflowParams
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
				return 0, ErrInvalidLengthParams
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupParams
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthParams
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthParams        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowParams          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupParams = fmt.Errorf("proto: unexpected end of group")
)