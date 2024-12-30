# @burnt-labs/xion-types

TypeScript definitions for Xion Protobuf files. This package provides TypeScript type definitions generated from the Protobuf files used in the Xion project, enabling developers to work with Xion-related data structures in a type-safe way.

## Table of Contents

- [@burnt-labs/xion-types](#burnt-labsxion-types)
  - [Table of Contents](#table-of-contents)
  - [Installation](#installation)
  - [Usage](#usage)
  - [Example](#example)
  - [Development](#development)
  - [License](#license)

---

## Installation

Install the **@burnt-labs/xion-types** package via npm or yarn:

```bash
# Using npm
npm install @burnt-labs/xion-types

# Using yarn
yarn add @burnt-labs/xion-types
```

---

## Usage

Once installed, you can import the type definitions in your TypeScript project. The types are generated from the Protobuf files used in the Xion project.

```typescript
import { MyProtobufType } from '@burnt-labs/xion-types/types/filename';

const myData: MyProtobufType = {
  field1: 'value',
  field2: 42
};
```

> **Note:** Replace `filename` with the appropriate file name where the type is defined.

---

## Example

Here is a full example of how you might use the **@burnt-labs/xion-types** package in a TypeScript project:

```typescript
import { MyProtobufType } from '@burnt-labs/xion-types/types/filename';

function processData(data: MyProtobufType): void {
  console.log(`Field 1: ${data.field1}`);
  console.log(`Field 2: ${data.field2}`);
}

const sampleData: MyProtobufType = {
  field1: 'Hello, Xion!',
  field2: 100
};

processData(sampleData);
```

> This simple example illustrates how you can work with the types generated from Xion's Protobuf definitions.

---

## Development

If you want to modify or regenerate the TypeScript definitions from Protobuf files, follow these steps:

1. **Clone the Repository**
   ```bash
   git clone https://github.com/burnt-labs/xion.git
   cd xion
   ```

2. **Install Dependencies**
   ```bash
   npm install
   ```

3. **Generate TypeScript Definitions**
   ```bash
   npx protoc --plugin=protoc-gen-ts=./node_modules/.bin/protoc-gen-ts \
     --ts_out=./generated \
     --proto_path=./proto \
     $(find ./proto -name '*.proto')
   ```

4. **Compile TypeScript Files**
   ```bash
   tsc --noEmit
   ```

> These steps will generate the TypeScript definitions from the Protobuf files located in the `proto` directory.

---

## License

This project is licensed under the MIT License. See the LICENSE file for details.

---

For more information, check out [Xion's GitHub repository](https://github.com/burnt-labs/xion).

