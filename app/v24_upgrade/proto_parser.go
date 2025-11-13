package v24_upgrade

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Protobuf wire types
const (
	WireVarint     = 0
	WireFixed64    = 1
	WireBytes      = 2
	WireStartGroup = 3
	WireEndGroup   = 4
	WireFixed32    = 5
)

// ParseProtobufFields parses raw protobuf data and returns a map of field number to field data
func ParseProtobufFields(data []byte) (map[int]*ProtobufField, error) {
	fields := make(map[int]*ProtobufField)
	offset := 0

	for offset < len(data) {
		// Read field tag (field number + wire type)
		tag, n := binary.Uvarint(data[offset:])
		if n <= 0 {
			return nil, fmt.Errorf("failed to read field tag at offset %d", offset)
		}
		offset += n

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)

		// Read field data based on wire type
		fieldData, bytesRead, err := readFieldData(data[offset:], wireType)
		if err != nil {
			return nil, fmt.Errorf("failed to read field %d data: %w", fieldNumber, err)
		}

		fields[fieldNumber] = &ProtobufField{
			Number:   fieldNumber,
			WireType: wireType,
			Data:     fieldData,
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

	case WireFixed64:
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

	case WireFixed32:
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

// SwapFields7And8 swaps the data at field positions 7 and 8 in the protobuf data
func SwapFields7And8(data []byte) ([]byte, error) {
	fields, err := ParseProtobufFields(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse protobuf: %w", err)
	}

	// Get fields 7 and 8
	field7, has7 := fields[7]
	field8, has8 := fields[8]

	if !has7 && !has8 {
		// Neither field exists - nothing to swap
		return data, nil
	}

	// Rebuild the protobuf with swapped fields
	result := make([]byte, 0, len(data))

	// Write all fields except 7 and 8 in order
	for fieldNum := 1; fieldNum <= 10; fieldNum++ { // Assuming max 10 fields in ContractInfo
		if fieldNum == 7 {
			// Write field 8's data with field number 7
			if has8 {
				result = append(result, EncodeFieldTag(7, field8.WireType)...)
				result = append(result, field8.Data...)
			}
		} else if fieldNum == 8 {
			// Write field 7's data with field number 8
			if has7 {
				result = append(result, EncodeFieldTag(8, field7.WireType)...)
				result = append(result, field7.Data...)
			}
		} else if field, ok := fields[fieldNum]; ok {
			// Write original field
			result = append(result, EncodeFieldTag(fieldNum, field.WireType)...)
			result = append(result, field.Data...)
		}
	}

	return result, nil
}

// ClearField8 removes field 8 from the protobuf data (nullifies ibc2_port_id)
func ClearField8(data []byte) ([]byte, error) {
	fields, err := ParseProtobufFields(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse protobuf: %w", err)
	}

	// If field 8 doesn't exist, nothing to do
	if _, has8 := fields[8]; !has8 {
		return data, nil
	}

	// Rebuild protobuf without field 8
	result := make([]byte, 0, len(data))

	for fieldNum := 1; fieldNum <= 10; fieldNum++ {
		if fieldNum == 8 {
			continue // Skip field 8
		}
		if field, ok := fields[fieldNum]; ok {
			result = append(result, EncodeFieldTag(fieldNum, field.WireType)...)
			result = append(result, field.Data...)
		}
	}

	return result, nil
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

// IsField8Empty checks if field 8 is empty or absent
func IsField8Empty(data []byte) bool {
	value, err := GetFieldValue(data, 8)
	if err != nil {
		return true // Error means no valid data
	}
	return len(value) == 0
}

// AddEmptyField7 adds an empty extension field at position 7
// This is used to migrate SchemaLegacy contracts to SchemaCanonical
func AddEmptyField7(data []byte) ([]byte, error) {
	fields, err := ParseProtobufFields(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse protobuf: %w", err)
	}

	// If field 7 already exists, nothing to do
	if _, has7 := fields[7]; has7 {
		return data, nil
	}

	// Create empty extension field (wire type 2 = length-delimited, length 0)
	emptyExtension := []byte{0} // Varint 0 = length 0

	// Rebuild the protobuf with field 7 inserted
	result := make([]byte, 0, len(data)+10) // +10 for field 7 tag + length

	// Write all fields in order, inserting field 7 in the correct position
	for fieldNum := 1; fieldNum <= 10; fieldNum++ {
		if fieldNum == 7 {
			// Insert empty field 7
			result = append(result, EncodeFieldTag(7, WireBytes)...)
			result = append(result, emptyExtension...)
		}

		if field, ok := fields[fieldNum]; ok && fieldNum != 7 {
			// Write original field
			result = append(result, EncodeFieldTag(fieldNum, field.WireType)...)
			result = append(result, field.Data...)
		}
	}

	return result, nil
}

// EnsureEmptyField8 ensures field 8 exists as an empty string
// This is different from ClearField8 which removes field 8 entirely
// After v24 migration, ALL contracts should have field 8 present but empty
func EnsureEmptyField8(data []byte) ([]byte, error) {
	fields, err := ParseProtobufFields(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse protobuf: %w", err)
	}

	// Check if field 8 already exists and is empty
	if _, has8 := fields[8]; has8 {
		value, err := GetFieldValue(data, 8)
		if err == nil && len(value) == 0 {
			// Field 8 already exists and is empty
			return data, nil
		}
		// Field 8 exists but has data - we'll replace it
	}

	// Create empty string field (wire type 2 = length-delimited, length 0)
	emptyString := []byte{0} // Varint 0 = length 0

	// Rebuild the protobuf with field 8 as empty string
	result := make([]byte, 0, len(data)+10)

	// Write all fields in order, setting/adding field 8 as empty
	for fieldNum := 1; fieldNum <= 10; fieldNum++ {
		if fieldNum == 8 {
			// Set field 8 to empty string (wire type 2 for string)
			result = append(result, EncodeFieldTag(8, WireBytes)...)
			result = append(result, emptyString...)
		} else if field, ok := fields[fieldNum]; ok {
			// Write original field
			result = append(result, EncodeFieldTag(fieldNum, field.WireType)...)
			result = append(result, field.Data...)
		}
	}

	return result, nil
}
