// Copyright (c) glassbox Authors.
// SPDX-License-Identifier: Apache-2.0

import { generateKeyPairSync, sign as nodeSign } from "crypto";

const { privateKey } = generateKeyPairSync("ed25519", {
  publicKeyEncoding: { type: "spki", format: "pem" },
  privateKeyEncoding: { type: "pkcs8", format: "pem" },
});

type MockFailure = {
  message: string;
  code?: number;
  method?: string;
  remaining: number;
};

type MockState = {
  loadedModule?: string;
  lastTemplate?: unknown;
  initialized: boolean;
  initializeCalls: number;
  openSessionCalls: number;
  loginCalls: number;
  findObjectsInitCalls: number;
  findObjectsCalls: number;
  findObjectsFinalCalls: number;
  signInitCalls: number;
  signCalls: number;
  closeSessionCalls: number;
  finalizeCalls: number;
  slots: number[];
  tokenLabels: Record<string, string>;
  keys: number[];
  failures: Record<string, MockFailure>;
};

const createDefaultState = (): MockState => ({
  loadedModule: undefined,
  lastTemplate: undefined,
  initialized: false,
  initializeCalls: 0,
  openSessionCalls: 0,
  loginCalls: 0,
  findObjectsInitCalls: 0,
  findObjectsCalls: 0,
  findObjectsFinalCalls: 0,
  signInitCalls: 0,
  signCalls: 0,
  closeSessionCalls: 0,
  finalizeCalls: 0,
  slots: [1],
  tokenLabels: { "1": "TestToken" },
  keys: [1],
  failures: {},
});

let state: MockState = createDefaultState();

const maybeThrow = (method: string): void => {
  const failure = state.failures[method];
  if (!failure || failure.remaining <= 0) {
    return;
  }

  failure.remaining -= 1;
  if (failure.remaining <= 0) {
    delete state.failures[method];
  }

  const err = new Error(failure.message) as Error & {
    code?: number;
    method?: string;
  };
  if (typeof failure.code === "number") {
    err.code = failure.code;
  }
  if (failure.method) {
    err.method = failure.method;
  }
  throw err;
};

export const CKF_SERIAL_SESSION = 0x00000004;
export const CKF_RW_SESSION = 0x00000002;
export const CKA_CLASS = 0x00000000;
export const CKO_PRIVATE_KEY = 0x00000003;
export const CKA_LABEL = 0x00000003;
export const CKA_ID = 0x00000102;
export const CKM_EDDSA = 0x00001050;
export const CKM_ECDSA = 0x00001041;

export class PKCS11 {
  load(modulePath: string): void {
    maybeThrow("load");
    state.loadedModule = modulePath;
  }

  C_Initialize(): void {
    maybeThrow("C_Initialize");
    state.initialized = true;
    state.initializeCalls += 1;
  }

  C_GetSlotList(_tokenPresent: boolean): number[] {
    maybeThrow("C_GetSlotList");
    return [...state.slots];
  }

  C_GetTokenInfo(slot: number): { label?: string } {
    maybeThrow("C_GetTokenInfo");
    return { label: state.tokenLabels[String(slot)] ?? `token-${slot}` };
  }

  C_OpenSession(_slot: number | Buffer, _flags: number): number {
    maybeThrow("C_OpenSession");
    state.openSessionCalls += 1;
    return state.openSessionCalls;
  }

  C_Login(_session: number, _userType: number, _pin?: string): void {
    maybeThrow("C_Login");
    state.loginCalls += 1;
  }

  C_FindObjectsInit(_session: number, template: unknown): void {
    maybeThrow("C_FindObjectsInit");
    state.findObjectsInitCalls += 1;
    state.lastTemplate = template;
  }

  C_FindObjects(_session: number, _count: number): number[] {
    maybeThrow("C_FindObjects");
    state.findObjectsCalls += 1;
    return [...state.keys];
  }

  C_FindObjectsFinal(_session: number): void {
    maybeThrow("C_FindObjectsFinal");
    state.findObjectsFinalCalls += 1;
  }

  C_SignInit(_session: number, _mechanism: unknown, _key: unknown): void {
    maybeThrow("C_SignInit");
    state.signInitCalls += 1;
  }

  C_Sign(_session: number, payload: Buffer): Buffer {
    maybeThrow("C_Sign");
    state.signCalls += 1;
    return nodeSign(null, payload, privateKey);
  }

  C_CloseSession(_session: number): void {
    maybeThrow("C_CloseSession");
    state.closeSessionCalls += 1;
  }

  C_Finalize(): void {
    maybeThrow("C_Finalize");
    state.initialized = false;
    state.finalizeCalls += 1;
  }
}

export const __getState = (): MockState => ({
  ...state,
  slots: [...state.slots],
  tokenLabels: { ...state.tokenLabels },
  keys: [...state.keys],
  failures: Object.fromEntries(
    Object.entries(state.failures).map(([name, failure]) => [
      name,
      { ...failure },
    ]),
  ),
});

export const __resetState = (): void => {
  state = createDefaultState();
};

export const __setFailure = (
  method: string,
  failure: { message: string; code?: number; method?: string; times?: number },
): void => {
  state.failures[method] = {
    message: failure.message,
    code: failure.code,
    method: failure.method,
    remaining: failure.times ?? 1,
  };
};

export const __setSlots = (slots: number[]): void => {
  state.slots = [...slots];
};

export const __setTokenLabels = (labels: Record<string, string>): void => {
  state.tokenLabels = { ...labels };
};

export const __setKeys = (keys: number[]): void => {
  state.keys = [...keys];
};

export default {
  PKCS11,
  CKF_SERIAL_SESSION,
  CKF_RW_SESSION,
  CKA_CLASS,
  CKO_PRIVATE_KEY,
  CKA_LABEL,
  CKA_ID,
  CKM_EDDSA,
  CKM_ECDSA,
  __getState,
  __resetState,
  __setFailure,
  __setSlots,
  __setTokenLabels,
  __setKeys,
};
