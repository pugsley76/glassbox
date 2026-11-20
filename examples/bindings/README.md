# Glassbox TypeScript Bindings

Generated TypeScript bindings for Soroban smart contracts, produced by
`glassbox generate-bindings`.

## Quick Start

```bash
glassbox generate-bindings contract.wasm --output ./generated --package my-contract
```

With debug metadata enabled:

```bash
glassbox generate-bindings contract.wasm \
  --output ./generated \
  --package my-contract \
  --debug-metadata \
  --wasm-source ./contract.wasm
```

## Environment Matrix Targets

Generate bindings specifically tailored for your execution runtime:

```bash
# 1. Node.js target (default): local subprocess simulation via child_process
glassbox generate-bindings contract.wasm --runtime node --output ./generated

# 2. Browser target: zero Node.js imports, fetch-based simulation, safe for Vite/Webpack
glassbox generate-bindings contract.wasm --runtime browser --output ./generated

# 3. Universal target: isomorphic environment detection for Node, Browser, and Electron
glassbox generate-bindings contract.wasm --runtime universal --output ./generated
```

## Generated Files

| File | Description |
|------|-------------|
| `types.ts` | TypeScript interfaces, enums, and typed error classes |
| `metadata.ts` | Per-function ABI descriptors with source-location hints |
| `client.ts` | Strongly-typed async client class |
| `Glassbox-integration.ts` | `ErstSimulator` wrapper for local simulation |
| `index.ts` | Barrel export |
| `package.json` | npm package manifest |
| `README.md` | Usage documentation |

## Type Safety

All Soroban types are mapped to idiomatic TypeScript:

| Soroban | TypeScript |
|---------|-----------|
| `Bool` | `boolean` |
| `U32`, `I32` | `number` |
| `U64`, `I64`, `U128`, `I128`, `U256`, `I256` | `bigint` |
| `String` | `string` |
| `Address`, `MuxedAddress` | `Address` (= `string`) |
| `Bytes` | `Bytes` (= `Uint8Array`) |
| `BytesN(N)` | `Uint8Array /* length: N */` |
| `Option<T>` | `T \| null` |
| `Vec<T>` | `Array<T>` |
| `Map<K,V>` | `Map<K, V>` |
| `Result<T,E>` | `Result<T, E>` |
| UDTs | Named interface / enum |

## Basic Usage

```typescript
import { MyContractClient } from './generated';

const client = new MyContractClient({
  contractId: 'CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQAHHAGCN4B2',
  network: 'testnet',
  enableSimulation: true,
});

// Call a contract method
const result = await client.transfer(
  sourceKeypair,
  'GDQP2KPQGKIHYJGXNUIYOMHARUARCA7DJT5FO2FFOOKY3B2WSQHG4W37', // from: Address
  'GBBD47IF6LWK7P7MDEVSCWR7DPUWV3NY3DTQEVFL4NAT4AQH3ZLLFLA5', // to: Address
  BigInt(1_000_000),                                             // amount: bigint
);
console.log('tx hash:', result.transactionHash);
```

## Simulation

```typescript
const simResult = await client.transfer(
  sourceKeypair,
  fromAddress,
  toAddress,
  BigInt(1_000_000),
  { simulate: true },
);
console.log('status:', simResult.simulation?.status);
console.log('CPU usage:', simResult.simulation?.budgetUsage?.cpuUsagePercent);
```

## Debug Metadata

When bindings are generated with `--debug-metadata`, each method accepts a
`withDebugMetadata` option that attaches ABI information to the call result.
This is useful for building debug tooling, logging, and IDE integrations.

```typescript
const result = await client.transfer(
  sourceKeypair,
  fromAddress,
  toAddress,
  BigInt(1_000_000),
  { withDebugMetadata: true, simulate: true },
);

// result.debugMetadata contains:
// {
//   name: 'transfer',
//   doc: 'Transfer tokens from one account to another.',
//   inputs: [
//     { name: 'from',   sorobanType: 'Address', tsType: 'Address' },
//     { name: 'to',     sorobanType: 'Address', tsType: 'Address' },
//     { name: 'amount', sorobanType: 'U128',    tsType: 'bigint'  },
//   ],
//   outputs: ['Void'],
//   source: { sourcePath: './contract.wasm', operationIndex: 0 },
// }
console.log('function:', result.debugMetadata?.name);
console.log('source:',   result.debugMetadata?.source);
```

You can also access the metadata registry directly:

```typescript
import { CONTRACT_METADATA } from './generated/metadata';

for (const [name, meta] of Object.entries(CONTRACT_METADATA)) {
  console.log(`${name}: ${meta.inputs.length} inputs → ${meta.outputs.join(', ')}`);
}
```

## Debugging with Glassbox

```typescript
import { ErstSimulator } from './generated/Glassbox-integration';

const simulator = new ErstSimulator({
  network: 'testnet',
  rpcUrl: 'https://soroban-testnet.stellar.org',
});

// Debug a failed transaction by hash
const debugResult = await simulator.debugTransaction(
  '5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab',
);
console.log('error:', debugResult.error);
console.log('events:', debugResult.events);
```

## Error Handling

Error enums generate both a TypeScript `enum` and a typed `Error` subclass:

```typescript
import { TokenError, TokenErrorError } from './generated/types';

try {
  await client.transfer(sourceKeypair, from, to, amount);
} catch (e) {
  if (e instanceof TokenErrorError) {
    switch (e.code) {
      case TokenError.InsufficientBalance:
        console.error('Not enough tokens');
        break;
      case TokenError.Unauthorized:
        console.error('Missing authorization');
        break;
    }
  }
}
```
