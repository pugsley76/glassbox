// Copyright (c) 2026 dotandev
// SPDX-License-Identifier: MIT OR Apache-2.0

import * as os from 'os';
import * as path from 'path';
import * as fs from 'fs/promises';
import { constants as fsConstants } from 'fs';
import { exec } from 'child_process';
import { promisify } from 'util';

const execAsync = promisify(exec);

const SUPPORTED_PLATFORMS = ['win32', 'darwin', 'linux'] as const;

export type ProtocolRegistrationStatus = 'ok' | 'degraded' | 'not_registered' | 'unsupported';

export interface ProtocolDiagnostics {
    platform: string;
    scheme: string;
    status: ProtocolRegistrationStatus;
    registered: boolean;
    cliPath: string | null;
    currentCliPath: string;
    pathExists: boolean;
    isExecutable: boolean;
    issues: string[];
    remediationSteps: string[];
}

/**
 * Raised when registration preconditions fail before OS artefacts are written.
 */
export class RegistrationValidationError extends Error {
    constructor(message: string) {
        super(message);
        this.name = 'RegistrationValidationError';
    }
}

/**
 * Returns a concise human-readable summary for reporting and status output.
 */
export function formatRegistrationSummary(diag: ProtocolDiagnostics): string {
    const issueCount = diag.issues.length;

    switch (diag.status) {
        case 'ok':
            return `Protocol handler ${diag.scheme}:// is registered and healthy on ${diag.platform}.`;
        case 'not_registered':
            if (issueCount === 0) {
                return `Protocol handler ${diag.scheme}:// is not registered on ${diag.platform}.`;
            }
            return `Protocol handler ${diag.scheme}:// is not registered on ${diag.platform} (${issueCount} issue(s)).`;
        case 'degraded':
            return `Protocol handler ${diag.scheme}:// is registered but degraded on ${diag.platform} (${issueCount} issue(s)).`;
        case 'unsupported':
            return `Protocol registration is not supported on ${diag.platform}.`;
        default:
            return `Protocol handler ${diag.scheme}:// status is ${diag.status} on ${diag.platform}.`;
    }
}

/**
 * ProtocolRegistrar handles the registration and unregistration of the
 * custom URI protocol handler (glassbox://) across different operating systems.
 */
export class ProtocolRegistrar {
    private readonly protocol = 'glassbox';
    private readonly cliPath: string;

    constructor() {
        this.cliPath = process.execPath;
    }

    /**
     * Validate inputs before writing OS registration artefacts.
     */
    async validateRegistrationPreconditions(): Promise<void> {
        const platform = os.platform();

        if (!SUPPORTED_PLATFORMS.includes(platform as typeof SUPPORTED_PLATFORMS[number])) {
            throw new RegistrationValidationError(
                `Protocol registration is not supported on ${platform}.\n` +
                '  Fix: use Linux, macOS, or Windows to register the glassbox:// handler.',
            );
        }

        if (!this.cliPath || this.cliPath.trim() === '') {
            throw new RegistrationValidationError(
                'Cannot register: CLI executable path is empty.\n' +
                '  Fix: invoke Glassbox from the installed binary, not a transient script.',
            );
        }

        if (this.cliPath.includes('\0')) {
            throw new RegistrationValidationError(
                'Cannot register: CLI executable path contains a null byte.\n' +
                '  Fix: reinstall Glassbox from a trusted source.',
            );
        }

        try {
            await fs.access(this.cliPath);
        } catch {
            throw new RegistrationValidationError(
                `Cannot register: CLI executable not found at ${this.cliPath}.\n` +
                '  Fix: reinstall Glassbox or verify the binary path is correct.',
            );
        }

        try {
            await fs.access(os.homedir());
        } catch {
            throw new RegistrationValidationError(
                `Cannot register: home directory is not accessible (${os.homedir()}).\n` +
                '  Fix: ensure your user home directory exists and is readable.',
            );
        }
    }

    /**
     * Register the glassbox:// protocol handler for the current OS
     */
    async register(): Promise<void> {
        await this.validateRegistrationPreconditions();
        const platform = os.platform();

        try {
            switch (platform) {
                case 'win32':
                    await this.registerWindows();
                    break;
                case 'darwin':
                    await this.registerMacOS();
                    break;
                case 'linux':
                    await this.registerLinux();
                    break;
                default:
                    throw new RegistrationValidationError(
                        `Protocol registration is not supported on ${platform}.\n` +
                        '  Fix: use Linux, macOS, or Windows to register the glassbox:// handler.',
                    );
            }

            console.log(` Protocol handler registered for ${this.protocol}://`);
        } catch (error) {
            if (error instanceof RegistrationValidationError) {
                throw error;
            }
            console.error('Failed to register protocol handler:', error);
            throw error;
        }
    }

