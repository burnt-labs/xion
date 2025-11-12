# Wire Type Analysis: Why v20/v21/v23 Worked But v22 Broke Everything

## Executive Summary

The v20-v23 ContractInfo corruption saga is a **protobuf wire type compatibility story**. The key insight: `google.protobuf.Any` (wire type 2) and `string` (wire type 2) use the same wire encoding, allowing mismatched schemas to "work" through **accidental binary compatibility** - until the fields were in the correct positions but the stored data was in the wrong format.

## Timeline with Wire Type Analysis

### v20 (wasmd v0.61.0) - Bug Introduced, But "Works"

**Schema Definition:**
```protobuf
message ContractInfo {
  // Fields 1-6: Standard fields
  string ibc_port_id = 7;              // NEW: wire type 2 (length-delimited)
  google.protobuf.Any extension = 8;   // MOVED: wire type 2 (length-delimited)
}
```

**What Contracts Actually Wrote:**
```
Field 7: (empty string, usually omitted)
Field 8: Extension data (protobuf Any message)
```

**Why It "Worked":**
- When **writing**: Extension data went to field 8 (wrong position, but valid protobuf)
- When **reading**: Code expected field 8 to have `Any` type
- **Binary compatibility**: Both `string` and `Any` use wire type 2 (length-delimited bytes)
- Result: Decoder reads field 8 as `Any`, gets `Any` data → ✅ Works!

### v21 (wasmd v0.61.4) - Same Bug, Still "Works"

Same as v20. Contracts continued writing `extension` to field 8, code continued reading from field 8. **No problems because read/write schemas matched.**

### v22 (wasmd v0.61.6-xion.1) - "Fixed" Schema, Everything BREAKS

**Schema Definition (Corrected):**
```protobuf
message ContractInfo {
  // Fields 1-6: Standard fields
  google.protobuf.Any extension = 7;   // MOVED BACK: wire type 2
  string ibc2_port_id = 8;             // NEW: wire type 2
}
```

**What v22 Code Expected to Read:**
```
Field 7: Extension (protobuf Any)
Field 8: ibc2_port_id (string)
```

**What Was Actually Stored (from v20/v21):**
```
Field 7: (empty or ibc_port_id string)
Field 8: Extension (protobuf Any)
```

**Why It BROKE:**

`★ Insight ─────────────────────────────────────`
**The Critical Mismatch**

The v22 code tried to read field 7 as `google.protobuf.Any`, but the stored data had either:
1. Nothing (empty field 7)
2. A string value (ibc_port_id)

When the decoder tried to parse a **simple string as a protobuf Any message**, it encountered invalid nested message structure:
- `Any` expects: `type_url` + `value` (specific format)
- Got instead: Empty or string bytes
- Result: **`proto: illegal wireType 7`** error

The error "wireType 7" is misleading - it's actually the decoder getting confused trying to parse string data as a complex Any message.
`─────────────────────────────────────────────────`

**Detailed Breakdown:**

1. **v20/v21 contracts** stored `extension` (Any) at field 8
2. **v22 code** reads field 7 expecting `extension` (Any)
3. Field 7 is empty or has string data (not Any format)
4. Protobuf decoder tries to parse field 7 as Any:
   ```
   Expected Any format:
   - type_url (string, field 1)
   - value (bytes, field 2)

   Got instead:
   - Empty bytes OR
   - String data (not nested message structure)
   ```
5. Decoder fails: "This doesn't look like an Any message!"
6. Error: `proto: illegal wireType 7`

### v23 (wasmd v0.61.6-xion.2) - Reverted to "Broken" Schema

**Schema Definition (REVERTED to v20/v21):**
```protobuf
message ContractInfo {
  // Fields 1-6: Standard fields
  string ibc_port_id = 7;              // BACK to wrong position
  google.protobuf.Any extension = 8;   // BACK to wrong position
}
```

**Why It "Works" Again:**
- Reverted code back to reading field 8 for extension
- Old v20/v21 contracts have extension at field 8 → ✅ Works!
- New v22 contracts written during the brief window need migration
- Schema and data positions match again (even though both are "wrong")

**The New Problem:**
```
v20/v21 era contracts: field 8 has Any data → Works with v23 ✅
v22 era contracts:     field 7 has Any data → Broken on v23 ✗
```

This is why "some contracts created on v21 (and possibly before) still have corruption issues on v23" - they were actually created or modified **during the v22 window**.

### v24 (wasmd v0.61.6-xion.3) - Migration to Correct Schema

**Schema Definition (Correct, like v22):**
```protobuf
message ContractInfo {
  // Fields 1-6: Standard fields
  google.protobuf.Any extension = 7;   // Correct position
  string ibc2_port_id = 8;             // Correct position (null)
}
```

**Migration Strategy:**
1. **Detect**: Check if field 8 has data (indicates v20/v21/v23 schema)
2. **Swap**: Move data from field 8 → field 7
3. **Clear**: Null field 8 (IBCv2 never used)
4. **Verify**: All contracts now have Any at field 7, empty field 8

## Wire Type Compatibility Table

| Type | Wire Type | Binary Format | Compatible With |
|------|-----------|---------------|-----------------|
| `string` | 2 (bytes) | length + data | `bytes`, `Any` |
| `bytes` | 2 (bytes) | length + data | `string`, `Any` |
| `google.protobuf.Any` | 2 (bytes) | length + nested message | `string`, `bytes` |
| `uint64` | 0 (varint) | variable int | Other varints |
| `fixed64` | 1 (64-bit) | 8 bytes | Nothing |

