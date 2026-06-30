// Copyright (c) 2026 dotandev
// SPDX-License-Identifier: MIT OR Apache-2.0

import {
    ProtocolRegistrar,
    RegistrationValidationError,
    formatRegistrationSummary,
} from '../protocol/register';
import * as fs from 'fs/promises';
import * as os from 'os';

jest.mock('fs/promises');
jest.mock('os', () => ({
    ...jest.requireActual('os'),
    platform: jest.fn(() => process.platform),
    homedir: jest.fn(() => (jest.requireActual('os') as typeof import('os')).homedir()),
}));
jest.mock('child_process', () => ({
    exec: jest.fn(),
}));
jest.mock('util', () => ({
    ...jest.requireActual('util'),
    promisify: jest.fn(() => jest.fn()),
}));

describe('formatRegistrationSummary', () => {
    it('should describe a healthy registration', () => {
        const summary = formatRegistrationSummary({
            platform: 'linux',
            scheme: 'glassbox',
            status: 'ok',
            registered: true,
            cliPath: '/usr/local/bin/glassbox',
            currentCliPath: '/usr/local/bin/glassbox',
            pathExists: true,
            isExecutable: true,
            issues: [],
            remediationSteps: [],
        });

        expect(summary).toContain('healthy');
        expect(summary).toContain('glassbox://');
    });

    it('should include issue counts for degraded registrations', () => {
        const summary = formatRegistrationSummary({
            platform: 'darwin',
            scheme: 'glassbox',
            status: 'degraded',
            registered: true,
            cliPath: '/tmp/glassbox',
            currentCliPath: '/usr/local/bin/glassbox',
            pathExists: true,
            isExecutable: false,
            issues: ['Binary is not executable'],
            remediationSteps: ['chmod +x /tmp/glassbox'],
        });

        expect(summary).toContain('degraded');
        expect(summary).toContain('1 issue');
    });

    it('should describe unsupported platforms', () => {
        const summary = formatRegistrationSummary({
            platform: 'freebsd',
            scheme: 'glassbox',
            status: 'unsupported',
            registered: false,
            cliPath: null,
            currentCliPath: '/usr/local/bin/glassbox',
            pathExists: false,
            isExecutable: false,
            issues: ['Protocol registration is not supported on freebsd'],
            remediationSteps: ['Use Linux, macOS, or Windows'],
        });

        expect(summary).toContain('not supported');
        expect(summary).toContain('freebsd');
    });
});

describe('ProtocolRegistrar.validateRegistrationPreconditions', () => {
    let registrar: ProtocolRegistrar;

    beforeEach(() => {
        jest.resetAllMocks();
        (os.platform as jest.Mock).mockReturnValue('linux');
        (os.homedir as jest.Mock).mockReturnValue('/home/user');
        registrar = new ProtocolRegistrar();
    });

    it('should reject unsupported platforms before touching OS state', async () => {
        (os.platform as jest.Mock).mockReturnValue('freebsd');

        await expect(registrar.validateRegistrationPreconditions()).rejects.toThrow(RegistrationValidationError);
        await expect(registrar.validateRegistrationPreconditions()).rejects.toThrow(/not supported on freebsd/);
    });

    it('should reject missing CLI executables', async () => {
        (fs.access as jest.Mock).mockRejectedValue(new Error('ENOENT'));

        await expect(registrar.validateRegistrationPreconditions()).rejects.toThrow(/not found at/);
    });

    it('should reject inaccessible home directories', async () => {
        (fs.access as jest.Mock)
            .mockResolvedValueOnce(undefined)
            .mockRejectedValueOnce(new Error('EACCES'));

        await expect(registrar.validateRegistrationPreconditions()).rejects.toThrow(/home directory is not accessible/);
    });

    it('should pass when platform, binary, and home directory are valid', async () => {
        (fs.access as jest.Mock).mockResolvedValue(undefined);

        await expect(registrar.validateRegistrationPreconditions()).resolves.toBeUndefined();
    });
});