    /**
     * Windows: Register via Registry
     */
    private async registerWindows(): Promise<void> {
        const regPath = `HKEY_CURRENT_USER\\Software\\Classes\\${this.protocol}`;

        const commands = [
            `reg add "${regPath}" /ve /d "URL:GLASSBOX Protocol" /f`,
            `reg add "${regPath}" /v "URL Protocol" /d "" /f`,
            `reg add "${regPath}\\shell\\open\\command" /ve /d "\\"${this.cliPath}\\" protocol-handler \\"%1\\"" /f`,
        ];

        for (const cmd of commands) {
            await execAsync(cmd);
        }
    }

    /**
     * macOS: Register via Info.plist
     */
    private async registerMacOS(): Promise<void> {
        const plistPath = path.join(
            os.homedir(),
            'Library',
            'LaunchAgents',
            `com.glassbox.protocol.plist`,
        );

        const plistContent = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.glassbox.protocol</string>
    <key>CFBundleURLTypes</key>
    <array>
        <dict>
            <key>CFBundleURLName</key>
            <string>GLASSBOX Protocol</string>
            <key>CFBundleURLSchemes</key>
            <array>
                <string>${this.protocol}</string>
            </array>
        </dict>
    </array>
    <key>ProgramArguments</key>
    <array>
        <string>${this.cliPath}</string>
        <string>protocol-handler</string>
    </array>
    <key>StandardInPath</key>
    <string>/dev/null</string>
    <key>StandardOutPath</key>
    <string>/tmp/glassbox-protocol.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/glassbox-protocol-error.log</string>
</dict>
</plist>`;

        await fs.writeFile(plistPath, plistContent, 'utf8');
        await execAsync(`launchctl load ${plistPath}`);
    }

    /**
     * Linux: Register via .desktop file
     */
    private async registerLinux(): Promise<void> {
        const desktopPath = path.join(
            os.homedir(),
            '.local',
            'share',
            'applications',
            'glassbox-protocol.desktop',
        );

        const desktopContent = `[Desktop Entry]
Version=1.0
Type=Application
Name=GLASSBOX Protocol Handler
Exec=${this.cliPath} protocol-handler %u
MimeType=x-scheme-handler/${this.protocol};
NoDisplay=true
Terminal=false`;

        await fs.mkdir(path.dirname(desktopPath), { recursive: true });
        await fs.writeFile(desktopPath, desktopContent, 'utf8');

        await execAsync(`xdg-mime default glassbox-protocol.desktop x-scheme-handler/${this.protocol}`);
        await execAsync('update-desktop-database ~/.local/share/applications/');
    }

    /**
     * Unregister protocol handler
     */
    async unregister(): Promise<void> {
        const platform = os.platform();

        if (!SUPPORTED_PLATFORMS.includes(platform as typeof SUPPORTED_PLATFORMS[number])) {
            throw new RegistrationValidationError(
                `Protocol unregistration is not supported on ${platform}.\n` +
                '  Fix: manually remove any custom protocol handler configuration on this platform.',
            );
        }

        try {
            switch (platform) {
                case 'win32':
                    await execAsync(`reg delete "HKEY_CURRENT_USER\\Software\\Classes\\${this.protocol}" /f`);
                    break;
                case 'darwin': {
                    const plistPath = path.join(os.homedir(), 'Library', 'LaunchAgents', 'com.glassbox.protocol.plist');
                    await execAsync(`launchctl unload ${plistPath}`);
                    await fs.unlink(plistPath);
                    break;
                }
                case 'linux': {
                    const desktopPath = path.join(os.homedir(), '.local', 'share', 'applications', 'glassbox-protocol.desktop');
                    await fs.unlink(desktopPath);
                    break;
                }
            }

            console.log(' Protocol handler unregistered');
        } catch (error) {
            console.error('Failed to unregister protocol handler:', error);
            throw error;
        }
    }

    /**
     * Check if protocol is already registered
     */
    async isRegistered(): Promise<boolean> {
        const platform = os.platform();

        if (!SUPPORTED_PLATFORMS.includes(platform as typeof SUPPORTED_PLATFORMS[number])) {
            return false;
        }

        try {
            switch (platform) {
                case 'win32': {
                    const { stdout } = await execAsync(`reg query "HKEY_CURRENT_USER\\Software\\Classes\\${this.protocol}"`);
                    return stdout.includes('URL Protocol');
                }
                case 'darwin': {
                    const plistPath = path.join(os.homedir(), 'Library', 'LaunchAgents', 'com.glassbox.protocol.plist');
                    await fs.access(plistPath);
                    return true;
                }
                case 'linux': {
                    const desktopPath = path.join(os.homedir(), '.local', 'share', 'applications', 'glassbox-protocol.desktop');
                    await fs.access(desktopPath);
                    return true;
                }
                default:
                    return false;
            }
        } catch {
            return false;
        }
    }

    async getRegisteredPath(): Promise<string | null> {
        const platform = os.platform();

        try {
            switch (platform) {
                case 'win32': {
                    const { stdout } = await execAsync(
                        `reg query "HKEY_CURRENT_USER\\Software\\Classes\\${this.protocol}\\shell\\open\\command" /ve`
                    );
                    const match = stdout.match(/"([^"]+)"\s+protocol-handler/);
                    return match ? match[1] : null;
                }
                case 'darwin': {
                    const plistPath = path.join(
                        os.homedir(), 'Library', 'LaunchAgents', 'com.glassbox.protocol.plist'
                    );
                    const content = await fs.readFile(plistPath, 'utf8');
                    const match = content.match(/<key>ProgramArguments<\/key>\s*<array>\s*<string>([^<]+)<\/string>/);
                    return match ? match[1] : null;
                }
                case 'linux': {
                    const desktopPath = path.join(
                        os.homedir(), '.local', 'share', 'applications', 'glassbox-protocol.desktop'
                    );
                    const content = await fs.readFile(desktopPath, 'utf8');
                    const match = content.match(/^Exec=(.+)\s+protocol-handler/m);
                    return match ? match[1] : null;
                }
                default:
                    return null;
            }
        } catch {
            return null;
        }
    }

    async diagnose(): Promise<ProtocolDiagnostics> {
        const platform = os.platform();
        const base: ProtocolDiagnostics = {
            platform,
            scheme: this.protocol,
            status: 'not_registered',
            registered: false,
            cliPath: null,
            currentCliPath: this.cliPath,
            pathExists: false,
            isExecutable: false,
            issues: [],
            remediationSteps: [],
        };

        if (!SUPPORTED_PLATFORMS.includes(platform as typeof SUPPORTED_PLATFORMS[number])) {
            base.status = 'unsupported';
            base.issues.push(`Protocol registration is not supported on ${platform}`);
            base.remediationSteps.push('Use Linux, macOS, or Windows to register the glassbox:// handler');
            return base;
        }

        const registered = await this.isRegistered();
        if (!registered) {
            base.issues.push('Protocol handler is not registered with the operating system');
            base.remediationSteps.push('Run "GLASSBOX Protocol:register" to enable dashboard integration');
            return base;
        }

        base.registered = true;
        const cliPath = await this.getRegisteredPath();
        base.cliPath = cliPath;

        if (!cliPath) {
            base.status = 'degraded';
            base.issues.push('Could not determine registered CLI path from OS artefacts');
            base.remediationSteps.push('Re-run "GLASSBOX Protocol:register" to refresh registration');
            return base;
        }

        try {
            await fs.access(cliPath);
            base.pathExists = true;
        } catch {
            base.status = 'degraded';
            base.issues.push(`Binary not found at ${cliPath}`);
            base.remediationSteps.push(`Ensure the Glassbox binary exists at ${cliPath}`);
            base.remediationSteps.push('Re-run "GLASSBOX Protocol:register" to update the registered path');
            return base;
        }

        try {
            if (platform === 'win32') {
                const ext = path.extname(cliPath).toLowerCase();
                base.isExecutable = ['.exe', '.cmd', '.bat', '.com'].includes(ext);
            } else {
                await fs.access(cliPath, fsConstants.X_OK);
                base.isExecutable = true;
            }
        } catch {
            base.isExecutable = false;
        }

        if (!base.isExecutable) {
            base.status = 'degraded';
            base.issues.push(`Binary at ${cliPath} is not executable`);
            if (platform === 'win32') {
                base.remediationSteps.push('Ensure the registered file is a runnable .exe, .cmd, .bat, or .com binary');
            } else {
                base.remediationSteps.push(`Restore execute permissions, for example: chmod +x ${cliPath}`);
            }
            base.remediationSteps.push('Re-run "GLASSBOX Protocol:register" if the binary moved or was replaced');
            return base;
        }

        base.status = 'ok';
        return base;
    }
}
