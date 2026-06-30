// Copyright (c) 2026 dotandev
// SPDX-License-Identifier: MIT OR Apache-2.0

import { ProtocolRegistrar } from '../register';
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

describe('ProtocolRegistrar.diagnose', () => {
    let registrar: ProtocolRegistrar;

    beforeEach(() => {
        jest.resetAllMocks();
        (os.platform as jest.Mock).mockReturnValue(process.platform);
        (os.homedir as jest.Mock).mockReturnValue(require('os').homedir());
        registrar = new ProtocolRegistrar();
    });

    it('should report not registered when protocol is unregistered', async () => {
        jest.spyOn(registrar, 'isRegistered').mockResolvedValue(false);

        const result = await registrar.diagnose();

        expect(result.registered).toBe(false);
        expect(result.cliPath).toBeNull();
        expect(result.pathExists).toBe(false);
        expect(result.isExecutable).toBe(false);
    });

    it('should report unknown path when registered path cannot be resolved', async () => {
        jest.spyOn(registrar, 'isRegistered').mockResolvedValue(true);
        jest.spyOn(registrar, 'getRegisteredPath').mockResolvedValue(null);

        const result = await registrar.diagnose();

        expect(result.registered).toBe(true);
        expect(result.cliPath).toBeNull();
        expect(result.pathExists).toBe(false);
    });

    it('should detect missing binary', async () => {
        jest.spyOn(registrar, 'isRegistered').mockResolvedValue(true);
        jest.spyOn(registrar, 'getRegisteredPath').mockResolvedValue('/usr/local/bin/Glassbox');
        (fs.access as jest.Mock).mockRejectedValue(new Error('ENOENT'));

        const result = await registrar.diagnose();

        expect(result.registered).toBe(true);
        expect(result.cliPath).toBe('/usr/local/bin/Glassbox');
        expect(result.pathExists).toBe(false);
        expect(result.isExecutable).toBe(false);
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
        expect(result.pathExists).toBe(true);
        expect(result.isExecutable).toBe(false);
    });

    it('should check file extension for executability on Windows', async () => {
        jest.spyOn(registrar, 'isRegistered').mockResolvedValue(true);
        jest.spyOn(registrar, 'getRegisteredPath').mockResolvedValue('C:\\Program Files\\Glassbox\\glassbox.exe');
        (os.platform as jest.Mock).mockReturnValue('win32');
        (fs.access as jest.Mock).mockResolvedValue(undefined);

        const result = await registrar.diagnose();

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

        expect(result.registered).toBe(true);
        expect(result.cliPath).toBe('/usr/local/bin/Glassbox');
        expect(result.pathExists).toBe(true);
        expect(result.isExecutable).toBe(true);
    });
});

describe('ProtocolRegistrar.register', () => {
    let registrar: ProtocolRegistrar;

    beforeEach(() => {
        jest.resetAllMocks();
        (os.platform as jest.Mock).mockReturnValue(process.platform);
        (os.homedir as jest.Mock).mockReturnValue(require('os').homedir());
    });

    it('should throw if CLI path is not absolute', async () => {
        registrar = new ProtocolRegistrar('relative/path');
        await expect(registrar.register()).rejects.toThrow("Registration failed: CLI path must be absolute, got 'relative/path'.");
    });

    it('should throw if CLI executable is not found', async () => {
        registrar = new ProtocolRegistrar('/usr/local/bin/nonexistent');
        (fs.access as jest.Mock).mockRejectedValue(new Error('ENOENT'));
        await expect(registrar.register()).rejects.toThrow("Registration failed: CLI executable not found at '/usr/local/bin/nonexistent'.");
    });

    it('should throw if Windows executable has invalid extension', async () => {
        (os.platform as jest.Mock).mockReturnValue('win32');
        registrar = new ProtocolRegistrar('C:\\invalid.txt');
        (fs.access as jest.Mock).mockResolvedValue(undefined);
        await expect(registrar.register()).rejects.toThrow("Registration failed: Invalid executable extension on Windows for 'C:\\invalid.txt'.");
    });

    it('should throw if Unix executable is not executable', async () => {
        (os.platform as jest.Mock).mockReturnValue('linux');
        registrar = new ProtocolRegistrar('/usr/local/bin/script.sh');
        (fs.access as jest.Mock).mockResolvedValueOnce(undefined); // First access check for existence
        (fs.access as jest.Mock).mockRejectedValueOnce(new Error('EACCES')); // Second for executability
        await expect(registrar.register()).rejects.toThrow("Registration failed: CLI file is not executable at '/usr/local/bin/script.sh'.");
    });
    
    it('should throw with clear message if OS registration fails', async () => {
        (os.platform as jest.Mock).mockReturnValue('linux');
        registrar = new ProtocolRegistrar('/usr/local/bin/glassbox');
        (fs.access as jest.Mock).mockResolvedValue(undefined);
        
        // Mock the internal registerLinux method to fail
        jest.spyOn(registrar as any, 'registerLinux').mockRejectedValue(new Error('Permission denied'));
        
        await expect(registrar.register()).rejects.toThrow("Protocol registration failed on linux: Permission denied");
    });
});