**Key Insight**: Wire type 2 (length-delimited) fields can be read as different types as long as the **decoder's expectations match the stored structure**.

## Why "illegal wireType 7" Error?

The error message is misleading. Wire type 7 doesn't officially exist in protobuf (only 0-5 are valid). Here's what really happened:

```
Stored data at field 7: "" (empty) or "some_string"
Decoder expects:        google.protobuf.Any

Decoder attempts to parse as Any:
1. Read type_url field (field 1 of Any)
2. Encounters unexpected byte sequence
3. Tries to read field tag
4. Gets corrupted/invalid tag that decodes to "7"
5. Error: "illegal wireType 7"
```

The "7" is actually a **corrupted field tag** from trying to parse non-Any data as an Any message.

## Binary Compatibility Examples

### Example 1: Why v20/v21 "Worked"

**Written by v20/v21:**
```
Field 8, Wire Type 2 (bytes):
  Length: 50
  Data: [Any message bytes: type_url + value]
```

**Read by v20/v21:**
```protobuf
google.protobuf.Any extension = 8;
```
- Expects field 8, wire type 2 ✓
- Expects Any message structure ✓
- Data IS Any message ✓
- **Result: Success!**

### Example 2: Why v22 Broke

**Stored by v20/v21:**
```
Field 7: (empty)
Field 8, Wire Type 2: [Any message bytes]
```

**Read by v22:**
```protobuf
google.protobuf.Any extension = 7;  // Looking at field 7!
```
- Expects field 7, wire type 2 ✓
- Expects Any message structure ✓
- Field 7 is EMPTY or has string ✗
- **Result: "illegal wireType 7" error**

### Example 3: Why v23 "Fixed" It

**Stored by v20/v21:**
```
Field 8, Wire Type 2: [Any message bytes]
```

**Read by v23 (reverted schema):**
```protobuf
google.protobuf.Any extension = 8;  // Back to reading field 8
```
- Expects field 8, wire type 2 ✓
- Expects Any message structure ✓
- Field 8 HAS Any message ✓
- **Result: Works again!**

## The v22 Contracts Problem on v23

Contracts created during the v22 window stored data "correctly":
```
Field 7, Wire Type 2: [Any message bytes]  // Correct position
Field 8: (empty)                            // Correct
```

But v23 reverted the schema, so it looks for extension at field 8:
```protobuf
google.protobuf.Any extension = 8;  // v23 schema
```

Reading v22 contract on v23:
- Looks for extension at field 8
- Field 8 is empty ✗
- Field 7 has the data (but v23 ignores it)
- **Result: Missing extension or parse errors**

## Why Block Height Doesn't Work

The README states:
> "Some contracts created on v21 (and possibly before) still have corruption issues on v23"

This happens because:

1. **Contract created in v21** → Extension at field 8 ✓
2. **Chain upgrades to v22** (brief window)
3. **Contract admin migrates/updates contract** during v22
4. **Contract info rewritten with v22 schema** → Extension moves to field 7
5. **Chain upgrades to v23** → Now looking at field 8 again
6. **Contract breaks** because extension is at field 7, not field 8

The contract's **creation block** says v21, but its **current state** reflects v22 schema. Block height doesn't capture when ContractInfo was last written.

## The v24 Solution: Universal Detection

Instead of relying on version or block height:

```go
func DetectSchemaVersion(data []byte) SchemaVersion {
    field8Value := GetFieldValue(data, 8)
    if field8Value != nil && len(field8Value) > 0 {
        return SchemaBroken  // Extension is at field 8 (v20/v21/v23)
    }
    return SchemaSafe  // Extension is at field 7 (pre-v20 or v22)
}
```

**This works because:**
- XION never used IBCv2, so field 8 should always be empty in correct schemas
- Any data in field 8 = extension is in wrong place
- Works regardless of when contract was created or modified
- Simple, fast, and reliable

## Wire Type Reference

### Protobuf Wire Types (0-5 valid)

| Wire Type | Name | Used For |
|-----------|------|----------|
| 0 | Varint | int32, int64, uint32, uint64, bool, enum |
| 1 | 64-bit | fixed64, sfixed64, double |
| 2 | Length-delimited | string, bytes, embedded messages, packed repeated fields |
| 3 | Start group | Deprecated |
| 4 | End group | Deprecated |
| 5 | 32-bit | fixed32, sfixed32, float |

**Wire Type 2 (Length-delimited) Format:**
```
[varint length][length bytes of data]
```

This is why `string` and `google.protobuf.Any` are binary-compatible - they both use this format. The difference is in **how the data bytes are interpreted**:
- `string`: UTF-8 text
- `bytes`: Raw bytes
- `Any`: Nested protobuf message (type_url + value)

## Lessons Learned

1. **Wire type compatibility is dangerous** - Accidental compatibility can hide bugs
2. **Version mismatches can "work"** - Until you fix them and break existing data
3. **State migrations are critical** - Code fixes don't fix stored data
4. **Test upgrades thoroughly** - Even "correct" schemas can break "incorrect" data
5. **Simple detection wins** - Field presence is more reliable than version tracking

## References

- [Protobuf Encoding Guide](https://protobuf.dev/programming-guides/encoding/)
- [wasmd PR #2123](https://github.com/CosmWasm/wasmd/pull/2123) - Introduced bug
- [wasmd PR #2390](https://github.com/CosmWasm/wasmd/pull/2390) - Fixed schema
- [ContractInfo Proto Definition](https://github.com/CosmWasm/wasmd/blob/main/proto/cosmwasm/wasm/v1/types.proto)
