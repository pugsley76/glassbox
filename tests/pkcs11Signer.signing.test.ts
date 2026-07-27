// Copyright (c) 2026 dotandev
// SPDX-License-Identifier: MIT OR Apache-2.0

import { Pkcs11Signer } from "../src/audit/signing/pkcs11Signer";

jest.mock("pkcs11js");

type Pkcs11MockState = {
  loadedModule?: string;
  lastTemplate?: unknown;
  initializeCalls: number;
  openSessionCalls: number;
  loginCalls: number;
  findObjectsInitCalls: number;
  findObjectsCalls: number;
  signInitCalls: number;
  signCalls: number;
  closeSessionCalls: number;
  finalizeCalls: number;
};

type Pkcs11MockModule = {
  __getState: () => Pkcs11MockState;
  __resetState: () => void;
  __setFailure: (
    method: string,
    failure: {
      message: string;
      code?: number;
      method?: string;
      times?: number;
    },
  ) => void;
};

const pkcs11Mock = (): Pkcs11MockModule =>
  jest.requireMock("pkcs11js") as Pkcs11MockModule;

describe("Pkcs11Signer signing (mock module)", () => {
  const originalEnv = process.env;

  beforeEach(() => {
    jest.resetModules();
    process.env = { ...originalEnv };
    process.env.GLASSBOX_PKCS11_MODULE = "/tmp/mock-pkcs11.so";
    process.env.GLASSBOX_AUDIT_HSM_RATE_LIMIT_FILE =
      "/tmp/GLASSBOX_audit_hsm_calls_test.json";
    process.env.GLASSBOX_PKCS11_PIN = "1234";
    process.env.GLASSBOX_PKCS11_KEY_LABEL = "test-key";
    process.env.GLASSBOX_PKCS11_PUBLIC_KEY_PEM =
      "-----BEGIN PUBLIC KEY-----\nMCowBQYDK2VwAyEA7k7O6Y8bD4b2M6h0x9YfM8Kq2jv4m7Y8Ww8U6F6K3qk=\n-----END PUBLIC KEY-----";
    delete process.env.GLASSBOX_PKCS11_ALGORITHM;
    pkcs11Mock().__resetState();
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  test("signs payload using mocked PKCS#11 module", async () => {
    const signer = new Pkcs11Signer();
    const payload = Buffer.from("hello-world");

    const signature = await signer.sign(payload);

    expect(signature).toBeInstanceOf(Uint8Array);
    expect(signature.length).toBe(64);
  });

  test("uses configured module path and key label for lookup", async () => {
    const signer = new Pkcs11Signer();
    await signer.sign(Buffer.from("another-payload"));

    const state = pkcs11Mock().__getState();

    expect(state.loadedModule).toBe("/tmp/mock-pkcs11.so");
    expect(JSON.stringify(state.lastTemplate)).toContain("test-key");
  });

  test("reuses the same initialized session across consecutive sign calls", async () => {
    const signer = new Pkcs11Signer();

    await signer.sign(Buffer.from("one"));
    await signer.sign(Buffer.from("two"));

    const state = pkcs11Mock().__getState();

    expect(state.initializeCalls).toBe(1);
    expect(state.openSessionCalls).toBe(1);
    expect(state.loginCalls).toBe(1);
    expect(state.findObjectsInitCalls).toBe(1);
    expect(state.findObjectsCalls).toBe(1);
    expect(state.signInitCalls).toBe(2);
    expect(state.signCalls).toBe(2);
    expect(state.closeSessionCalls).toBe(0);
    expect(state.finalizeCalls).toBe(0);
  });

  test("reconnects once when the cached session becomes invalid", async () => {
    const signer = new Pkcs11Signer();

    await signer.sign(Buffer.from("first"));

    pkcs11Mock().__setFailure("C_SignInit", {
      message: "stale session",
      code: 0x000000b3,
      method: "C_SignInit",
      times: 1,
    });

    const signature = await signer.sign(Buffer.from("second"));
    expect(signature).toBeInstanceOf(Uint8Array);

    const state = pkcs11Mock().__getState();
    expect(state.initializeCalls).toBe(2);
    expect(state.openSessionCalls).toBe(2);
    expect(state.loginCalls).toBe(2);
    expect(state.findObjectsInitCalls).toBe(2);
    expect(state.findObjectsCalls).toBe(2);
    expect(state.signInitCalls).toBe(2);
    expect(state.signCalls).toBe(2);
    expect(state.closeSessionCalls).toBe(1);
    expect(state.finalizeCalls).toBe(1);
  });

  test("close releases the cached PKCS#11 session", async () => {
    const signer = new Pkcs11Signer();

    await signer.sign(Buffer.from("cleanup"));
    await signer.close();

    const state = pkcs11Mock().__getState();
    expect(state.closeSessionCalls).toBe(1);
    expect(state.finalizeCalls).toBe(1);
  });
});
