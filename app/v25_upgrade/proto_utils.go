package v25_upgrade

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
)

// ParseProtobufFields parses raw protobuf data and returns a map of field number to field data
// Returns error if the protobuf structure is invalid
func ParseProtobufFields(data []byte) (map[int]*ProtobufField, error) {
	fields := make(map[int]*ProtobufField)
	offset := 0

	for offset < len(data) {
		startOffset := offset

		// Read field tag (field number + wire type)
		tag, n := binary.Uvarint(data[offset:])
		if n <= 0 {
			return nil, fmt.Errorf("failed to read field tag at offset %d", offset)
		}
		offset += n

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)

		// Validate wire type (6 and 7 are invalid in proto3)
		if wireType == 6 || wireType == 7 {
			return nil, fmt.Errorf("invalid wire type %d for field %d at offset %d", wireType, fieldNumber, startOffset)
		}

		// Read field data based on wire type
		fieldData, bytesRead, err := readFieldData(data[offset:], wireType)
		if err != nil {
			return nil, fmt.Errorf("failed to read field %d data at offset %d: %w", fieldNumber, offset, err)
		}

		totalLength := (offset - startOffset) + bytesRead

		fields[fieldNumber] = &ProtobufField{
			Number:   fieldNumber,
			WireType: wireType,
			Data:     fieldData,
			Offset:   startOffset,
			Length:   totalLength,
		}

		offset += bytesRead
	}

	return fields, nil
}

// readFieldData reads the data for a field based on its wire type
func readFieldData(data []byte, wireType int) ([]byte, int, error) {
	switch wireType {
	case WireVarint:
		// Read varint
		_, n := binary.Uvarint(data)
		if n <= 0 {
			return nil, 0, fmt.Errorf("failed to read varint")
		}
		return data[:n], n, nil

	case Wire64Bit:
		// Read 8 bytes
		if len(data) < 8 {
			return nil, 0, io.ErrUnexpectedEOF
		}
		return data[:8], 8, nil

	case WireBytes:
		// Read length-delimited data
		length, n := binary.Uvarint(data)
		if n <= 0 {
			return nil, 0, fmt.Errorf("failed to read length")
		}
		totalLen := n + int(length)
		if len(data) < totalLen {
			return nil, 0, io.ErrUnexpectedEOF
		}
		return data[:totalLen], totalLen, nil

	case Wire32Bit:
		// Read 4 bytes
		if len(data) < 4 {
			return nil, 0, io.ErrUnexpectedEOF
		}
		return data[:4], 4, nil

	default:
		return nil, 0, fmt.Errorf("unsupported wire type: %d", wireType)
	}
}

// EncodeFieldTag encodes a field number and wire type into a protobuf tag
func EncodeFieldTag(fieldNumber, wireType int) []byte {
	tag := uint64(fieldNumber<<3 | wireType)
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, tag)
	return buf[:n]
}

// GetFieldValue extracts the raw value of a specific field
func GetFieldValue(data []byte, fieldNumber int) ([]byte, error) {
	fields, err := ParseProtobufFields(data)
	if err != nil {
		return nil, err
	}

	field, ok := fields[fieldNumber]
	if !ok {
		return nil, nil // Field doesn't exist
	}

	// For length-delimited fields (wire type 2), extract the actual data without the length prefix
	if field.WireType == WireBytes {
		length, n := binary.Uvarint(field.Data)
		if n <= 0 {
			return nil, fmt.Errorf("failed to read field length")
		}
		return field.Data[n : n+int(length)], nil
	}

	return field.Data, nil
}

// HasField checks if a field exists in the protobuf data
func HasField(data []byte, fieldNumber int) bool {
	fields, err := ParseProtobufFields(data)
	if err != nil {
		return false
	}
	_, ok := fields[fieldNumber]
	return ok
}

// IsFieldEmpty checks if a field is empty or absent
func IsFieldEmpty(data []byte, fieldNumber int) bool {
	value, err := GetFieldValue(data, fieldNumber)
	if err != nil {
		return true // Error means no valid data
	}
	return len(value) == 0
}

// GetFieldDataWithTag returns the field data including its tag
// Used for raw manipulation
func GetFieldDataWithTag(data []byte, fieldNumber int) ([]byte, error) {
	fields, err := ParseProtobufFields(data)
	if err != nil {
		return nil, err
	}

	field, ok := fields[fieldNumber]
	if !ok {
		return nil, fmt.Errorf("field %d not found", fieldNumber)
	}

	// Extract the complete field (tag + data) from original data
	return data[field.Offset : field.Offset+field.Length], nil
}

// HexDump returns a hex dump of data (for debugging)
// Limits to maxBytes to avoid huge outputs
func HexDump(data []byte, maxBytes int) string {
	if len(data) > maxBytes {
		return hex.EncodeToString(data[:maxBytes]) + "..."
	}
	return hex.EncodeToString(data)
}

// AnalyzeProtobufStructure provides detailed analysis of protobuf structure
// Returns human-readable information about fields
func AnalyzeProtobufStructure(data []byte) (string, error) {
	fields, err := ParseProtobufFields(data)
	if err != nil {
		return "", fmt.Errorf("failed to parse protobuf: %w", err)
	}

	result := fmt.Sprintf("Total size: %d bytes\n", len(data))
	result += fmt.Sprintf("Fields found: %d\n\n", len(fields))

	for i := 1; i <= 10; i++ { // ContractInfo has up to 8 fields
		if field, ok := fields[i]; ok {
			wireTypeName := getWireTypeName(field.WireType)
			value, _ := GetFieldValue(data, i)
			result += fmt.Sprintf("Field %d: %s (wire type %d)\n", i, wireTypeName, field.WireType)
			result += fmt.Sprintf("  Offset: %d, Length: %d\n", field.Offset, field.Length)
			result += fmt.Sprintf("  Data size: %d bytes\n", len(value))
			if len(value) > 0 && len(value) <= 100 {
				result += fmt.Sprintf("  Value (hex): %s\n", hex.EncodeToString(value))
			}
			result += "\n"
		}
	}

	return result, nil
}

// getWireTypeName returns the name of a wire type
func getWireTypeName(wireType int) string {
	switch wireType {
	case WireVarint:
		return "Varint"
	case Wire64Bit:
		return "64-bit"
	case WireBytes:
		return "Length-delimited"
	case Wire32Bit:
		return "32-bit"
	default:
		return "Unknown"
	}
}

// FormatAddress formats an address for display
// Truncates long addresses for readability
func FormatAddress(address string) string {
	if len(address) > 50 {
		return address[:20] + "..." + address[len(address)-20:]
	}
	return address
}
