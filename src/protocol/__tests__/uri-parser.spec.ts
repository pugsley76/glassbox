// Copyright (c) 2026 dotandev
// SPDX-License-Identifier: MIT OR Apache-2.0

import { URIParser } from '../uri-parser';

describe('URIParser', () => {
    let parser: URIParser;

    beforeEach(() => {
        parser = new URIParser();
    });

    describe('parse', () => {
        it('should parse a valid URI with all optional parameters', () => {
            const uri = 'glassbox://debug/a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdeff?network=testnet&operation=0&source=dashboard';

            const result = parser.parse(uri);

            expect(result.transactionHash).toBe('a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdeff');
            expect(result.network).toBe('testnet');
            expect(result.operation).toBe(0);
            expect(result.source).toBe('dashboard');
        });

        it('should correctly parse a URI with only the mandatory parameters', () => {
            const uri = 'glassbox://debug/a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdeff?network=mainnet';

            const result = parser.parse(uri);

            expect(result.transactionHash).toBeDefined();
            expect(result.network).toBe('mainnet');
            expect(result.operation).toBeUndefined();
            expect(result.source).toBeUndefined();
        });

        it('should throw an error if the transaction hash is missing', () => {
            const uri = 'glassbox://debug/?network=testnet';

            expect(() => parser.parse(uri)).toThrow('Missing transaction hash');
        });

        it('should throw an error if the transaction hash format is invalid', () => {
            const uri = 'glassbox://debug/invalid_hash_format?network=testnet';

            expect(() => parser.parse(uri)).toThrow('Invalid transaction hash format');
        });

        it('should throw an error if the network parameter is missing', () => {
            const uri = 'glassbox://debug/a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdeff';

            expect(() => parser.parse(uri)).toThrow('Missing required parameter: network');
        });

        it('should throw an error for an unsupported network value', () => {
            const uri = 'glassbox://debug/a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdeff?network=devnet';

            expect(() => parser.parse(uri)).toThrow('Invalid network');
        });

        it('should throw an error for a negative operation index', () => {
            const uri = 'glassbox://debug/a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdeff?network=testnet&operation=-5';

            expect(() => parser.parse(uri)).toThrow('Invalid operation index');
        });

        it('should throw an error if the protocol is not glassbox://', () => {
            const uri = 'https://debug/hash?network=testnet';

            expect(() => parser.parse(uri)).toThrow('Invalid protocol');
        });
    });

    describe('validateSignature', () => {
        it('should correctly validate a valid HMAC-SHA256 signature', () => {
            const secret = 'super-secret-key';
            const parsedData = {
                transactionHash: 'a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdeff',
                network: 'testnet' as const,
                operation: 0,
                source: 'dashboard',
                signature: '',
                raw: '',
            };

            // Manually generate the signature for testing
            parsedData.signature = parser.generateSignature(parsedData, secret);

            // Validation should succeed
            expect(parser.validateSignature(parsedData, secret)).toBe(true);
        });

        it('should reject an incorrect or tampered signature', () => {
            const parsedData = {
                transactionHash: 'a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdeff',
                network: 'testnet' as const,
                operation: 0,
                source: 'dashboard',
                signature: 'c0ffee-invalid-signature-deadbeef',
                raw: '',
            };

            expect(parser.validateSignature(parsedData, 'some-secret')).toBe(false);
        });
    });

    describe('sanitize', () => {
        it('should strip away potentially malicious characters from the URI', () => {
            const maliciousUri = 'glassbox://debug/hash?network=testnet; rm -rf /';

            const sanitized = parser.sanitize(maliciousUri);

            expect(sanitized).not.toContain(';');
            expect(sanitized).not.toContain(' ');
        });

        it('should truncate URIs that exceed the maximum allowed length', () => {
            const overlyLongUri = 'glassbox://debug/' + 'b'.repeat(1000);

            const sanitized = parser.sanitize(overlyLongUri);

            expect(sanitized.length).toBeLessThanOrEqual(500);
        });
    });

    describe('new parameters for regression and mock harness', () => {
        it('should parse valid protocol-version, mock-ledger-manifest, and mock-ledger-entry', () => {
            const uri = 'glassbox://debug/a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdeff?network=testnet&protocol-version=22&mock-ledger-manifest=/path/to/manifest.json&mock-ledger-entry=key1:AAAA&mock-ledger-entry=key2:BBBB';

            const result = parser.parse(uri);

            expect(result.protocolVersion).toBe(22);
            expect(result.mockLedgerManifest).toBe('/path/to/manifest.json');
            expect(result.mockLedgerEntries).toEqual(['key1:AAAA', 'key2:BBBB']);
        });

        it('should throw error for invalid protocol-version', () => {
            const uri1 = 'glassbox://debug/a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdeff?network=testnet&protocol-version=-5';
            expect(() => parser.parse(uri1)).toThrow('Invalid protocol-version: must be a positive integer');

            const uri2 = 'glassbox://debug/a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdeff?network=testnet&protocol-version=99';
            expect(() => parser.parse(uri2)).toThrow('Invalid protocol-version: unsupported version');
        });

        it('should throw error for invalid mock-ledger-manifest containing null bytes', () => {
            const uri = 'glassbox://debug/a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdeff?network=testnet&mock-ledger-manifest=/path/to\0manifest.json';
            expect(() => parser.parse(uri)).toThrow('Invalid mock-ledger-manifest: cannot contain null bytes');
        });

        it('should throw error for invalid mock-ledger-entry format or values', () => {
            const uri1 = 'glassbox://debug/a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdeff?network=testnet&mock-ledger-entry=invalid';
            expect(() => parser.parse(uri1)).toThrow('Invalid mock-ledger-entry format');

            const uri2 = 'glassbox://debug/a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdeff?network=testnet&mock-ledger-entry=key:';
            expect(() => parser.parse(uri2)).toThrow('Invalid mock-ledger-entry value');

            const uri3 = 'glassbox://debug/a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdeff?network=testnet&mock-ledger-entry=key:invalid_base64_!@#$';
            expect(() => parser.parse(uri3)).toThrow('Invalid mock-ledger-entry value');
        });
    });
});