describe('ProtocolRegistrar.diagnose', () => {
    let registrar: ProtocolRegistrar;

    beforeEach(() => {
        jest.resetAllMocks();
        (os.platform as jest.Mock).mockReturnValue(process.platform);
        (os.homedir as jest.Mock).mockReturnValue(require('os').homedir());
        registrar = new ProtocolRegistrar();
    });

    it('should report unsupported platforms with remediation steps', async () => {
        (os.platform as jest.Mock).mockReturnValue('freebsd');

        const result = await registrar.diagnose();

        expect(result.status).toBe('unsupported');
        expect(result.issues.length).toBeGreaterThan(0);
        expect(result.remediationSteps.length).toBeGreaterThan(0);
        expect(formatRegistrationSummary(result)).toContain('not supported');
    });

    it('should report not registered when protocol is unregistered', async () => {
        jest.spyOn(registrar, 'isRegistered').mockResolvedValue(false);

        const result = await registrar.diagnose();

        expect(result.registered).toBe(false);
        expect(result.status).toBe('not_registered');
        expect(result.cliPath).toBeNull();
        expect(result.issues).toContain('Protocol handler is not registered with the operating system');
        expect(result.remediationSteps.length).toBeGreaterThan(0);
    });

    it('should report unknown path when registered path cannot be resolved', async () => {
        jest.spyOn(registrar, 'isRegistered').mockResolvedValue(true);
        jest.spyOn(registrar, 'getRegisteredPath').mockResolvedValue(null);

        const result = await registrar.diagnose();

        expect(result.registered).toBe(true);
        expect(result.status).toBe('degraded');
        expect(result.cliPath).toBeNull();
        expect(result.issues.length).toBeGreaterThan(0);
    });

    it('should detect missing binary', async () => {
        jest.spyOn(registrar, 'isRegistered').mockResolvedValue(true);
        jest.spyOn(registrar, 'getRegisteredPath').mockResolvedValue('/usr/local/bin/Glassbox');
        (fs.access as jest.Mock).mockRejectedValue(new Error('ENOENT'));

        const result = await registrar.diagnose();

        expect(result.registered).toBe(true);
        expect(result.status).toBe('degraded');
        expect(result.cliPath).toBe('/usr/local/bin/Glassbox');
        expect(result.pathExists).toBe(false);
        expect(result.issues.some((issue) => issue.includes('Binary not found'))).toBe(true);
    });

    it('should detect non-executable binary on Unix', async () => {
        jest.spyOn(registrar, 'isRegistered').mockResolvedValue(true);
        jest.spyOn(registrar, 'getRegisteredPath').mockResolvedValue('/usr/local/bin/Glassbox');
        (os.platform as jest.Mock).mockReturnValue('linux');
        (fs.access as jest.Mock)
            .mockResolvedValueOnce(undefined)
            .mockRejectedValueOnce(new Error('EACCES'));

        const result = await registrar.diagnose();

        expect(result.registered).toBe(true);
        expect(result.status).toBe('degraded');
        expect(result.pathExists).toBe(true);
        expect(result.isExecutable).toBe(false);
        expect(result.remediationSteps.some((step) => step.includes('chmod +x'))).toBe(true);
    });

    it('should check file extension for executability on Windows', async () => {
        jest.spyOn(registrar, 'isRegistered').mockResolvedValue(true);
        jest.spyOn(registrar, 'getRegisteredPath').mockResolvedValue('C:\\Program Files\\Glassbox\\glassbox.exe');
        (os.platform as jest.Mock).mockReturnValue('win32');
        (fs.access as jest.Mock).mockResolvedValue(undefined);

        const result = await registrar.diagnose();

        expect(result.status).toBe('ok');
        expect(result.registered).toBe(true);
        expect(result.pathExists).toBe(true);
        expect(result.isExecutable).toBe(true);
    });

    it('should reject non-executable extension on Windows', async () => {
        jest.spyOn(registrar, 'isRegistered').mockResolvedValue(true);
        jest.spyOn(registrar, 'getRegisteredPath').mockResolvedValue('C:\\Glassbox\\Glassbox.txt');
        (os.platform as jest.Mock).mockReturnValue('win32');
        (fs.access as jest.Mock).mockResolvedValue(undefined);

        const result = await registrar.diagnose();

        expect(result.status).toBe('degraded');
        expect(result.registered).toBe(true);
        expect(result.pathExists).toBe(true);
        expect(result.isExecutable).toBe(false);
    });

    it('should confirm fully healthy registration', async () => {
        jest.spyOn(registrar, 'isRegistered').mockResolvedValue(true);
        jest.spyOn(registrar, 'getRegisteredPath').mockResolvedValue('/usr/local/bin/Glassbox');
        (os.platform as jest.Mock).mockReturnValue('linux');
        (fs.access as jest.Mock).mockResolvedValue(undefined);

        const result = await registrar.diagnose();

        expect(result.status).toBe('ok');
        expect(result.registered).toBe(true);
        expect(result.cliPath).toBe('/usr/local/bin/Glassbox');
        expect(result.pathExists).toBe(true);
        expect(result.isExecutable).toBe(true);
        expect(result.issues).toHaveLength(0);
    });
});
